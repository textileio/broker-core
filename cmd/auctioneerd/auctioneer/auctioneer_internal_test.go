package auctioneer

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/textileio/broker-core/auctioneer"
)

func TestAcceptBid(t *testing.T) {
	require.True(t, acceptBid(&auctioneer.Auction{}, &auctioneer.Bid{}))
	require.True(t, acceptBid(&auctioneer.Auction{}, &auctioneer.Bid{StartEpoch: 90000}))
	require.True(t, acceptBid(&auctioneer.Auction{FilEpochDeadline: 89999}, &auctioneer.Bid{StartEpoch: 89999}))
	require.False(t, acceptBid(&auctioneer.Auction{FilEpochDeadline: 89999}, &auctioneer.Bid{StartEpoch: 90000}))
	// this shouldn't happen because zero StartEpoch won't pass core.Bid
	// validation, but just in case.
	require.False(t, acceptBid(&auctioneer.Auction{FilEpochDeadline: 89999}, &auctioneer.Bid{}))

	require.True(t, acceptBid(&auctioneer.Auction{}, &auctioneer.Bid{MinerAddr: "f0001"}))
	require.True(t, acceptBid(&auctioneer.Auction{
		ExcludedStorageProviders: []string{"f0002"}},
		&auctioneer.Bid{MinerAddr: "f0001"}))
	require.False(t, acceptBid(&auctioneer.Auction{
		ExcludedStorageProviders: []string{"f0001"},
	}, &auctioneer.Bid{MinerAddr: "f0001"}))
}
