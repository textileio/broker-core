package store

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	blockstore "github.com/ipfs/go-ipfs-blockstore"
	util "github.com/ipfs/go-ipfs-util"
	cbor "github.com/ipfs/go-ipld-cbor"
	format "github.com/ipfs/go-ipld-format"
	"github.com/ipld/go-car"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multihash"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/logging"
	"github.com/textileio/broker-core/marketpeer"
	badger "github.com/textileio/go-ds-badger3"
	golog "github.com/textileio/go-log/v2"
)

func init() {
	if err := logging.SetLogLevels(map[string]golog.LogLevel{
		"bidbot/store": golog.LevelDebug,
	}); err != nil {
		panic(err)
	}

	DataURIFetchTimeout = time.Second * 5
}

func TestStore_ListBids(t *testing.T) {
	t.Parallel()
	s, _, _ := newStore(t)

	sk, _, err := crypto.GenerateEd25519Key(rand.Reader)
	require.NoError(t, err)
	auctioneerID, err := peer.IDFromPrivateKey(sk)
	require.NoError(t, err)

	limit := 100
	now := time.Now()
	ids := make([]broker.BidID, limit)
	for i := 0; i < limit; i++ {
		now = now.Add(time.Millisecond)
		id := broker.BidID(strings.ToLower(ulid.MustNew(ulid.Timestamp(now), rand.Reader).String()))
		aid := broker.AuctionID(strings.ToLower(ulid.MustNew(ulid.Now(), rand.Reader).String()))
		err := s.SaveBid(Bid{
			ID:               id,
			AuctionID:        aid,
			AuctioneerID:     auctioneerID,
			DataURI:          "https://foo.com/cid/bafyreifwqq6gi4fs6t2o4myssyxdy4nbhc4p4zkz3sesqmploueynskzfq",
			DealSize:         1024,
			DealDuration:     1000,
			AskPrice:         100,
			VerifiedAskPrice: 100,
			StartEpoch:       2000,
		})
		require.NoError(t, err)
		ids[i] = id
	}

	// Empty query, should return newest 10 records
	l, err := s.ListBids(Query{})
	require.NoError(t, err)
	assert.Len(t, l, 10)
	assert.Equal(t, ids[limit-1], l[0].ID)
	assert.Equal(t, ids[limit-10], l[9].ID)

	// Get next page, should return next 10 records
	offset := l[len(l)-1].ID
	l, err = s.ListBids(Query{Offset: string(offset)})
	require.NoError(t, err)
	assert.Len(t, l, 10)
	assert.Equal(t, ids[limit-11], l[0].ID)
	assert.Equal(t, ids[limit-20], l[9].ID)

	// Get previous page, should return the first page in reverse order
	offset = l[0].ID
	l, err = s.ListBids(Query{Offset: string(offset), Order: OrderAscending})
	require.NoError(t, err)
	assert.Len(t, l, 10)
	assert.Equal(t, ids[limit-10], l[0].ID)
	assert.Equal(t, ids[limit-1], l[9].ID)
}

