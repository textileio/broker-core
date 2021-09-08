package auctioneer

import (
	"context"
	"testing"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	core "github.com/textileio/bidbot/lib/auction"
	"github.com/textileio/broker-core/auctioneer"
	"github.com/textileio/broker-core/msgbroker/fakemsgbroker"
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

func TestSelectWinners(t *testing.T) {
	t.Skip()
	failureRates := map[string]int{
		"sp1": 1,
		"sp2": 2,
		"sp3": 2,
		"sp4": 10,
		"sp5": 20,
		"sp6": 200,
		"sp7": 200,
		"sp8": 100,
	}
	winningRates := map[string]int{
		"sp1": 200,
		"sp2": 100,
		"sp3": 20,
		"sp4": 10,
		"sp5": 10,
		"sp6": 2,
		"sp7": 2,
		"sp8": 1,
	}
	bids := []auctioneer.Bid{
		auctioneer.Bid{ID: "bid0", StorageProviderID: "sp0"},
		auctioneer.Bid{ID: "bid1", StorageProviderID: "sp1"},
		auctioneer.Bid{ID: "bid2", StorageProviderID: "sp2"},
		auctioneer.Bid{ID: "bid3", StorageProviderID: "sp3"},
		auctioneer.Bid{ID: "bid4", StorageProviderID: "sp4"},
		auctioneer.Bid{ID: "bid5", StorageProviderID: "sp5"},
		auctioneer.Bid{ID: "bid6", StorageProviderID: "sp6"},
		auctioneer.Bid{ID: "bid7", StorageProviderID: "sp7"},
		auctioneer.Bid{ID: "bid8", StorageProviderID: "sp8"},
	}
	for _, testCase := range []struct {
		name    string
		auction auctioneer.Auction
		winners []core.BidID
	}{
		{
			"one replica, randomly choose one from 5 with low failure rates",
			auctioneer.Auction{DealReplication: 1},
			[]core.BidID{"bid0", "bid1", "bid2", "bid3", "bid4"},
		},
		{
			"two replicas, choose one with low failure rates and another with low winning rates",
			auctioneer.Auction{DealReplication: 2},
			[]core.BidID{"bid0", "bid1", "bid2", "bid3", "bid4",
				"bid4", "bid5", "bid6", "bid7", "bid8"},
		},
		{
			"three replicas, randomly choose one apart from the above",
			auctioneer.Auction{DealReplication: 3},
			[]core.BidID{"bid0", "bid1", "bid2", "bid3", "bid4",
				"bid4", "bid5", "bid6", "bid7", "bid8",
				"bid0", "bid1", "bid2", "bid3", "bid4", "bid5", "bid6", "bid7", "bid8"},
		},
		{
			"more replicas, choose more random ones",
			auctioneer.Auction{DealReplication: 4},
			[]core.BidID{"bid0", "bid1", "bid2", "bid3", "bid4",
				"bid4", "bid5", "bid6", "bid7", "bid8",
				"bid0", "bid1", "bid2", "bid3", "bid4", "bid5", "bid6", "bid7", "bid8",
				"bid0", "bid1", "bid2", "bid3", "bid4", "bid5", "bid6", "bid7", "bid8"},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			rounds, total := 1000, 0
			expectedWins := make(map[core.BidID]int)
			for _, id := range testCase.winners {
				total++
				expectedWins[id]++
			}
			for id, _ := range expectedWins {
				expectedWins[id] = expectedWins[id] * rounds / total
			}
			a := &Auctioneer{mb: fakemsgbroker.New(),
				winsPublisher: func(ctx context.Context, id core.ID, bid core.BidID, bidder peer.ID) error { return nil },
			}
			a.providerFailureRates.Store(failureRates)
			a.providerWinningRates.Store(winningRates)
			spWins := make(map[core.BidID]int)
			for i := 0; i < rounds; i++ {
				winningBids, err := a.selectWinners(context.Background(), testCase.auction, bids)
				assert.NoError(t, err)
				assert.Len(t, winningBids, int(testCase.auction.DealReplication))
				for id, _ := range winningBids {
					assert.Contains(t, expectedWins, id)
					spWins[id]++
				}
			}
			t.Log(spWins)
			assert.Len(t, spWins, len(expectedWins))
			for id, n := range spWins {
				assert.InDelta(t, n, expectedWins[id], float64(expectedWins[id]/2))
			}
		})
	}
}
