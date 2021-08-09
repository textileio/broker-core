package service

import (
	"context"
	"crypto/ecdsa"
	"net"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	jwt "github.com/golang-jwt/jwt"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/textileio/bidbot/lib/logging"
	ethjwt "github.com/textileio/broker-core/cmd/authd/eth"
	pb "github.com/textileio/broker-core/gen/broker/auth/v1"
	mocks "github.com/textileio/broker-core/mocks/chainapi"
	"github.com/textileio/broker-core/tests"
	golog "github.com/textileio/go-log/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const bufSize = 1024 * 1024

var privateKey *ecdsa.PrivateKey
var token string
var kid string

func init() {
	if err := logging.SetLogLevels(map[string]golog.LogLevel{
		"auth/service": golog.LevelDebug,
	}); err != nil {
		panic(err)
	}
	var err error
	privateKey, err = crypto.GenerateKey()
	if err != nil {
		panic(err)
	}

	addr := crypto.PubkeyToAddress(privateKey.PublicKey)
	kid = "eth:1337:" + addr.Hex()

	now := time.Now()
	claims := &jwt.StandardClaims{
		Issuer:    kid,
		Subject:   kid,
		Audience:  "provider",
		NotBefore: now.Unix(),
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(time.Hour).Unix(),
	}
	t := &jwt.Token{
		Header: map[string]interface{}{
			"typ": "JWT",
			"alg": ethjwt.SigningMethod.Alg(),
			"kid": kid,
		},
		Claims: claims,
		Method: ethjwt.SigningMethod,
	}
	token, err = t.SignedString(privateKey)
	if err != nil {
		panic(err)
	}
}

func TestService_validateToken(t *testing.T) {
	// Valid token
	output, err := validateToken(token)
	require.NoError(t, err)
	require.Equal(t, output.Iss, kid)
	require.Equal(t, output.Sub, kid)
	require.Equal(t, output.Aud, "provider")

	// Invalid token
	invalidToken := "INVALID_TOKEN"
	output, err = validateToken(invalidToken)
	require.Error(t, err)
	require.Nil(t, output)
}
func TestService_detectInput(t *testing.T) {
	// Valid token
	input := detectInput(token)
	require.Equal(t, token, input.token)
	require.Equal(t, chainToken, input.tokenType)

	// Raw token
	invalidToken := "RAW_TOKEN"
	input = detectInput(invalidToken)
	require.Equal(t, invalidToken, input.token)
	require.Equal(t, rawToken, input.tokenType)
}

func TestService_RawAuthToken(t *testing.T) {
	s := newService(t)
	ctx := context.Background()
	c := newClient(t, s.Config.Listener.(*bufconn.Listener))
	err := s.store.CreateAuthToken(ctx, "TOKEN-1", "IDENTITY-1", "ORIGIN-1")
	require.NoError(t, err)

	// Valid case.
	req := &pb.AuthRequest{Token: "TOKEN-1"}
	res, err := c.Auth(ctx, req)
	require.NoError(t, err)
	require.Equal(t, "IDENTITY-1", res.Identity)
	require.Equal(t, "ORIGIN-1", res.Origin)

	// Error case.
	_, err = s.Auth(ctx, &pb.AuthRequest{Token: "TOKEN-X"})
	require.Error(t, err)
}

func TestService_ValidateLockedFunds(t *testing.T) {
	// Funds ok
	sub := "sub"
	aud := "aud"
	chainID := "id"
	mockChain := &mocks.ChainAPI{}
	mockChain.On(
		"HasDeposit",
		mock.Anything, // this is the ctx, can't use AnythingOfType because context.Context is an interface.
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string"),
		mock.AnythingOfType("string"),
	).Return(true, nil)
	ok, err := validateDepositedFunds(context.Background(), aud, sub, chainID, mockChain)
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
		mock.AnythingOfType("string"),
	).Return(false, nil)
	ok, err = validateDepositedFunds(context.Background(), aud, sub, chainID, mockChain)
	require.Error(t, err)
	require.False(t, ok)
	mockChain.AssertExpectations(t)
}

// TODO: Test different types of tokens, not just eth-1337.
func TestClient_Token(t *testing.T) {
	s := newService(t)
	c := newClient(t, s.Config.Listener.(*bufconn.Listener))
	req := &pb.AuthRequest{Token: token}
	res, err := c.Auth(context.Background(), req)
	require.NoError(t, err)
	require.Equal(t, res.Identity, kid)
	require.Equal(t, res.Origin, "eth-1337")
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
	deps := Deps{NearAPI: newChainAPIClientMock(), EthAPI: newChainAPIClientMock(), PolyAPI: newChainAPIClientMock()}
	serv, err := New(config, deps)
	require.NoError(t, err)
	return serv
}
