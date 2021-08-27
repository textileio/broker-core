package auctioneer

import (
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
