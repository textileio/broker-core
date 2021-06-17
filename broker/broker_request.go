package broker

import (
	"context"
	"time"

	"github.com/ipfs/go-cid"
)

// BrokerRequestor alows to create and query BrokerRequests.
type BrokerRequestor interface {
	// TODO(jsign): rename and reorganize?
	// Create creates a new BrokerRequest for data `c` and
	// configuration `meta`.
	Create(ctx context.Context, c cid.Cid, meta Metadata) (BrokerRequest, error)
	// Get returns a broker request from an id.
	Get(ctx context.Context, ID BrokerRequestID) (BrokerRequest, error)
}

// BrokerRequestID is the type used for broker request identity.
type BrokerRequestID string

// BrokerRequest references a storage request for a Cid.
type BrokerRequest struct {
	ID            BrokerRequestID     `json:"id"`
	DataCid       cid.Cid             `json:"data_cid"`
	Status        BrokerRequestStatus `json:"status"`
	Metadata      Metadata            `json:"metadata"`
	StorageDealID StorageDealID       `json:"storage_deal_id,omitempty"`
	CreatedAt     time.Time           `json:"created_at"`
	UpdatedAt     time.Time           `json:"updated_at"`
}

// Metadata provides storage and bidding configuration.
type Metadata struct {
	Region string `json:"region"`
}

// BrokerRequestStatus describe the current status of a
// BrokerRequest.
type BrokerRequestStatus int

const (
	// RequestUnknown is an invalid status value. Defined for safety.
	RequestUnknown BrokerRequestStatus = iota
	// RequestBatching indicates that a broker request is being batched.
	RequestBatching
	// RequestPreparing indicates that a broker request is being prepared.
	RequestPreparing
	// RequestAuctioning indicates that a broker request is in bidding stage.
	RequestAuctioning
	// RequestDealMaking indicates that the storage deal deals are being executed.
	RequestDealMaking
	// RequestSuccess indicates that the storage deal was successfully stored in Filecoin.
	RequestSuccess
)

// String returns a string-encoded status.
func (brs BrokerRequestStatus) String() string {
	switch brs {
	case RequestUnknown:
		return "unknown"
	case RequestBatching:
		return "batching"
	case RequestPreparing:
		return "preparing"
	case RequestAuctioning:
		return "auctioning"
	case RequestDealMaking:
		return "deal-making"
	case RequestSuccess:
		return "success"
	default:
		return invalidStatus
	}
}
