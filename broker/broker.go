package broker

import (
	"context"
	"errors"
	"path"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
)

const (
	invalidStatus        = "invalid"
	epochsPerDay  uint64 = 60 * 24 * 2 // 1 epoch = ~30s

	// MinDealDuration is the minimum allowed deal duration in epochs requested of miners.
	MinDealDuration = epochsPerDay * 365 / 2 // ~6 months
	// MaxDealDuration is the maximum allowed deal duration in epochs requested of miners.
	MaxDealDuration = epochsPerDay * 365 // ~1 year
	// MinDealReplication is the minimum allowed deal replication requested of miners.
	MinDealReplication = 1
	// MaxDealReplication is the maximum allowed deal replication requested of miners.
	MaxDealReplication = 10
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

	// StorageDealFinalizedDeal signals to the broker results about deal making.
	StorageDealFinalizedDeal(ctx context.Context, fad FinalizedAuctionDeal) error

	// StorageDealProposalAccepted signals the broker that a miner has accepted a deal proposal.
	StorageDealProposalAccepted(ctx context.Context, sdID StorageDealID, miner string, proposalCid cid.Cid) error
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
	// StorageDealError indicates that the storage deal has errored.
	StorageDealError
)

// String returns a string-encoded status.
func (sds StorageDealStatus) String() string {
	switch sds {
	case StorageDealUnkown:
		return "unknown"
	case StorageDealPreparing:
		return "preparing"
	case StorageDealAuctioning:
		return "auctioning"
	case StorageDealDealMaking:
		return "deal making"
	case StorageDealSuccess:
		return "success"
	default:
		return invalidStatus
	}
}

// StorageDeal is the underlying entity that gets into bidding and
// store data in the Filecoin network. It groups one or multiple
// BrokerRequests.
type StorageDeal struct {
	ID               StorageDealID
	Status           StorageDealStatus
	BrokerRequestIDs []BrokerRequestID
	RepFactor        int
	DealDuration     int
	CreatedAt        time.Time
	UpdatedAt        time.Time
	Error            string

	// Packer calculates this field after batching storage requests.
	PayloadCid cid.Cid

	// Piecer calculates these fields after preparing the batched DAG.
	PieceCid  cid.Cid
	PieceSize uint64

	// Dealer populates this field
	Deals []MinerDeal
}

// MinerDeal contains information about a miner deal resulted from
// winned auctions:
// If ErrCause is not empty, is a failed deal.
// If ErrCause is empty, and DealID is zero then the deal is in progress.
// IF ErrCause is empty, and DealID is not zero then is final.
type MinerDeal struct {
	StorageDealID StorageDealID
	AuctionID     AuctionID
	BidID         BidID
	CreatedAt     time.Time
	UpdatedAt     time.Time

	Miner          string
	DealID         int64
	DealExpiration uint64
	ErrorCause     string
}

// DataPreparationResult is the result of preparing a StorageDeal.
type DataPreparationResult struct {
	PieceSize uint64
	PieceCid  cid.Cid
}

// Validate returns an error if the struct contain invalid fields.
func (dpr DataPreparationResult) Validate() error {
	if dpr.PieceSize == 0 {
		return errors.New("piece size is zero")
	}
	if !dpr.PieceCid.Defined() {
		return errors.New("piece cid is undefined")
	}
	return nil
}

// AuctionTopic is used by brokers to publish and by miners to subscribe to deal auctions.
const AuctionTopic string = "/textile/auction/0.0.1"

// BidsTopic is used by miners to submit deal auction bids.
// "/textile/auction/0.0.1/<auction_id>/bids".
func BidsTopic(auctionID AuctionID) string {
	return path.Join(AuctionTopic, string(auctionID), "bids")
}

// WinsTopic is used by brokers to notify a bidbot that it has won the deal auction.
// "/textile/auction/0.0.1/<peer_id>/wins".
func WinsTopic(pid peer.ID) string {
	return path.Join(AuctionTopic, pid.String(), "wins")
}

// ProposalsTopic is used by brokers to notify a bidbot of the proposal cid.Cid for an accepted deal auction.
// "/textile/auction/0.0.1/<peer_id>/proposals".
func ProposalsTopic(pid peer.ID) string {
	return path.Join(AuctionTopic, pid.String(), "proposals")
}

// AuctionID is a unique identifier for an Auction.
type AuctionID string

// Auction defines the core auction model.
type Auction struct {
	ID              AuctionID
	StorageDealID   StorageDealID
	DataURI         string
	DealSize        uint64
	DealDuration    uint64
	DealReplication uint32
	DealVerified    bool
	Status          AuctionStatus
	Bids            map[BidID]Bid
	WinningBids     map[BidID]WinningBid
	StartedAt       time.Time
	UpdatedAt       time.Time
	Duration        time.Duration
	Attempts        uint32
	ErrorCause      string
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
	// AuctionStatusFinalized indicates the auction has reached a final state.
	// If ErrorCause is empty, the auction has received a sufficient number of bids.
	// If ErrorCause is not empty, a fatal error has occurred and the auction should be considered abandoned.
	AuctionStatusFinalized
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
	case AuctionStatusFinalized:
		return "finalized"
	default:
		return invalidStatus
	}
}

// BidID is a unique identifier for a Bid.
type BidID string

// Bid defines the core bid model.
type Bid struct {
	MinerAddr        string
	WalletAddrSig    []byte
	BidderID         peer.ID
	AskPrice         int64 // attoFIL per GiB per epoch
	VerifiedAskPrice int64 // attoFIL per GiB per epoch
	StartEpoch       uint64
	FastRetrieval    bool
	ReceivedAt       time.Time
}

// WinningBid contains details about a winning bid.
type WinningBid struct {
	BidderID                peer.ID
	Acknowledged            bool // Whether or not the bidder acknowledged receipt of the win
	ProposalCid             cid.Cid
	ProposalCidAcknowledged bool // Whether or not the bidder acknowledged receipt of the proposal Cid
}

// FinalizedAuctionDeal contains information about a finalized deal.
type FinalizedAuctionDeal struct {
	StorageDealID  StorageDealID
	Miner          string
	DealID         int64
	DealExpiration uint64
	ErrorCause     string
}
