package packer

import "github.com/textileio/broker-core/broker"

// Packer batches and prepares data to be stored in Filecoin.
type Packer interface {
	// ReadyToPack signals the packer that this broker request
	// is ready to be prepared. Depending on the packer configuration,
	// this broker request data might be batched with others.
	//
	// This API only allows packer to know about this BrokerRequest, the real
	// work is done asyc. At some point  in the future, the packer will notify
	// the Broker that a new `StorageDeal` got prepared (which includes the
	// provided BrokerRequest), so it can continue with bidding.
	ReadyToPack(br broker.BrokerRequest) error
}
