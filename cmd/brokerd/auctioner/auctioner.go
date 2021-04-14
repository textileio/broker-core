package auctioner

import (
	"context"

	"github.com/textileio/broker-core/auctioner"
	"github.com/textileio/broker-core/broker"
)

// Auctioner creates auctions and decides on winning bids.
type Auctioner struct {
}

var _ auctioner.Auctioner = (*Auctioner)(nil)

// New returns a new auctioner.
func New() (*Auctioner, error) {
	return &Auctioner{}, nil
}

// ReadyToAuction signals the auctioner that this storage deal is ready to be included
// in an auction. At some point it will call the broker telling who wons the bids to continue
// with deal making.
func (a *Auctioner) ReadyToAuction(ctx context.Context, sd broker.StorageDeal) error {
	// @sander connects pipes here.

	return nil
}
