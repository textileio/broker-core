package auth

import (
	"context"

	pb "github.com/textileio/broker-core/auth/pb/auth"
)

// Service is a gRPC service for buckets.
type Service struct {
}

// Go trick
// this service struct is implementing the interface pb.API...
// if it's not, will see errors
var _ pb.APIServiceServer = (*Service)(nil)

// NewService returns a new service.
func NewService() *Service {
	// put the indexer here
	return &Service{}
}

func (s *Service) Auth(ctx context.Context, req *pb.AuthRequest) (*pb.AuthResponse, error) {
	// check expiration
	// call s.indexer.
	return &pb.AuthResponse{}, nil
}
