package account

import (
	"context"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	"github.com/textileio/broker-core/cmd/neard/nearclient/types"

	"testing"
)

var ctx = context.Background()

func TestViewState(t *testing.T) {
	c, cleanup := makeAccount(t)
	defer cleanup()
	res, err := c.ViewState(ctx, ViewStateWithFinality("final"))
	require.NoError(t, err)
	require.NotNil(t, res)
}

func TestState(t *testing.T) {
	c, cleanup := makeAccount(t)
	defer cleanup()
	res, err := c.State(ctx, StateWithFinality("final"))
	require.NoError(t, err)
	require.NotNil(t, res)
}

func makeAccount(t *testing.T) (*Account, func()) {
	rpcClient, err := rpc.DialContext(ctx, "https://rpc.testnet.near.org")
	require.NoError(t, err)
	config := &types.Config{
		RPCClient: rpcClient,
		NetworkID: "testnet",
	}
	a := NewAccount(config, "lock-box.testnet")
	return a, func() {
		rpcClient.Close()
	}
}
