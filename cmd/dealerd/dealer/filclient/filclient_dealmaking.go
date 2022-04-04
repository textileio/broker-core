package filclient

import (
	"context"
	"fmt"

	"github.com/dustin/go-humanize"
	"github.com/filecoin-project/go-address"
	cborutil "github.com/filecoin-project/go-cbor-util"
	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/filecoin-project/go-fil-markets/storagemarket/network"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/v7/actors/builtin/market"
	"github.com/ipfs/go-cid"
	"github.com/jsign/go-filsigner/wallet"
	"github.com/textileio/broker-core/cmd/dealerd/store"
	"github.com/textileio/broker-core/metrics"
	"github.com/textileio/go-auctions-client/propsigner"
	logger "github.com/textileio/go-log/v2"
	"go.opentelemetry.io/otel/attribute"
)

const dealProtocol = "/fil/storage/mk/1.1.0"

// ExecuteAuctionDeal creates a deal with a storage-provider using the data described in an auction deal.
func (fc *FilClient) ExecuteAuctionDeal(
	ctx context.Context,
	ad store.AuctionData,
	aud store.AuctionDeal,
	rw *store.RemoteWallet) (propCid cid.Cid, retriable bool, err error) {
	log.Debugf(
		"executing auction deal for data-cid %s, piece-cid %s and size %s...",
		ad.PayloadCid, ad.PieceCid, humanize.IBytes(ad.PieceSize))
	defer func() {
		var attrs []attribute.KeyValue
		if rw != nil {
			attrs = []attribute.KeyValue{attrWalletSignature.String(rw.WalletAddr), attrRemoteWallet}
		} else {
			attrs = []attribute.KeyValue{attrWalletSignature.String(fc.conf.pubKey.String()), attrLocalWallet}
		}
		metrics.MetricIncrCounter(ctx, err, fc.metricExecAuctionDeal, attrs...)
	}()

	p, err := fc.createDealProposal(ctx, ad, aud, rw)
	if err != nil {
		// Any error here deserves retries.
		log.Errorf("creating deal proposal: %s", err)
		return cid.Undef, true, nil
	}
	log.Debugf("created proposal (remote-wallet: %t): %s", rw != nil, logger.MustJSONIndent(p))
	pr, err := fc.sendProposal(ctx, p)
	if err != nil {
		log.Errorf("sending proposal to storage-provider: %s", err)
		// Any error here deserves retries.
		return cid.Undef, true, nil
	}

	switch pr.Response.State {
	case storagemarket.StorageDealWaitingForData, storagemarket.StorageDealProposalAccepted:
		log.Debugf("proposal %s accepted: %s", pr.Response.Proposal, logger.MustJSONIndent(p))
	default:
		log.Warnf("proposal failed: %s", pr.Response.Proposal, logger.MustJSONIndent(p))
		return cid.Undef,
			false,
			fmt.Errorf("failed proposal (%s): %s",
				storagemarket.DealStates[pr.Response.State],
				pr.Response.Message)
	}

	return pr.Response.Proposal, false, nil
}

