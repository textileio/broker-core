package filclient

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/filecoin-project/go-address"
	cborutil "github.com/filecoin-project/go-cbor-util"
	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/filecoin-project/go-fil-markets/storagemarket/impl/clientutils"
	"github.com/filecoin-project/go-fil-markets/storagemarket/network"
	"github.com/filecoin-project/go-state-types/abi"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/wallet"
	"github.com/filecoin-project/specs-actors/actors/builtin/market"
	"github.com/ipfs/go-cid"
	logger "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/host"
	inet "github.com/libp2p/go-libp2p-core/network"
	"github.com/libp2p/go-libp2p-core/peer"
	protocol "github.com/libp2p/go-libp2p-protocol"
	"github.com/multiformats/go-multiaddr"
	"github.com/textileio/broker-core/cmd/dealerd/dealer/store"
	"github.com/textileio/broker-core/logging"
	"golang.org/x/xerrors"
)

var (
	log = logger.Logger("dealer/filclient")
)

const DealStatusProtocol = "/fil/storage/status/1.1.0"
const DealProtocol = "/fil/storage/mk/1.1.0"

type FilClient struct {
	walletAddr address.Address
	wallet     *wallet.LocalWallet
	api        api.GatewayAPI
	host       host.Host
}

func New(opts ...Option) (*FilClient, error) {
	cfg := defaultConfig
	for _, op := range opts {
		if err := op(&cfg); err != nil {
			return nil, fmt.Errorf("applying option: %s", err)
		}
	}

	memks := wallet.NewMemKeyStore()
	w, err := wallet.NewWallet(memks)
	if err != nil {
		return nil, fmt.Errorf("creating local wallet: %s", err)
	}

	ki := &types.KeyInfo{
		Type:       cfg.keyType,
		PrivateKey: cfg.keyPrivate,
	}
	waddr, err := w.WalletImport(context.Background(), ki)
	if err != nil {
		return nil, fmt.Errorf("importing wallet addr: %s", err)
	}

	h, err := libp2p.New(context.Background(),
		libp2p.ConnectionManager(connmgr.NewConnManager(500, 800, time.Minute)),
	)
	if err != nil {
		return nil, fmt.Errorf("creating host: %s", err)
	}

	fc := &FilClient{
		walletAddr: waddr,
		wallet:     w,
		host:       h,
	}

	return fc, nil
}

func (fc *FilClient) ExecuteAuctionDeal(ctx context.Context, ad store.AuctionData, aud store.AuctionDeal) (cid.Cid, error) {
	p, err := fc.createDealProposal(ctx, ad, aud)
	if err != nil {
		return cid.Undef, fmt.Errorf("creating deal proposal: %s", err)
	}
	log.Debugf("created proposal: %s", logging.MustJsonIndent(p))
	pr, err := fc.sendProposal(ctx, p)
	if err != nil {
		return cid.Undef, fmt.Errorf("sending proposal to miner: %s", err)
	}

	switch pr.Response.State {
	case storagemarket.StorageDealWaitingForData, storagemarket.StorageDealProposalAccepted:
		log.Debugf("proposal accepted: %s", logging.MustJsonIndent(p))
	default:
		log.Warnf("proposal failed: %s", logging.MustJsonIndent(p))
		return cid.Undef,
			fmt.Errorf("failed proposal (%s): %s",
				storagemarket.DealStates[pr.Response.State],
				pr.Response.Message)
	}

	return pr.Response.Proposal, nil
}

func (fc *FilClient) GetChainHeight(ctx context.Context) (int64, error) {
	tip, err := fc.api.ChainHead(ctx)
	if err != nil {
		return 0, fmt.Errorf("getting chain head: %s", err)
	}

	return int64(tip.Height()), nil
}

