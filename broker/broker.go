package broker

import (
	"context"
	"time"

	"github.com/ipfs/go-cid"
)

// Broker provides full set of functionalities for Filecoin brokering.
type Broker interface {
	BrokerRequestor

	// CreateStorageDeal creates a new StorageDeal. It is called
	// by the Packer after batching a set of BrokerRequest properly.
	CreateStorageDeal(ctx context.Context, batchCid cid.Cid, srids []BrokerRequestID) (StorageDeal, error)

	// StorageDealPrepared signals the broker that a StorageDeal was prepared and it's ready to auction.
	StorageDealPrepared(ctx context.Context, id StorageDealID, pr DataPreparationResult) error
}

// BrokerRequestor alows to create and query BrokerRequests.
type BrokerRequestor interface {
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

// Validate validates if the instance is valid.
func (m Metadata) Validate() error {
	// TODO: we can validate regions, or any other
	// fields that might exist.
	return nil
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
		return "invalid"
	}
}

// StorageDealID is the type of a StorageDeal identifier.
type StorageDealID string

// StorageDealStatus is the type of a broker status.
type StorageDealStatus int

const (
	// StorageDealUnkown is an invalid status value. Defined for safety.
	StorageDealUnkown StorageDealStatus = iota
	// StorageDealPreparing indicates that the storage deal is being prepared.
	StorageDealPreparing
	// StorageDealAuctioning indicates that the storage deal is being auctioned.
	StorageDealAuctioning
	// StorageDealDealMaking indicates that the storage deal deals are being executed.
	StorageDealDealMaking
	// StorageDealSuccess indicates that the storage deal was successfully stored in Filecoin.
	StorageDealSuccess
)

// StorageDeal is the underlying entity that gets into bidding and
// store data in the Filecoin network. It groups one or multiple
// BrokerRequests.
type StorageDeal struct {
	ID               StorageDealID
	Cid              cid.Cid
	Status           StorageDealStatus
	BrokerRequestIDs []BrokerRequestID
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// DataPreparationResult is the result of preparing a StorageDeal.
type DataPreparationResult struct {
	PieceSize uint64
	PieceCid  cid.Cid
}
