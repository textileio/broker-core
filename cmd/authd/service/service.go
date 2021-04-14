package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	jwt "github.com/dgrijalva/jwt-go"
	logging "github.com/ipfs/go-log/v2"
	mbase "github.com/multiformats/go-multibase"
	varint "github.com/multiformats/go-varint"
	"github.com/ockam-network/did"

	crypto "github.com/libp2p/go-libp2p-core/crypto"
	pb "github.com/textileio/broker-core/gen/broker/auth/v1"
	chainapi "github.com/textileio/broker-core/gen/broker/chainapi/v1"

	// This import runs the init, which registers the algo with jwt-go.
	_ "github.com/textileio/jwt-go-eddsa"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const (
	// LogName is our auth logger.
	LogName = "auth/service"
)

var (
	log = logging.Logger(LogName)
)

// Service is a gRPC service for authorization.
type Service struct {
	pb.UnimplementedAPIServiceServer
	Config
	Deps
	server *grpc.Server
}

// Config is the service config.
type Config struct {
	ListenAddr string
}

// Deps comprises the service dependencies.
type Deps struct {
	ChainAPIServiceClient chainapi.ChainApiServiceClient
}

var _ pb.APIServiceServer = (*Service)(nil)

// New returns a new service.
func New(config Config, deps Deps) (*Service, error) {
	s := &Service{
		server: grpc.NewServer(),
		Config: config,
		Deps:   deps,
	}
	listener, err := net.Listen("tcp", config.ListenAddr)
	if err != nil {
		return nil, fmt.Errorf("getting net listener: %v", err)
	}
	go func() {
		pb.RegisterAPIServiceServer(s.server, s)
		if err := s.server.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			log.Errorf("server error: %v", err)
		}
	}()

	log.Infof("service listening at %s", config.ListenAddr)
	return s, nil
}

// Close the service.
func (s *Service) Close() error {
	stopped := make(chan struct{})
	go func() {
		s.server.GracefulStop()
		close(stopped)
	}()
	timer := time.NewTimer(10 * time.Second)
	select {
	case <-timer.C:
		s.server.Stop()
	case <-stopped:
		timer.Stop()
	}
	log.Info("service was shutdown")
	return nil
}

// ValidatedInput represents input that has been validated.
type ValidatedInput struct {
	JwtBase64URL string
}

// ValidatedToken represents token data that has been validated.
type ValidatedToken struct {
	X   string
	Sub string
	Iss string
	Blk uint64
}

// AuthClaims defines a verifiable claim to an identity.
type AuthClaims struct {
	jwt.StandardClaims
	Blk uint64 `json:"blk,omitempty"`
}

// ValidateInput sanity checks the raw inputs to the service.
func ValidateInput(input *pb.AuthRequest) (*ValidatedInput, error) {
	parts := strings.Split(input.JwtBase64URL, ".")
	if len(parts) != 3 {
		return nil, status.Errorf(codes.InvalidArgument, "Token contains invalid number of segments")
	}
	return &ValidatedInput{
		JwtBase64URL: input.JwtBase64URL,
	}, nil
}

// ValidateToken validates the JWT token.
func ValidateToken(jwtBase64URL string) (*ValidatedToken, error) {
	jwkMap := map[string]string{}
	token, err := jwt.ParseWithClaims(jwtBase64URL, &AuthClaims{}, func(token *jwt.Token) (interface{}, error) {
		jwk := token.Header["jwk"]
		for k, v := range jwk.(map[string]interface{}) {
			foo, ok := v.(string)
			if !ok {
				continue
			} else {
				jwkMap[k] = foo
			}
		}
		x := jwkMap["x"]
		dx, _ := base64.URLEncoding.DecodeString(x)
		pkey, err := crypto.UnmarshalEd25519PublicKey(dx)
		return pkey, err
	})
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Parsing JWT: %s", err))
	}
	if !token.Valid {
		return nil, status.Errorf(codes.Unauthenticated, "JWT invalid")
	}

	claims, ok := token.Claims.(*AuthClaims)
	if claims.Valid() != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Invalid claims: %s", err))
	}
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "Invalid claims")
	}

	validatedToken := &ValidatedToken{
		X:   jwkMap["x"],
		Sub: claims.Subject,
		Iss: claims.Issuer,
		Blk: claims.Blk,
	}
	return validatedToken, err
}

// ValidateKeyDID validates the key DID.
func ValidateKeyDID(sub string, x string) (bool, error) {
	subDID, err := did.Parse(sub)
	if err != nil {
		return false, status.Errorf(codes.Unauthenticated, "Error parsing DID")
	}
	_, bytes, err := mbase.Decode(subDID.ID)
	if err != nil {
		return false, status.Errorf(codes.Unauthenticated, "Error decoding DID")
	}
	_, n, err := varint.FromUvarint(bytes)
	if err != nil {
		return false, status.Errorf(codes.Unauthenticated, "DID multiformat error")
	}
	if n != 2 {
		return false, status.Errorf(codes.Unauthenticated, "Key DID format error")
	}
	dx, _ := base64.URLEncoding.DecodeString(x)
	if string(dx) != string(bytes[2:]) {
		return false, status.Errorf(codes.Unauthenticated, "Key DID does not match the public key")
	}
	return true, nil
}

// ValidateLockedFunds validates that the user has locked funds on chain.
func ValidateLockedFunds(ctx context.Context, iss string, blk uint64, s chainapi.ChainApiServiceClient) (bool, error) {
	var chainReq = &chainapi.HasFundsRequest{
		BlockHeight: blk,
		AccountId:   iss,
	}
	chainRes, err := s.HasFunds(ctx, chainReq)
	if err != nil {
		return false, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Locked funds error: %s", err))
	}
	if !chainRes.HasFunds {
		return false, status.Errorf(codes.InvalidArgument, "Doesn't have locked funds")
	}
	return true, nil
}

// Auth takes in a base64 encoded JWT, verifies it, checks the indentity (by comparing key DIDs) and returns DID.
func (s *Service) Auth(ctx context.Context, req *pb.AuthRequest) (*pb.AuthResponse, error) {
	// Validate the request input.
	validInput, InputErr := ValidateInput(req)
	if InputErr != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Invalid request input: %s (%s)", req, InputErr))
	}
	// Validate the JWT token.
	token, tokenErr := ValidateToken(validInput.JwtBase64URL)
	if tokenErr != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Invalid JWT: %s", tokenErr))
	}
	// Validate the key DID.
	keyOk, keyErr := ValidateKeyDID(token.Sub, token.X)
	if !keyOk || keyErr != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Invalid Key DID: %s", keyErr))
	}
	// Check for locked funds
	fundsOk, fundsErr := ValidateLockedFunds(ctx, token.Iss, token.Blk, s.Deps.ChainAPIServiceClient)
	if !fundsOk || fundsErr != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("Locked funds error: %s", fundsErr))
	}
	return &pb.AuthResponse{
		Identity: token.Sub,
	}, nil
}
