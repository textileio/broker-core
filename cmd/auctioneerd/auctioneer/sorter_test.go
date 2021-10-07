package auctioneer

import (
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
			s := BidsSorter(testCase.auction, testCase.bids)
			b, ok := s.Iterate(LowerPrice()).Next()
			assert.True(t, ok)
			assert.Equal(t, core.BidID(testCase.winner), b.ID)
		})
	}
}

func TestSortRandom(t *testing.T) {
	hits := make(map[core.BidID]int)
	for i := 0; i < 1000; i++ {
		s := BidsSorter(
			&auctioneer.Auction{},
			[]auctioneer.Bid{auctioneer.Bid{ID: "a"}, auctioneer.Bid{ID: "b"}, auctioneer.Bid{ID: "c"}, auctioneer.Bid{ID: "d"}})

		b, ok := s.Iterate(Random()).Next()
		assert.True(t, ok)
		hits[b.ID]++
	}
	// there should be some randomness, albeit very uneven.
	assert.Len(t, hits, 4)
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

	it := BidsSorter(&auctioneer.Auction{}, bids).Iterate(LowerProviderRate(rateTable))
	assert.EqualValues(t, "zero", it.MustNext().ID)
	assert.EqualValues(t, "low", it.MustNext().ID)
	assert.EqualValues(t, "low2", it.MustNext().ID)
	assert.EqualValues(t, "medium", it.MustNext().ID)
	assert.EqualValues(t, "high", it.MustNext().ID)
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
			it := BidsSorter(&auctioneer.Auction{}, bids).Iterate(testCase.cmp)
			for {
				if b, ok := it.Next(); ok {
					result = append(result, string(b.ID))
					continue
				}
				break
			}
			assert.Equal(t, testCase.expected, result)
		})
	}
}

func TestRandomTopN(t *testing.T) {
	rateTable := map[string]int{
		"f0001": 100,
		"f0002": 1,
		"f0003": 2,
		"f0004": 9999,
	}
	bids := []auctioneer.Bid{
		auctioneer.Bid{ID: "medium", StorageProviderID: "f0001"},
		auctioneer.Bid{ID: "low", StorageProviderID: "f0002"},
		auctioneer.Bid{ID: "low2", StorageProviderID: "f0003"},
		auctioneer.Bid{ID: "high", StorageProviderID: "f0004"},
		auctioneer.Bid{ID: "zero", StorageProviderID: "f0005"},
	}

	hits := make(map[core.BidID]int)
	for i := 0; i < 1000; i++ {
		it := BidsSorter(&auctioneer.Auction{}, bids).RandomTopN(4, LowerProviderRate(rateTable))
		hits[it.MustNext().ID]++
	}
	assert.InDelta(t, 250, hits["zero"], 50)
	assert.InDelta(t, 250, hits["low"], 50)
	assert.InDelta(t, 250, hits["low2"], 50)
	assert.InDelta(t, 250, hits["medium"], 50)
}

func TestRandom(t *testing.T) {
	// can cause CI failure occasionally
	t.Skip()
	hits := make(map[core.BidID]int)
	for i := 0; i < 1000; i++ {
		s := BidsSorter(
			&auctioneer.Auction{},
			[]auctioneer.Bid{auctioneer.Bid{ID: "a"}, auctioneer.Bid{ID: "b"}, auctioneer.Bid{ID: "c"}, auctioneer.Bid{ID: "d"}})

		b := s.Random().MustNext()
		hits[b.ID]++
	}
	assert.Len(t, hits, 4)
	for id, n := range hits {
		// we should have fairly good randomness
		assert.InDelta(t, 250, n, 50, id)
	}
}

func TestUniqueResults(t *testing.T) {
	rateTable := map[string]int{
		"f0001": 100,
		"f0002": 1,
		"f0003": 2,
		"f0004": 9999,
	}
	bids := []auctioneer.Bid{
		auctioneer.Bid{ID: "medium", StorageProviderID: "f0001"},
		auctioneer.Bid{ID: "low", StorageProviderID: "f0002"},
		auctioneer.Bid{ID: "low2", StorageProviderID: "f0003"},
		auctioneer.Bid{ID: "high", StorageProviderID: "f0004"},
		auctioneer.Bid{ID: "zero", StorageProviderID: "f0005"},
	}

	for i := 0; i < 100; i++ {
		hits := make(map[core.BidID]int)
		s := BidsSorter(&auctioneer.Auction{}, bids)
		id := s.Iterate(LowerProviderRate(rateTable)).MustNext().ID
		hits[id]++
		id = s.RandomTopN(3, LowerProviderRate(rateTable)).MustNext().ID
		assert.NotContains(t, hits, id)
		hits[id]++
		id = s.RandomTopN(2, LowerProviderRate(rateTable)).MustNext().ID
		assert.NotContains(t, hits, id)
		hits[id]++
		id = s.Random().MustNext().ID
		assert.NotContains(t, hits, id)
		hits[id]++
	}
}
