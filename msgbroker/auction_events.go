package msgbroker

import (
	"context"

	"github.com/textileio/broker-core/auctioneer"
)

// AuctionEventsListener is a handler for auction events related topics.
type AuctionEventsListener interface {
	OnAuctionStarted(context.Context, auctioneer.Auction) error
	OnAuctionBidReceived(context.Context, auctioneer.Auction, auctioneer.Bid) error
	OnAuctionWinnerSelected(context.Context, auctioneer.Auction, auctioneer.WinningBid) error
	AuctionWinnerAcked(context.Context, auctioneer.Auction, auctioneer.WinningBid) error
	AuctionProposalCidDelivered(ctx context.Context, auctionID, bidID, proposalCid, errorCause string) error
	AuctionClosedListener
}