func (fc *FilClient) createDealProposal(
	ctx context.Context,
	ad store.AuctionData,
	aud store.AuctionDeal,
	rw *store.RemoteWallet) (*network.Proposal, error) {
	collBounds, err := fc.api.StateDealProviderCollateralBounds(
		ctx,
		abi.PaddedPieceSize(ad.PieceSize),
		aud.Verified,
		types.EmptyTSK)
	if err != nil {
		return nil, fmt.Errorf("calculating provider collateral: %s", err)
	}

	pricePerEpoch := big.Div(
		big.Mul(big.NewInt(int64(ad.PieceSize)), big.NewInt(aud.PricePerGibPerEpoch)),
		big.NewInt(1<<30),
	)

	label, err := labelField(ad.PayloadCid)
	if err != nil {
		return nil, fmt.Errorf("failed to construct label field: %w", err)
	}

	sp, err := address.NewFromString(aud.StorageProviderID)
	if err != nil {
		return nil, fmt.Errorf("parsing storage-provider address: %s", err)
	}

	clientAddr := fc.conf.pubKey
	if rw != nil {
		waddr, err := address.NewFromString(rw.WalletAddr)
		if err != nil {
			return nil, fmt.Errorf("parsing remote wallet addr: %s", err)
		}
		clientAddr = waddr
	}

	// set provider collateral 10% above minimum to avoid fluctuations causing deal failure
	provCol := big.Div(big.Mul(collBounds.Min, big.NewInt(11)), big.NewInt(10))
	proposal := &market.DealProposal{
		PieceCID:     ad.PieceCid,
		PieceSize:    abi.PaddedPieceSize(ad.PieceSize), // Check padding vs not padding.
		VerifiedDeal: aud.Verified,
		Client:       clientAddr,
		Provider:     sp,

		Label: label,

		StartEpoch: abi.ChainEpoch(aud.StartEpoch),
		EndEpoch:   abi.ChainEpoch(aud.StartEpoch + ad.Duration),

		StoragePricePerEpoch: pricePerEpoch,
		ProviderCollateral:   provCol,
		ClientCollateral:     big.Zero(),
	}

	if err := fc.validateProposal(proposal); err != nil {
		return nil, fmt.Errorf("proposal validation: %s", err)
	}

	var sig *crypto.Signature
	if rw == nil {
		raw, err := cborutil.Dump(proposal)
		if err != nil {
			return nil, fmt.Errorf("encoding proposal in cbor: %s", err)
		}
		sig, err = wallet.WalletSign(fc.conf.exportedHexKey, raw)
		if err != nil {
			return nil, fmt.Errorf("locally signing proposal: %s", err)
		}
	} else {
		peerID, err := fc.connectToRemoteWallet(ctx, rw)
		if err != nil {
			return nil, fmt.Errorf("connecting to remote wallet: %s", err)
		}

		log.Debugf("requesting remote signature to %s", peerID)
		sig, err = propsigner.RequestDealProposalSignatureV1(ctx, fc.host, rw.AuthToken, *proposal, peerID)
		if err != nil {
			log.Errorf("remote signature ask for %s failed with: %s", peerID, err)
			return nil, fmt.Errorf("remote signing proposal: %s", err)
		}
		log.Debugf("remote signature from %s received successfully", peerID)
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
			PieceCid:     &ad.PieceCid,
		},
		FastRetrieval: aud.FastRetrieval,
	}, nil
}

func (fc *FilClient) sendProposal(
	ctx context.Context,
	proposal *network.Proposal) (res *network.SignedResponse, err error) {
	s, err := fc.streamToStorageProvider(ctx, proposal.DealProposal.Proposal.Provider, dealProtocol)
	if err != nil {
		return nil, fmt.Errorf("opening stream to storage-provider: %w", err)
	}

	defer func() {
		if err := s.Close(); err != nil {
			log.Errorf("closing stream: %s", err)
		}
	}()

	if err := cborutil.WriteCborRPC(s, proposal); err != nil {
		return nil, fmt.Errorf("failed to write proposal to storage-provider: %w", err)
	}

	var resp network.SignedResponse
	if err := cborutil.ReadCborRPC(s, &resp); err != nil {
		return nil, fmt.Errorf("failed to read response from storage-provider: %w", err)
	}

	return &resp, nil
}

func (fc *FilClient) validateProposal(p *market.DealProposal) error {
	switch p.VerifiedDeal {
	case true:
		if big.Cmp(p.StoragePricePerEpoch, fc.conf.maxVerifiedPricePerGiBPerEpoch) == 1 {
			return fmt.Errorf(
				"the verified proposal has %d price per epoch and max is %d",
				p.StoragePricePerEpoch,
				fc.conf.maxVerifiedPricePerGiBPerEpoch.Int64())
		}
	case false:
		if !fc.conf.allowUnverifiedDeals {
			return fmt.Errorf("only verified deals are allowed")
		}

		if big.Cmp(p.StoragePricePerEpoch, fc.conf.maxUnverifiedPricePerGiBPerEpoch) == 1 {
			return fmt.Errorf(
				"the unverified proposal has %d price per epoch and max is %d",
				p.StoragePricePerEpoch,
				fc.conf.maxUnverifiedPricePerGiBPerEpoch.Int64())
		}
	}

	return nil
}
