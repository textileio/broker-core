package brokerauth

import (
	"context"

	"github.com/textileio/broker-core/auth"
)

// BrokerAuth provides authorization resolution for the storage service.
type BrokerAuth struct {
}

// New returns a new BrokerAuth.
func New() (*BrokerAuth, error) {
	return &BrokerAuth{}, nil
}

var _ auth.Authorizer = (*BrokerAuth)(nil)

// IsAuthorized resolves if the provided identity is authorized to use
// the storage service. If that isn't the case, an string is return explaining why.
func (bs *BrokerAuth) IsAuthorized(ctx context.Context, identity string) (bool, string, error) {
	// TODO: Fill this implementation when the Auth API is ready.
	// For now, authorize every request.

	return true, "", nil
}
