package filclient

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	smtypes "github.com/filecoin-project/boost/storagemarket/types"
	boosttypes "github.com/filecoin-project/boost/transport/types"
	"github.com/filecoin-project/go-address"
	cborutil "github.com/filecoin-project/go-cbor-util"
	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/filecoin-project/go-fil-markets/storagemarket/network"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/v7/actors/builtin/market"
	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	"github.com/jsign/go-filsigner/wallet"
	"github.com/multiformats/go-multibase"
	"github.com/textileio/broker-core/cmd/dealerd/store"
	"github.com/textileio/broker-core/metrics"
	"github.com/textileio/go-auctions-client/propsigner"
	logger "github.com/textileio/go-log/v2"
	"go.opentelemetry.io/otel/attribute"
)

const (
	dealProtocolv110 = "/fil/storage/mk/1.1.0"
	dealProtocolv120 = "/fil/storage/mk/1.2.0"
)

// ExecuteAuctionDeal creates a deal with a storage-provider using the data described in an auction deal.
func (fc *FilClient) ExecuteAuctionDeal(
	ctx context.Context,
	ad store.AuctionData,
	aud store.AuctionDeal,
	rw *store.RemoteWallet,
	allowBoost bool) (didentifier string, retriable bool, err error) {
	log.Debugf("executing auction deal for data-cid %s, piece-cid %s and sp %s...",
		ad.PayloadCid, ad.PieceCid, aud.StorageProviderID)
	defer func() {
		var attrs []attribute.KeyValue
		if rw != nil {
			attrs = []attribute.KeyValue{attrWalletSignature.String(rw.WalletAddr), attrRemoteWallet}
		} else {
			attrs = []attribute.KeyValue{attrWalletSignature.String(fc.conf.pubKey.String()), attrLocalWallet}
		}
		metrics.MetricIncrCounter(ctx, err, fc.metricExecAuctionDeal, attrs...)
	}()

	p, err := fc.createDealProposalV110(ctx, ad, aud, rw)
	if err != nil {
		// Any error here deserves retries.
		log.Errorf("creating deal proposal: %s", err)
		return "", true, nil
	}

	sp, err := address.NewFromString(aud.StorageProviderID)
	if err != nil {
		return "", false, fmt.Errorf("parsing storage-provider address: %s", err)
	}
	spDealProtocol, err := fc.dealProtocolForStorageProvider(ctx, sp)
	if err != nil {
		return "", true, fmt.Errorf("detecting supporting deal protocol: %s", err)
	}
	if spDealProtocol == dealProtocolv120 && !allowBoost {
		log.Info("downgrading %s from boost to legacy deal proposal protocol", sp)
		spDealProtocol = dealProtocolv110
	}

	var dealStatus storagemarket.StorageDealStatus
	var proposalMsg, dealIdentifier string
	switch spDealProtocol {
	case dealProtocolv110:
		pr, err := fc.sendProposalV110(ctx, p)
		if err != nil {
			log.Errorf("sending proposal to storage-provider: %s", err)
			// Any error here deserves retries.
			return "", true, nil
		}
		dealIdentifier = pr.Response.Proposal.String()
		dealStatus = pr.Response.State
		proposalMsg = pr.Response.Message
	case dealProtocolv120:
		dealUUID, err := uuid.NewRandom()
		if err != nil {
			return "", true, fmt.Errorf("generating random uuid: %s", err)
		}
		dealIdentifier = dealUUID.String()
		dealStatus, proposalMsg, err =
			fc.createAndSendProposalV120(ctx, *p.DealProposal, p.Piece.Root, dealUUID, ad.CARURL)
		if err != nil {
			return "", true, fmt.Errorf("sending proposal v1.2.0: %s", err)
		}
	default:
		return "", false, fmt.Errorf("unsupported deal protocol %s", spDealProtocol)
	}

	switch dealStatus {
	case storagemarket.StorageDealWaitingForData, storagemarket.StorageDealProposalAccepted:
		log.Debugf("proposal %s for %s accepted: %s", dealIdentifier, aud.StorageProviderID, proposalMsg)
	default:
		log.Warnf("proposal %s for %s failed: %s", dealIdentifier, aud.StorageProviderID, proposalMsg)
		return "",
			false,
			fmt.Errorf("failed proposal %s (%s) for %s: %s",
				dealIdentifier,
				storagemarket.DealStates[dealStatus],
				aud.StorageProviderID,
				proposalMsg)
	}

	return dealIdentifier, false, nil
}

