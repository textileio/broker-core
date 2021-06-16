package auctioneer

import (
	"context"

	"github.com/ipfs/go-cid"
	"github.com/textileio/broker-core/broker"
)

// Auctioneer creates auctions and decides on winning bids.
type Auctioneer interface {
	// ReadyToAuction signals the auctioneer that this storage deal is ready to be included in a broker.Auction.
	// dealDuration is in units of Filecoin epochs (~30s).
	ReadyToAuction(
		ctx context.Context,
		id broker.StorageDealID,
		dataCid cid.Cid,
		dealSize int,
		dealDuration int,
		dealReplication int,
		dealVerified bool,
		excludedMiners []string,
	) (broker.AuctionID, error)

	// GetAuction returns an auction by broker.AuctionID.
	GetAuction(ctx context.Context, id broker.AuctionID) (broker.Auction, error)

	// ProposalAccepted notifies about an accepted deal proposal by a miner.
	ProposalAccepted(ctx context.Context, auID broker.AuctionID, bidID broker.BidID, proposalCid cid.Cid) error
}
