package broker

import (
	"context"

	"github.com/ipfs/go-cid"
	"github.com/textileio/broker-core/cmd/storaged/storage"
)

type Broker interface {
	Create(ctx context.Context, c cid.Cid, meta storage.Metadata) (BrokerRequest, error)
}

type BrokerRequest struct {
	ID     string              `json:"id"`
	Status BrokerRequestStatus `json:"status"`
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
