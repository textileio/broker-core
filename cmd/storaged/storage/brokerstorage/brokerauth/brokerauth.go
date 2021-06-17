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
	client       authd.AuthAPIServiceClient
	bearerTokens []string
}

// New returns a new BrokerAuth.
func New(addr string, bearerTokens []string) (*AuthService, error) {
	conn, connErr := grpc.Dial(addr, grpc.WithInsecure())
	if connErr != nil {
		return nil, fmt.Errorf("creating authd client connection: %v", connErr)
	}
	client := authd.NewAuthAPIServiceClient(conn)

	tokens := make([]string, len(bearerTokens))
	for i := range bearerTokens {
		tokens[i] = "Bearer " + bearerTokens[i]
	}
	return &AuthService{
		client:       client,
		bearerTokens: bearerTokens,
	}, nil
}

var _ auth.Authorizer = (*AuthService)(nil)

// IsAuthorized returns the identity that is authorized to use the storage service, otherwise returning an error.
// The token is a base64 URL encoded JWT.
func (a *AuthService) IsAuthorized(ctx context.Context, token string) (bool, string, error) {
	for _, staticToken := range a.bearerTokens {
		if token == staticToken {
			return true, "", nil
		}
	}

	req := &authd.AuthRequest{Token: token}
	_, err := a.client.Auth(ctx, req)
	if status.Code(err) == codes.Unauthenticated {
		return false, err.Error(), nil
	} else if err != nil {
		return false, err.Error(), err
	}
	return true, "", nil
}