// ResolveDealIDFromMessage looks for a publish deal message by its Cid and resolves the deal-id from the receipt.
// If the message isn't found, the return parameters are 0,nil.
// If the message is found, a deal-id is returned which can be trusted to match the original ProposalCid.
// This method is mostly what Estuary does with some tweaks, kudos to them!
func (fc *FilClient) ResolveDealIDFromMessage(
	ctx context.Context,
	aud store.AuctionDeal,
	publishDealMessage cid.Cid) (int64, error) {
	mlookup, err := fc.api.StateSearchMsg(ctx, publishDealMessage)
	if err != nil {
		return 0, xerrors.Errorf("could not find published deal on chain: %w", err)
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
			return 0, xerrors.Errorf("failed to compute deal proposal ipld node: %w", err)
		}

		// If we find a proposal in the message that matches our AuctionDeal proposal cid, we can be sure
		// that this deal-id is the one we're looking for. The proposal cid summarizes the deal information.
		if nd.Cid() == aud.ProposalCid {
			dealix = i
			break
		}
	}

	if dealix == -1 {
		return 0, fmt.Errorf("deal isn't part of the published message")
	}

	if mlookup.Receipt.ExitCode != 0 {
		return 0, xerrors.Errorf("the message failed to execute (exit: %d)", mlookup.Receipt.ExitCode)
	}

	var retval market.PublishStorageDealsReturn
	if err := retval.UnmarshalCBOR(bytes.NewReader(mlookup.Receipt.Return)); err != nil {
		return 0, xerrors.Errorf("publish deal return was improperly formatted: %w", err)
	}

	if len(retval.IDs) != len(params.Deals) {
		return 0, fmt.Errorf("return value from publish deals did not match length of params")
	}

	return int64(retval.IDs[dealix]), nil
}

func (fc *FilClient) createDealProposal(ctx context.Context, ad store.AuctionData, aud store.AuctionDeal) (*network.Proposal, error) {
	collBounds, err := fc.api.StateDealProviderCollateralBounds(
		ctx,
		abi.PaddedPieceSize(ad.PieceSize),
		aud.Verified,
		types.EmptyTSK)
	if err != nil {
		return nil, fmt.Errorf("calculating provider collateral: %s", err)
	}

	pricePerEpoch := big.Div(
		big.Mul(big.NewInt(ad.PieceSize), big.NewInt(aud.PricePerGiBPerEpoch)),
		big.NewInt(1<<30),
	)

	label, err := clientutils.LabelField(ad.PayloadCid)
	if err != nil {
		return nil, fmt.Errorf("failed to construct label field: %w", err)
	}

	miner, err := address.NewFromString(aud.Miner)
	if err != nil {
		return nil, fmt.Errorf("parsing miner address: %s", err)
	}

	// set provider collateral 10% above minimum to avoid fluctuations causing deal failure
	provCol := big.Div(big.Mul(collBounds.Min, big.NewInt(11)), big.NewInt(10))
	proposal := &market.DealProposal{
		PieceCID:     ad.PieceCid,
		PieceSize:    abi.PaddedPieceSize(ad.PieceSize), // Check padding vs not padding.
		VerifiedDeal: aud.Verified,
		Client:       fc.walletAddr,
		Provider:     miner,

		Label: label,

		StartEpoch: abi.ChainEpoch(aud.StartEpoch),
		EndEpoch:   abi.ChainEpoch(aud.StartEpoch + ad.Duration),

		StoragePricePerEpoch: abi.TokenAmount(pricePerEpoch),
		ProviderCollateral:   provCol,
		ClientCollateral:     big.Zero(),
	}

	raw, err := cborutil.Dump(proposal)
	if err != nil {
		return nil, err
	}
	sig, err := fc.wallet.WalletSign(ctx, fc.walletAddr, raw, api.MsgMeta{Type: api.MTDealProposal})
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
		FastRetrieval: aud.FastRetrieval,
	}, nil
}

