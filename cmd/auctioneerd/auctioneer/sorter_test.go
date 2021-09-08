package auctioneer

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	core "github.com/textileio/bidbot/lib/auction"
	"github.com/textileio/broker-core/auctioneer"
)

func TestSortByPrice(t *testing.T) {
	for _, testCase := range []struct {
		name    string
		bids    []auctioneer.Bid
		auction *auctioneer.Auction
		winner  string
	}{
		{"unverified", []auctioneer.Bid{
			auctioneer.Bid{ID: "medium", AskPrice: 1},
			auctioneer.Bid{ID: "low", AskPrice: 0},
			auctioneer.Bid{ID: "high", AskPrice: 2},
		}, &auctioneer.Auction{DealVerified: false}, "low"},
		{"verified", []auctioneer.Bid{
			auctioneer.Bid{ID: "high", VerifiedAskPrice: 2},
			auctioneer.Bid{ID: "low", VerifiedAskPrice: 0},
			auctioneer.Bid{ID: "medium", VerifiedAskPrice: 1},
		}, &auctioneer.Auction{DealVerified: true}, "low"},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			s := BidsSorter(testCase.auction, testCase.bids)
			b := <-s.Iterate(ctx, LowerPrice())
			assert.Equal(t, core.BidID(testCase.winner), b.ID)
		})
	}
}

func TestSortRandom(t *testing.T) {
	hits := make(map[core.BidID]int)
	for i := 0; i < 100; i++ {
		s := BidsSorter(
			&auctioneer.Auction{},
			[]auctioneer.Bid{auctioneer.Bid{ID: "a", AskPrice: 1}, auctioneer.Bid{ID: "b", AskPrice: 1}})

		ctx, cancel := context.WithCancel(context.Background())
		b := <-s.Iterate(ctx, Random())
		hits[b.ID]++
		cancel()
	}
	for id, n := range hits {
		// we should have some randomness
		assert.InDelta(t, 50, n, 40, id)
	}
}

func TestSortByProviderRates(t *testing.T) {
	rateTable := map[string]int{
		"f0001": 100,
		"f0002": 1,
		"f0003": 1,
		"f0004": 9999,
	}
	bids := []auctioneer.Bid{
		auctioneer.Bid{ID: "medium", StorageProviderID: "f0001"},
		auctioneer.Bid{ID: "low", StorageProviderID: "f0002"},
		auctioneer.Bid{ID: "low2", StorageProviderID: "f0003"},
		auctioneer.Bid{ID: "high", StorageProviderID: "f0004"},
		auctioneer.Bid{ID: "zero", StorageProviderID: "f0005"},
	}

	ch := BidsSorter(&auctioneer.Auction{}, bids).Iterate(context.Background(), LowerProviderRate(rateTable))
	assert.EqualValues(t, "zero", (<-ch).ID)
	assert.EqualValues(t, "low", (<-ch).ID)
	assert.EqualValues(t, "low2", (<-ch).ID)
	assert.EqualValues(t, "medium", (<-ch).ID)
	assert.EqualValues(t, "high", (<-ch).ID)
}

func TestCombination(t *testing.T) {
	bids := []auctioneer.Bid{
		auctioneer.Bid{ID: "1", AskPrice: 1, StartEpoch: 3},
		auctioneer.Bid{ID: "2", AskPrice: 3, StartEpoch: 1},
		auctioneer.Bid{ID: "3", AskPrice: 2, StartEpoch: 2},
		auctioneer.Bid{ID: "4", AskPrice: 2, StartEpoch: 1},
	}
	for _, testCase := range []struct {
		name     string
		cmp      Cmp
		expected []string
	}{
		{
			"epoch-over-price",
			Weighed{}.Add(EarlierStartEpoch(1), 10).Add(LowerPrice(), 6),
			[]string{"4", "2", "3", "1"},
		},
		{
			"price-over-epoch",
			Weighed{}.Add(LowerPrice(), 10).Add(EarlierStartEpoch(1), 6),
			[]string{"4", "1", "3", "2"},
		},
		{
			"epoch-then-price",
			Ordered(EarlierStartEpoch(1), LowerPrice()),
			[]string{"4", "2", "3", "1"},
		},
		{
			"price-then-epoch",
			Ordered(LowerPrice(), EarlierStartEpoch(1)),
			[]string{"1", "4", "3", "2"},
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			var result []string
			for b := range BidsSorter(&auctioneer.Auction{}, bids).Iterate(context.Background(), testCase.cmp) {
				result = append(result, string(b.ID))
			}
			assert.Equal(t, testCase.expected, result)
		})
	}
}

func TestRandomTopN(t *testing.T) {
	hits := make(map[core.BidID]int)
	for i := 0; i < 100; i++ {
		s := BidsSorter(
			&auctioneer.Auction{},
			[]auctioneer.Bid{
				auctioneer.Bid{ID: "a", AskPrice: 2},
				auctioneer.Bid{ID: "b", AskPrice: 10},
				auctioneer.Bid{ID: "c", AskPrice: 1},
				auctioneer.Bid{ID: "d", AskPrice: 1},
			})
		ctx, cancel := context.WithCancel(context.Background())
		b := <-s.RandomTopN(ctx, 3, LowerPrice())
		hits[b.ID]++
		cancel()
	}
	assert.NotContains(t, hits, "b", "the bid with higher price should not be chosen")
	assert.Len(t, hits, 3)
	for id, n := range hits {
		// we should have some randomness
		assert.InDelta(t, 33, n, 10, id)
	}
}
