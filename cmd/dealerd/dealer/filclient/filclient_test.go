package filclient

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/filecoin-project/go-address"
	cborutil "github.com/filecoin-project/go-cbor-util"
	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	"github.com/jsign/go-filsigner/wallet"
	"github.com/libp2p/go-libp2p"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/peer"
	swarmt "github.com/libp2p/go-libp2p-swarm/testing"
	bhost "github.com/libp2p/go-libp2p/p2p/host/basic"
	"github.com/stretchr/testify/require"
	"github.com/textileio/broker-core/cmd/dealerd/store"
	"github.com/textileio/go-auctions-client/localwallet"
	"github.com/textileio/go-auctions-client/propsigner"
)

const authToken = "veryhardtokentoguess"

func TestRemoteDealProposalSigningV110(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Libp2p hosts wiring.
	h1, err := bhost.NewHost(swarmt.GenSwarm(t), nil) // dealer
	require.NoError(t, err)
	h2, err := bhost.NewHost(swarmt.GenSwarm(t), nil) // remote wallet
	require.NoError(t, err)
	err = h2.Connect(ctx, peer.AddrInfo{ID: h1.ID(), Addrs: h1.Addrs()})
	require.NoError(t, err)

	// Remote wallet config.
	walletKeys := []string{
		// Secp256k1 exported private key in Lotus format.
		"7b2254797065223a22736563703235366b31222c22507269766174654b6579223a226b35507976337148327349586343595a58594f5775453149326e32554539436861556b6c4e36695a5763453d227d", // nolint:lll
	}
	lwallet, err := localwallet.New(walletKeys)
	require.NoError(t, err)

	err = propsigner.NewDealSignerService(h2, authToken, lwallet)
	require.NoError(t, err)

	// Dealer filclient.
	client, err := New(createFilClient(t), h1, WithMaxPriceLimits(10, 10))
	require.NoError(t, err)

	// Create proposal targeting remote wallet.
	pieceCid, err := cid.Decode("baga6ea4seaqmj45fgnl36bep72gnf4ib5degrslzuq6zyk2hjbhakt6cas464pi")
	require.NoError(t, err)
	payloadCid, err := cid.Decode("uAXASIFxC4XOlV43b01pJO6ptOSxf8E_JjXhQXdgW-oMQxkUF")
	require.NoError(t, err)
	ad := store.AuctionData{
		PayloadCid: payloadCid,
		PieceCid:   pieceCid,
		PieceSize:  34359738368,
		Duration:   525600,
		CARURL:     "https://my.car.url",
	}
	aud := store.AuctionDeal{
		StorageProviderID:   "f0127896",
		PricePerGibPerEpoch: 0,
		StartEpoch:          754395,
		Verified:            true,
		FastRetrieval:       true,
		AuctionID:           "auction-1",
		BidID:               "bid-1",
	}
	maddrs := make([]string, len(h2.Addrs()))
	for i, maddr := range h2.Addrs() {
		maddrs[i] = maddr.String()
	}
	waddrPubKey, err := wallet.PublicKey(walletKeys[0])
	require.NoError(t, err)
	rw := &store.RemoteWallet{
		PeerID:     h2.ID().String(),
		AuthToken:  authToken,
		WalletAddr: waddrPubKey.String(),
		Multiaddrs: maddrs,
	}
	sp, err := client.createDealProposalV110(ctx, ad, aud, rw)
	require.NoError(t, err)

	// Validate signature.
	err = propsigner.ValidateDealProposalSignature(sp.DealProposal.Proposal, &sp.DealProposal.ClientSignature)
	require.NoError(t, err)
}

func TestRemoteDealStatusSigningV110(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Libp2p hosts wiring.
	h1, err := bhost.NewHost(swarmt.GenSwarm(t), nil) // dealer
	require.NoError(t, err)
	h2, err := bhost.NewHost(swarmt.GenSwarm(t), nil) // remote wallet
	require.NoError(t, err)
	err = h2.Connect(ctx, peer.AddrInfo{ID: h1.ID(), Addrs: h1.Addrs()})
	require.NoError(t, err)

	// Remote wallet config.
	walletKeys := []string{
		// Secp256k1 exported private key in Lotus format.
		"7b2254797065223a22736563703235366b31222c22507269766174654b6579223a226b35507976337148327349586343595a58594f5775453149326e32554539436861556b6c4e36695a5763453d227d", // nolint:lll
	}
	lwallet, err := localwallet.New(walletKeys)
	require.NoError(t, err)

	err = propsigner.NewDealSignerService(h2, authToken, lwallet)
	require.NoError(t, err)

	// Dealer filclient.
	client, err := New(createFilClient(t), h1, WithMaxPriceLimits(10, 10))
	require.NoError(t, err)

	// Remote wallet.
	maddrs := make([]string, len(h2.Addrs()))
	for i, maddr := range h2.Addrs() {
		maddrs[i] = maddr.String()
	}
	waddrPubKey, err := wallet.PublicKey(walletKeys[0])
	require.NoError(t, err)
	rw := &store.RemoteWallet{
		PeerID:     h2.ID().String(),
		AuthToken:  authToken,
		WalletAddr: waddrPubKey.String(),
		Multiaddrs: maddrs,
	}

	// Create proposal targeting remote wallet.
	propCid, err := cid.Decode("bafyreifydfjfbkcszmeyz72zu66an2lc4glykhrjlq7r7ir75mplwpqoxu")
	require.NoError(t, err)

	sig, err := client.signDealStatusRequest(ctx, propCid, uuid.Nil, rw)
	require.NoError(t, err)

	// Validate signature.
	payload, err := cborutil.Dump(propCid)
	require.NoError(t, err)
	err = propsigner.ValidateDealStatusSignature(waddrPubKey.String(), payload, sig)
	require.NoError(t, err)
}

