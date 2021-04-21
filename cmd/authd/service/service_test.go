package service_test

// @todo: on save file, run "goimports" to clean up imports

import (
	"context"
	"testing"

	golog "github.com/ipfs/go-log/v2"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/textileio/broker-core/logging"
	mocks "github.com/textileio/broker-core/mocks/broker/chainapi/v1"

	"github.com/textileio/broker-core/cmd/authd/service"
	pb "github.com/textileio/broker-core/gen/broker/auth/v1"
	chainapi "github.com/textileio/broker-core/gen/broker/chainapi/v1"
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
//     "aud": "https://broker.staging.textile.io/",
// }

func TestService_validateKeyDID(t *testing.T) {
	// Valid sub, valid x
	sub := "did:key:z6MkmabiunAzWE4ZqoX4AmPxgWEvn9Q4vrTM8bjX43hBiCX4"
	x := "aeMfwYNaIFeslhQdotW8QBuc3Mqy-hAVpOu4cNewGWM="
	ok, err := service.ValidateKeyDID(sub, x)
	require.NoError(t, err)
	require.True(t, ok)

	// Valid sub, invalid x
	sub = "did:key:z6MkmabiunAzWE4ZqoX4AmPxgWEvn9Q4vrTM8bjX43hBiCX4"
	x = "INVALID_X"
	ok, err = service.ValidateKeyDID(sub, x)
	require.Error(t, err)
	require.False(t, ok)

	// Invalid sub, valid x
	sub = "INVALID_SUB"
	x = "aeMfwYNaIFeslhQdotW8QBuc3Mqy-hAVpOu4cNewGWM="
	ok, err = service.ValidateKeyDID(sub, x)
	require.Error(t, err)
	require.False(t, ok)

	// Invalid sub, Invalid x
	sub = "INVALID_SUB"
	x = "INVALID_X"
	ok, err = service.ValidateKeyDID(sub, x)
	require.Error(t, err)
	require.False(t, ok)
}

func TestService_validateToken(t *testing.T) {
	// Valid token
	token := TOKEN
	output, err := service.ValidateToken(token)
	require.NoError(t, err)
	require.Equal(t, output.Iss, "carsonfarmer.testnet")
	require.Equal(t, output.Sub, "did:key:z6MkmabiunAzWE4ZqoX4AmPxgWEvn9Q4vrTM8bjX43hBiCX4")
	require.Equal(t, output.X, "aeMfwYNaIFeslhQdotW8QBuc3Mqy-hAVpOu4cNewGWM=")

	// Invalid token
	token = "INVALID_TOKEN"
	output, err = service.ValidateToken(token)
	require.Error(t, err)
	require.Nil(t, output)
}
func TestService_validateInput(t *testing.T) {
	// Valid token
	token := TOKEN
	req := &pb.AuthRequest{
		JwtBase64URL: token}
	input, err := service.ValidateInput(req)
	require.NoError(t, err)
	require.Equal(t, req.JwtBase64URL, input.JwtBase64URL)

	// Invalid token with valid height
	token = "INVALID_TOKEN"
	req = &pb.AuthRequest{
		JwtBase64URL: token}
	input, err = service.ValidateInput(req)
	require.Error(t, err)
	require.Nil(t, input)

	// Valid token with no height
	token = TOKEN
	req = &pb.AuthRequest{
		JwtBase64URL: token}
	input, err = service.ValidateInput(req)
	require.NoError(t, err)
	require.Equal(t, req.JwtBase64URL, input.JwtBase64URL)
}

func TestClient_ValidateLockedFunds(t *testing.T) {
	// Funds ok
	sub := "sub"
	mockChain := &mocks.ChainApiServiceClient{}
	mockChain.On(
		"HasFunds",
		mock.Anything, // this is the ctx, can't use AnythingOfType because context.Context is an interface.
		mock.AnythingOfType("*chainapi.HasFundsRequest"),
	).Return(&chainapi.HasFundsResponse{
		HasFunds: true,
	}, nil)
	ok, err := service.ValidateLockedFunds(context.Background(), sub, mockChain)
	require.NoError(t, err)
	require.True(t, ok)
	mockChain.AssertExpectations(t)

	// Funds not ok
	mockChain = &mocks.ChainApiServiceClient{}
	mockChain.On(
		"HasFunds",
		mock.Anything, // this is the ctx, can't use AnythingOfType because context.Context is an interface.
		mock.AnythingOfType("*chainapi.HasFundsRequest"),
	).Return(&chainapi.HasFundsResponse{
		HasFunds: false,
	}, nil)
	ok, err = service.ValidateLockedFunds(context.Background(), sub, mockChain)
	require.Error(t, err)
	require.False(t, ok)
	mockChain.AssertExpectations(t)
}
