package dealer

import (
	"context"

	"github.com/ipfs/go-cid"
	"github.com/textileio/bidbot/lib/auction"
)

// Dealer creates deals in the Filecoin network.
type Dealer interface {
	ReadyToCreateDeals(ctx context.Context, sdb AuctionDeals) error
}

// AuctionDeals describes a set of deals for some prepared data.
type AuctionDeals struct {
	StorageDealID auction.StorageDealID
	PayloadCid    cid.Cid
	PieceCid      cid.Cid
	PieceSize     uint64
	Duration      uint64
	Targets       []AuctionDealsTarget
}

// AuctionDealsTarget describes a target miner for making deals.
type AuctionDealsTarget struct {
	Miner               string
	PricePerGiBPerEpoch int64
	StartEpoch          uint64
	Verified            bool
	FastRetrieval       bool
}
