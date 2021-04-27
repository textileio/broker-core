package nearclient

import (
	"context"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	"github.com/textileio/broker-core/cmd/neard/nearclient/types"

	"testing"
)

var ctx = context.Background()

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
	config := &types.Config{
		RPCClient: rpcClient,
	}
	c, err := NewClient(config)
	require.NoError(t, err)
	return c, func() {
		rpcClient.Close()
	}
}
