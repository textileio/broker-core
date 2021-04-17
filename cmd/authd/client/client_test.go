package client_test

// @todo: on save file, run "goimports" to clean up imports

import (
	"context"
	"fmt"
	"testing"

	golog "github.com/ipfs/go-log/v2"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/require"
	"github.com/textileio/broker-core/cmd/authd/client"
	"github.com/textileio/broker-core/logging"
	"github.com/textileio/broker-core/rpc"

	"github.com/textileio/broker-core/cmd/authd/service"
	pb "github.com/textileio/broker-core/gen/broker/auth/v1"
)

func init() {
	if err := logging.SetLogLevels(map[string]golog.LogLevel{
		"auth/service": golog.LevelDebug,
	}); err != nil {
		panic(err)
	}
}

// TOKEN is the JWT token for testing.
var TOKEN = "eyJhbGciOiJFZERTQVNoYTI1NiIsInR5cCI6IkpXVCIsImp3ayI6eyJrdHkiOiJPS1AiLCJjcnYiOiJFZDI1NTE5IiwieCI6" +
	"ImFlTWZ3WU5hSUZlc2xoUWRvdFc4UUJ1YzNNcXktaEFWcE91NGNOZXdHV009IiwidXNlIjoic2lnIn19.eyJpc3MiOiJjYXJzb25mYXJ" +
	"tZXIudGVzdG5ldCIsInN1YiI6ImRpZDprZXk6ejZNa21hYml1bkF6V0U0WnFvWDRBbVB4Z1dFdm45UTR2clRNOGJqWDQzaEJpQ1g0Iiw" +
	"ibmJmIjoxNjE4NTg5ODU2LCJpYXQiOjE2MTg1ODk4NTYsImV4cCI6MTAxNjE4NTg5ODU2LCJhdWQiOiJodHRwczovL2Jyb2tlci5zdGF" +
	"naW5nLnRleHRpbGUuaW8vIn0=.ffEXF27CDug7F85JzpvHObAaALcV4X9_cTyfvpDqPWNejTT9SNceGD20TP6IOIDlHLZ20DLpVDamDwL" +
	"FyiPFBA=="

// Header:
// {
//     "alg": "EdDSA",
//     "typ": "JWT",
//     "jwk": {
//         "kty": "OKP",
//         "crv": "Ed25519",
//         "x": "aeMfwYNaIFeslhQdotW8QBuc3Mqy-hAVpOu4cNewGWM=",
//         "use": "sig"
//     }
// }
// Payload:
// {
//     "iss": "carsonfarmer.testnet",
//     "sub": "did:key:z6MkmabiunAzWE4ZqoX4AmPxgWEvn9Q4vrTM8bjX43hBiCX4",
//     "nbf": 1618517489,
//     "iat": 1618517489,
//     "exp": 101618517489,
//     "aud": "https://broker.staging.textile.io/"
// }

func TestClient_Create(t *testing.T) {
	jwtBase64URL := TOKEN
	c := newClient(t)
	req := &pb.AuthRequest{JwtBase64URL: jwtBase64URL}

	res, err := c.Auth(context.Background(), req)
	require.NoError(t, err)
	require.NotNil(t, res)
}

func newClient(t *testing.T) *client.Client {
	listenPort, err := freeport.GetFreePort()
	require.NoError(t, err)
	addr := fmt.Sprintf("127.0.0.1:%d", listenPort)
	s, err := service.New(addr)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, s.Close())
	})

	c, err := client.NewClient(addr, rpc.GetClientOpts(addr)...)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, c.Close())
	})

	return c
}
