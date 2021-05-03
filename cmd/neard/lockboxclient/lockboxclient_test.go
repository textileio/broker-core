package lockboxclient

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	"github.com/textileio/broker-core/cmd/neard/nearclient"
	"github.com/textileio/broker-core/cmd/neard/nearclient/keys"
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
// 	require.NotNil(t, res)
// }

// func TestLockFunds(t *testing.T) {
// 	c, cleanup := makeClient(t)
// 	defer cleanup()
// 	res, err := c.LockFunds(ctx, "aaronbroker")
// 	require.NoError(t, err)
// 	require.NotNil(t, res)
// }

// func TestHasLocked(t *testing.T) {
// 	c, cleanup := makeClient(t)
// 	defer cleanup()
// 	res, err := c.HasLocked(ctx, "aaronbroker", "lock-box.testnet")
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

func makeClient(t *testing.T) (*Client, func()) {
	rpcClient, err := rpc.DialContext(ctx, "https://rpc.testnet.near.org")
	require.NoError(t, err)

	keys, err := keys.NewKeyPairFromString(
		"ed25519:xxxx",
	)
	require.NoError(t, err)

	config := &types.Config{
		RPCClient: rpcClient,
		NetworkID: "testnet",
		Signer:    keys,
	}
	nc, err := nearclient.NewClient(config)
	require.NoError(t, err)
	c, err := NewClient(nc, "lock-box.testnet")
	require.NoError(t, err)
	return c, func() {
		rpcClient.Close()
	}
}
