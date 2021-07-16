package auctioneer

import (
	"context"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/textileio/bidbot/lib/auction"
	"github.com/textileio/broker-core/broker"
)

// Auctioneer creates auctions and decides on winning bids.
type Auctioneer interface {
	// ReadyToAuction signals the auctioneer that this storage deal is ready to be included in a auction.
	// dealDuration is in units of Filecoin epochs (~30s).
	ReadyToAuction(
		ctx context.Context,
		id broker.StorageDealID,
		payloadCid cid.Cid,
		dealSize int,
		dealDuration int,
		dealReplication int,
		dealVerified bool,
		excludedMiners []string,
		filEpochDeadline uint64,
		sources auction.Sources,
	) (auction.AuctionID, error)

	// ProposalAccepted notifies about an accepted deal proposal by a miner.
	ProposalAccepted(ctx context.Context, auID auction.AuctionID, bidID auction.BidID, proposalCid cid.Cid) error
}

// Auction defines the core auction model.
type Auction struct {
	ID               auction.AuctionID // TODO(jsign): verify assignement and tests.
	StorageDealID    broker.StorageDealID
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
