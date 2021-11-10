package filapi

import (
	"context"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
	logger "github.com/textileio/go-log/v2"
)

var (
	log = logger.Logger("dealer/filapi")
)

type retryable struct {
	apis []FilAPI
}

// Retryable has a list of APIs and implements a retry strategy in case of error.
func Retryable(apis []FilAPI) FilAPI {
	return retryable{apis}
}

func (r retryable) ChainHead(ctx context.Context) (ts *types.TipSet, err error) {
	r.callWithRetry(func(index int) error {
		ts, err = r.apis[index].ChainHead(ctx)
		return err
	})
	return ts, err
}

func (r retryable) StateSearchMsg(ctx context.Context, cid cid.Cid) (ml *api.MsgLookup, err error) {
	r.callWithRetry(func(index int) error {
		ml, err = r.apis[index].StateSearchMsg(ctx, cid)
		return err
	})
	return ml, err
}

func (r retryable) ChainGetMessage(ctx context.Context, cid cid.Cid) (msg *types.Message, err error) {
	r.callWithRetry(func(index int) error {
		msg, err = r.apis[index].ChainGetMessage(ctx, cid)
		return err
	})
	return msg, err
}

func (r retryable) StateMarketStorageDeal(ctx context.Context, dealID abi.DealID,
	tsk types.TipSetKey) (md *api.MarketDeal, err error) {
	r.callWithRetry(func(index int) error {
		md, err = r.apis[index].StateMarketStorageDeal(ctx, dealID, tsk)
		return err
	})

	return md, err
}

func (r retryable) StateDealProviderCollateralBounds(ctx context.Context, pieceSize abi.PaddedPieceSize,
	verified bool, tsk types.TipSetKey) (dcb api.DealCollateralBounds, err error) {
	r.callWithRetry(func(index int) error {
		dcb, err = r.apis[index].StateDealProviderCollateralBounds(ctx, pieceSize, verified, tsk)
		return err
	})

	return dcb, err
}

func (r retryable) StateMinerInfo(ctx context.Context, addr address.Address,
	tsk types.TipSetKey) (mi miner.MinerInfo, err error) {
	r.callWithRetry(func(index int) error {
		mi, err = r.apis[index].StateMinerInfo(ctx, addr, tsk)
		return err
	})

	return mi, err
}

func (r retryable) NetFindPeer(ctx context.Context, peerID peer.ID) (ai peer.AddrInfo, err error) {
	r.callWithRetry(func(index int) error {
		ai, err = r.apis[index].NetFindPeer(ctx, peerID)
		return err
	})

	return ai, err
}

func (r retryable) hasMoreAttempts(index int) bool {
	return index < len(r.apis)-1
}

// it starts at the first API and goes until it succeeds or there are no more APIs to try.
func (r retryable) callWithRetry(f func(index int) error) {
	currentAttempt := 0
	err := f(currentAttempt)
	for err != nil && r.hasMoreAttempts(currentAttempt) {
		log.Warnf("call to gateway %d failed. retrying...: %s", currentAttempt, err)
		currentAttempt++
		err = f(currentAttempt)
	}
}
