package auctioneer

import (
	"container/heap"
	"math/rand"
	"time"

	"github.com/textileio/broker-core/auctioneer"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

// Cmp is the interface for a comparator.
type Cmp interface {
	// Cmp returns arbitrary number with the following semantics:
	// negative: i is considered to be less than j
	// zero: i is considered to be equal to j
	// positive: i is considered to be greater than j
	Cmp(a *auctioneer.Auction, i auctioneer.Bid, j auctioneer.Bid) int
}

// CmpFn is a helper which turns a function to a Cmp interface.
func CmpFn(f func(a *auctioneer.Auction, i auctioneer.Bid, j auctioneer.Bid) int) Cmp {
	return fnCmp{f: f}
}

type fnCmp struct {
	f func(*auctioneer.Auction, auctioneer.Bid, auctioneer.Bid) int
}

func (c fnCmp) Cmp(a *auctioneer.Auction, i auctioneer.Bid, j auctioneer.Bid) int {
	return c.f(a, i, j)
}

type ordered struct {
	cmps []Cmp
}

// Ordered executes each comparator in order, i.e., if the first comparator
// judges the two bids to be equal, continues to the next comparator, and so
// on. It considers two bids to be equal if all comparators are exhausted.
func Ordered(cmps ...Cmp) Cmp {
	return ordered{cmps}
}

func (c ordered) Cmp(a *auctioneer.Auction, i auctioneer.Bid, j auctioneer.Bid) int {
	for _, c := range c.cmps {
		result := c.Cmp(a, i, j)
		switch result {
		case 0:
			continue
		default:
			return result
		}
	}
	return 0
}

// Weighed combines comparators togethers with different weights. The result
// depends on both the weights given to each comparator and the scale of the
// comparison result of each comparator. Be aware to not cause integer overflow.
type Weighed struct {
	cmps    []Cmp
	weights []int
}

// Add returns a new weighed comparator with the comparator being added with
// the given weight.
func (wc Weighed) Add(cmp Cmp, weight int) Weighed {
	w := Weighed{cmps: wc.cmps, weights: wc.weights}
	w.cmps = append(w.cmps, cmp)
	w.weights = append(w.weights, weight)
	return w
}

// Cmp adds up the result of calling Cmp of each comparators with their respective weights.
func (wc Weighed) Cmp(a *auctioneer.Auction, i auctioneer.Bid, j auctioneer.Bid) int {
	var weighed int
	for k, cmp := range wc.cmps {
		weighed += wc.weights[k] * cmp.Cmp(a, i, j)
	}
	return weighed
}

// LowerPrice returns a comparator which prefers lower ask price or verified ask
// price depending on if the auction is verified. The price difference is returned.
func LowerPrice() Cmp {
	return CmpFn(func(a *auctioneer.Auction, i auctioneer.Bid, j auctioneer.Bid) int {
		if a.DealVerified {
			return int(i.VerifiedAskPrice - j.VerifiedAskPrice)
		}
		return int(i.AskPrice - j.AskPrice)
	})
}

// EarlierStartEpoch returns a comparator which prefers bid with earlier start
// epoch. The difference between the start epochs in the scale of the
// resolution is returned.
func EarlierStartEpoch(resolution time.Duration) Cmp {
	epochResolution := int(resolution.Seconds() / 30)
	if epochResolution == 0 {
		epochResolution = 1
	}
	return CmpFn(func(a *auctioneer.Auction, i auctioneer.Bid, j auctioneer.Bid) int {
		return (int(i.StartEpoch) - int(j.StartEpoch)) / epochResolution
	})
}

type providerRate struct {
	rates map[string]int
}

func (wc providerRate) Cmp(a *auctioneer.Auction, i auctioneer.Bid, j auctioneer.Bid) int {
	return wc.rates[i.StorageProviderID] - wc.rates[j.StorageProviderID]
}

// LowerProviderRate returns a comparator which considers some rate of the storage provider. Provider with lower rate
// gets a higher chance to win. Provider not in the provided rates table are considered to have zero rate.
func LowerProviderRate(rates map[string]int) Cmp {
	return providerRate{rates}
}

// Random returns a comparator which randomly returns -1, 0, or 1 using the global random source.
// Note that it violates the invariants of heap operations and the result distribution is very uneven.
func Random() Cmp {
	return CmpFn(func(a *auctioneer.Auction, i auctioneer.Bid, j auctioneer.Bid) int {
		return rand.Intn(3) - 1
	})
}

// BidsIter provides a general way to iterate bids sorted by sorter.
type BidsIter func() (auctioneer.Bid, bool)

// Next gets the next bid from the iterator, if exists.
func (i BidsIter) Next() (auctioneer.Bid, bool) {
	return i()
}

// MustNext is similar to Next, but panics when the next bid doesn't exist.
func (i BidsIter) MustNext() auctioneer.Bid {
	b, ok := i.Next()
	if !ok {
		panic("no next item")
	}
	return b
}

// BidsSorter constructs a sorter from the given comparator and bids.
func BidsSorter(auction *auctioneer.Auction, bids []auctioneer.Bid) *Sorter {
	h := make([]auctioneer.Bid, 0, len(bids))
	h = append(h, bids...)
	return &Sorter{&bidHeap{a: auction, h: h}}
}

// Sorter has a single sort method which takes an aunction and some bids, then
// sort the bids based on the comparator given.
type Sorter struct {
	bh *bidHeap
}

// Len returns the number of bids remain in the sorter right now.
func (s Sorter) Len() int {
	return s.bh.Len()
}

// Select returns a new sorter with the bids don't pass the filter being removed.
func (s Sorter) Select(filter func(*auctioneer.Bid) bool) Sorter {
	h := make([]auctioneer.Bid, 0, len(s.bh.h))
	for _, b := range s.bh.h {
		if filter(&b) {
			h = append(h, b)
		}
	}
	return Sorter{&bidHeap{a: s.bh.a, h: h}}
}

// Iterate returns an iterator which gives the sorted result one by one.
func (s *Sorter) Iterate(cmp Cmp) BidsIter {
	s.bh = &bidHeap{a: s.bh.a, h: s.bh.h, cmp: cmp}
	heap.Init(s.bh)
	return func() (auctioneer.Bid, bool) {
		if s.bh.Len() == 0 {
			return auctioneer.Bid{}, false
		}

		return heap.Pop(s.bh).(auctioneer.Bid), true
	}
}

// RandomTopN returns an iterator which randomly choose one of the top N from the sorted list to return.
func (s *Sorter) RandomTopN(n int, cmp Cmp) BidsIter {
	s.bh = &bidHeap{a: s.bh.a, h: s.bh.h, cmp: cmp}
	heap.Init(s.bh)
	if n > s.bh.Len() {
		n = s.bh.Len()
	}
	return func() (auctioneer.Bid, bool) {
		if n == 0 {
			return auctioneer.Bid{}, false
		}
		i := rand.Intn(n)
		temp := make([]auctioneer.Bid, 0, i)
		for j := 0; j < i; j++ {
			temp = append(temp, heap.Pop(s.bh).(auctioneer.Bid))
		}
		b := heap.Pop(s.bh).(auctioneer.Bid)
		for _, t := range temp {
			heap.Push(s.bh, t)
		}
		n--
		return b, true
	}
}

// Random returns an iterator which randomly choose one bid to return.
func (s *Sorter) Random() BidsIter {
	s.bh = &bidHeap{a: s.bh.a, h: s.bh.h, cmp: Random()}
	heap.Init(s.bh)
	return func() (auctioneer.Bid, bool) {
		if s.bh.Len() == 0 {
			return auctioneer.Bid{}, false
		}
		return heap.Remove(s.bh, rand.Intn(s.bh.Len())).(auctioneer.Bid), true
	}
}

// bidHeap is used to efficiently select auction winners.
type bidHeap struct {
	a   *auctioneer.Auction
	h   []auctioneer.Bid
	cmp Cmp
}

// Len returns the length of h.
func (bh bidHeap) Len() int {
	return len(bh.h)
}

// Less returns true if the value at i is less than the value at j.
func (bh bidHeap) Less(i, j int) bool {
	return bh.cmp.Cmp(bh.a, bh.h[i], bh.h[j]) < 0
}

// Swap index i and j.
func (bh bidHeap) Swap(i, j int) {
	bh.h[i], bh.h[j] = bh.h[j], bh.h[i]
}

// Push adds x to h.
func (bh *bidHeap) Push(x interface{}) {
	bh.h = append(bh.h, x.(auctioneer.Bid))
}

// Pop removes and returns the last element in h.
func (bh *bidHeap) Pop() (x interface{}) {
	x, bh.h = bh.h[len(bh.h)-1], bh.h[:len(bh.h)-1]
	return x
}
