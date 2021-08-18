package providerclient

import (
	"context"
	"fmt"
	"strconv"

	api "github.com/textileio/near-api-go"
)

// HasDeposit calls the contract hasLocked function.
func (c *Client) HasDeposit(ctx context.Context, depositee string) (bool, error) {
	res, err := c.NearClient.CallFunction(
		ctx,
		c.contractAccountID,
		"hasDeposit",
		api.CallFunctionWithFinality("final"),
		api.CallFunctionWithArgs(map[string]interface{}{
			"depositee": depositee,
		}),
	)
	if err != nil {
		return false, fmt.Errorf("calling call function: %v", err)
	}
	b, err := strconv.ParseBool(string(res.Result))
	if err != nil {
		return false, fmt.Errorf("parsing result %v: %v", string(res.Result), err)
	}
	return b, nil
}

// ReleaseDeposits unlocks all funds from expired sessions in the contract.
func (c *Client) ReleaseDeposits(ctx context.Context) error {
	_, err := c.NearClient.Account(c.clientAccountID).FunctionCall(ctx, c.contractAccountID, "releaseDeposits")
	if err != nil {
		return fmt.Errorf("calling rpc unlock funds: %v", err)
	}
	return nil
}
