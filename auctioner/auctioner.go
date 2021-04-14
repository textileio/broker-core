package auctioner

import (
	"context"

	"github.com/textileio/broker-core/broker"
)

// Auctioner creates auctions and decides on winning bids.
type Auctioner interface {
	ReadyToAuction(ctx context.Context, sr broker.StorageDeal) error
}