func (fc *FilClient) CheckDealStatusWithMiner(ctx context.Context, minerAddr string, propCid cid.Cid) (*storagemarket.ProviderDealState, error) {
	miner, err := address.NewFromString(minerAddr)
	if err != nil {
		return nil, fmt.Errorf("invalid miner address %s: %s", minerAddr, err)
	}
	cidb, err := cborutil.Dump(propCid)
	if err != nil {
		return nil, err
	}

	sig, err := fc.wallet.WalletSign(ctx, fc.walletAddr, cidb, api.MsgMeta{Type: api.MTUnknown})
	if err != nil {
		return nil, xerrors.Errorf("signing status request failed: %w", err)
	}

	req := &network.DealStatusRequest{
		Proposal:  propCid,
		Signature: *sig,
	}

	s, err := fc.streamToMiner(ctx, miner, DealStatusProtocol)
	if err != nil {
		return nil, err
	}

	if err := cborutil.WriteCborRPC(s, req); err != nil {
		return nil, xerrors.Errorf("failed to write status request: %w", err)
	}

	var resp network.DealStatusResponse
	if err := cborutil.ReadCborRPC(s, &resp); err != nil {
		return nil, xerrors.Errorf("reading response: %w", err)
	}

	// TODO: check the signatures and stuff?

	return &resp.DealState, nil
}

func (fc *FilClient) streamToMiner(ctx context.Context, maddr address.Address, protocol protocol.ID) (inet.Stream, error) {
	mpid, err := fc.connectToMiner(ctx, maddr)
	if err != nil {
		return nil, err
	}

	s, err := fc.host.NewStream(ctx, mpid, protocol)
	if err != nil {
		return nil, xerrors.Errorf("failed to open stream to peer: %w", err)
	}

	return s, nil
}

func (fc *FilClient) connectToMiner(ctx context.Context, maddr address.Address) (peer.ID, error) {
	minfo, err := fc.api.StateMinerInfo(ctx, maddr, types.EmptyTSK)
	if err != nil {
		return "", err
	}

	if minfo.PeerId == nil {
		return "", fmt.Errorf("miner %s has no peer ID set", maddr)
	}

	var maddrs []multiaddr.Multiaddr
	for _, mma := range minfo.Multiaddrs {
		ma, err := multiaddr.NewMultiaddrBytes(mma)
		if err != nil {
			return "", fmt.Errorf("miner %s had invalid multiaddrs in their info: %w", maddr, err)
		}
		maddrs = append(maddrs, ma)
	}

	if err := fc.host.Connect(ctx, peer.AddrInfo{
		ID:    *minfo.PeerId,
		Addrs: maddrs,
	}); err != nil {
		return "", err
	}

	return *minfo.PeerId, nil
}

func (fc *FilClient) sendProposal(ctx context.Context, netprop *network.Proposal) (*network.SignedResponse, error) {
	s, err := fc.streamToMiner(ctx, netprop.DealProposal.Proposal.Provider, DealProtocol)
	if err != nil {
		return nil, xerrors.Errorf("opening stream to miner: %w", err)
	}

	defer s.Close()

	if err := cborutil.WriteCborRPC(s, netprop); err != nil {
		return nil, xerrors.Errorf("failed to write proposal to miner: %w", err)
	}

	var resp network.SignedResponse
	if err := cborutil.ReadCborRPC(s, &resp); err != nil {
		return nil, xerrors.Errorf("failed to read response from miner: %w", err)
	}

	return &resp, nil
}

func (fc *FilClient) CheckChainDeal(ctx context.Context, dealid int64) (bool, error) {
	deal, err := fc.api.StateMarketStorageDeal(ctx, abi.DealID(dealid), types.EmptyTSK)
	if err != nil {
		nfs := fmt.Sprintf("deal %d not found", dealid)
		if strings.Contains(err.Error(), nfs) {
			return false, nil
		}

		return false, err
	}

	if deal.State.SlashEpoch > 0 {
		return false, nil
	}

	return true, nil
}
