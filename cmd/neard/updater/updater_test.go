package updater

import (
	"context"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	"github.com/textileio/broker-core/cmd/neard/contractclient"
	"github.com/textileio/broker-core/cmd/neard/nearclient"
	nctypes "github.com/textileio/broker-core/cmd/neard/nearclient/types"
	"github.com/textileio/broker-core/cmd/neard/statecache"
)

var ctx = context.Background()

func TestIt(t *testing.T) {
	_, _, cleanup := makeClient(t)
	defer cleanup()

	// time.Sleep(time.Second * 1000)

	// state := sc.GetState()
	// require.NotNil(t, state)
	// require.Greater(t, u.blockHeight, 0)
}

func makeClient(t *testing.T) (*statecache.StateCache, *Updater, func()) {
	rpcClient, err := rpc.DialContext(ctx, "https://rpc.testnet.near.org")
	require.NoError(t, err)

	// keys, err := keys.NewKeyPairFromString(
	// 	"ed25519:xxx",
	// )
	require.NoError(t, err)

	config := &nctypes.Config{
		RPCClient: rpcClient,
		NetworkID: "testnet",
		// Signer:    keys,
	}
	nc, err := nearclient.NewClient(config)
	require.NoError(t, err)
	c, err := contractclient.NewClient(nc, "filecoin-bridge.testnet", "filecoin-bridge.testnet")
	require.NoError(t, err)

	sc, err := statecache.NewStateCache()
	require.NoError(t, err)

	updaterConfig := Config{
		Contract:        c,
		UpdateFrequency: time.Millisecond * 500,
		RequestTimeout:  time.Minute,
		Delegate:        sc,
	}
	u := NewUpdater(updaterConfig)

	return sc, u, func() {
		rpcClient.Close()
		require.NoError(t, u.Close())
	}
}
