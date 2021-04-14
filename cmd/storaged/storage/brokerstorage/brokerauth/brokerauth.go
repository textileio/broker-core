package brokerauth

import (
	"context"
	"fmt"

	"github.com/textileio/broker-core/auth"
	"github.com/textileio/broker-core/cmd/authd/client"
	pb "github.com/textileio/broker-core/gen/broker/auth/v1"
	"github.com/textileio/broker-core/rpc"
)

// BrokerAuth provides authentication resolution for the storage service.
type BrokerAuth struct {
	c *client.Client
}

// New returns a new BrokerAuth.
func New(addr string) (*BrokerAuth, error) {
	c, err := client.NewClient(addr, rpc.GetClientOpts(addr)...)
	if err != nil {
		return nil, fmt.Errorf("creating client: %v", err)
	}
	return &BrokerAuth{c: c}, nil
}

var _ auth.Authorizer = (*BrokerAuth)(nil)

// IsAuthorized returns the identity that is authorized to use the storage service, otherwise returning an error.
func (ba *BrokerAuth) IsAuthorized(ctx context.Context, jwtBase64URL string) (bool, string, error) {
	req := &pb.AuthRequest{JwtBase64URL: jwtBase64URL}
	res, err := ba.c.Auth(ctx, req)
	if err != nil {
		return false, fmt.Sprintf("IsAuthorized error: %s", err), err
	}
	return true, res.Identity, nil
}
