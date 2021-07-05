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

// BrokerRequestID is the type used for broker request identity.
type BrokerRequestID string

// BrokerRequest references a storage request for a Cid.
type BrokerRequest struct {
	ID            BrokerRequestID
	DataCid       cid.Cid
	Status        BrokerRequestStatus
	StorageDealID StorageDealID
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// BrokerRequestInfo returns information about a broker request.
type BrokerRequestInfo struct {
	BrokerRequest BrokerRequest
	Deals         []BrokerRequestDeal
}

// BrokerRequestDeal describes on-chain deals of a broker-request.
type BrokerRequestDeal struct {
	Miner      string
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
	// RequestDealMaking indicates that the broker request deals are being executed.
	RequestDealMaking
	// RequestSuccess indicates that the broker request was successfully stored in Filecoin.
	RequestSuccess
	// RequestError indicates that the broker request storage errored.
	RequestError
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
