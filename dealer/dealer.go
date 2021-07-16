package dealer

import (
	"context"

	"github.com/ipfs/go-cid"
	"github.com/textileio/broker-core/broker"
)

// Dealer creates deals in the Filecoin network.
type Dealer interface {
	ReadyToCreateDeals(ctx context.Context, sdb AuctionDeals) error
}

// AuctionDeals describes a set of deals for some prepared data.
type AuctionDeals struct {
	StorageDealID broker.StorageDealID
	PayloadCid    cid.Cid
	PieceCid      cid.Cid
	PieceSize     uint64
	Duration      uint64
	Proposals     []Proposal
}

// Proposal describes information for deal making with a miner.
type Proposal struct {
	Miner               string
	PricePerGiBPerEpoch int64
	StartEpoch          uint64
	Verified            bool
	FastRetrieval       bool
}