func TestRemoteDealStatusSigningV120(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Libp2p hosts wiring.
	h1, err := bhost.NewHost(swarmt.GenSwarm(t), nil) // dealer
	require.NoError(t, err)
	h2, err := bhost.NewHost(swarmt.GenSwarm(t), nil) // remote wallet
	require.NoError(t, err)
	err = h2.Connect(ctx, peer.AddrInfo{ID: h1.ID(), Addrs: h1.Addrs()})
	require.NoError(t, err)

	// Remote wallet config.
	walletKeys := []string{
		// Secp256k1 exported private key in Lotus format.
		"7b2254797065223a22736563703235366b31222c22507269766174654b6579223a226b35507976337148327349586343595a58594f5775453149326e32554539436861556b6c4e36695a5763453d227d", // nolint:lll
	}
	lwallet, err := localwallet.New(walletKeys)
	require.NoError(t, err)

	err = propsigner.NewDealSignerService(h2, authToken, lwallet)
	require.NoError(t, err)

	// Dealer filclient.
	client, err := New(createFilClient(t), h1, WithMaxPriceLimits(10, 10))
	require.NoError(t, err)

	// Remote wallet.
	maddrs := make([]string, len(h2.Addrs()))
	for i, maddr := range h2.Addrs() {
		maddrs[i] = maddr.String()
	}
	waddrPubKey, err := wallet.PublicKey(walletKeys[0])
	require.NoError(t, err)
	rw := &store.RemoteWallet{
		PeerID:     h2.ID().String(),
		AuthToken:  authToken,
		WalletAddr: waddrPubKey.String(),
		Multiaddrs: maddrs,
	}

	// Create proposal targeting remote wallet.
	dealUID, err := uuid.NewRandom()
	require.NoError(t, err)

	sig, err := client.signDealStatusRequest(ctx, cid.Undef, dealUID, rw)
	require.NoError(t, err)

	// Validate signature.
	payload, err := dealUID.MarshalBinary()
	require.NoError(t, err)
	err = propsigner.ValidateDealStatusSignature(waddrPubKey.String(), payload, sig)
	require.NoError(t, err)
}