func (fc *FilClient) createAndSendProposalV120(
	ctx context.Context,
	dealProposal market.ClientDealProposal,
	payloadCid cid.Cid,
	dealUUID uuid.UUID,
	carURL string) (storagemarket.StorageDealStatus, string, error) {
	transferParams, err := json.Marshal(boosttypes.HttpRequest{URL: carURL})
	if err != nil {
		return 0, "", fmt.Errorf("marshaling deal transfer params: %w", err)
	}

	carHTTPHeader, err := http.Get(carURL)
	if err != nil {
		return 0, "", fmt.Errorf("get http header of %s: %s", carURL, err)
	}
	_ = carHTTPHeader.Body.Close()
	clHeader, ok := carHTTPHeader.Header["Content-Length"]
	if !ok {
		return 0, "", fmt.Errorf("http header of %s doesn't have Content-Length attr", carURL)
	}
	carTotalBytes, err := strconv.ParseUint(clHeader[0], 10, 64)
	if err != nil {
		return 0, "", fmt.Errorf("parsing Content-Length header %s: %s", clHeader[0], err)
	}
	// Send proposal to storage provider using deal protocol v1.2.0 format
	params := smtypes.DealParams{
		DealUUID:           dealUUID,
		ClientDealProposal: dealProposal,
		DealDataRoot:       payloadCid,
		Transfer: smtypes.Transfer{
			Type:     "http",
			ClientID: "",
			Params:   transferParams,
			Size:     carTotalBytes,
		},
	}

	resp, err := fc.sendProposalV120(ctx, &params)
	if err != nil {
		return 0, "", fmt.Errorf("send proposal v1.2.0 rpc: %s", err)
	}

	// Check if the deal proposal was accepted
	if !resp.Accepted {
		return storagemarket.StorageDealProposalRejected, resp.Message, nil
	}

	return storagemarket.StorageDealWaitingForData, resp.Message, nil
}

func (fc *FilClient) createDealProposalV110(
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

	label, err := labelField(ad.PayloadCid)
	if err != nil {
		return nil, fmt.Errorf("failed to construct label field: %w", err)
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

	p := &network.Proposal{
		DealProposal: sigprop,
		Piece: &storagemarket.DataRef{
			TransferType: storagemarket.TTManual,
			Root:         ad.PayloadCid,
			PieceCid:     &ad.PieceCid,
		},
		FastRetrieval: aud.FastRetrieval,
	}
	log.Debugf("created proposal (remote-wallet: %t): %s", rw != nil, logger.MustJSONIndent(p))

	return p, nil
}

func (fc *FilClient) sendProposalV120(
	ctx context.Context,
	proposal *smtypes.DealParams) (res *smtypes.DealResponse, err error) {
	log.Debugf("sending proposal v1.2.0: %s", logger.MustJSONIndent(proposal))
	s, err := fc.streamToStorageProvider(ctx, proposal.ClientDealProposal.Proposal.Provider, dealProtocolv120)
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

	var resp smtypes.DealResponse
	if err := cborutil.ReadCborRPC(s, &resp); err != nil {
		return nil, fmt.Errorf("failed to read response from storage-provider: %w", err)
	}
	log.Debugf("sent proposal v1.2.0: %s", logger.MustJSONIndent(proposal))

	return &resp, nil
}

func (fc *FilClient) sendProposalV110(
	ctx context.Context,
	proposal *network.Proposal) (res *network.SignedResponse, err error) {
	log.Debugf("sending proposal v1.1.0 to %s", proposal.DealProposal.Proposal.Provider)
	s, err := fc.streamToStorageProvider(ctx, proposal.DealProposal.Proposal.Provider, dealProtocolv110)
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
	log.Debugf("sent proposal v1.1.0 to %s: %s",
		proposal.DealProposal.Proposal.Provider, logger.MustJSONIndent(proposal))

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

func (fc *FilClient) dealProtocolForStorageProvider(
	ctx context.Context,
	storageProvider address.Address) (string, error) {
	log.Debugf("detecting preferred protocol for %s", storageProvider)
	mpid, err := fc.connectToStorageProvider(ctx, storageProvider)
	if err != nil {
		return "", fmt.Errorf("connecting to storage-provider %s: %s", storageProvider, err)
	}

	proto, err := fc.host.Peerstore().FirstSupportedProtocol(mpid, dealProtocolv120, dealProtocolv110)
	if err != nil {
		return "", fmt.Errorf("getting deal protocol for %s: %w", storageProvider, err)
	}
	if proto == "" {
		return "", fmt.Errorf("%s does not support any deal making protocol", storageProvider)
	}
	log.Debugf("preferred protocol for %s is %s", storageProvider, proto)

	return proto, nil
}

func labelField(c cid.Cid) (string, error) {
	if c.Version() == 0 {
		return c.StringOfBase(multibase.Base58BTC)
	}
	return c.StringOfBase(multibase.Base64)
}
