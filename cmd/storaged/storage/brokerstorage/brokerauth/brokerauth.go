package brokerauth

import (
	"context"
	"fmt"

	"github.com/textileio/broker-core/auth"
	authd "github.com/textileio/broker-core/gen/broker/auth/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// AuthService provides authentication resolution for the storage service.
type AuthService struct {
	client authd.AuthAPIServiceClient
}

// New returns a new BrokerAuth.
func New(addr string) (*AuthService, error) {
	conn, connErr := grpc.Dial(addr, grpc.WithInsecure())
	if connErr != nil {
		return nil, fmt.Errorf("creating authd client connection: %v", connErr)
	}
	client := authd.NewAuthAPIServiceClient(conn)
	return &AuthService{client: client}, nil
}

var _ auth.Authorizer = (*AuthService)(nil)

// IsAuthorized returns the identity that is authorized to use the storage service, otherwise returning an error.
// The token is a base64 URL encoded JWT.
func (a *AuthService) IsAuthorized(ctx context.Context, token string) (bool, string, error) {
	req := &authd.AuthRequest{Token: token}
	res, err := a.client.Auth(ctx, req)
	if status.Code(err) == codes.Unauthenticated {
		return false, err.Error(), nil
	} else if err != nil {
		return false, err.Error(), err
	}
	return true, res.Identity, nil
}
