package lockboxclient

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	"github.com/textileio/broker-core/cmd/neard/nearclient"
	"github.com/textileio/broker-core/cmd/neard/nearclient/types"
)

var ctx = context.Background()

func TestIt(t *testing.T) {
	c, cleanup := makeClient(t)
	defer cleanup()
	require.NotNil(t, c)
}

// func TestSetBroker(t *testing.T) {
// 	c, cleanup := makeClient(t)
// 	defer cleanup()
// 	res, err := c.SetBroker(ctx, "aaronbroker", []string{"address1"})
// 	require.NoError(t, err)
// 	require.NotNil(t, res)
// }

// func TestDeleteBroker(t *testing.T) {
// 	c, cleanup := makeClient(t)
// 	defer cleanup()
// 	err := c.DeleteBroker(ctx, "aaronbroker")
// 	require.NoError(t, err)
// }

// func TestGetBroker(t *testing.T) {
// 	c, cleanup := makeClient(t)
// 	defer cleanup()
// 	res, err := c.GetBroker(ctx, "aaronbroker")
// 	require.NoError(t, err)
// 	require.NotNil(t, res)
// }

// func TestListBrokers(t *testing.T) {
// 	c, cleanup := makeClient(t)
// 	defer cleanup()
// 	res, err := c.ListBrokers(ctx)
// 	require.NoError(t, err)
// 	require.NotEmpty(t, res)
// }

// func TestAddDeposit(t *testing.T) {
// 	c, cleanup := makeClient(t)
// 	defer cleanup()
// 	res, err := c.AddDeposit(ctx, "aaronbroker")
// 	require.NoError(t, err)
// 	require.NotNil(t, res)
// }

// func TestHasDeposit(t *testing.T) {
// 	c, cleanup := makeClient(t)
// 	defer cleanup()
// 	res, err := c.HasDeposit(ctx, "aaronbroker", "asutula.testnet")
// 	require.NoError(t, err)
// 	require.True(t, res)
// }

// func TestGetAccount(t *testing.T) {
// 	c, cleanup := makeClient(t)
// 	defer cleanup()
// 	res, err := c.GetAccount(ctx)
// 	require.NoError(t, err)
// 	require.NotNil(t, res)
// }

// func TestGetState(t *testing.T) {
// 	c, cleanup := makeClient(t)
// 	defer cleanup()
// 	res, err := c.GetState(ctx)
// 	require.NoError(t, err)
// 	require.NotNil(t, res)
// }

// func TestUpdatePayload(t *testing.T) {
// 	c, cleanup := makeClient(t)
// 	defer cleanup()

// 	info, err := c.SetBroker(ctx, "asutula.testnet", []string{"addr1"})
// 	require.NoError(t, err)
// 	require.NotNil(t, info)

// 	opts := PayloadOptions{
// 		PieceCid: "pieceCid2",
// 		Deals: []DealInfo{
// 			{
// 				DealID:     "dealId",
// 				MinerID:    "minerId",
// 				Expiration: 100,
// 			},
// 		},
// 		DataCids: []string{"cid1, cid2, cid3, cid4"},
// 	}
// 	err = c.UpdatePayload(ctx, "payloadCid", opts)
// 	require.NoError(t, err)
// }

// func TestListPayloads(t *testing.T) {
// 	c, cleanup := makeClient(t)
// 	defer cleanup()
// 	res, err := c.ListPayloads(ctx, 0, 100)
// 	require.NoError(t, err)
// 	require.NotNil(t, res)
// }

// func TestPayloadJSON(t *testing.T) {
// 	p := &PayloadInfo{
// 		PayloadCid: "payloadCid",
// 		PieceCid:   "pieceCid",
// 		Deals: []DealInfo{
// 			{
// 				DealID:     "dealId",
// 				MinerID:    "minerId",
// 				Expiration: 100,
// 			},
// 		},
// 	}
// 	bytes, err := json.MarshalIndent(p, "", "  ")
// 	require.NoError(t, err)
// 	require.NotEmpty(t, bytes)

// 	var q PayloadInfo
// 	err = json.Unmarshal(bytes, &q)
// 	require.NoError(t, err)
// 	require.Equal(t, *p, q)
// }

func makeClient(t *testing.T) (*Client, func()) {
	rpcClient, err := rpc.DialContext(ctx, "https://rpc.testnet.near.org")
	require.NoError(t, err)

	// keys, err := keys.NewKeyPairFromString(
	// 	"ed25519xxxx",
	// )
	// require.NoError(t, err)

	config := &types.Config{
		RPCClient: rpcClient,
		NetworkID: "testnet",
		// Signer:    keys,
	}
	nc, err := nearclient.NewClient(config)
	require.NoError(t, err)
	c, err := NewClient(nc, "asutula.testnet", "asutula.testnet")
	require.NoError(t, err)
	return c, func() {
		rpcClient.Close()
	}
}
