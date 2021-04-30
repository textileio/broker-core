package dealer

import (
	"context"

	"github.com/ipfs/go-cid"
	"github.com/textileio/broker-core/broker"
)

type Dealer interface {
	ReadyToCreateDeals(ctx context.Context, sdb AuctionDeals) error
}

type AuctionDeals struct {
	StorageDealID broker.StorageDealID
	PayloadCid    cid.Cid
	PieceCid      cid.Cid
	PieceSize     uint64
	Duration      uint64
	Targets       []AuctionDealsTarget
}

type AuctionDealsTarget struct {
	Miner               string
	PricePerGiBPerEpoch int64
	StartEpoch          uint64
	Verified            bool
	FastRetrieval       bool
}
