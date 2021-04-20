package service

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
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

// Auth takes in a base64 encoded JWT, verifies it, checks the indentity (by comparing key DIDs) and returns DID.
func (s *Service) Auth(ctx context.Context, req *pb.AuthRequest) (*pb.AuthResponse, error) {
	jwtBase64URL := req.JwtBase64URL

	// token.header.x
	var x string

	// Validate the JWT.
	// TODO break this out to another function.
	token, err := jwt.ParseWithClaims(jwtBase64URL, &jwt.StandardClaims{}, func(token *jwt.Token) (interface{}, error) {
		jwk := token.Header["jwk"]
		jwkMap := map[string]string{}
		for k, v := range jwk.(map[string]interface{}) {
			foo, ok := v.(string)
			if !ok {
				continue
			} else {
				jwkMap[k] = foo
			}
		}
		x = jwkMap["x"]
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

	// Get the claims.
	// TODO break out to another function.
	claims, ok := token.Claims.(*jwt.StandardClaims)
	if claims.Valid() != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Invalid claims: %s", err))
	}
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "Invalid claims")
	}

	// Verify the key DID.
	// TODO: break this out to another function.
	// TODO: break out the error handling to another function.
	sub := claims.Subject
	subDID, _ := did.Parse(sub)
	_, bytes, err := mbase.Decode(subDID.ID)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated,
			"The sub key DID did not match the key DID produced using the public key")
	}
	_, n, err := varint.FromUvarint(bytes)
	if err != nil {
		return nil, status.Errorf(codes.Unauthenticated,
			"The sub key DID did not match the key DID produced using the public key")
	}
	if n != 2 {
		return nil, status.Errorf(codes.Unauthenticated,
			"The sub key DID did not match the key DID produced using the public key")
	}
	dx, _ := base64.URLEncoding.DecodeString(x)
	if string(dx) != string(bytes[2:]) {
		return nil, status.Errorf(codes.Unauthenticated,
			"The sub key DID did not match the key DID produced using the public key")
	}

	// Check the chain for locked funds.
	// TODO break this out into another function
	iss := claims.Issuer
	blockHeight := req.BlockHeight
	var chainReq = &chainapi.HasFundsRequest{
		BlockHeight: blockHeight,
		AccountId:   iss,
	}
	chainRes, err := s.Deps.ChainAPIServiceClient.HasFunds(ctx, chainReq)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Locked funds: %s", err))
	}
	if !chainRes.HasFunds {
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("Locked funds: %s", err))
	}

	// Return the subscriber identity
	return &pb.AuthResponse{
		Identity: sub,
	}, nil
}
