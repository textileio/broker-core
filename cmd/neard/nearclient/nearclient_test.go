package nearclient

import (
	"context"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"

	"testing"
)

var ctx = context.Background()

func TestViewState(t *testing.T) {
	c, cleanup := makeClient(t)
	defer cleanup()
	res, err := c.ViewState(ctx, "lock-box.testnet", ViewStateWithFinality("final"))
	require.NoError(t, err)
	require.NotNil(t, res)
}

func TestViewAccount(t *testing.T) {
	c, cleanup := makeClient(t)
	defer cleanup()
	res, err := c.ViewAccount(ctx, "lock-box.testnet", ViewAccountWithFinality("final"))
	require.NoError(t, err)
	require.NotNil(t, res)
}

func TestDataChanges(t *testing.T) {
	c, cleanup := makeClient(t)
	defer cleanup()
	res, err := c.DataChanges(ctx, []string{"lock-box.testnet"}, DataChangesWithFinality("final"))
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
