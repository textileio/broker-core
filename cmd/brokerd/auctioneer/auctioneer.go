package auctioneer

import (
	"context"
	"fmt"

	logger "github.com/ipfs/go-log/v2"
	"github.com/textileio/broker-core/auctioneer"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/auctioneerd/client"
	"github.com/textileio/broker-core/rpc"
)

var log = logger.Logger("auctioneer")

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
func (a *Auctioneer) ReadyToAuction(ctx context.Context, id broker.StorageDealID, dealSize uint64, dealDuration uint64) error {
	res, err := a.c.CreateAuction(ctx, id, dealSize, dealDuration)
	if err != nil {
		return fmt.Errorf("creating auction: %v", err)
	}

	log.Debugf("created auction %s", res.Id)
	return nil
}
