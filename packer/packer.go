package packer

import (
	"context"

	"github.com/ipfs/go-cid"
	"github.com/textileio/broker-core/broker"
)

// Packer batches and prepares data to be stored in Filecoin.
type Packer interface {
	// ReadyToBatch signals the packer that this broker request
	// is ready to be prepared. Depending on the packer configuration,
	// this broker request data might be batched with others.
	//
	// This API only allows packer to know about this BrokerRequest, the real
	// work is done asyc. After some BrokerRequests are batched, the packer will notify
	// the Broker that a new `StorageDeal` got prepared (which includes the
	// provided BrokerRequest), so it can continue with bidding.
	ReadyToBatch(ctx context.Context, id broker.BrokerRequestID, dataCid cid.Cid) error
}
