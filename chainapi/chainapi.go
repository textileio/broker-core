package chainapi

import (
	"context"
)

// ChainAPI provides blockchain interactions.
type ChainAPI interface {
	HasDeposit(ctx context.Context, depositee string, chainID string) (bool, error)
}
