package queue

import (
	"context"
	"crypto/rand"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/ipfs/go-cid"
	util "github.com/ipfs/go-ipfs-util"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/textileio/bidbot/lib/auction"
	"github.com/textileio/bidbot/lib/logging"
	"github.com/textileio/broker-core/broker"
	badger "github.com/textileio/go-ds-badger3"
	golog "github.com/textileio/go-log/v2"
)

var testCid cid.Cid
var testSources auction.Sources

func init() {
	if err := logging.SetLogLevels(map[string]golog.LogLevel{
		"auctioneer/queue": golog.LevelDebug,
	}); err != nil {
		panic(err)
	}

	testCid, _ = cid.Parse("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jH1")
	carURL, _ := url.Parse("https://foo.com/cid/123")
	testSources = auction.Sources{CARURL: &auction.CARURL{URL: *carURL}}
}

func TestQueue_newID(t *testing.T) {
	t.Parallel()
	q := newQueue(t)

	// Ensure monotonic
	var last auction.AuctionID
	for i := 0; i < 10000; i++ {
		id, err := q.newID(time.Now())
		require.NoError(t, err)

		if i > 0 {
			assert.Greater(t, id, last)
		}
		last = id
	}
}

func TestQueue_ListAuctions(t *testing.T) {
	t.Parallel()
	q := newQueue(t)

	limit := 100
	now := time.Now()

	ids := make([]auction.AuctionID, limit)
	for i := 0; i < limit; i++ {
		now = now.Add(time.Millisecond)
		id, err := q.CreateAuction(auction.Auction{
			StorageDealID:   auction.StorageDealID(strings.ToLower(ulid.MustNew(ulid.Now(), rand.Reader).String())),
			PayloadCid:      testCid,
			DealSize:        1024,
			DealDuration:    1,
			DealReplication: 1,
			Sources:         testSources,
			Duration:        time.Second,
		})
		require.NoError(t, err)
		ids[i] = id
	}

	// Allow all to finish
	time.Sleep(time.Second)

	// Empty query, should return newest 10 records
	l, err := q.ListAuctions(Query{})
	require.NoError(t, err)
	assert.Len(t, l, 10)
	assert.Equal(t, ids[limit-1], l[0].ID)
	assert.Equal(t, ids[limit-10], l[9].ID)

	// Get next page, should return next 10 records
	offset := l[len(l)-1].ID
	l, err = q.ListAuctions(Query{Offset: string(offset)})
	require.NoError(t, err)
	assert.Len(t, l, 10)
	assert.Equal(t, ids[limit-11], l[0].ID)
	assert.Equal(t, ids[limit-20], l[9].ID)

	// Get previous page, should return the first page in reverse order
	offset = l[0].ID
	l, err = q.ListAuctions(Query{Offset: string(offset), Order: OrderAscending})
	require.NoError(t, err)
	assert.Len(t, l, 10)
	assert.Equal(t, ids[limit-10], l[0].ID)
	assert.Equal(t, ids[limit-1], l[9].ID)
}

func TestQueue_CreateAuction(t *testing.T) {
	t.Parallel()
	q := newQueue(t)

	id, err := q.CreateAuction(auction.Auction{
		StorageDealID:   auction.StorageDealID(strings.ToLower(ulid.MustNew(ulid.Now(), rand.Reader).String())),
		PayloadCid:      testCid,
		DealSize:        1024,
		DealDuration:    1,
		DealReplication: 1,
		Sources:         testSources,
		Duration:        time.Millisecond,
	})
	require.NoError(t, err)

	// Allow to finish
	time.Sleep(time.Second)

	got, err := q.GetAuction(id)
	require.NoError(t, err)
	assert.NotEmpty(t, got.ID)
	assert.NotEmpty(t, got.StorageDealID)
	assert.Equal(t, auction.AuctionStatusFinalized, got.Status)
	assert.Equal(t, "https://foo.com/cid/123", got.Sources.CARURL.URL.String())
	assert.Equal(t, 1024, int(got.DealSize))
	assert.Equal(t, 1, int(got.DealDuration))
	assert.Equal(t, 1, int(got.DealReplication))
	assert.False(t, got.DealVerified)
	assert.Len(t, got.Bids, 3)
	assert.Len(t, got.WinningBids, 1)
	assert.Equal(t, 1, int(got.Attempts))
	assert.Empty(t, got.ErrorCause)
	assert.False(t, got.StartedAt.IsZero())
	assert.False(t, got.UpdatedAt.IsZero())
}

