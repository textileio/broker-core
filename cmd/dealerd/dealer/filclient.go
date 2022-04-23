package dealer

import (
	"context"

	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/ipfs/go-cid"
	"github.com/textileio/broker-core/cmd/dealerd/store"
)

// FilClient provides functionalities to create and monitor Filecoin deals.
type FilClient interface {
	ExecuteAuctionDeal(
		ctx context.Context,
		ad store.AuctionData,
		aud store.AuctionDeal,
		rw *store.RemoteWallet,
		allowBoost bool) (string, bool, error)
	GetChainHeight(ctx context.Context) (uint64, error)
	ResolveDealIDFromMessage(
		ctx context.Context,
		publishDealMessage cid.Cid,
		spID string,
		pieceCid cid.Cid,
		startEpoch uint64) (int64, error)
	CheckChainDeal(ctx context.Context, dealID int64) (bool, uint64, bool, error)
	CheckDealStatusWithStorageProvider(
		ctx context.Context,
		storageProviderID string,
		dealIdentifier string,
		rw *store.RemoteWallet) (*cid.Cid, storagemarket.StorageDealStatus, error)
}
