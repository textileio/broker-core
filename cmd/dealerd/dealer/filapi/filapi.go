package filapi

import (
	"context"
	"errors"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/actors/builtin/miner"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/libp2p/go-libp2p-core/routing"
)

// FilAPI contains the minimum set of Filecoin network API used by fil client.
type FilAPI interface {
	ChainHead(context.Context) (*types.TipSet, error)
	StateSearchMsg(context.Context, cid.Cid) (*api.MsgLookup, error)
	ChainGetMessage(context.Context, cid.Cid) (*types.Message, error)
	StateMarketStorageDeal(context.Context, abi.DealID, types.TipSetKey) (*api.MarketDeal, error)
	StateDealProviderCollateralBounds(context.Context, abi.PaddedPieceSize, bool,
		types.TipSetKey) (api.DealCollateralBounds, error)
	StateMinerInfo(context.Context, address.Address, types.TipSetKey) (miner.MinerInfo, error)
	NetFindPeer(context.Context, peer.ID) (peer.AddrInfo, error)
}

// MetricsCollector is how the FilAPI metrics are collected.
type MetricsCollector func(ctx context.Context, methodName string, err error, duration time.Duration)

type measured struct {
	wrapped   FilAPI
	collector MetricsCollector
}

// Measured wraps FilAPI to measure the number and duration of each API calls.
func Measured(api FilAPI, collector MetricsCollector) FilAPI {
	return measured{api, collector}
}

func (m measured) ChainHead(ctx context.Context) (*types.TipSet, error) {
	start := time.Now()
	ts, err := m.wrapped.ChainHead(ctx)
	m.collector(ctx, "ChainHead", err, time.Since(start))
	return ts, err
}
func (m measured) StateSearchMsg(ctx context.Context, cid cid.Cid) (*api.MsgLookup, error) {
	start := time.Now()
	ml, err := m.wrapped.StateSearchMsg(ctx, cid)
	m.collector(ctx, "StateSearchMsg", err, time.Since(start))
	return ml, err
}
func (m measured) ChainGetMessage(ctx context.Context, cid cid.Cid) (*types.Message, error) {
	start := time.Now()
	msg, err := m.wrapped.ChainGetMessage(ctx, cid)
	m.collector(ctx, "ChainGetMessage", err, time.Since(start))
	return msg, err
}
func (m measured) StateMarketStorageDeal(ctx context.Context, dealID abi.DealID,
	tsk types.TipSetKey) (*api.MarketDeal, error) {
	start := time.Now()
	md, err := m.wrapped.StateMarketStorageDeal(ctx, dealID, tsk)
	m.collector(ctx, "StateMarketStorageDeal", err, time.Since(start))
	return md, err
}
func (m measured) StateDealProviderCollateralBounds(ctx context.Context, pieceSize abi.PaddedPieceSize,
	verified bool, tsk types.TipSetKey) (api.DealCollateralBounds, error) {
	start := time.Now()
	dcb, err := m.wrapped.StateDealProviderCollateralBounds(ctx, pieceSize, verified, tsk)
	m.collector(ctx, "StateDealProviderCollateralBounds", err, time.Since(start))
	return dcb, err
}
func (m measured) StateMinerInfo(ctx context.Context, addr address.Address,
	tsk types.TipSetKey) (miner.MinerInfo, error) {
	start := time.Now()
	mi, err := m.wrapped.StateMinerInfo(ctx, addr, tsk)
	m.collector(ctx, "StateMinerInfo", err, time.Since(start))
	return mi, err
}
func (m measured) NetFindPeer(ctx context.Context, peerID peer.ID) (peer.AddrInfo, error) {
	start := time.Now()
	ai, err := m.wrapped.NetFindPeer(ctx, peerID)
	if errors.Is(err, routing.ErrNotFound) {
		// this shouldn't be considered as an API error
		m.collector(ctx, "NetFindPeer", nil, time.Since(start))
	} else {
		m.collector(ctx, "NetFindPeer", err, time.Since(start))
	}
	return ai, err
}
