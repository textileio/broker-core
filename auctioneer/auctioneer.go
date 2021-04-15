package auctioneer

import (
	"context"

	"github.com/libp2p/go-libp2p-core/protocol"
	"github.com/textileio/broker-core/broker"
)

const ProtocolDeals protocol.ID = "/textile/deals/0.0.1"

type Auctioneer interface {
	ReadyToAuction(ctx context.Context, deal broker.StorageDeal) error
}

type Bid struct {
	StorageDealID string
}
