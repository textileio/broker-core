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
	"github.com/textileio/broker-core/auctioneer"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/tests"
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
	StartDelay = 0
}

func TestQueue_newID(t *testing.T) {
	t.Parallel()
	q := newQueue(t)

	// Ensure monotonic
	var last auction.ID
	for i := 0; i < 10000; i++ {
		id, err := q.newID(time.Now())
		require.NoError(t, err)

		if i > 0 {
			assert.Greater(t, id, last)
		}
		last = id
	}
}

func TestQueue_CreateAuction(t *testing.T) {
	t.Parallel()
	q := newQueue(t)

	ctx := context.Background()
	id := auction.ID("ID-1")
	err := q.CreateAuction(ctx, auctioneer.Auction{
		ID:              id,
		BatchID:         broker.BatchID(strings.ToLower(ulid.MustNew(ulid.Now(), rand.Reader).String())),
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

	got, err := q.GetAuction(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
	assert.NotEmpty(t, got.BatchID)
	assert.Equal(t, broker.AuctionStatusFinalized, got.Status)
	assert.Equal(t, "https://foo.com/cid/123", got.Sources.CARURL.URL.String())
	assert.Equal(t, 1024, int(got.DealSize))
	assert.Equal(t, 1, int(got.DealDuration))
	assert.Equal(t, 1, int(got.DealReplication))
	assert.False(t, got.DealVerified)
	assert.Len(t, got.Bids, 3)
	assert.Len(t, got.WinningBids, 1)
	assert.Empty(t, got.ErrorCause)
	assert.False(t, got.StartedAt.IsZero())
	assert.False(t, got.UpdatedAt.IsZero())
}

func TestQueue_GetBidderID(t *testing.T) {
	t.Parallel()
	q := newQueue(t)

	ctx := context.Background()
	id := auction.ID("ID-1")
	err := q.CreateAuction(ctx, auctioneer.Auction{
		ID:              id,
		BatchID:         broker.BatchID(strings.ToLower(ulid.MustNew(ulid.Now(), rand.Reader).String())),
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

	got, err := q.GetAuction(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, broker.AuctionStatusFinalized, got.Status)
	assert.Empty(t, got.ErrorCause)
	assert.Len(t, got.Bids, 3)
	assert.Len(t, got.WinningBids, int(got.DealReplication))

	pcid := cid.NewCidV1(cid.Raw, util.Hash([]byte("proposal")))

	// Test auction not found
	_, err = q.GetBidderID(ctx, "foo", "bar")
	require.ErrorIs(t, err, ErrAuctionNotFound)

	// Test bid not found
	_, err = q.GetBidderID(ctx, got.ID, "foo")
	require.ErrorIs(t, err, ErrBidNotFound)

	for id := range got.WinningBids {
		bidderID, err := q.GetBidderID(ctx, got.ID, id)
		require.NoError(t, err)
		assert.NotEmpty(t, bidderID)

		// make sure proposal cid is only published once
		err = q.SetProposalCidDelivered(ctx, got.ID, id, pcid)
		require.NoError(t, err)
		bidderID, err = q.GetBidderID(ctx, got.ID, id)
		require.ErrorIs(t, err, ErrProposalDelivered)
		assert.Empty(t, bidderID)
	}

	got, err = q.GetAuction(ctx, id)
	require.NoError(t, err)
	for _, wb := range got.WinningBids {
		assert.True(t, wb.ProposalCid.Defined())
		assert.Empty(t, wb.ErrorCause)
	}
}

func newQueue(t *testing.T) *Queue {
	u, err := tests.PostgresURL()
	require.NoError(t, err)
	q, err := NewQueue(u, runner, finalizer)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, q.Close())
	})
	return q
}

func runner(
	_ context.Context,
	a auctioneer.Auction,
	addBid func(bid auctioneer.Bid) (auction.BidID, error),
) (map[auction.BidID]auctioneer.WinningBid, error) {
	time.Sleep(time.Millisecond * 100)

	result := make(map[auction.BidID]auctioneer.WinningBid)
	receivedBids := []auctioneer.Bid{
		{
			StorageProviderID: "miner1",
			WalletAddrSig:     []byte("sig1"),
			BidderID:          randomPeerID(),
			AskPrice:          100,
			VerifiedAskPrice:  100,
			StartEpoch:        1,
			FastRetrieval:     true,
			ReceivedAt:        time.Now(),
		},
		{
			StorageProviderID: "miner2",
			WalletAddrSig:     []byte("sig2"),
			BidderID:          randomPeerID(),
			AskPrice:          200,
			VerifiedAskPrice:  200,
			StartEpoch:        1,
			FastRetrieval:     true,
			ReceivedAt:        time.Now(),
		},
		{
			StorageProviderID: "miner3",
			WalletAddrSig:     []byte("sig3"),
			BidderID:          randomPeerID(),
			AskPrice:          300,
			VerifiedAskPrice:  300,
			StartEpoch:        1,
			FastRetrieval:     true,
			ReceivedAt:        time.Now(),
		},
	}

	// Select winners
	for i, bid := range receivedBids {
		id, err := addBid(bid)
		if err != nil {
			return nil, err
		}
		if i < int(a.DealReplication) {
			result[id] = auctioneer.WinningBid{
				BidderID: receivedBids[0].BidderID,
			}
		}
	}
	return result, nil
}

func finalizer(_ context.Context, _ *auctioneer.Auction) error {
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
