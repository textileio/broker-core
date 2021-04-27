package dealer

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-address"
	cborutil "github.com/filecoin-project/go-cbor-util"
	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/filecoin-project/go-fil-markets/storagemarket/impl/clientutils"
	"github.com/filecoin-project/go-fil-markets/storagemarket/network"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/actors/builtin/market"
	"github.com/textileio/broker-core/cmd/dealerd/dealer/store"
)

func (d *Dealer) executeBid(ctx context.Context, ad store.AuctionData, aud store.AuctionDeal) (*network.Proposal, error) {
	collBounds, err := d.gateway.StateDealProviderCollateralBounds(ctx, abi.PaddedPieceSize(ad.PieceSize), aud.Verified, types.EmptyTSK)
	if err != nil {
		return nil, fmt.Errorf("calculating provider collateral: %s", err)
	}

	// set provider collateral 10% above minimum to avoid fluctuations causing deal failure
	provCol := big.Div(big.Mul(collBounds.Min, big.NewInt(11)), big.NewInt(10))

	dealStart := aud.StartEpoch

	end := dealStart + ad.Duration

	pricePerEpoch := big.Div(big.Mul(big.NewInt(ad.PieceSize), big.NewInt(aud.PricePerGiBPerEpoch)), big.NewInt(1<<30))

	label, err := clientutils.LabelField(ad.PayloadCid)
	if err != nil {
		return nil, fmt.Errorf("failed to construct label field: %w", err)
	}

	miner, err := address.NewFromString(aud.Miner)
	if err != nil {
		return nil, fmt.Errorf("parsing miner address: %s", err)
	}
	proposal := &market.DealProposal{
		PieceCID:     ad.PieceCid,
		PieceSize:    abi.PaddedPieceSize(ad.PieceSize), // Check padding vs not padding.
		VerifiedDeal: aud.Verified,
		Client:       d.walletAddr,
		Provider:     miner,

		Label: label,

		StartEpoch: abi.ChainEpoch(dealStart),
		EndEpoch:   abi.ChainEpoch(end),

		StoragePricePerEpoch: abi.TokenAmount(pricePerEpoch),
		ProviderCollateral:   provCol,
		ClientCollateral:     big.Zero(),
	}

	raw, err := cborutil.Dump(proposal)
	if err != nil {
		return nil, err
	}
	sig, err := d.wallet.WalletSign(ctx, d.walletAddr, raw, api.MsgMeta{Type: api.MTDealProposal})
	if err != nil {
		return nil, err
	}

	sigprop := &market.ClientDealProposal{
		Proposal:        *proposal,
		ClientSignature: *sig,
	}

	return &network.Proposal{
		DealProposal: sigprop,
		Piece: &storagemarket.DataRef{
			TransferType: storagemarket.TTManual,
			Root:         ad.PayloadCid,
		},
		FastRetrieval: true,
	}, nil
}
