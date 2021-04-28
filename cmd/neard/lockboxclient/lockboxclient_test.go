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
	res, err := c.GetState(ctx)
	require.NoError(t, err)
	require.NotNil(t, res)
	require.NotEmpty(t, res.BlockHash)
	require.NotEmpty(t, res.BlockHeight)
	require.NotEmpty(t, res.LockedFunds)
}

func TestHasLocked(t *testing.T) {
	c, cleanup := makeClient(t)
	defer cleanup()
	res, err := c.HasLocked(ctx, "eightysteele.testnet")
	require.NoError(t, err)
	require.True(t, res)
}

func TestGetAccount(t *testing.T) {
	c, cleanup := makeClient(t)
	defer cleanup()
	res, err := c.GetAccount(ctx)
	require.NoError(t, err)
	require.NotNil(t, res)
}

func TestGetState(t *testing.T) {
	c, cleanup := makeClient(t)
	defer cleanup()
	res, err := c.GetState(ctx)
	require.NoError(t, err)
	require.NotNil(t, res)
}

func makeClient(t *testing.T) (*Client, func()) {
	rpcClient, err := rpc.DialContext(ctx, "https://rpc.testnet.near.org")
	require.NoError(t, err)
	nc, err := nearclient.NewClient(&types.Config{RPCClient: rpcClient})
	require.NoError(t, err)
	c, err := NewClient(nc, "lock-box.testnet")
	require.NoError(t, err)
	return c, func() {
		rpcClient.Close()
	}
}
