package broker

import (
	"context"
	"time"

	"github.com/ipfs/go-cid"
)

// Broker allows to create and track BrokerRequest.
type Broker interface {
	// Create creates a new BrokerRequest for data `c` and
	// configuration `meta`.
	Create(ctx context.Context, c cid.Cid, meta Metadata) (BrokerRequest, error)
	// Get returns a broker request from an id.
	Get(ctx context.Context, ID BrokerRequestID) (BrokerRequest, error)

	// CreateStorageDeal creates a new StorageDeal. It is called
	// by the Packer after batching a set of BrokerRequest properly.
	CreateStorageDeal(ctx context.Context, brg BrokerRequestGroup) error
}

// BrokerRequestID is the type used for broker request identity.
type BrokerRequestID string

// BrokerRequest references a storage request for a Cid.
type BrokerRequest struct {
	ID            BrokerRequestID     `json:"id"`
	Status        BrokerRequestStatus `json:"status"`
	Metadata      Metadata            `json:"metadata"`
	StorageDealID StorageDealID       `json:"storage_deal_id"`
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
	// BrokerRequestUnknown is an invalid status value. Defined for safety.
	BrokerRequestUnknown BrokerRequestStatus = iota
	// BrokerRequestBatching indicates that a broker request is being batched.
	BrokerRequestBatching
	// BrokerRequestPreparing indicates that a broker request is being prepared.
	BrokerRequestPreparing
	// BrokerRequestAuctioning indicates that a broker request is in bidding stage.
	BrokerRequestAuctioning
	// BrokerRequestDealMaking indicates that the storage deal deals are being executed.
	BrokerRequestDealMaking
	// BrokerRequestSuccess indicates that the storage deal was successfully stored in Filecoin.
	BrokerRequestSuccess
)

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

// BrokerRequestGroup is an ephemeral entity that Packer utilizes
// to ask the Broker to create a definitive StorageDeal from it.
type BrokerRequestGroup struct {
	Cid                    cid.Cid
	GroupedStorageRequests []BrokerRequestID
}

// DataPreparationResult is the result of preparing a StorageDeal.
type DataPreparationResult struct {
	PieceSize int64
	CommP     cid.Cid
}
