package registryclient

import (
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	api "github.com/textileio/near-api-go"
	"github.com/textileio/near-api-go/types"
)

var ctx = context.Background()

func TestIt(t *testing.T) {
	c, cleanup := makeClient(t)
	defer cleanup()
	require.NotNil(t, c)
}

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

	// keys, err := keys.NewKeyPairFromString(
	// 	"ed25519:",
	// )
	// require.NoError(t, err)

	config := &types.Config{
		RPCClient: rpcClient,
		NetworkID: "testnet",
		// Signer:    keys,
	}
	nc, err := api.NewClient(config)
	require.NoError(t, err)
	c, err := NewClient(nc, "bridge-registry.testnet", "asutula.testnet")
	require.NoError(t, err)
	return c, func() {
		rpcClient.Close()
	}
}
