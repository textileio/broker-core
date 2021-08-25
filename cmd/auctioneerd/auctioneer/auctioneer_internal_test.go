package auctioneer

import (
	"container/heap"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	core "github.com/textileio/bidbot/lib/auction"
	"github.com/textileio/broker-core/auctioneer"
)

func TestNewID(t *testing.T) {
	t.Parallel()
	a := &Auctioneer{}

	// Ensure monotonic
	var last core.BidID
	for i := 0; i < 10000; i++ {
		id, err := a.newID()
		require.NoError(t, err)

		if i > 0 {
			assert.Greater(t, id, last)
		}
		last = id
	}
}

func TestAcceptBid(t *testing.T) {
	require.True(t, acceptBid(&auctioneer.Auction{}, &auctioneer.Bid{}))
	require.True(t, acceptBid(&auctioneer.Auction{}, &auctioneer.Bid{StartEpoch: 90000}))
	require.True(t, acceptBid(&auctioneer.Auction{FilEpochDeadline: 89999}, &auctioneer.Bid{StartEpoch: 89999}))
	require.False(t, acceptBid(&auctioneer.Auction{FilEpochDeadline: 89999}, &auctioneer.Bid{StartEpoch: 90000}))
	// this shouldn't happen because zero StartEpoch won't pass core.Bid
	// validation, but just in case.
	require.False(t, acceptBid(&auctioneer.Auction{FilEpochDeadline: 89999}, &auctioneer.Bid{}))

	require.True(t, acceptBid(&auctioneer.Auction{}, &auctioneer.Bid{StorageProviderID: "f0001"}))
	require.True(t, acceptBid(&auctioneer.Auction{
		ExcludedStorageProviders: []string{"f0002"}},
		&auctioneer.Bid{StorageProviderID: "f0001"}))
	require.False(t, acceptBid(&auctioneer.Auction{
		ExcludedStorageProviders: []string{"f0001"},
	}, &auctioneer.Bid{StorageProviderID: "f0001"}))
}

func TestSortBids(t *testing.T) {
	for _, testCase := range []struct {
		name         string
		bids         map[core.BidID]auctioneer.Bid
		dealVerified bool
		winner       string
	}{
		{"unverified", map[core.BidID]auctioneer.Bid{
			"higher": auctioneer.Bid{AskPrice: 1},
			"lower":  auctioneer.Bid{AskPrice: 0},
		}, false, "lower"},
		{"verified", map[core.BidID]auctioneer.Bid{
			"higher": auctioneer.Bid{VerifiedAskPrice: 1},
			"lower":  auctioneer.Bid{VerifiedAskPrice: 0},
		}, true, "lower"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			bh := heapifyBids(testCase.bids, testCase.dealVerified)
			b := heap.Pop(bh).(rankedBid)
			assert.Equal(t, core.BidID(testCase.winner), b.ID)
		})
	}

	hits := make(map[core.BidID]int)
	for i := 0; i < 100; i++ {
		bids := map[core.BidID]auctioneer.Bid{"a": auctioneer.Bid{AskPrice: 1}, "b": auctioneer.Bid{AskPrice: 1}}
		bh := heapifyBids(bids, false)
		b := heap.Pop(bh).(rankedBid)
		hits[b.ID]++
	}
	for _, n := range hits {
		// we should have some randomness, no matter how unfair it can be
		assert.InDelta(t, 50, n, 49)
	}
}
