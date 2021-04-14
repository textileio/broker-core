package piecer

import (
	"context"

	"github.com/ipfs/go-cid"
	"github.com/textileio/broker-core/broker"
)

type Piecer interface {
	ReadyToPrepare(ctx context.Context, id broker.StorageDealID, c cid.Cid) error
}
