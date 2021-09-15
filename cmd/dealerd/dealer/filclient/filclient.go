package filclient

import (
	"bytes"
	"context"
	"fmt"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/filecoin-project/go-address"
	cborutil "github.com/filecoin-project/go-cbor-util"
	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/filecoin-project/go-fil-markets/storagemarket/network"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/specs-actors/actors/builtin/market"
	"github.com/ipfs/go-cid"
	"github.com/jsign/go-filsigner/wallet"
	"github.com/libp2p/go-libp2p-core/host"
	inet "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	protocol "github.com/libp2p/go-libp2p-core/protocol"
	"github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multibase"
	"github.com/textileio/bidbot/lib/logging"
	"github.com/textileio/broker-core/cmd/dealerd/store"
	"github.com/textileio/broker-core/metrics"
	"github.com/textileio/go-auctions-client/propsigner"
	logger "github.com/textileio/go-log/v2"
	"go.opentelemetry.io/otel/metric"
)

var (
	log = logger.Logger("dealer/filclient")
)

const dealStatusProtocol = "/fil/storage/status/1.1.0"
const dealProtocol = "/fil/storage/mk/1.1.0"

// FilClient provides API to interact with the Filecoin network.
type FilClient struct {
	conf config

	api  v0api.FullNode
	host host.Host

	metricExecAuctionDeal                    metric.Int64Counter
	metricGetChainHeight                     metric.Int64Counter
	metricResolveDealIDFromMessage           metric.Int64Counter
	metricCheckDealStatusWithStorageProvider metric.Int64Counter
	metricCheckChainDeal                     metric.Int64Counter
}

// New returns a new FilClient.
func New(api v0api.FullNode, h host.Host, opts ...Option) (*FilClient, error) {
	cfg := defaultConfig
	for _, op := range opts {
		if err := op(&cfg); err != nil {
			return nil, fmt.Errorf("applying option: %s", err)
		}
	}

	fc := &FilClient{
		conf: cfg,
		host: h,
		api:  api,
	}
	fc.initMetrics()

	return fc, nil
}

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
		metrics.MetricIncrCounter(ctx, err, fc.metricExecAuctionDeal)
	}()

	p, err := fc.createDealProposal(ctx, ad, aud, rw)
	if err != nil {
		// Any error here deserves retries.
		log.Errorf("creating deal proposal: %s", err)
		return cid.Undef, true, nil
	}
	log.Debugf("created proposal (remote-wallet: %t): %s", rw != nil, logging.MustJSONIndent(p))
	pr, err := fc.sendProposal(ctx, p)
	if err != nil {
		log.Errorf("sending proposal to storage-provider: %s", err)
		// Any error here deserves retries.
		return cid.Undef, true, nil
	}

	switch pr.Response.State {
	case storagemarket.StorageDealWaitingForData, storagemarket.StorageDealProposalAccepted:
		log.Debugf("proposal %s accepted: %s", pr.Response.Proposal, logging.MustJSONIndent(p))
	default:
		log.Warnf("proposal failed: %s", pr.Response.Proposal, logging.MustJSONIndent(p))
		return cid.Undef,
			false,
			fmt.Errorf("failed proposal (%s): %s",
				storagemarket.DealStates[pr.Response.State],
				pr.Response.Message)
	}

	return pr.Response.Proposal, false, nil
}

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

