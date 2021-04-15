package lockboxclient

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	"github.com/textileio/broker-core/cmd/neard/nearclient"
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

func makeClient(t *testing.T) (*Client, func()) {
	rpcClient, err := rpc.DialContext(ctx, "https://rpc.testnet.near.org")
	require.NoError(t, err)
	nc, err := nearclient.NewClient(rpcClient)
	require.NoError(t, err)
	c, err := NewClient(nc, "lock-box.testnet")
	require.NoError(t, err)
	return c, func() {
		rpcClient.Close()
	}
}
