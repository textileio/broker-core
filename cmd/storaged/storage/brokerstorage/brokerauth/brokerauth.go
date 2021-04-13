package brokerauth

import (
	"context"

	"github.com/textileio/broker-core/auth"
)

type BrokerAuth struct {
}

func New() (*BrokerAuth, error) {
	return &BrokerAuth{}, nil
}

var _ auth.Authorizer = (*BrokerAuth)(nil)

func (bs *BrokerAuth) IsAuthorized(ctx context.Context, identity string) (bool, string, error) {
	// TODO: Fill this implementation when the Auth API is ready.
	// For now, authorize every request.

	return true, "", nil
}
