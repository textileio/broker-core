package broker

import (
	"context"
	"net/url"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multiaddr"
)

const (
	CodecFilCommitmentUnsealed = 0xf101
	MaxPieceSize               = 32 << 30
)

// TODO(jsign): rename and reorganize?
// BrokerRequestor allows to create and query BrokerRequests.
type BrokerRequestor interface {
	// Create creates a new BrokerRequest for data `c` and
	// configuration `meta`.
	Create(ctx context.Context, c cid.Cid, meta Metadata, pc *PreparedCAR) (BrokerRequest, error)
	// Get returns a broker request from an id.
	Get(ctx context.Context, ID BrokerRequestID) (BrokerRequest, error)
}

// BrokerRequestID is the type used for broker request identity.
type BrokerRequestID string

// BrokerRequest references a storage request for a Cid.
type BrokerRequest struct {
	ID            BrokerRequestID
	DataCid       cid.Cid
	Status        BrokerRequestStatus
	Metadata      Metadata
	StorageDealID StorageDealID
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// TODO(jsign): delete metadata.
// Metadata provides storage and bidding configuration.
type Metadata struct {
	Region string
}

type PreparedCAR struct {
	PieceCid  cid.Cid
	PieceSize int64
	RepFactor int
	Deadline  time.Time
	CARURL    *CARURL
	CARIPFS   *CARIPFS
}

// CARURL contains details of a CAR file stored in an HTTP endpoint.
type CARURL struct {
	URL url.URL
}

// CARIPFS contains details of a CAR file Cid stored in an HTTP endpoint.
type CARIPFS struct {
	Cid            cid.Cid
	NodesMultiaddr []multiaddr.Multiaddr
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
