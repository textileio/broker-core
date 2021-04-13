package broker

import (
	"context"

	"github.com/ipfs/go-cid"
)

type Broker interface {
	Create(ctx context.Context, c cid.Cid, meta Metadata) (BrokerRequest, error)
}

type BrokerRequest struct {
	ID     string              `json:"id"`
	Status BrokerRequestStatus `json:"status"`
}

type Metadata struct {
	Region string `json:"region"`
}

type BrokerRequestStatus int

// Not complete yet.
const (
	StatusUnkown BrokerRequestStatus = iota
	StatusBatching
	StatusPreparing
	StatusBidding
	StatusMonitoring
	StatusSuccess
)
