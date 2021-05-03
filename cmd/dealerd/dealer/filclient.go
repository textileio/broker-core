package dealer

import (
	"context"

	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/ipfs/go-cid"
	"github.com/textileio/broker-core/cmd/dealerd/dealer/store"
)

type Filclient interface {
	ExecuteAuctionDeal(ctx context.Context, ad store.AuctionData, aud store.AuctionDeal) (cid.Cid, error)
	GetChainHeight(ctx context.Context) (uint64, error)
	ResolveDealIDFromMessage(ctx context.Context, proposalCid cid.Cid, publishDealMessage cid.Cid) (int64, error)
	CheckChainDeal(ctx context.Context, dealid int64) (bool, uint64, error)
	CheckDealStatusWithMiner(
		ctx context.Context,
		minerAddr string,
		propCid cid.Cid) (*storagemarket.ProviderDealState, error)
}
