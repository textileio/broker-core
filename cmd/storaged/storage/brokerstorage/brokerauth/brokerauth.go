package brokerauth

import (
	"context"
	"fmt"

	"github.com/textileio/broker-core/auth"
	authd "github.com/textileio/broker-core/gen/broker/auth/v1"
	"google.golang.org/grpc"
)

// AuthdService provides authentication resolution for the storage service.
type AuthdService struct {
	client authd.AuthAPIServiceClient
}

// New returns a new BrokerAuth.
func New(addr string) (*AuthdService, error) {
	conn, connErr := grpc.Dial(addr, grpc.WithInsecure())
	if connErr != nil {
		return nil, fmt.Errorf("creating authd client connection: %v", connErr)
	}
	client := authd.NewAuthAPIServiceClient(conn)
	return &AuthdService{client: client}, nil
}

var _ auth.Authorizer = (*AuthdService)(nil)

// IsAuthorized returns the identity that is authorized to use the storage service, otherwise returning an error.
func (a *AuthdService) IsAuthorized(ctx context.Context, jwtBase64URL string) (bool, string, error) {
	req := &authd.AuthRequest{JwtBase64URL: jwtBase64URL}
	res, err := a.client.Auth(ctx, req)
	if err != nil {
		return false, fmt.Sprintf("IsAuthorized error: %s", err), err
	}
	return true, res.Identity, nil
}
