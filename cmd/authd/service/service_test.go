package service

import (
	"context"
	"net"
	"testing"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/textileio/bidbot/lib/logging"
	pb "github.com/textileio/broker-core/gen/broker/auth/v1"
	mocks "github.com/textileio/broker-core/mocks/chainapi"
	"github.com/textileio/broker-core/tests"
	golog "github.com/textileio/go-log/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

func init() {
	if err := logging.SetLogLevels(map[string]golog.LogLevel{
		"auth/service": golog.LevelDebug,
	}); err != nil {
		panic(err)
	}
}

// TOKEN is the JWT token for testing.
var TOKEN = "eyJhbGciOiJFZERTQVNoYTI1NiIsInR5cCI6IkpXVCIsImp3ayI6eyJrdHkiOiJPS1" +
	"AiLCJjcnYiOiJFZDI1NTE5IiwieCI6IjZURnVRRzFGTHZ4UGxPdGFVbllFQlRlU3ha" +
	"a09GZ3VSSGZwNlN1Q1ZDbG89IiwidXNlIjoic2lnIn19.eyJhdWQiOiJhYXJvbmJyb2" +
	"tlciIsImlzcyI6ImNhcnNvbmZhcm1lci50ZXN0bmV0Iiwic3ViIjoiZGlkOmtleTp6N" +
	"k1rdjlZa25rMzZlUzhwY1pkZjgyWXhIcnBpWmJZZDFFYlNld0R2WEM3amhRRDciLCJu" +
	"YmYiOjE2MjAwODY2NDMsImlhdCI6MTYyMDA4NjY0MywiZXhwIjozNjAwMDAwMDE2MjA" +
	"wODY2NjB9.XcGW8z7HEVy6gZl2ZP0yGPyetlcXal8d86_YKvIor8vFQWYS9zSu4vxYm" +
	"KutmsVkVu2gsopkdF3hsw0_qjCLDQ=="

// The unecoded TOKEN:
//
// Header:
// {
//     "alg": "EdDSASha256",
//     "typ": "JWT",
//     "jwk": {
//         "kty": "OKP",
//         "crv": "Ed25519",
//         "x": "6TFuQG1FLvxPlOtaUnYEBTeSxZkOFguRHfp6SuCVClo=",
//         "use": "sig"
//     }
// }
// Payload:
// {
//     "aud": "aaronbroker",
//     "iss": "carsonfarmer.testnet",
//     "sub": "did:key:z6Mkv9Yknk36eS8pcZdf82YxHrpiZbYd1EbSewDvXC7jhQD7",
//     "nbf": 1620086643,
//     "iat": 1620086643,
//     "exp": 360000001620086660
// }

func TestService_validateKeyDID(t *testing.T) {
	// Valid sub, valid x
	sub := "did:key:z6MkmabiunAzWE4ZqoX4AmPxgWEvn9Q4vrTM8bjX43hBiCX4"
	x := "aeMfwYNaIFeslhQdotW8QBuc3Mqy-hAVpOu4cNewGWM="
	ok, err := validateKeyDID(sub, x)
	require.NoError(t, err)
	require.True(t, ok)

	// Valid sub, invalid x
	sub = "did:key:z6MkmabiunAzWE4ZqoX4AmPxgWEvn9Q4vrTM8bjX43hBiCX4"
	x = "INVALID_X"
	ok, err = validateKeyDID(sub, x)
	require.Error(t, err)
	require.False(t, ok)

	// Invalid sub, valid x
	sub = "INVALID_SUB"
	x = "aeMfwYNaIFeslhQdotW8QBuc3Mqy-hAVpOu4cNewGWM="
	ok, err = validateKeyDID(sub, x)
	require.Error(t, err)
	require.False(t, ok)

	// Invalid sub, Invalid x
	sub = "INVALID_SUB"
	x = "INVALID_X"
	ok, err = validateKeyDID(sub, x)
	require.Error(t, err)
	require.False(t, ok)
}

func TestService_validateToken(t *testing.T) {
	// Valid token
	token := TOKEN
	output, err := validateToken(token)
	require.NoError(t, err)
	require.Equal(t, output.Iss, "carsonfarmer.testnet")
	require.Equal(t, output.Sub, "did:key:z6Mkv9Yknk36eS8pcZdf82YxHrpiZbYd1EbSewDvXC7jhQD7")
	require.Equal(t, output.X, "6TFuQG1FLvxPlOtaUnYEBTeSxZkOFguRHfp6SuCVClo=")
	require.Equal(t, output.Aud, "aaronbroker")

	// Invalid token
	token = "INVALID_TOKEN"
	output, err = validateToken(token)
	require.Error(t, err)
	require.Nil(t, output)
}
func TestService_detectInput(t *testing.T) {
	// Valid token
	token := TOKEN
	input, err := detectInput(token)
	require.NoError(t, err)
	require.Equal(t, token, input.token)
	require.Equal(t, chainToken, input.tokenType)

	// Raw token
	token = "RAW_TOKEN"
	input, err = detectInput(token)
	require.NoError(t, err)
	require.Equal(t, token, input.token)
	require.Equal(t, rawToken, input.tokenType)

	// Valid token with no height
	token = TOKEN
	input, err = detectInput(token)
	require.NoError(t, err)
	require.Equal(t, token, input.token)
	require.Equal(t, chainToken, input.tokenType)
}

func TestService_ValidateLockedFunds(t *testing.T) {
	// Funds ok
	sub := "sub"
	aud := "aud"
	mockChain := &mocks.ChainAPI{}
	mockChain.On(
		"HasDeposit",
		mock.Anything, // this is the ctx, can't use AnythingOfType because context.Context is an interface.
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string"),
	).Return(true, nil)
	ok, err := validateDepositedFunds(context.Background(), aud, sub, mockChain)
	require.NoError(t, err)
	require.True(t, ok)
	mockChain.AssertExpectations(t)

	// Funds not ok
	mockChain = &mocks.ChainAPI{}
	mockChain.On(
		"HasDeposit",
		mock.Anything, // this is the ctx, can't use AnythingOfType because context.Context is an interface.
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string"),
	).Return(false, nil)
	ok, err = validateDepositedFunds(context.Background(), aud, sub, mockChain)
	require.Error(t, err)
	require.False(t, ok)
	mockChain.AssertExpectations(t)
}

func TestClient_Setup(t *testing.T) {
	s := newService(t)
	c := newClient(t, s.Config.Listener.(*bufconn.Listener))
	req := &pb.AuthRequest{Token: TOKEN}
	res, err := c.Auth(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, res.Identity, "did:key:z6Mkv9Yknk36eS8pcZdf82YxHrpiZbYd1EbSewDvXC7jhQD7")
}

func newClient(t *testing.T, listener *bufconn.Listener) pb.AuthAPIServiceClient {
	bufDialer := func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}
	conn, err := grpc.Dial("bufnet", grpc.WithContextDialer(bufDialer), grpc.WithInsecure())
	require.NoError(t, err)
	return pb.NewAuthAPIServiceClient(conn)
}

func newChainAPIClientMock() *mocks.ChainAPI {
	mockChain := &mocks.ChainAPI{}
	mockChain.On(
		"HasDeposit",
		mock.Anything, // this is the ctx, can't use AnythingOfType because context.Context is an interface.
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string"),
	).Return(true, nil)
	return mockChain
}

func newService(t *testing.T) *Service {
	listener := bufconn.Listen(bufSize)
	u, err := tests.PostgresURL()
	require.NoError(t, err)
	config := Config{
		Listener:    listener,
		PostgresURI: u,
	}
	deps := Deps{NearAPI: newChainAPIClientMock()}
	serv, err := New(config, deps)
	require.NoError(t, err)
	return serv
}
