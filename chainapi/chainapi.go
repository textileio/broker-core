package chainapi

import (
	"context"
)

// ChainAPI provides blockchain interactions.
type ChainAPI interface {
	HasDeposit(ctx context.Context, depositee, chainID string) (bool, error)
	OwnsPublicKey(ctx context.Context, accountID, publicKey, chainID string) (bool, error)
}
