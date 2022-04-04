package filclient

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	cborutil "github.com/filecoin-project/go-cbor-util"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/v7/actors/builtin/market"
	"github.com/ipfs/go-cid"
	"github.com/textileio/broker-core/metrics"
)

// GetChainHeight returns the current chain height.
func (fc *FilClient) GetChainHeight(ctx context.Context) (height uint64, err error) {
	defer func() {
		metrics.MetricIncrCounter(ctx, err, fc.metricGetChainHeight)
	}()
	tip, err := fc.api.ChainHead(ctx)
	if err != nil {
		return 0, fmt.Errorf("getting chain head: %s", err)
	}

	return uint64(tip.Height()), nil
}

// ResolveDealIDFromMessage looks for a publish deal message by its Cid and resolves the deal-id from the receipt.
// If the message isn't found, the return parameters are 0,nil.
// If the message is found, a deal-id is returned which can be trusted to match the original ProposalCid.
// This method is mostly what Estuary does with some tweaks, kudos to them!
func (fc *FilClient) ResolveDealIDFromMessage(
	ctx context.Context,
	proposalCid cid.Cid,
	publishDealMessage cid.Cid) (dealID int64, err error) {
	defer func() {
		metrics.MetricIncrCounter(ctx, err, fc.metricResolveDealIDFromMessage)
	}()
	mlookup, err := fc.api.StateSearchMsg(ctx, publishDealMessage)
	if err != nil {
		return 0, fmt.Errorf("could not find published deal on chain: %w", err)
	}

	if mlookup == nil {
		return 0, nil
	}

	msg, err := fc.api.ChainGetMessage(ctx, publishDealMessage)
	if err != nil {
		return 0, fmt.Errorf("get chain message; %s", err)
	}

	var params market.PublishStorageDealsParams
	if err := params.UnmarshalCBOR(bytes.NewReader(msg.Params)); err != nil {
		return 0, fmt.Errorf("unmarshaling publish storage deal params: %s", err)
	}

	dealix := -1
	for i, pd := range params.Deals {
		nd, err := cborutil.AsIpld(&pd)
		if err != nil {
			return 0, fmt.Errorf("failed to compute deal proposal ipld node: %w", err)
		}

		// If we find a proposal in the message that matches our AuctionDeal proposal cid, we can be sure
		// that this deal-id is the one we're looking for. The proposal cid summarizes the deal information.
		if nd.Cid() == proposalCid {
			dealix = i
			break
		}
	}

	if dealix == -1 {
		return 0, fmt.Errorf("deal isn't part of the published message")
	}

	if mlookup.Receipt.ExitCode != 0 {
		return 0, fmt.Errorf("the message failed to execute (exit: %s)", mlookup.Receipt.ExitCode)
	}

	var retval market.PublishStorageDealsReturn
	if err := retval.UnmarshalCBOR(bytes.NewReader(mlookup.Receipt.Return)); err != nil {
		return 0, fmt.Errorf("publish deal return was improperly formatted: %w", err)
	}

	if len(retval.IDs) != len(params.Deals) {
		return 0, fmt.Errorf("return value from publish deals did not match length of params")
	}

	return int64(retval.IDs[dealix]), nil
}

// CheckChainDeal checks if a deal is active on-chain. If that's the case, it also returns the
// deal expiration as a second parameter.
func (fc *FilClient) CheckChainDeal(
	ctx context.Context,
	dealid int64) (active bool, expiration uint64, slashed bool, err error) {
	log.Debugf("checking deal %d on-chain...", dealid)
	defer func() {
		metrics.MetricIncrCounter(ctx, err, fc.metricCheckChainDeal)
	}()
	deal, err := fc.api.StateMarketStorageDeal(ctx, abi.DealID(dealid), types.EmptyTSK)
	if err != nil {
		nfs := fmt.Sprintf("deal %d not found", dealid)
		if strings.Contains(err.Error(), nfs) {
			log.Debugf("deal %d still isn't on-chain", dealid)
			return false, 0, false, nil
		}

		return false, 0, false, fmt.Errorf("calling state market storage deal: %s", err)
	}

	if deal.State.SlashEpoch > 0 {
		log.Warnf("deal %d is on-chain but slashed", dealid)
		return false, 0, true, fmt.Errorf("is active on chain but slashed: %d", deal.State.SlashEpoch)
	}

	if deal.State.SectorStartEpoch > 0 {
		return true, uint64(deal.Proposal.EndEpoch), false, nil
	}

	return false, 0, false, nil
}
