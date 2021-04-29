package account

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/stretchr/testify/require"
	"github.com/textileio/broker-core/cmd/neard/nearclient/keys"
	"github.com/textileio/broker-core/cmd/neard/nearclient/transaction"
	"github.com/textileio/broker-core/cmd/neard/nearclient/types"

	"testing"
)

var ctx = context.Background()

func TestViewState(t *testing.T) {
	a, cleanup := makeAccount(t)
	defer cleanup()
	res, err := a.ViewState(ctx, ViewStateWithFinality("final"))
	require.NoError(t, err)
	require.NotNil(t, res)
}

func TestState(t *testing.T) {
	a, cleanup := makeAccount(t)
	defer cleanup()
	res, err := a.State(ctx, StateWithFinality("final"))
	require.NoError(t, err)
	require.NotNil(t, res)
}

func TestFindAccessKey(t *testing.T) {
	a, cleanup := makeAccount(t)
	defer cleanup()
	pubKey, accessKeyView, err := a.FindAccessKey(ctx, "", nil)
	require.NoError(t, err)
	require.NotNil(t, pubKey)
	require.NotNil(t, accessKeyView)
}

func TestSignTransaction(t *testing.T) {
	a, cleanup := makeAccount(t)
	defer cleanup()
	amt := big.NewInt(1000)
	sendAction := transaction.TransferAction(*amt)
	hash, signedTxn, err := a.SignTransaction(ctx, "carsonfarmer.testnet", sendAction)
	require.NoError(t, err)
	require.NotEmpty(t, hash)
	require.NotNil(t, signedTxn)
}

func TestSignAndSendTransaction(t *testing.T) {
	a, cleanup := makeAccount(t)
	defer cleanup()
	amt, ok := (&big.Int{}).SetString("1000000000000000000000000", 10)
	require.True(t, ok)
	sendAction := transaction.TransferAction(*amt)
	res, err := a.SignAndSendTransaction(ctx, "carsonfarmer.testnet", sendAction)
	require.NoError(t, err)
	require.NotNil(t, res)

	status, ok := res.GetStatus()
	fmt.Println(status, ok)

	status2, ok := res.GetStatusBasic()
	fmt.Println(status2, ok)
}

func makeAccount(t *testing.T) (*Account, func()) {
	rpcClient, err := rpc.DialContext(ctx, "https://rpc.testnet.near.org")
	require.NoError(t, err)

	keys, err := keys.NewKeyPairFromString(
		"ed25519:4TuypHBtJ3ZzLb1ooPtAncJjyJ5gNJ96j59wY62kr3qusxca51XqbxzCoG4VJmC3JmfAiGje1sW1CrpZvNCWsWu6",
	)
	require.NoError(t, err)

	config := &types.Config{
		RPCClient: rpcClient,
		NetworkID: "testnet",
		Signer:    keys,
	}
	a := NewAccount(config, "asutula.testnet")
	return a, func() {
		rpcClient.Close()
	}
}
