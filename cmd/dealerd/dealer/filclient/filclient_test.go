package filclient

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"
	"github.com/textileio/broker-core/cmd/dealerd/dealer/store"
	"github.com/textileio/broker-core/logging"
)

func TestExecuteAuctionDeal(t *testing.T) {
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
	}
	aud := store.AuctionDeal{
		Miner:               "f01278",
		PricePerGiBPerEpoch: 0,
		StartEpoch:          754395,
		Verified:            false,
		FastRetrieval:       true,
	}
	propCid, err := client.ExecuteAuctionDeal(ctx, ad, aud)
	require.NoError(t, err)
	fmt.Printf("propCid: %s", propCid)
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

	dealID, err := client.ResolveDealIDFromMessage(ctx, proposalCid, publishedMessage)
	require.NoError(t, err)
	require.Equal(t, int64(1919949), dealID)

	active, expiration, err := client.CheckChainDeal(ctx, dealID)
	require.NoError(t, err)
	require.True(t, active)
	require.Equal(t, uint64(1279995), expiration)
}

func TestCheckStatusWithMiner(t *testing.T) {
	t.Parallel()
	t.SkipNow()

	client := create(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	proposalCid, err := cid.Decode("bafyreibru2chqj7wanixo6m5qnmamovvgby7672ws3yojzyttimu7fl72q")
	require.NoError(t, err)
	status, err := client.CheckDealStatusWithMiner(ctx, "f01278", proposalCid)
	require.NoError(t, err)
	fmt.Printf("%s\n", logging.MustJSONIndent(status))
	fmt.Printf("%s\n", storagemarket.DealStatesDescriptions[status.State])
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

func create(t *testing.T) *FilClient {
	var api v0api.FullNodeStruct
	closer, err := jsonrpc.NewMergeClient(context.Background(), "https://api.node.glif.io", "Filecoin",
		[]interface{}{
			&api.CommonStruct.Internal,
			&api.Internal,
		},
		http.Header{},
	)
	require.NoError(t, err)
	t.Cleanup(closer)

	exportedKey := "7b2254797065223a22736563703235366b31222c22507269766174654b6579223a226b35507976337148327349" +
		"586343595a58594f5775453149326e32554539436861556b6c4e36695a5763453d227d"
	require.NoError(t, err)
	client, err := New(&api, WithExportedKey(exportedKey))
	require.NoError(t, err)

	return client
}
