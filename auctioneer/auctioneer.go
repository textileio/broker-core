package auctioneer

import (
	"context"

	"github.com/ipfs/go-cid"
	"github.com/textileio/bidbot/lib/auction"
)

// Auctioneer creates auctions and decides on winning bids.
type Auctioneer interface {
	// ReadyToAuction signals the auctioneer that this storage deal is ready to be included in a auction.Auction.
	// dealDuration is in units of Filecoin epochs (~30s).
	ReadyToAuction(
		ctx context.Context,
		id auction.StorageDealID,
		payloadCid cid.Cid,
		dealSize int,
		dealDuration int,
		dealReplication int,
		dealVerified bool,
		excludedMiners []string,
		filEpochDeadline uint64,
		sources auction.Sources,
	) (auction.AuctionID, error)

	// GetAuction returns an auction by auction.AuctionID.
	GetAuction(ctx context.Context, id auction.AuctionID) (auction.Auction, error)

	// ProposalAccepted notifies about an accepted deal proposal by a miner.
	ProposalAccepted(ctx context.Context, auID auction.AuctionID, bidID auction.BidID, proposalCid cid.Cid) error
}
