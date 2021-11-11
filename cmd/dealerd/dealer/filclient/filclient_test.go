package filclient

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/filecoin-project/lotus/chain/types"
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
	logger "github.com/textileio/go-log/v2"
)

func TestRemoteDealProposalSigning(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Libp2p hosts wiring.
	h1, err := bhost.NewHost(ctx, swarmt.GenSwarm(t, ctx), nil) // dealer
	require.NoError(t, err)
	h2, err := bhost.NewHost(ctx, swarmt.GenSwarm(t, ctx), nil) // remote wallet
	require.NoError(t, err)
	err = h2.Connect(ctx, peer.AddrInfo{ID: h1.ID(), Addrs: h1.Addrs()})
	require.NoError(t, err)

	// Remote wallet config.
	walletKeys := []string{
		// Secp256k1 exported private key in Lotus format.
		"7b2254797065223a22736563703235366b31222c22507269766174654b6579223a226b35507976337148327349586343595a58594f5775453149326e32554539436861556b6c4e36695a5763453d227d", // nolint:lll
	}
	authToken := "veryhardtokentoguess"
	lwallet, err := localwallet.New(walletKeys)
	require.NoError(t, err)

	err = propsigner.NewDealSignerService(h2, authToken, lwallet)
	require.NoError(t, err)

	// Dealer filclient.
	api := createFilClient(t, "https://api.node.glif.io/")
	client, err := New([]v0api.FullNode{api}, h1, WithMaxPriceLimits(10, 10))
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
	}
	aud := store.AuctionDeal{
		StorageProviderID:   "f01278",
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
	sp, err := client.createDealProposal(ctx, ad, aud, rw)
	require.NoError(t, err)

	// Validate signature.
	err = propsigner.ValidateDealProposalSignature(sp.DealProposal.Proposal, &sp.DealProposal.ClientSignature)
	require.NoError(t, err)
}

func TestRemoteDealStatusSigning(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	// Libp2p hosts wiring.
	h1, err := bhost.NewHost(ctx, swarmt.GenSwarm(t, ctx), nil) // dealer
	require.NoError(t, err)
	h2, err := bhost.NewHost(ctx, swarmt.GenSwarm(t, ctx), nil) // remote wallet
	require.NoError(t, err)
	err = h2.Connect(ctx, peer.AddrInfo{ID: h1.ID(), Addrs: h1.Addrs()})
	require.NoError(t, err)

	// Remote wallet config.
	walletKeys := []string{
		// Secp256k1 exported private key in Lotus format.
		"7b2254797065223a22736563703235366b31222c22507269766174654b6579223a226b35507976337148327349586343595a58594f5775453149326e32554539436861556b6c4e36695a5763453d227d", // nolint:lll
	}
	authToken := "veryhardtokentoguess"
	lwallet, err := localwallet.New(walletKeys)
	require.NoError(t, err)

	err = propsigner.NewDealSignerService(h2, authToken, lwallet)
	require.NoError(t, err)

	// Dealer filclient.
	api := createFilClient(t, "https://api.node.glif.io/")
	client, err := New([]v0api.FullNode{api}, h1, WithMaxPriceLimits(10, 10))
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

	dsr, err := client.createDealStatusRequest(ctx, propCid, rw)
	require.NoError(t, err)

	// Validate signature.
	err = propsigner.ValidateDealStatusSignature(waddrPubKey.String(), propCid, &dsr.Signature)
	require.NoError(t, err)
}

func TestExecuteAuctionDeal(t *testing.T) {
	t.Parallel()
	t.SkipNow()

	client := create(t, "https://api.node.glif.io/")
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
	propCid, retry, err := client.ExecuteAuctionDeal(ctx, ad, aud, nil)
	require.NoError(t, err)
	require.False(t, retry)
	fmt.Printf("propCid: %s", propCid)
}

func TestConnectWithStorageProvider(t *testing.T) {
	t.Parallel()
	t.SkipNow()

	ctx := context.Background()
	fc := create(t, "https://api.node.glif.io/")

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

	client := create(t, "https://api.node.glif.com/", "https://api.node.glif.io/")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	publishedMessage, err := cid.Decode("bafy2bzacec6yztd2zzhgef77tqtx6qv5nldzziylq7iizpmgx4577boclftig")
	require.NoError(t, err)
	proposalCid, err := cid.Decode("bafyreibru2chqj7wanixo6m5qnmamovvgby7672ws3yojzyttimu7fl72q")
	require.NoError(t, err)

	dealID, err := client.ResolveDealIDFromMessage(ctx, proposalCid, publishedMessage)
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

	client := create(t, "https://api.node.glif.io/")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	proposalCid, err := cid.Decode("bafyreieakjjn6kv36zfo23e67mvn2mrjgjz34w2awjaivskfhf4okjhdva")
	require.NoError(t, err)
	status, err := client.CheckDealStatusWithStorageProvider(ctx, "f0840770", proposalCid, nil)
	require.NoError(t, err)
	fmt.Printf("%s\n", logger.MustJSONIndent(status))
	fmt.Printf("%s\n", storagemarket.DealStatesDescriptions[status.State])
}

func TestGetChainHeight(t *testing.T) {
	t.Parallel()

	client := create(t, "https://api.node.glif.io/")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	height, err := client.GetChainHeight(ctx)
	require.NoError(t, err)
	require.Greater(t, height, uint64(0))
}

func TestGetChainHeightWithSuccessulRetry(t *testing.T) {
	t.Parallel()

	client := create(t, "https://api.node.glif.com/", "https://api.node.glif.io/") // the first gateway is invalid
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	height, err := client.GetChainHeight(ctx)
	require.NoError(t, err)
	require.Greater(t, height, uint64(0))
}

func TestGetChainHeightWithUnsuccessulRetry(t *testing.T) {
	t.Parallel()

	client := create(t, "https://api.node.glif.com/", "https://api.node.glif.com/") // both gateways are invalid
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	_, err := client.GetChainHeight(ctx)
	require.Error(t, err)
}

func createFilClient(t *testing.T, url string) *v0api.FullNodeStruct {
	var api v0api.FullNodeStruct
	closer, err := jsonrpc.NewMergeClient(context.Background(), url, "Filecoin",
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

func create(t *testing.T, urls ...string) *FilClient {
	apis := []v0api.FullNode{}
	for _, url := range urls {
		apis = append(apis, createFilClient(t, url))
	}

	exportedKey := "7b2254797065223a22736563703235366b31222c22507269766174654b6579223a226b35507976337148327349" +
		"586343595a58594f5775453149326e32554539436861556b6c4e36695a5763453d227d"
	h, err := libp2p.New(context.Background(),
		libp2p.ConnectionManager(connmgr.NewConnManager(500, 800, time.Minute)),
	)
	require.NoError(t, err)
	client, err := New(apis, h, WithExportedKey(exportedKey))
	require.NoError(t, err)

	return client
}
