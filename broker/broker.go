package broker

import (
	"context"

	"github.com/ipfs/go-cid"
)

// Broker allows to create and track BrokerRequest.
type Broker interface {
	Create(ctx context.Context, c cid.Cid, meta Metadata) (BrokerRequest, error)
}

// BrokerRequest references a storage request for a Cid.
type BrokerRequest struct {
	ID       string              `json:"id"`
	Status   BrokerRequestStatus `json:"status"`
	Metadata Metadata            `json:"metadata"`
}

// Metadata provides storage and bidding configuration.
type Metadata struct {
	Region string `json:"region"`
}

// BrokerRequestStatus describe the current status of a
// BrokerRequest.
type BrokerRequestStatus int

const (
	StatusUnkown BrokerRequestStatus = iota
	StatusIdle
	StatusFetchingData
	StatusBatching
	StatusPreparing
	StatusBidding
	StatusMonitoring
	StatusSuccess
)
