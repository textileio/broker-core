package store

import (
	"context"
	"crypto/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

	DataCidFetchTimeout = time.Second * 5
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
			DataCid:          cid.NewCidV1(cid.Raw, util.Hash([]byte("howdy"))),
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
	dataCid := cid.NewCidV1(cid.Raw, util.Hash([]byte("data")))
	err = s.SaveBid(Bid{
		ID:               id,
		AuctionID:        aid,
		AuctioneerID:     auctioneerID,
		DataCid:          dataCid,
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
	assert.True(t, got.DataCid.Equals(dataCid))
	assert.Equal(t, 1024, int(got.DealSize))
	assert.Equal(t, 1000, int(got.DealDuration))
	assert.Equal(t, BidStatusSubmitted, got.Status)
	assert.Equal(t, 100, int(got.AskPrice))
	assert.Equal(t, 100, int(got.VerifiedAskPrice))
	assert.Equal(t, 2000, int(got.StartEpoch))
	assert.False(t, got.FastRetrieval)
	assert.False(t, got.ProposalCid.Defined())
	assert.Equal(t, 0, int(got.DataCidFetchAttempts))
	assert.False(t, got.CreatedAt.IsZero())
	assert.False(t, got.UpdatedAt.IsZero())
	assert.Empty(t, got.ErrorCause)
}

func TestStore_StatusProgression(t *testing.T) {
	t.Parallel()
	s, dag, bs := newStore(t)

	sk, _, err := crypto.GenerateEd25519Key(rand.Reader)
	require.NoError(t, err)
	auctioneerID, err := peer.IDFromPrivateKey(sk)
	require.NoError(t, err)

	t.Run("happy path", func(t *testing.T) {
		id := broker.BidID(strings.ToLower(ulid.MustNew(ulid.Now(), rand.Reader).String()))
		aid := broker.AuctionID(strings.ToLower(ulid.MustNew(ulid.Now(), rand.Reader).String()))

		// Create data node and add to ipfs
		dnode, err := cbor.WrapObject([]byte("foo"), multihash.SHA2_256, -1)
		require.NoError(t, err)
		err = dag.Add(context.Background(), dnode)
		require.NoError(t, err)

		err = s.SaveBid(Bid{
			ID:               id,
			AuctionID:        aid,
			AuctioneerID:     auctioneerID,
			DataCid:          dnode.Cid(),
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
		f, err := os.Open(filepath.Join(s.dealDataDirectory, dnode.Cid().String()))
		require.NoError(t, err)
		defer func() { _ = f.Close() }()
		h, err := car.LoadCar(bs, f)
		require.NoError(t, err)
		require.Len(t, h.Roots, 1)
		require.True(t, h.Roots[0].Equals(dnode.Cid()))
	})

	t.Run("unreachable data cid", func(t *testing.T) {
		id := broker.BidID(strings.ToLower(ulid.MustNew(ulid.Now(), rand.Reader).String()))
		aid := broker.AuctionID(strings.ToLower(ulid.MustNew(ulid.Now(), rand.Reader).String()))
		err = s.SaveBid(Bid{
			ID:               id,
			AuctionID:        aid,
			AuctioneerID:     auctioneerID,
			DataCid:          cid.NewCidV1(cid.Raw, util.Hash([]byte("unreachable"))),
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
		assert.Equal(t, 2, int(got.DataCidFetchAttempts))
	})
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
	s, err := NewStore(ds, p.DAGService(), t.TempDir(), 2)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, s.Close())
		require.NoError(t, ds.Close())
	})
	return s, p.DAGService(), p.BlockStore()
}
