package reporter

import (
	"context"

	"github.com/ipfs/go-cid"
)

// Reporter allows to publish updated deal information on-chain.
type Reporter interface {
	ReportStorageInfo(
		ctx context.Context,
		payloadCid cid.Cid,
		pieceCid cid.Cid,
		deals []DealInfo,
		dataCids []cid.Cid,
	) error
}

// DealInfo contains information from an active deal.
type DealInfo struct {
	DealID     int64
	MinerID    string
	Expiration uint64
}
