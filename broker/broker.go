package broker

import (
	"context"
	"errors"
	"time"

	"github.com/ipfs/go-cid"

	"github.com/textileio/bidbot/lib/auction"
)

const (
	invalidStatus = "invalid"
)

// WinningBid contains details about a winning bid in a closed auction.
type WinningBid struct {
	MinerAddr     string
	Price         int64
	StartEpoch    uint64
	FastRetrieval bool
}

// ClosedAuction contains closed auction details auctioneer reports back to the broker.
type ClosedAuction struct {
	ID              auction.AuctionID
	StorageDealID   StorageDealID
	DealDuration    uint64
	DealReplication uint32
	DealVerified    bool
	Status          AuctionStatus
	WinningBids     map[auction.BidID]WinningBid
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
		return "invalid"
	}
}

// Broker provides full set of functionalities for Filecoin brokering.
type Broker interface {
	// Create creates a new BrokerRequest for a cid.
	Create(ctx context.Context, dataCid cid.Cid) (BrokerRequest, error)

	// CreatePrepared creates a new BrokerRequest for prepared data.
	CreatePrepared(ctx context.Context, payloadCid cid.Cid, pc PreparedCAR) (BrokerRequest, error)

	// GetBrokerRequestInfo returns a broker request information by id.
	GetBrokerRequestInfo(ctx context.Context, ID BrokerRequestID) (BrokerRequestInfo, error)

	// StorageDealAuctioned signals to the broker that StorageDeal auction has completed.
	StorageDealAuctioned(ctx context.Context, auction ClosedAuction) error

	// StorageDealFinalizedDeal signals to the broker results about deal making.
	StorageDealFinalizedDeal(ctx context.Context, fad FinalizedAuctionDeal) error

	// StorageDealProposalAccepted signals the broker that a miner has accepted a deal proposal.
	StorageDealProposalAccepted(ctx context.Context, sdID StorageDealID, miner string, proposalCid cid.Cid) error
}

// StorageDeal is the underlying entity that gets into bidding and
// store data in the Filecoin network. It groups one or multiple
// BrokerRequests.
type StorageDeal struct {
	ID                 StorageDealID
	Status             StorageDealStatus
	RepFactor          int
	DealDuration       int
	Sources            auction.Sources
	DisallowRebatching bool
	AuctionRetries     int
	FilEpochDeadline   uint64
	Error              string

	// Packer calculates this field after batching storage requests.
	PayloadCid cid.Cid

	// Piecer calculates these fields after preparing the batched DAG.
	PieceCid  cid.Cid
	PieceSize uint64
	CreatedAt time.Time
	UpdatedAt time.Time
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

// MinerDeal contains information about a miner deal resulted from
// winned auctions:
// If ErrCause is not empty, is a failed deal.
// If ErrCause is empty, and DealID is zero then the deal is in progress.
// IF ErrCause is empty, and DealID is not zero then is final.
type MinerDeal struct {
	StorageDealID StorageDealID
	AuctionID     auction.AuctionID
	BidID         auction.BidID

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

// FinalizedAuctionDeal contains information about a finalized deal.
type FinalizedAuctionDeal struct {
	StorageDealID  StorageDealID
	Miner          string
	DealID         int64
	DealExpiration uint64
	ErrorCause     string
}
