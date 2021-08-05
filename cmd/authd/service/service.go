package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"strings"

	eth "github.com/ethereum/go-ethereum/common"
	jwt "github.com/golang-jwt/jwt"
	crypto "github.com/libp2p/go-libp2p-core/crypto"
	"github.com/textileio/bidbot/lib/common"
	"github.com/textileio/broker-core/chainapi"
	"github.com/textileio/broker-core/cmd/authd/store"
	pb "github.com/textileio/broker-core/gen/broker/auth/v1"
	"github.com/textileio/broker-core/rpc"
	golog "github.com/textileio/go-log/v2"

	// These imports run init functions, which register each algo with jwt-go.
	_ "github.com/textileio/broker-core/cmd/authd/eth"
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
	store  *store.Store
}

// Config is the service config.
type Config struct {
	Listener    net.Listener
	PostgresURI string
}

// Deps comprises the service dependencies.
type Deps struct {
	NearAPI chainapi.ChainAPI
}

var _ pb.AuthAPIServiceServer = (*Service)(nil)

// New returns a new service.
func New(config Config, deps Deps) (*Service, error) {
	store, err := store.New(config.PostgresURI)
	if err != nil {
		return nil, fmt.Errorf("creating store: %s", err)
	}
	s := &Service{
		server: grpc.NewServer(grpc.UnaryInterceptor(common.GrpcLoggerInterceptor(log))),
		Config: config,
		Deps:   deps,
		store:  store,
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

type tokenType int

const (
	chainToken tokenType = iota
	rawToken
)

type detectedInput struct {
	token     string
	tokenType tokenType
}

// ValidatedToken represents token data that has been validated.
type ValidatedToken struct {
	Sub string
	Iss string
	Aud string
}

// AuthClaims defines standard claims for authentication.
type AuthClaims struct {
	jwt.StandardClaims
}

func detectInput(token string) *detectedInput {
	parts := strings.Split(token, ".")
	if len(parts) == 3 {
		return &detectedInput{
			token:     token,
			tokenType: chainToken,
		}
	}

	return &detectedInput{
		token:     token,
		tokenType: rawToken,
	}
}

// validateToken validates the JWT token.
func validateToken(jwtBase64URL string) (*ValidatedToken, error) {
	token, err := jwt.ParseWithClaims(jwtBase64URL, &AuthClaims{}, func(token *jwt.Token) (interface{}, error) {
		header, ok := token.Header["jwk"]
		if ok {
			for k, v := range header.(map[string]interface{}) {
				if val, ok := v.(string); !ok {
					continue
				} else if k == "x" {
					dx, _ := base64.URLEncoding.DecodeString(val)
					return crypto.UnmarshalEd25519PublicKey(dx)
				}
			}
		}
		header, ok = token.Header["kid"]
		if ok {
			var val string
			if val, ok = header.(string); !ok {
				return nil, errors.New("invalid key info")
			}
			addrString := strings.Split(val, ":")[2]
			addrBytes := eth.FromHex(addrString)
			return eth.BytesToAddress(addrBytes), nil
		}
		return nil, errors.New("invalid key info")
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
		Sub: claims.Subject,
		Iss: claims.Issuer,
		Aud: claims.Audience,
	}
	return validatedToken, err
}

// // validateKeyDID validates the key DID.
// // See the spec: https://w3c-ccg.github.io/did-method-key/
// func validateKeyDID(sub string, x string) (bool, error) {
// 	subDID, err := did.Parse(sub)
// 	if err != nil {
// 		return false, fmt.Errorf("parsing DID: %v", err)
// 	}
// 	_, bytes, err := mbase.Decode(subDID.ID)
// 	if err != nil {
// 		return false, fmt.Errorf("decoding DID: %v", err)
// 	}
// 	// Checks that the first two bytes are multicodec prefix values (according to spec)
// 	_, n, err := varint.FromUvarint(bytes)
// 	if err != nil {
// 		return false, fmt.Errorf("DID multiformat: %v", err)
// 	}
// 	if n != 2 {
// 		return false, errors.New("key DID format")
// 	}
// 	dx, _ := base64.URLEncoding.DecodeString(x)
// 	if string(dx) != string(bytes[2:]) {
// 		return false, errors.New("key DID does not match the public key")
// 	}
// 	return true, nil
// }

// validateDepositedFunds validates that the user has locked funds on chain.
func validateDepositedFunds(
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

// Auth authenticates a user storage request.
// It detects raw bearer tokens or JTWs associated with the NEAR blockchain.
// The latter is an URL encoded base64 JWT with an Ed25519 signature.
// 1. Validates the JWT
// 2. Validates that the key DID in the JWT ("sub" in the payload) was created with the public key ("x" in the header)
// 3. Validates that the user has locked funds on-chain using a service provided by neard.
// It returns the key DID.
func (s *Service) Auth(ctx context.Context, req *pb.AuthRequest) (*pb.AuthResponse, error) {
	input := detectInput(req.Token)

	switch input.tokenType {
	case rawToken:
		rt, ok, err := s.store.GetAuthToken(ctx, input.token)
		if err != nil {
			return nil, fmt.Errorf("find raw token: %s", err)
		}
		if !ok {
			return nil, fmt.Errorf("raw token doesn't exist")
		}
		log.Debugf("successful raw authentication identity %s, origin %s", rt.Identity, rt.Origin)
		return &pb.AuthResponse{
			Identity: rt.Identity,
			Origin:   rt.Origin,
		}, nil
	case chainToken:
		// Validate the JWT token.
		token, err := validateToken(input.token)
		if err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "invalid JWT: %v", err)
		}
		// TODO: On NEAR, check that the account is indeed associated with the given public key
		// Check for locked funds
		fundsOk, err := validateDepositedFunds(ctx, token.Aud, token.Iss, s.Deps.NearAPI)
		if !fundsOk || err != nil {
			return nil, status.Errorf(codes.Unauthenticated, "locked funds: %v", err)
		}
		log.Debugf("successful chain authentication: %s", token.Iss)
		return &pb.AuthResponse{
			Identity: token.Sub,
			Origin:   "NEAR",
		}, nil
	default:
		return nil, status.Errorf(codes.InvalidArgument, "unknown token type")
	}
}
