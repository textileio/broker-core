package broker

import (
	"context"
	"errors"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/ipfs/go-cid"
	"github.com/textileio/bidbot/lib/auction"
)

const (
	invalidStatus = "invalid"
)

// WinningBid contains details about a winning bid in a closed auction.
type WinningBid struct {
	StorageProviderID string
	Price             int64
	StartEpoch        uint64
	FastRetrieval     bool
}

// ClosedAuction contains closed auction details auctioneer reports back to the broker.
type ClosedAuction struct {
	ID              auction.ID
	BatchID         BatchID
	DealDuration    uint64
	DealReplication uint32
	DealVerified    bool
	Status          AuctionStatus
	WinningBids     map[auction.BidID]WinningBid
	ErrorCause      string
}

// AuctionStatus is the status of an auction.
type AuctionStatus string

const (
	// AuctionStatusUnspecified indicates the initial or invalid status of an auction.
	AuctionStatusUnspecified AuctionStatus = ""
	// AuctionStatusQueued indicates the auction is currently queued.
	AuctionStatusQueued AuctionStatus = "queued"
	// AuctionStatusStarted indicates the auction has started.
	AuctionStatusStarted AuctionStatus = "started"
	// AuctionStatusFinalized indicates the auction has reached a final state.
	// If ErrorCause is empty, the auction has received a sufficient number of bids.
	// If ErrorCause is not empty, a fatal error has occurred and the auction should be considered abandoned.
	AuctionStatusFinalized AuctionStatus = "finalized"
)

// Broker provides full set of functionalities for Filecoin brokering.
type Broker interface {
	// Create creates a new StorageRequest for a cid.
	Create(ctx context.Context, dataCid cid.Cid, origin string) (StorageRequest, error)

	// CreatePrepared creates a new StorageRequest for prepared data.
	CreatePrepared(
		ctx context.Context,
		payloadCid cid.Cid,
		pc PreparedCAR,
		m BatchMetadata,
		rw *RemoteWallet) (StorageRequest, error)

	// GetStorageRequestInfo returns a storage request information by id.
	GetStorageRequestInfo(ctx context.Context, ID StorageRequestID) (StorageRequestInfo, error)
}

// Batch is the underlying entity that gets into bidding and
// store data in the Filecoin network. It groups one or multiple
// StorageRequests.
type Batch struct {
	ID                 BatchID
	Status             BatchStatus
	RepFactor          int
	DealDuration       int
	Sources            auction.Sources
	DisallowRebatching bool
	FilEpochDeadline   uint64
	Error              string
	Origin             string
	Tags               map[string]string
	Providers          []address.Address

	// Packer calculates this field after batching storage requests.
	PayloadCid  cid.Cid
	PayloadSize *int64

	// Piecer calculates these fields after preparing the batched DAG.
	PieceCid  cid.Cid
	PieceSize uint64
	CreatedAt time.Time
	UpdatedAt time.Time
}

// BatchID is the type of a batch identifier.
type BatchID string

// BatchStatus is the type of a broker status.
type BatchStatus string

const (
	// BatchStatusUnkown is an invalid status value. Defined for safety.
	BatchStatusUnkown BatchStatus = "unknown"
	// BatchStatusPreparing indicates that the storage deal is being prepared.
	BatchStatusPreparing BatchStatus = "preparing"
	// BatchStatusAuctioning indicates that the storage deal is being auctioned.
	BatchStatusAuctioning BatchStatus = "auctioning"
	// BatchStatusDealMaking indicates that the storage deal deals are being executed.
	BatchStatusDealMaking BatchStatus = "deal_making"
	// BatchStatusSuccess indicates that the storage deal was successfully stored in Filecoin.
	BatchStatusSuccess BatchStatus = "success"
	// BatchStatusError indicates that the storage deal has errored.
	BatchStatusError BatchStatus = "error"
)

// DataPreparationResult is the result of preparing a batch.
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

// FinalizedDeal contains information about a finalized deal.
type FinalizedDeal struct {
	BatchID           BatchID
	StorageProviderID string
	DealID            int64
	DealExpiration    uint64
	ErrorCause        string
	AuctionID         auction.ID
	BidID             auction.BidID
}
