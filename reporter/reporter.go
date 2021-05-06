package reporter

import (
	"context"

	"github.com/ipfs/go-cid"
)

type Reporter interface {
	ReportStorageInfo(
		ctx context.Context,
		payloadCid cid.Cid,
		pieceCid cid.Cid,
		deals []DealInfo,
		dataCids []cid.Cid,
	) error
}

type DealInfo struct {
	DealID     int64
	MinerID    string
	Expiration uint64
}
