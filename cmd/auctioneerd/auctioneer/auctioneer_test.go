package auctioneer

import (
	"testing"

	"github.com/stretchr/testify/require"
	core "github.com/textileio/bidbot/lib/auction"
	"github.com/textileio/broker-core/auctioneer"
)

func TestAcceptBid(t *testing.T) {
	require.True(t, acceptBid(&auctioneer.Auction{}, &core.Bid{}))
	require.True(t, acceptBid(&auctioneer.Auction{}, &core.Bid{StartEpoch: 90000}))
	require.True(t, acceptBid(&auctioneer.Auction{FilEpochDeadline: 89999}, &core.Bid{StartEpoch: 89999}))
	require.False(t, acceptBid(&auctioneer.Auction{FilEpochDeadline: 89999}, &core.Bid{StartEpoch: 90000}))
	// this shouldn't happen because zero StartEpoch won't pass core.Bid
	// validation, but just in case.
	require.False(t, acceptBid(&auctioneer.Auction{FilEpochDeadline: 89999}, &core.Bid{}))

	require.True(t, acceptBid(&auctioneer.Auction{}, &core.Bid{MinerAddr: "f0001"}))
	require.True(t, acceptBid(&auctioneer.Auction{ExcludedMiners: []string{"f0002"}}, &core.Bid{MinerAddr: "f0001"}))
	require.False(t, acceptBid(&auctioneer.Auction{ExcludedMiners: []string{"f0001"}}, &core.Bid{MinerAddr: "f0001"}))
}
