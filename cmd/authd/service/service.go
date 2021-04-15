package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	logging "github.com/ipfs/go-log/v2"
	pb "github.com/textileio/broker-core/cmd/authd/pb/auth"
	"google.golang.org/grpc"
)

var (
	log = logging.Logger("auth/service")
)

// Service is a gRPC service for buckets.
type Service struct {
	pb.UnimplementedAPIServiceServer

	server *grpc.Server
}

// Go trick
// this service struct is implementing the interface pb.API...
// if it's not, will see errors.
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

// Auth provides an API endpoint to resolve authorization for users.
func (s *Service) Auth(ctx context.Context, req *pb.AuthRequest) (*pb.AuthResponse, error) {
	// check expiration
	// call s.indexer.
	return &pb.AuthResponse{}, nil
}