func TestStore_SaveBid(t *testing.T) {
	t.Parallel()
	s, _, _ := newStore(t)

	sk, _, err := crypto.GenerateEd25519Key(rand.Reader)
	require.NoError(t, err)
	auctioneerID, err := peer.IDFromPrivateKey(sk)
	require.NoError(t, err)

	id := broker.BidID(strings.ToLower(ulid.MustNew(ulid.Now(), rand.Reader).String()))
	aid := broker.AuctionID(strings.ToLower(ulid.MustNew(ulid.Now(), rand.Reader).String()))
	dataURI := "https://foo.com/cid/bafyreifwqq6gi4fs6t2o4myssyxdy4nbhc4p4zkz3sesqmploueynskzfq"

	err = s.SaveBid(Bid{
		ID:               id,
		AuctionID:        aid,
		AuctioneerID:     auctioneerID,
		DataURI:          dataURI,
		DealSize:         1024,
		DealDuration:     1000,
		AskPrice:         100,
		VerifiedAskPrice: 100,
		StartEpoch:       2000,
	})
	require.NoError(t, err)

	got, err := s.GetBid(id)
	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
	assert.Equal(t, aid, got.AuctionID)
	assert.True(t, got.AuctioneerID.MatchesPrivateKey(sk))
	assert.Equal(t, dataURI, got.DataURI)
	assert.Equal(t, 1024, int(got.DealSize))
	assert.Equal(t, 1000, int(got.DealDuration))
	assert.Equal(t, BidStatusSubmitted, got.Status)
	assert.Equal(t, 100, int(got.AskPrice))
	assert.Equal(t, 100, int(got.VerifiedAskPrice))
	assert.Equal(t, 2000, int(got.StartEpoch))
	assert.False(t, got.FastRetrieval)
	assert.False(t, got.ProposalCid.Defined())
	assert.Equal(t, 0, int(got.DataURIFetchAttempts))
	assert.False(t, got.CreatedAt.IsZero())
	assert.False(t, got.UpdatedAt.IsZero())
	assert.Empty(t, got.ErrorCause)
}

func TestStore_StatusProgression(t *testing.T) {
	t.Parallel()
	s, _, bs := newStore(t)
	gwurl := newHTTPDataURIGateway(t)

	sk, _, err := crypto.GenerateEd25519Key(rand.Reader)
	require.NoError(t, err)
	auctioneerID, err := peer.IDFromPrivateKey(sk)
	require.NoError(t, err)

	t.Run("happy path", func(t *testing.T) {
		id := broker.BidID(strings.ToLower(ulid.MustNew(ulid.Now(), rand.Reader).String()))
		aid := broker.AuctionID(strings.ToLower(ulid.MustNew(ulid.Now(), rand.Reader).String()))
		dataCid := "bafyreifwqq6gi4fs6t2o4myssyxdy4nbhc4p4zkz3sesqmploueynskzfq"

		err = s.SaveBid(Bid{
			ID:               id,
			AuctionID:        aid,
			AuctioneerID:     auctioneerID,
			DataURI:          gwurl + "/cid/" + dataCid,
			DealSize:         1024,
			DealDuration:     1000,
			AskPrice:         100,
			VerifiedAskPrice: 100,
			StartEpoch:       2000,
		})
		require.NoError(t, err)

		got, err := s.GetBid(id)
		require.NoError(t, err)
		assert.Equal(t, BidStatusSubmitted, got.Status)

		err = s.SetAwaitingProposalCid(id)
		require.NoError(t, err)
		got, err = s.GetBid(id)
		require.NoError(t, err)
		assert.Equal(t, BidStatusAwaitingProposal, got.Status)

		err = s.SetProposalCid(id, cid.NewCidV1(cid.Raw, util.Hash([]byte("howdy"))))
		require.NoError(t, err)
		got, err = s.GetBid(id)
		require.NoError(t, err)
		assert.Equal(t, BidStatusFetchingData, got.Status)

		// Allow to finish
		time.Sleep(time.Second * 5)

		got, err = s.GetBid(id)
		require.NoError(t, err)
		assert.Equal(t, BidStatusFinalized, got.Status)
		assert.Empty(t, got.ErrorCause)

		// Check if car file was written to proposal data directory
		f, err := os.Open(filepath.Join(s.dealDataDirectory, dataCid))
		require.NoError(t, err)
		defer func() { _ = f.Close() }()
		h, err := car.LoadCar(bs, f)
		require.NoError(t, err)
		require.Len(t, h.Roots, 1)
		require.Equal(t, h.Roots[0].String(), dataCid)
	})

	t.Run("unreachable data uri", func(t *testing.T) {
		id := broker.BidID(strings.ToLower(ulid.MustNew(ulid.Now(), rand.Reader).String()))
		aid := broker.AuctionID(strings.ToLower(ulid.MustNew(ulid.Now(), rand.Reader).String()))
		dataCid := "bafyreifwqq6gi4fs6t2o4myssyxdy4nbhc4p4zkz3sesqmploueynskzfq"

		err = s.SaveBid(Bid{
			ID:               id,
			AuctionID:        aid,
			AuctioneerID:     auctioneerID,
			DataURI:          "https://foo.com/cid/" + dataCid,
			DealSize:         1024,
			DealDuration:     1000,
			AskPrice:         100,
			VerifiedAskPrice: 100,
			StartEpoch:       2000,
		})
		require.NoError(t, err)

		got, err := s.GetBid(id)
		require.NoError(t, err)
		assert.Equal(t, BidStatusSubmitted, got.Status)

		err = s.SetAwaitingProposalCid(id)
		require.NoError(t, err)
		got, err = s.GetBid(id)
		require.NoError(t, err)
		assert.Equal(t, BidStatusAwaitingProposal, got.Status)

		err = s.SetProposalCid(id, cid.NewCidV1(cid.Raw, util.Hash([]byte("howdy"))))
		require.NoError(t, err)
		got, err = s.GetBid(id)
		require.NoError(t, err)
		assert.Equal(t, BidStatusFetchingData, got.Status)

		// Allow to finish
		time.Sleep(time.Second * 12)

		got, err = s.GetBid(id)
		require.NoError(t, err)
		assert.Equal(t, BidStatusFinalized, got.Status)
		assert.NotEmpty(t, got.ErrorCause)
		assert.Equal(t, 2, int(got.DataURIFetchAttempts))
	})
}

