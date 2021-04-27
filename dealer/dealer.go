package dealer

import (
	"context"

	"github.com/ipfs/go-cid"
	"github.com/textileio/broker-core/broker"
)

type Dealer interface {
	ReadyToExecuteBids(ctx context.Context)
}

type StorageDealBundle struct {
	StorageDealID broker.StorageDealID
	PayloadCid    cid.Cid
	PieceCid      cid.Cid
	PaddedSize    int64
}
