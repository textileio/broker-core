package broker

import (
	"context"
	"path"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
)

const (
	epochsPerDay uint64 = 60 * 24 * 2 // 1 epoch = ~30s

	// MinDealEpochs is the minimum allowed deal duration requested of miners.
	MinDealEpochs = epochsPerDay * 365 / 2 // ~6 months
	// MaxDealEpochs is the maximum allowed deal duration requested of miners.
	MaxDealEpochs = epochsPerDay * 365 // ~1 year
)

// Broker provides full set of functionalities for Filecoin brokering.
type Broker interface {
	BrokerRequestor

	// CreateStorageDeal creates a new StorageDeal. It is called
	// by the Packer after batching a set of BrokerRequest properly.
	CreateStorageDeal(ctx context.Context, batchCid cid.Cid, srids []BrokerRequestID) (StorageDealID, error)

	// StorageDealPrepared signals the broker that a StorageDeal was prepared and it's ready to auction.
	StorageDealPrepared(ctx context.Context, id StorageDealID, pr DataPreparationResult) error

	// StorageDealAuctioned signals to the broker that StorageDeal auction has completed.
	StorageDealAuctioned(ctx context.Context, auction Auction) error
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

// AuctionTopic is used by brokers to publish and by miners to subscribe to deal auctions.
const AuctionTopic string = "/textile/auction/0.0.1"

// BidsTopic is used by miners to submit deal auction bids.
// "/textile/auction/0.0.1/<auction_id>/bids".
func BidsTopic(auctionID AuctionID) string {
	return path.Join(AuctionTopic, string(auctionID), "bids")
}

// WinsTopic is used by brokers to notify a miner thay have won the deal auction.
// "/textile/auction/0.0.1/<peer_id>/wins".
func WinsTopic(pid peer.ID) string {
	return path.Join(AuctionTopic, pid.String(), "wins")
}

// AuctionID is a unique identifier for an Auction.
type AuctionID string

// Auction defines the core auction model.
type Auction struct {
	ID           AuctionID
	DealID       string
	DealSize     uint64
	DealDuration uint64
	Status       AuctionStatus
	Bids         map[BidID]Bid
	WinningBids  []BidID
	StartedAt    time.Time
	Duration     time.Duration
	Error        string
}

// AuctionStatus is the status of an auction.
type AuctionStatus int

const (
	// AuctionStatusUnspecified indicates the initial or invalid status of an auction.
	AuctionStatusUnspecified AuctionStatus = iota
	// AuctionStatusQueued indicates the auction is currently queued.
	AuctionStatusQueued
	// AuctionStatusStarted indicates the auction has started.
	AuctionStatusStarted
	// AuctionStatusEnded indicates the auction has ended.
	AuctionStatusEnded
	// AuctionStatusError indicates the auction resulted in an error.
	AuctionStatusError
)

// String returns a string-encoded status.
func (as AuctionStatus) String() string {
	switch as {
	case AuctionStatusUnspecified:
		return "unspecified"
	case AuctionStatusQueued:
		return "queued"
	case AuctionStatusStarted:
		return "started"
	case AuctionStatusEnded:
		return "ended"
	case AuctionStatusError:
		return "error"
	default:
		return "invalid"
	}
}

// BidID is a unique identifier for a Bid.
type BidID string

// Bid defines the core bid model.
type Bid struct {
	Broker     peer.ID
	From       peer.ID
	AskPrice   int64 // attoFIL per GiB per epoch
	ReceivedAt time.Time
}
