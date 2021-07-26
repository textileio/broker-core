package auctioneer

import (
	"time"

	"github.com/ipfs/go-cid"
	"github.com/textileio/bidbot/lib/auction"
	"github.com/textileio/broker-core/broker"
)

// Auction defines the core auction model.
type Auction struct {
	ID               auction.AuctionID
	BatchID          broker.BatchID
	PayloadCid       cid.Cid
	DealSize         uint64
	DealDuration     uint64
	DealReplication  uint32
	DealVerified     bool
	FilEpochDeadline uint64
	ExcludedMiners   []string
	Sources          auction.Sources
	Status           broker.AuctionStatus
	Bids             map[auction.BidID]auction.Bid
	WinningBids      map[auction.BidID]auction.WinningBid
	StartedAt        time.Time
	UpdatedAt        time.Time
	Duration         time.Duration
	Attempts         uint32
	ErrorCause       string
	// Ugly trick: a workaround to avoid calling Auctioneer.finalizeAuction
	// twice, because auction are enqueued to the Queue again indirectly
	// by Auctioneer.DeliverProposal.
	BrokerAlreadyNotifiedByClosedAuction bool
}
