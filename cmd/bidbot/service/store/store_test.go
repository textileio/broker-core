package store

import (
	"context"
	"crypto/rand"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	blockstore "github.com/ipfs/go-ipfs-blockstore"
	cbor "github.com/ipfs/go-ipld-cbor"
	format "github.com/ipfs/go-ipld-format"
	golog "github.com/ipfs/go-log/v2"
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
)

func init() {
	if err := logging.SetLogLevels(map[string]golog.LogLevel{
		"bidbot/store": golog.LevelDebug,
	}); err != nil {
		panic(err)
	}

	ProposalFetchTimeout = time.Second * 5
}

func TestStore_ListBids(t *testing.T) {
	t.Parallel()
	s, _, _ := newStore(t)

	sk, _, err := crypto.GenerateEd25519Key(rand.Reader)
	require.NoError(t, err)
	auctioneerID, err := peer.IDFromPrivateKey(sk)
	require.NoError(t, err)

	t.Run("pagination", func(t *testing.T) {
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
	})
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
		err = s.SaveBid(Bid{
			ID:               id,
			AuctionID:        aid,
			AuctioneerID:     auctioneerID,
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

		// Create node and add to ipfs
		pnode, err := cbor.WrapObject([]byte("foo"), multihash.SHA2_256, -1)
		require.NoError(t, err)
		err = dag.Add(context.Background(), pnode)
		require.NoError(t, err)

		err = s.SetProposalCid(id, pnode.Cid())
		require.NoError(t, err)
		got, err = s.GetBid(id)
		require.NoError(t, err)
		assert.Equal(t, BidStatusFetchingProposal, got.Status)

		// Allow to finish
		time.Sleep(time.Second * 5)

		got, err = s.GetBid(id)
		require.NoError(t, err)
		assert.Equal(t, BidStatusFinalized, got.Status)
		assert.Empty(t, got.ErrorCause)

		// Check if car file was written to proposal data directory
		f, err := os.Open(filepath.Join(s.proposalDataDirectory, pnode.Cid().String()))
		require.NoError(t, err)
		defer func() { _ = f.Close() }()
		h, err := car.LoadCar(bs, f)
		require.NoError(t, err)
		require.Len(t, h.Roots, 1)
		require.True(t, h.Roots[0].Equals(pnode.Cid()))
	})

	t.Run("unreachable proposal cid", func(t *testing.T) {
		id := broker.BidID(strings.ToLower(ulid.MustNew(ulid.Now(), rand.Reader).String()))
		aid := broker.AuctionID(strings.ToLower(ulid.MustNew(ulid.Now(), rand.Reader).String()))
		err = s.SaveBid(Bid{
			ID:               id,
			AuctionID:        aid,
			AuctioneerID:     auctioneerID,
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

		// Create node but don't add to ipfs
		pnode, err := cbor.WrapObject([]byte("bar"), multihash.SHA2_256, -1)
		require.NoError(t, err)

		err = s.SetProposalCid(id, pnode.Cid())
		require.NoError(t, err)
		got, err = s.GetBid(id)
		require.NoError(t, err)
		assert.Equal(t, BidStatusFetchingProposal, got.Status)

		// Allow to finish
		time.Sleep(time.Second * 12)

		got, err = s.GetBid(id)
		require.NoError(t, err)
		assert.Equal(t, BidStatusFinalized, got.Status)
		assert.NotEmpty(t, got.ErrorCause)
		assert.Equal(t, 2, int(got.ProposalCidFetchAttempts))
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
