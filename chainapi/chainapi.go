package chainapi

import (
	"context"
)

// ChainAPI provides blockchain interactions.
type ChainAPI interface {
	HasDeposit(ctx context.Context, brokerID, accountID string, chainID string) (bool, error)
}
