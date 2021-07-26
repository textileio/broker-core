package broker

import (
	"time"

	"github.com/ipfs/go-cid"
	"github.com/textileio/bidbot/lib/auction"
)

const (
	// CodecFilCommitmentUnsealed is the IPLD codec for PieceCid cids.
	CodecFilCommitmentUnsealed = 0xf101
	// DefaultPreparedCARDeadline is the default deadline for prepared CAR deals.
	DefaultPreparedCARDeadline = time.Hour * 48
	// MaxPieceSize is the maximum piece size accepted for prepared data.
	MaxPieceSize = 32 << 30
	// MinDealReplication is the minimum allowed deal replication requested of miners.
	MinDealReplication = 1
	// MaxDealReplication is the maximum allowed deal replication requested of miners.
	MaxDealReplication = 10
)

// StorageRequestID is the type used for storage request identity.
type StorageRequestID string

// StorageRequest references a storage request for a Cid.
type StorageRequest struct {
	ID        StorageRequestID
	DataCid   cid.Cid
	Status    StorageRequestStatus
	BatchID   BatchID
	CreatedAt time.Time
	UpdatedAt time.Time
}

// StorageRequestInfo returns information about a storage request.
type StorageRequestInfo struct {
	StorageRequest StorageRequest
	Deals          []StorageRequestDeal
}

// StorageRequestDeal describes on-chain deals of a storage-request.
type StorageRequestDeal struct {
	MinerID    string
	DealID     int64
	Expiration uint64
}

// PreparedCAR contains information about prepared data.
type PreparedCAR struct {
	PieceCid  cid.Cid
	PieceSize uint64
	RepFactor int
	Deadline  time.Time
	Sources   auction.Sources
}

// StorageRequestStatus describe the current status of a
// StorageRequest.
type StorageRequestStatus int

const (
	// RequestUnknown is an invalid status value. Defined for safety.
	RequestUnknown StorageRequestStatus = iota
	// RequestBatching indicates that a storage request is being batched.
	RequestBatching
	// RequestPreparing indicates that a storage request is being prepared.
	RequestPreparing
	// RequestAuctioning indicates that a storage request is in bidding stage.
	RequestAuctioning
	// RequestDealMaking indicates that the storage request deals are being executed.
	RequestDealMaking
	// RequestSuccess indicates that the storage request was successfully stored in Filecoin.
	RequestSuccess
	// RequestError indicates that the storage request storage errored.
	RequestError
)

// String returns a string-encoded status.
func (brs StorageRequestStatus) String() string {
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
