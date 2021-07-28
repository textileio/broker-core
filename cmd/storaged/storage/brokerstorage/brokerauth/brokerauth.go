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

	return &AuthService{
		client: client,
	}, nil
}

var _ auth.Authorizer = (*AuthService)(nil)

// IsAuthorized evaluates a authentication token and returns associated information about it.
// If the authorization token is invalid, the second parameter will return false and the third one will
// explain why. If the authorization is valid, the returned values are true and "" respectively.
// The returned AuthorizedEntity value contains valid information only on successful authentications.
func (a *AuthService) IsAuthorized(ctx context.Context, token string) (auth.AuthorizedEntity, bool, string, error) {
	req := &authd.AuthRequest{Token: token}
	res, err := a.client.Auth(ctx, req)
	if status.Code(err) == codes.Unauthenticated {
		return auth.AuthorizedEntity{}, false, err.Error(), nil
	} else if err != nil {
		return auth.AuthorizedEntity{}, false, err.Error(), err
	}
	return auth.AuthorizedEntity{
		Identity: res.Identity,
		Origin:   res.Origin,
	}, true, "", nil
}
