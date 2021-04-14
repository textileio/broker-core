package broker

import (
	"context"
	"time"

	"github.com/ipfs/go-cid"
)

// Broker allows to create and track BrokerRequest.
type Broker interface {
	Create(ctx context.Context, c cid.Cid, meta Metadata) (BrokerRequest, error)
	Get(ctx context.Context, ID BrokerRequestID) (BrokerRequest, error)
}

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
	StatusUnkown BrokerRequestStatus = iota
	StatusBatching
	StatusPreparing
	StatusBidding
	StatusMonitoring
	StatusSuccess
)

// Events
type EventType int

const (
	StorageRequestReadyToBatch EventType = iota
	BatchCreated
)

type Event struct {
	ID      BrokerRequestID
	Type    EventType
	Payload interface{}
}

// StorageDeal
type StorageDealID string

type StorageDealStatus int

const (
	StorageDealUnkown StorageDealStatus = iota
	StorageDealPreparing
)

type StorageDeal struct {
	ID               StorageDealID
	Cid              cid.Cid
	Status           StorageDealStatus
	BrokerRequestIDs []BrokerRequestID
	CreatedAt        time.Time
	UpdatedAt        time.Time
}
type BrokerRequestGroup struct {
	Cid                    cid.Cid
	GroupedStorageRequests []BrokerRequestID
}