// CheckDealStatusWithStorageProvider checks a deal proposal status with a storage-provider.
// The caller should be aware that shouldn't fully trust data from storage-providers.
// To fully confirm the deal, a call to CheckChainDeal must be made after the storage-provider
// publishes the deal on-chain.
func (fc *FilClient) CheckDealStatusWithStorageProvider(
	ctx context.Context,
	storageProviderID string,
	propCid cid.Cid) (status *storagemarket.ProviderDealState, err error) {
	log.Debugf("checking status of proposal %s with storage-provider %s", propCid, storageProviderID)
	defer func() {
		metrics.MetricIncrCounter(ctx, err, fc.metricCheckDealStatusWithStorageProvider)
	}()
	sp, err := address.NewFromString(storageProviderID)
	if err != nil {
		return nil, fmt.Errorf("invalid storage-provider address %s: %s", storageProviderID, err)
	}
	cidb, err := cborutil.Dump(propCid)
	if err != nil {
		return nil, fmt.Errorf("encoding proposal in cbor: %s", err)
	}

	sig, err := wallet.WalletSign(fc.conf.exportedHexKey, cidb)
	if err != nil {
		return nil, fmt.Errorf("signing status request failed: %w", err)
	}

	req := &network.DealStatusRequest{
		Proposal:  propCid,
		Signature: *sig,
	}

	s, err := fc.streamToStorageProvider(ctx, sp, dealStatusProtocol)
	if err != nil {
		return nil, fmt.Errorf("opening stream with %s: %s", storageProviderID, err)
	}

	if err := cborutil.WriteCborRPC(s, req); err != nil {
		return nil, fmt.Errorf("failed to write status request: %w", err)
	}

	var resp network.DealStatusResponse
	if err := cborutil.ReadCborRPC(s, &resp); err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	log.Debugf("storage-provider %s replied proposal %s status check: %s",
		storageProviderID, propCid, storagemarket.DealStates[resp.DealState.State])

	if resp.DealState.State == storagemarket.StorageDealError {
		log.Warnf("deal error: %s", resp.DealState.Message)
	}

	return &resp.DealState, nil
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
		peerID, err := peer.Decode(rw.PeerID)
		if err != nil {
			return nil, fmt.Errorf("decoding remote peer-id: %s", err)
		}
		maddrs := make([]multiaddr.Multiaddr, len(rw.Multiaddrs))
		for i, maddr := range rw.Multiaddrs {
			maddr, err := multiaddr.NewMultiaddr(maddr)
			if err != nil {
				log.Warnf("parsing multiaddr %s: %s", maddr, err)
				continue
			}
			maddrs[i] = maddr
		}

		if fc.conf.relayMaddr != "" {
			relayed, err := multiaddr.NewMultiaddr(fc.conf.relayMaddr + "/p2p-circuit/p2p/" + peerID.String())
			if err != nil {
				return nil, fmt.Errorf("creating relayed maddr: %s", err)
			}
			maddrs = append(maddrs, relayed)
		}
		pi := peer.AddrInfo{
			ID:    peerID,
			Addrs: maddrs,
		}
		if err := fc.host.Connect(ctx, pi); err != nil {
			return nil, fmt.Errorf("connecting with remote wallet: %s", err)
		}

		log.Debugf("requesting remote signature to %s", peerID)

		log.Debugf("requesting remote signature to %s", peerID)
		sig, err = propsigner.RequestSignatureV1(ctx, fc.host, rw.AuthToken, *proposal, peerID)
		if err != nil {
			return nil, fmt.Errorf("remote signing proposal: %s", err)
		}
		log.Debugf("remote signature to %s received successfully", peerID)
		if err := validateSignature(*proposal, sig); err != nil {
			return nil, fmt.Errorf("remote signature is invalid: %s", err)
		}
		log.Debugf("remote signature from %s is valid", peerID)

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

func (fc *FilClient) streamToStorageProvider(
	ctx context.Context,
	maddr address.Address,
	protocol protocol.ID) (inet.Stream, error) {
	mpid, err := fc.connectToStorageProvider(ctx, maddr)
	if err != nil {
		return nil, fmt.Errorf("connecting with storage-provider: %s", err)
	}

	s, err := fc.host.NewStream(ctx, mpid, protocol)
	if err != nil {
		return nil, fmt.Errorf("failed to open stream to peer: %w", err)
	}

	return s, nil
}

func (fc *FilClient) connectToStorageProvider(ctx context.Context, maddr address.Address) (peer.ID, error) {
	minfo, err := fc.api.StateMinerInfo(ctx, maddr, types.EmptyTSK)
	if err != nil {
		return "", fmt.Errorf("state storage-provider info call: %s", err)
	}

	if minfo.PeerId == nil {
		log.Warnf("storage-provider %s doesn't have a PeerID on-chain", maddr)
		return "", fmt.Errorf("storage-provider %s has no peer ID set", maddr)
	}

	addrInfo, err := fc.api.NetFindPeer(ctx, *minfo.PeerId)
	if err != nil {
		log.Warnf("net-find-peer %s api call failed: %s", *minfo.PeerId, err)
		return "", fmt.Errorf("find peer by id: %s", err)
	}

	maddrs := addrInfo.Addrs
	if len(maddrs) == 0 {
		log.Debugf("resolving multiaddresses for %s in DHT failed, querying the chain for available ones...", maddr)
		// Try checking on-chain as a last resource.
		for _, mma := range minfo.Multiaddrs {
			ma, err := multiaddr.NewMultiaddrBytes(mma) //nolint:typecheck
			if err != nil {
				return "", fmt.Errorf("storage-provider %s had invalid multiaddrs in their info: %w", maddr, err)
			}
			maddrs = append(maddrs, ma)
		}
	}

	if len(maddrs) == 0 {
		return "", fmt.Errorf("no available multiaddresses for storage-provider %s", maddr)
	}

	if err := fc.host.Connect(ctx, peer.AddrInfo{
		ID:    *minfo.PeerId,
		Addrs: maddrs,
	}); err != nil {
		log.Warnf("failed connecting with storage-provider %s", maddr)
		return "", fmt.Errorf("connecting to storage-provider: %s", err)
	}

	return *minfo.PeerId, nil
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

func labelField(c cid.Cid) (string, error) {
	if c.Version() == 0 {
		return c.StringOfBase(multibase.Base58BTC)
	}
	return c.StringOfBase(multibase.Base64)
}

func validateSignature(proposal market.DealProposal, sig *crypto.Signature) error {
	msg := &bytes.Buffer{}
	err := proposal.MarshalCBOR(msg)
	if err != nil {
		return fmt.Errorf("marshaling proposal: %s", err)
	}
	sigBytes, err := sig.MarshalBinary()
	if err != nil {
		return fmt.Errorf("marshaling signature: %s", err)
	}
	ok, err := wallet.WalletVerify(proposal.Client, msg.Bytes(), sigBytes)
	if err != nil {
		return fmt.Errorf("verifying signature: %s", err)
	}
	if !ok {
		return fmt.Errorf("signature is invalid: %s", err)
	}
	return nil
}
