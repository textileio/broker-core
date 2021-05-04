package filclient

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/filecoin-project/lotus/api/client"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"
)

func TestExecuteAuctionDeal(t *testing.T) {
	t.Parallel()
	t.Skip()
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

func TestResolveDealIDFromMessage(t *testing.T) {
	t.Parallel()
	client := create(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	messageCid, err := cid.Decode("bafy2bzaceblpg3v7ay45wf2lrvhwnjv2cldticixzekh5wft6mklyzg6vwjbw")
	require.NoError(t, err)
	proposalCid, err := cid.Decode("bafyreiewepxw6o2cka5jovcvpjh72a2gdhhvpt5xsiov7g4ejiyaawnti4")
	require.NoError(t, err)
	dealID, err := client.ResolveDealIDFromMessage(ctx, proposalCid, messageCid)
	require.NoError(t, err)
	require.Equal(t, int64(1856658), dealID)
}

func TestCheckDealStatusWithMiner(t *testing.T) {
	t.Parallel()
	t.SkipNow()

	client := create(t)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*30)
	defer cancel()

	proposalCid, err := cid.Decode("bafyreig54fltdzbl6f7yyvhyu6gyayheawgpeqsyczmdh4obeaqszrdrpm")
	require.NoError(t, err)
	_, err = client.CheckDealStatusWithMiner(ctx, "f022352", proposalCid)
	require.NoError(t, err)
}

func create(t *testing.T) *FilClient {
	api, closer, err := client.NewGatewayRPC(context.Background(), "https://api.node.glif.io", http.Header{})
	require.NoError(t, err)
	t.Cleanup(closer)

	exportedKey := "7b2254797065223a22736563703235366b31222c22507269766174654b6579223a226b35507976337148327349" +
		"586343595a58594f5775453149326e32554539436861556b6c4e36695a5763453d227d"
	require.NoError(t, err)
	client, err := New(api, WithExportedKey(exportedKey))
	require.NoError(t, err)

	return client
}
