package nearclient

import (
	"context"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"testing"
)

var ctx = context.Background()

func TestIt(t *testing.T) {
	c, cleanup := makeClient(t)
	defer cleanup()
	res, err := c.ViewState(ctx, "lock-box.testnet", WithFinality("final"))
	require.NoError(t, err)
	require.NotNil(t, res)
}

func makeClient(t *testing.T) (*Client, func()) {
	rpcClient, err := rpc.DialContext(ctx, "https://rpc.testnet.near.org")
	require.NoError(t, err)
	c, err := NewClient(rpcClient)
	require.NoError(t, err)
	return c, func() {
		rpcClient.Close()
	}
}
