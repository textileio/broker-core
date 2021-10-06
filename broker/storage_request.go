package broker

import (
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/textileio/bidbot/lib/auction"
)

const (
	// CodecFilCommitmentUnsealed is the IPLD codec for PieceCid cids.
	CodecFilCommitmentUnsealed = 0xf101
	// DefaultPreparedCARDeadline is the default deadline for prepared CAR deals.
	DefaultPreparedCARDeadline = time.Hour * 48
	// MaxPieceSize is the maximum piece size accepted for prepared data.
	MaxPieceSize = 32 << 30
	// MinDealReplication is the minimum allowed deal replication requested of storage-providers.
	MinDealReplication = 1
	// MaxDealReplication is the maximum allowed deal replication requested of storage-providers.
	MaxDealReplication = 10
)

// StorageRequestID is the type used for storage request identity.
type StorageRequestID string

// StorageRequest references a storage request for a Cid.
type StorageRequest struct {
	ID        StorageRequestID
	DataCid   cid.Cid
	Status    StorageRequestStatus
	Origin    string
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
	StorageProviderID string
	DealID            int64
	Expiration        uint64
}

// PreparedCAR contains information about prepared data.
type PreparedCAR struct {
	PieceCid  cid.Cid
	PieceSize uint64
	RepFactor int
	Deadline  time.Time
	Sources   auction.Sources
}

// BatchMetadata is metadata about a batch.
type BatchMetadata struct {
	Origin    string
	Tags      map[string]string
	Providers []address.Address
}

// RemoteWallet contains configuration of a remote wallet.
type RemoteWallet struct {
	PeerID     peer.ID
	WalletAddr address.Address
	AuthToken  string
	Multiaddrs []multiaddr.Multiaddr
}

// StorageRequestStatus describe the current status of a
// StorageRequest.
type StorageRequestStatus string

const (
	// RequestUnknown is an invalid status value. Defined for safety.
	RequestUnknown StorageRequestStatus = "unknown"
	// RequestBatching indicates that a storage request is being batched.
	RequestBatching StorageRequestStatus = "batching"
	// RequestPreparing indicates that a storage request is being prepared.
	RequestPreparing StorageRequestStatus = "preparing"
	// RequestAuctioning indicates that a storage request is in bidding stage.
	RequestAuctioning StorageRequestStatus = "auctioning"
	// RequestDealMaking indicates that the storage request deals are being executed.
	RequestDealMaking StorageRequestStatus = "deal_making"
	// RequestSuccess indicates that the storage request was successfully stored in Filecoin.
	RequestSuccess StorageRequestStatus = "success"
	// RequestError indicates that the storage request storage errored.
	RequestError StorageRequestStatus = "error"
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
