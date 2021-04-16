package auctioneer

import (
	"context"
	"fmt"

	"github.com/textileio/broker-core/auctioneer"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/auctioneerd/client"
	"github.com/textileio/broker-core/rpc"
)

// Auctioneer creates auctions and decides on winning bids.
type Auctioneer struct {
	c *client.Client
}

var _ auctioneer.Auctioneer = (*Auctioneer)(nil)

// New returns a new auctioneer.
func New(addr string) (*Auctioneer, error) {
	c, err := client.NewClient(addr, rpc.GetClientOpts(addr)...)
	if err != nil {
		return nil, fmt.Errorf("creating client: %v", err)
	}
	return &Auctioneer{c: c}, nil
}

// ReadyToAuction signals the auctioneer that this storage deal is ready to be included
// in an auction. At some point it will call the broker telling who wons the bids to continue
// with deal making.
func (a *Auctioneer) ReadyToAuction(ctx context.Context, sd broker.StorageDeal) error {
	// TODO (@sander): leverage sd in create request
	_, err := a.c.CreateAuction(ctx)
	if err != nil {
		return fmt.Errorf("creating auction: %v", err)
	}
	return nil
}
