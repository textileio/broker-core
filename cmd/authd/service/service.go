package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"strings"

	jwt "github.com/dgrijalva/jwt-go"
	golog "github.com/ipfs/go-log/v2"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	mbase "github.com/multiformats/go-multibase"
	varint "github.com/multiformats/go-varint"
	"github.com/ockam-network/did"
	"github.com/textileio/broker-core/chainapi"
	pb "github.com/textileio/broker-core/gen/broker/auth/v1"
	"github.com/textileio/broker-core/rpc"

	// This import runs the init, which registers the algo with jwt-go.
	_ "github.com/textileio/jwt-go-eddsa"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var log = golog.Logger("auth/service")

// Service is a gRPC service for authorization.
type Service struct {
	pb.UnimplementedAuthAPIServiceServer
	Config
	Deps
	server *grpc.Server
}

// Config is the service config.
type Config struct {
	Listener net.Listener
}

// Deps comprises the service dependencies.
type Deps struct {
	NearAPI chainapi.ChainAPI
}

var _ pb.AuthAPIServiceServer = (*Service)(nil)

// New returns a new service.
func New(config Config, deps Deps) (*Service, error) {
	s := &Service{
		server: grpc.NewServer(),
		Config: config,
		Deps:   deps,
	}
	go func() {
		pb.RegisterAuthAPIServiceServer(s.server, s)
		if err := s.server.Serve(config.Listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			log.Errorf("creating new server: %v", err)
		}
	}()
	return s, nil
}

// Close the service.
func (s *Service) Close() error {
	rpc.StopServer(s.server)
	log.Info("service was shutdown")
	return nil
}

// ValidatedInput represents input that has been validated.
type ValidatedInput struct {
	Token string // The base64 URL encoded JWT token
}

// ValidatedToken represents token data that has been validated.
type ValidatedToken struct {
	X   string
	Sub string
	Iss string
	Aud string
}

// AuthClaims defines standard claims for authentication.
type AuthClaims struct {
	jwt.StandardClaims
}

// ValidateInput sanity checks the raw inputs to the service.
func ValidateInput(jwtBase64URL string) (*ValidatedInput, error) {
	parts := strings.Split(jwtBase64URL, ".")
	if len(parts) != 3 {
		return nil, errors.New("token contains invalid number of segments")
	}
	return &ValidatedInput{
		Token: jwtBase64URL,
	}, nil
}

// ValidateToken validates the JWT token.
func ValidateToken(jwtBase64URL string) (*ValidatedToken, error) {
	jwkMap := map[string]string{}
	token, err := jwt.ParseWithClaims(jwtBase64URL, &AuthClaims{}, func(token *jwt.Token) (interface{}, error) {
		jwk := token.Header["jwk"]
		for k, v := range jwk.(map[string]interface{}) {
			if val, ok := v.(string); !ok {
				continue
			} else {
				jwkMap[k] = val
			}
		}
		x := jwkMap["x"]
		dx, _ := base64.URLEncoding.DecodeString(x)
		pkey, err := crypto.UnmarshalEd25519PublicKey(dx)
		return pkey, err
	})
	if err != nil {
		return nil, fmt.Errorf("unable to parse JWT: %v", err)
	}
	if !token.Valid {
		return nil, errors.New("the JWT is invalid")
	}

	claims, ok := token.Claims.(*AuthClaims)
	if claims.Valid() != nil {
		return nil, errors.New("invalid JWT claims")
	}
	if !ok {
		return nil, errors.New("invalid JWT claims")
	}

	validatedToken := &ValidatedToken{
		X:   jwkMap["x"],
		Sub: claims.Subject,
		Iss: claims.Issuer,
		Aud: claims.Audience,
	}
	return validatedToken, err
}

// ValidateKeyDID validates the key DID.
// See the spec: https://w3c-ccg.github.io/did-method-key/
func ValidateKeyDID(sub string, x string) (bool, error) {
	subDID, err := did.Parse(sub)
	if err != nil {
		return false, fmt.Errorf("parsing DID: %v", err)
	}
	_, bytes, err := mbase.Decode(subDID.ID)
	if err != nil {
		return false, fmt.Errorf("decoding DID: %v", err)
	}
	// Checks that the first two bytes are multicodec prefix values (according to spec)
	_, n, err := varint.FromUvarint(bytes)
	if err != nil {
		return false, fmt.Errorf("DID multiformat: %v", err)
	}
	if n != 2 {
		return false, errors.New("key DID format")
	}
	dx, _ := base64.URLEncoding.DecodeString(x)
	if string(dx) != string(bytes[2:]) {
		return false, errors.New("key DID does not match the public key")
	}
	return true, nil
}

// ValidateDepositedFunds validates that the user has locked funds on chain.
func ValidateDepositedFunds(
	ctx context.Context,
	brokerID string,
	accountID string,
	s chainapi.ChainAPI,
) (bool, error) {
	hasDeposit, err := s.HasDeposit(ctx, brokerID, accountID)
	if err != nil {
		return false, fmt.Errorf("locked funds: %v", err)
	}
	if !hasDeposit {
		return false, errors.New("account doesn't have deposited funds")
	}
	return true, nil
}

// Auth authenticates a user storage request containing a URL encoded base64 JWT with an Ed25519 signature.
// 1. Validates the JWT
// 2. Validates that the key DID in the JWT ("sub" in the payload) was created with the public key ("x" in the header)
// 3. Validates that the user has locked funds on-chain using a service provided by neard.
// It returns the key DID.
func (s *Service) Auth(ctx context.Context, req *pb.AuthRequest) (*pb.AuthResponse, error) {
	// Validate the request input.
	validInput, InputErr := ValidateInput(req.Token)
	if InputErr != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid request input: %s (%v)", req, InputErr)
	}
	// Validate the JWT token.
	token, tokenErr := ValidateToken(validInput.Token)
	if tokenErr != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid JWT: %v", tokenErr)
	}
	// Validate the key DID.
	keyOk, keyErr := ValidateKeyDID(token.Sub, token.X)
	if !keyOk || keyErr != nil {
		return nil, status.Errorf(codes.Unauthenticated, "invalid Key DID: %v", keyErr)
	}
	// Check for locked funds
	fundsOk, fundsErr := ValidateDepositedFunds(ctx, token.Aud, token.Iss, s.Deps.NearAPI)
	if !fundsOk || fundsErr != nil {
		return nil, status.Errorf(codes.Unauthenticated, "locked funds: %v", fundsErr)
	}
	log.Info(fmt.Sprintf("Authenticated successfully: %s", token.Iss))
	return &pb.AuthResponse{
		Identity: token.Sub,
	}, nil
}