func TestExecuteAuctionDealV110(t *testing.T) {
	t.Parallel()
	t.SkipNow()

	client := create(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	pieceCid, err := cid.Decode("baga6ea4seaqmj45fgnl36bep72gnf4ib5degrslzuq6zyk2hjbhakt6cas464pi")
	require.NoError(t, err)
	payloadCid, err := cid.Decode("uAXASIFxC4XOlV43b01pJO6ptOSxf8E_JjXhQXdgW-oMQxkUF")
	require.NoError(t, err)
	ad := store.AuctionData{
		PayloadCid: payloadCid,
		PieceCid:   pieceCid,
		PieceSize:  34359738368,
		Duration:   525600,
		CARURL:     "https://fill.me.url",
	}
	aud := store.AuctionDeal{
		StorageProviderID:   "f01278",
		PricePerGibPerEpoch: 0,
		StartEpoch:          754395,
		Verified:            false,
		FastRetrieval:       true,
		AuctionID:           "auction-1",
		BidID:               "bid-1",
	}
	dealIdentifier, retry, err := client.ExecuteAuctionDeal(ctx, ad, aud, nil)
	require.NoError(t, err)
	require.False(t, retry)
	fmt.Printf("dealIdentifier: %s", dealIdentifier)
}

func TestExecuteAuctionDealV120(t *testing.T) {
	t.Parallel()
	t.SkipNow()

	client := create(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	pieceCid, err := cid.Decode("baga6ea4seaqmj45fgnl36bep72gnf4ib5degrslzuq6zyk2hjbhakt6cas464pi")
	require.NoError(t, err)
	payloadCid, err := cid.Decode("uAXASIFxC4XOlV43b01pJO6ptOSxf8E_JjXhQXdgW-oMQxkUF")
	require.NoError(t, err)
	ad := store.AuctionData{
		PayloadCid: payloadCid,
		PieceCid:   pieceCid,
		PieceSize:  34359738368,
		Duration:   525600,
		CARURL:     "https://cargo.web3.storage/deal-cars/bafybeid5lurqs2na2deipmqv2zfbj7gudnaq7vd6qh5y34hcijf5igsw7m_baga6ea4seaqfo2mmbs42pwthsku4emzc4advqfpydw43pzt4p3xg22q52grucpa.car", //nolint
	}
	aud := store.AuctionDeal{
		StorageProviderID:   "f047419",
		PricePerGibPerEpoch: 0,
		StartEpoch:          754395,
		Verified:            true,
		FastRetrieval:       true,
		AuctionID:           "auction-1",
		BidID:               "bid-1",
	}
	dealIdentifier, retry, err := client.ExecuteAuctionDeal(ctx, ad, aud, nil)
	require.NoError(t, err)
	require.False(t, retry)
	fmt.Printf("dealIdentifier: %s", dealIdentifier)
}

func TestConnectWithStorageProvider(t *testing.T) {
	t.Parallel()
	t.SkipNow()

	ctx := context.Background()
	fc := create(t)

	maddr, _ := address.NewFromString("f010446")
	minfo, err := fc.api.StateMinerInfo(ctx, maddr, types.EmptyTSK)
	require.NoError(t, err)
	require.NotNil(t, minfo.PeerId)
	ai := peer.AddrInfo{
		ID: *minfo.PeerId,
	}
	err = fc.host.Connect(context.Background(), ai)
	require.NoError(t, err)
}

func TestPublishedMessageAndDealOnChain(t *testing.T) {
	t.Parallel()
	t.SkipNow()

	client := create(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	publishedMessage, err := cid.Decode("bafy2bzacec6yztd2zzhgef77tqtx6qv5nldzziylq7iizpmgx4577boclftig")
	require.NoError(t, err)
	proposalCid, err := cid.Decode("bafyreibru2chqj7wanixo6m5qnmamovvgby7672ws3yojzyttimu7fl72q")
	require.NoError(t, err)

	dealID, err := client.ResolveDealIDFromMessage(ctx, proposalCid.String(), publishedMessage)
	require.NoError(t, err)
	require.Equal(t, int64(1919949), dealID)

	active, expiration, _, err := client.CheckChainDeal(ctx, dealID)
	require.NoError(t, err)
	require.True(t, active)
	require.Equal(t, uint64(1279995), expiration)
}

func TestCheckStatusWithStorageProvider(t *testing.T) {
	t.Parallel()
	t.SkipNow()

	client := create(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	proposalCid, err := cid.Decode("bafyreieakjjn6kv36zfo23e67mvn2mrjgjz34w2awjaivskfhf4okjhdva")
	require.NoError(t, err)
	pcid, status, err := client.CheckDealStatusWithStorageProvider(ctx, "f0840770", proposalCid.String(), nil)
	require.NoError(t, err)
	fmt.Printf("pcid: %s\n", pcid)
	fmt.Printf("status: %s\n", storagemarket.DealStatesDescriptions[status])
}

func TestGetChainHeight(t *testing.T) {
	t.Parallel()

	client := create(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	height, err := client.GetChainHeight(ctx)
	require.NoError(t, err)
	require.Greater(t, height, uint64(0))
}

func createFilClient(t *testing.T) *v0api.FullNodeStruct {
	var api v0api.FullNodeStruct
	closer, err := jsonrpc.NewMergeClient(context.Background(), "https://api.node.glif.io", "Filecoin",
		[]interface{}{
			&api.CommonStruct.Internal,
			&api.NetStruct.Internal,
			&api.Internal,
		},
		http.Header{},
	)
	require.NoError(t, err)
	t.Cleanup(closer)

	return &api
}

func create(t *testing.T) *FilClient {
	api := createFilClient(t)
	exportedKey := "7b2254797065223a22736563703235366b31222c22507269766174654b6579223a226b35507976337148327349" +
		"586343595a58594f5775453149326e32554539436861556b6c4e36695a5763453d227d"
	cm, err := connmgr.NewConnManager(500, 800, connmgr.WithGracePeriod(time.Minute))
	require.NoError(t, err)
	h, err := libp2p.New(libp2p.ConnectionManager(cm))
	require.NoError(t, err)
	opts := []Option{
		WithExportedKey(exportedKey),
		WithMaxPriceLimits(1<<31, 1<<31),
	}
	client, err := New(api, h, opts...)
	require.NoError(t, err)

	return client
}