func TestStore_GenerateCarData(t *testing.T) {
	t.Skip()
	_, dag, _ := newStore(t)

	node, err := cbor.WrapObject([]byte(uuid.NewString()), multihash.SHA2_256, -1)
	require.NoError(t, err)
	err = dag.Add(context.Background(), node)
	require.NoError(t, err)
	buff := &bytes.Buffer{}
	err = car.WriteCar(context.Background(), dag, []cid.Cid{node.Cid()}, buff)
	require.NoError(t, err)

	fmt.Printf("cid: %s\n", node.Cid())
	fmt.Printf("data: %s\n", base64.StdEncoding.EncodeToString(buff.Bytes()))
}

func newStore(t *testing.T) (*Store, format.DAGService, blockstore.Blockstore) {
	ds, err := badger.NewDatastore(t.TempDir(), &badger.DefaultOptions)
	require.NoError(t, err)
	sk, _, err := crypto.GenerateEd25519Key(rand.Reader)
	require.NoError(t, err)
	p, err := marketpeer.New(marketpeer.Config{
		RepoPath: t.TempDir(),
		PrivKey:  sk,
	})
	require.NoError(t, err)
	s, err := NewStore(ds, p.Host(), p.DAGService(), nil, t.TempDir(), 2)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, s.Close())
		require.NoError(t, ds.Close())
	})
	return s, p.DAGService(), p.BlockStore()
}

func newHTTPDataURIGateway(t *testing.T) (url string) {
	mux := http.NewServeMux()

	mux.HandleFunc("/cid/", func(w http.ResponseWriter, r *http.Request) {
		var (
			id   = path.Base(r.URL.Path)
			data string
		)
		switch id {
		case "bafyreifwqq6gi4fs6t2o4myssyxdy4nbhc4p4zkz3sesqmploueynskzfq":
			data = "OqJlcm9vdHOB2CpYJQABcRIgtoQ8ZHCy9PTuMxKWLjxxoTi4/mVZ3IkoMet1CYbJWSxndmVyc2lvbgFKAXESILaEPGRwsv" +
				"T07jMSli48caE4uP5lWdyJKDHrdQmGyVksWCQ4NzY4MGFkNC1mODIzLTQ0ZTktOWNlZi03OTU2NDlhZDYwMzE="
		default:
			t.Fatal("invalid request")
		}
		decoded, err := base64.StdEncoding.DecodeString(data)
		require.NoError(t, err)
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", "attachment; filename="+strconv.Quote(id)+".car")
		_, err = w.Write(decoded)
		require.NoError(t, err)
	})

	ts := httptest.NewServer(mux)
	t.Cleanup(func() {
		ts.Close()
	})
	return ts.URL
}