func TestQueue_SetWinningBidProposalCid(t *testing.T) {
	t.Parallel()
	q := newQueue(t)

	id, err := q.CreateAuction(auction.Auction{
		StorageDealID:   auction.StorageDealID(strings.ToLower(ulid.MustNew(ulid.Now(), rand.Reader).String())),
		PayloadCid:      testCid,
		Sources:         testSources,
		DealSize:        1024,
		DealDuration:    1,
		DealReplication: 2,
		Duration:        time.Millisecond,
	})
	require.NoError(t, err)

	// Allow to finish
	time.Sleep(time.Second)

	got, err := q.GetAuction(id)
	require.NoError(t, err)
	assert.Equal(t, auction.AuctionStatusFinalized, got.Status)
	assert.Empty(t, got.ErrorCause)
	assert.Len(t, got.Bids, 3)
	assert.Len(t, got.WinningBids, int(got.DealReplication))

	pcid := cid.NewCidV1(cid.Raw, util.Hash([]byte("proposal")))

	// Test auction not found
	err = q.SetWinningBidProposalCid("foo", "bar", pcid)
	require.ErrorIs(t, err, ErrAuctionNotFound)

	// Test bid not found
	err = q.SetWinningBidProposalCid(got.ID, "foo", pcid)
	require.ErrorIs(t, err, ErrBidNotFound)

	for id := range got.WinningBids {
		// Test bad proposal cid
		err = q.SetWinningBidProposalCid(got.ID, id, cid.Undef)
		require.Error(t, err)

		err = q.SetWinningBidProposalCid(got.ID, id, pcid)
		require.NoError(t, err)
	}

	// Allow to finish
	time.Sleep(time.Second)

	_, err = q.GetAuction(id)
	require.ErrorIs(t, err, ErrAuctionNotFound)
}

func newQueue(t *testing.T) *Queue {
	s, err := badger.NewDatastore(t.TempDir(), &badger.DefaultOptions)
	require.NoError(t, err)
	q := NewQueue(s, runner, finalizer, 2)
	t.Cleanup(func() {
		require.NoError(t, q.Close())
		require.NoError(t, s.Close())
	})
	return q
}

func runner(
	_ context.Context,
	a auction.Auction,
	addBid func(bid auction.Bid) (auction.BidID, error),
) (map[auction.BidID]auction.WinningBid, error) {
	time.Sleep(time.Millisecond * 100)

	result := make(map[auction.BidID]auction.WinningBid)
	receivedBids := []auction.Bid{
		{
			MinerAddr:        "miner1",
			WalletAddrSig:    []byte("sig1"),
			BidderID:         randomPeerID(),
			AskPrice:         100,
			VerifiedAskPrice: 100,
			StartEpoch:       1,
			FastRetrieval:    true,
			ReceivedAt:       time.Now(),
		},
		{
			MinerAddr:        "miner2",
			WalletAddrSig:    []byte("sig2"),
			BidderID:         randomPeerID(),
			AskPrice:         200,
			VerifiedAskPrice: 200,
			StartEpoch:       1,
			FastRetrieval:    true,
			ReceivedAt:       time.Now(),
		},
		{
			MinerAddr:        "miner3",
			WalletAddrSig:    []byte("sig3"),
			BidderID:         randomPeerID(),
			AskPrice:         300,
			VerifiedAskPrice: 300,
			StartEpoch:       1,
			FastRetrieval:    true,
			ReceivedAt:       time.Now(),
		},
	}

	if len(a.WinningBids) == 0 {
		// First run... select winners
		for i, bid := range receivedBids {
			id, err := addBid(bid)
			if err != nil {
				return nil, err
			}
			if i < int(a.DealReplication) {
				result[id] = auction.WinningBid{
					BidderID:     receivedBids[0].BidderID,
					Acknowledged: true,
				}
			}
		}
	} else {
		// Second run from setting proposal cid
		for id, bid := range a.WinningBids {
			if bid.ProposalCid.Defined() {
				bid.ProposalCidAcknowledged = true
				result[id] = bid
			}
		}
	}
	return result, nil
}

func finalizer(_ context.Context, _ broker.ClosedAuction) error {
	return nil
}

func randomPeerID() peer.ID {
	sk, _, err := crypto.GenerateEd25519Key(rand.Reader)
	if err != nil {
		panic(err)
	}
	id, err := peer.IDFromPrivateKey(sk)
	if err != nil {
		panic(err)
	}
	return id
}
