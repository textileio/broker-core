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
	StorageDealID   auction.StorageDealID
	DealDuration    uint64
	DealReplication uint32
	DealVerified    bool
	Status          auction.AuctionStatus
	WinningBids     map[auction.BidID]WinningBid
	ErrorCause      string
}

// Broker provides full set of functionalities for Filecoin brokering.
type Broker interface {
	// Create creates a new BrokerRequest for a cid.
	Create(ctx context.Context, dataCid cid.Cid) (BrokerRequest, error)

	// CreatePrepared creates a new BrokerRequest for prepared data.
	CreatePrepared(ctx context.Context, payloadCid cid.Cid, pc PreparedCAR) (BrokerRequest, error)

	// GetBrokerRequestInfo returns a broker request information by id.
	GetBrokerRequestInfo(ctx context.Context, ID BrokerRequestID) (BrokerRequestInfo, error)

	// CreateStorageDeal creates a new StorageDeal. It is called
	// by the Packer after batching a set of BrokerRequest properly.
	CreateStorageDeal(ctx context.Context, batchCid cid.Cid, srids []BrokerRequestID) (auction.StorageDealID, error)

	// StorageDealPrepared signals the broker that a StorageDeal was prepared and it's ready to auction.
	StorageDealPrepared(ctx context.Context, id auction.StorageDealID, pr DataPreparationResult) error

	// StorageDealAuctioned signals to the broker that StorageDeal auction has completed.
	StorageDealAuctioned(ctx context.Context, auction ClosedAuction) error

	// StorageDealFinalizedDeal signals to the broker results about deal making.
	StorageDealFinalizedDeal(ctx context.Context, fad FinalizedAuctionDeal) error

	// StorageDealProposalAccepted signals the broker that a miner has accepted a deal proposal.
	StorageDealProposalAccepted(ctx context.Context, sdID auction.StorageDealID, miner string, proposalCid cid.Cid) error
}

// StorageDeal is the underlying entity that gets into bidding and
// store data in the Filecoin network. It groups one or multiple
// BrokerRequests.
type StorageDeal struct {
	ID                 auction.StorageDealID
	Status             StorageDealStatus
	BrokerRequestIDs   []BrokerRequestID
	RepFactor          int
	DealDuration       int
	Sources            auction.Sources
	DisallowRebatching bool
	AuctionRetries     int
	FilEpochDeadline   uint64
	CreatedAt          time.Time
	UpdatedAt          time.Time
	Error              string

	// Packer calculates this field after batching storage requests.
	PayloadCid cid.Cid

	// Piecer calculates these fields after preparing the batched DAG.
	PieceCid  cid.Cid
	PieceSize uint64

	// Dealer populates this field
	Deals []MinerDeal
}

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
	StorageDealID auction.StorageDealID
	AuctionID     auction.AuctionID
	BidID         auction.BidID
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

// FinalizedAuctionDeal contains information about a finalized deal.
type FinalizedAuctionDeal struct {
	StorageDealID  auction.StorageDealID
	Miner          string
	DealID         int64
	DealExpiration uint64
	ErrorCause     string
}
