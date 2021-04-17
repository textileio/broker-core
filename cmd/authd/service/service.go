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

// Service is a gRPC service for buckets.
type Service struct {
	pb.UnimplementedAPIServiceServer

	server *grpc.Server
}

var _ pb.APIServiceServer = (*Service)(nil)

// New returns a new service.
func New(listenAddr string) (*Service, error) {
	// put the indexer here

	s := &Service{
		server: grpc.NewServer(),
	}
	listener, err := net.Listen("tcp", listenAddr)
	if err != nil {
		return nil, fmt.Errorf("getting net listener: %v", err)
	}
	go func() {
		pb.RegisterAPIServiceServer(s.server, s)
		if err := s.server.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			log.Errorf("server error: %v", err)
		}
	}()

	log.Infof("service listening at %s", listenAddr)
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

	// token.payload.claims.sub
	var sub string

	// Validate the JWT.
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
		return nil, status.Errorf(codes.InvalidArgument, fmt.Sprintf("ParseWithClaims: %s", err))
	}

	if claims, ok := token.Claims.(*jwt.StandardClaims); ok && token.Valid {
		sub = claims.Subject
		subDID, _ := did.Parse(sub)
		_, bytes, err := mbase.Decode(subDID.ID)
		if err != nil {
			sub = ""
		}
		_, n, err := varint.FromUvarint(bytes)
		if err != nil {
			sub = ""
		}
		if n != 2 {
			sub = ""
		}
		dx, _ := base64.URLEncoding.DecodeString(x)
		if string(dx) != string(bytes[2:]) {
			sub = ""
		}
	} else {
		sub = ""
	}

	if sub == "" {
		return nil, status.Errorf(codes.Unauthenticated,
			"The sub key DID did not match the key DID produced using the public key")
	}

	// @todo: hit indexerd to check for locked funds

	// Return the subscriber identity
	return &pb.AuthResponse{
		Identity: sub,
	}, nil
}
