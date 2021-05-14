package chainapi

import (
	"context"

	"github.com/ipfs/go-cid"
)

// ChainAPI provides blockchain interactions.
type ChainAPI interface {
	HasDeposit(ctx context.Context, brokerID, accountID string) (bool, error)
	UpdatePayload(ctx context.Context, payloadCid cid.Cid, opts ...UpdatePayloadOption) error
}

// UpdatePayloadOptions holds the various payload properties that can be updated.
type UpdatePayloadOptions struct {
	PieceCid *cid.Cid
	Deals    []DealInfo
	DataCids []cid.Cid
}

// UpdatePayloadOption updates a updatePayloadOptions.
type UpdatePayloadOption func(*UpdatePayloadOptions)

// UpdatePayloadWithPieceCid allows you to update the payload piece cid.
func UpdatePayloadWithPieceCid(pieceCid cid.Cid) UpdatePayloadOption {
	return func(upo *UpdatePayloadOptions) {
		upo.PieceCid = &pieceCid
	}
}

// UpdatePayloadWithDeals allows you to add deals to the payload.
func UpdatePayloadWithDeals(deals []DealInfo) UpdatePayloadOption {
	return func(upo *UpdatePayloadOptions) {
		upo.Deals = deals
	}
}

// UpdatePayloadWithDataCids allows you to associate more data cids with a payload.
func UpdatePayloadWithDataCids(dataCids []cid.Cid) UpdatePayloadOption {
	return func(upo *UpdatePayloadOptions) {
		upo.DataCids = dataCids
	}
}

// DealInfo contains information from an active deal.
type DealInfo struct {
	DealID     int64
	MinerID    string
	Expiration uint64
}
