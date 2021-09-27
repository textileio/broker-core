package auctioneer

import (
	"time"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/textileio/bidbot/lib/auction"
	"github.com/textileio/broker-core/broker"
)

// Auction defines the core auction model.
type Auction struct {
	ID                       auction.ID
	BatchID                  broker.BatchID
	PayloadCid               cid.Cid
	DealSize                 uint64
	DealDuration             uint64
	DealReplication          uint32
	DealVerified             bool
	FilEpochDeadline         uint64
	ExcludedStorageProviders []string
	Sources                  auction.Sources
	ClientAddress            string
	Status                   broker.AuctionStatus
	Bids                     map[auction.BidID]Bid
	WinningBids              map[auction.BidID]WinningBid
	StartedAt                time.Time
	UpdatedAt                time.Time
	Duration                 time.Duration
	ErrorCause               string
}

// Bid defines the core bid model.
type Bid struct {
	ID                       auction.BidID
	StorageProviderID        string
	WalletAddrSig            []byte
	BidderID                 peer.ID
	AskPrice                 int64 // attoFIL per GiB per epoch
	VerifiedAskPrice         int64 // attoFIL per GiB per epoch
	StartEpoch               uint64
	FastRetrieval            bool
	ReceivedAt               time.Time
	WonAt                    time.Time
	ProposalCid              cid.Cid
	ProposalCidDeliveredAt   time.Time
	ProposalCidDeliveryError string
}

// WinningBid contains details about a winning bid.
type WinningBid struct {
	BidderID    peer.ID
	ProposalCid cid.Cid
	ErrorCause  string // an error that may have occurred when delivering the proposal cid
}
