package identity

import (
	"testing"

	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/mr-tron/base58/base58"
	"github.com/stretchr/testify/require"
)

func TestIt(t *testing.T) {
	pkey := "4TuypHBtJ3ZzLb1ooPtAncJjyJ5gNJ96j59wY62kr3qusxca51XqbxzCoG4VJmC3JmfAiGje1sW1CrpZvNCWsWu6"
	bytes, err := base58.Decode(pkey)
	require.NoError(t, err)
	require.NotEmpty(t, bytes)
	privKey, err := crypto.UnmarshalEd25519PrivateKey(bytes)
	require.NoError(t, err)
	require.NotNil(t, privKey)
	pubKey := privKey.GetPublic()
	require.NotNil(t, pubKey)
}
