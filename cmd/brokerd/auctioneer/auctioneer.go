package auctioneer

import (
	"context"

	"github.com/textileio/broker-core/auctioneer"
	"github.com/textileio/broker-core/broker"
)

// Auctioneer creates auctions and decides on winning bids.
type Auctioneer struct {
}

var _ auctioneer.Auctioneer = (*Auctioneer)(nil)

// New returns a new auctioneer.
func New() (*Auctioneer, error) {
	return &Auctioneer{}, nil
}

// ReadyToAuction signals the auctioneer that this storage deal is ready to be included
// in an auction. At some point it will call the broker telling who wons the bids to continue
// with deal making.
func (a *Auctioneer) ReadyToAuction(ctx context.Context, sd broker.StorageDeal) error {
	// @sander connects pipes here.

	return nil
}
