package dealer

import (
	"context"

	"github.com/ipfs/go-cid"
)

type Dealer interface {
	ReadyToCreateDeals(ctx context.Context, sdb AuctionDeals) error
}

type AuctionDeals struct {
	AuctionID  string // TODO: would be good to make a type
	PayloadCid cid.Cid
	PieceCid   cid.Cid
	PieceSize  int64
	Duration   int64
	Targets    []AuctionDealsTarget
}

type AuctionDealsTarget struct {
	Miner               string
	PricePerGiBPerEpoch int64
	StartEpoch          int64
	Verified            bool
	FastRetrieval       bool
}
