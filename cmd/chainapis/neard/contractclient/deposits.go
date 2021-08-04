package contractclient

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"

	"github.com/textileio/broker-core/cmd/chainapis/neard/nearclient"
	"github.com/textileio/broker-core/cmd/chainapis/neard/nearclient/transaction"
)

// Deposit holds information about a deposit.
type Deposit struct {
	Sender     string   `json:"sender"`
	Expiration uint64   `json:"expiration"`
	Amount     *big.Int `json:"amount"`
}

// MarshalJSON implements MarshalJSON.
func (d *Deposit) MarshalJSON() ([]byte, error) {
	type Alias Deposit
	return json.Marshal(&struct {
		Expiration string `json:"expiration"`
		Amount     string `json:"amount"`
		*Alias
	}{
		Expiration: strconv.FormatUint(d.Expiration, 10),
		Amount:     d.Amount.String(),
		Alias:      (*Alias)(d),
	})
}

// UnmarshalJSON implements UnmarshalJSON.
func (d *Deposit) UnmarshalJSON(data []byte) error {
	type Alias Deposit
	aux := &struct {
		Expiration string `json:"expiration"`
		Amount     string `json:"amount"`
		*Alias
	}{
		Alias: (*Alias)(d),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	expiration, err := strconv.ParseUint(aux.Expiration, 10, 64)
	if err != nil {
		return fmt.Errorf("parsing expiration: %v", err)
	}
	d.Expiration = expiration
	a, success := (&big.Int{}).SetString(aux.Amount, 10)
	if !success {
		return fmt.Errorf("failed to create big int from value %s", aux.Amount)
	}
	d.Amount = a
	return nil
}

// DepositInfo models user locked funds.
type DepositInfo struct {
	AccountID string  `json:"accountId"`
	BrokerID  string  `json:"brokerId"`
	Deposit   Deposit `json:"deposit"`
}

// HasDeposit calls the contract hasLocked function.
func (c *Client) HasDeposit(ctx context.Context, brokerID, accountID string) (bool, error) {
	res, err := c.nc.CallFunction(
		ctx,
		c.contractAccountID,
		"hasDeposit",
		nearclient.CallFunctionWithFinality("final"),
		nearclient.CallFunctionWithArgs(map[string]interface{}{
			"brokerId":  brokerID,
			"accountId": accountID,
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

// AddDeposit locks funds with the contract.
func (c *Client) AddDeposit(ctx context.Context, brokerID string) (*DepositInfo, error) {
	deposit, ok := (&big.Int{}).SetString("1000000000000000000000000", 10)
	if !ok {
		return nil, fmt.Errorf("creating depoist amount")
	}
	res, err := c.nc.Account(c.clientAccountID).FunctionCall(
		ctx,
		c.contractAccountID,
		"addDeposit",
		transaction.FunctionCallWithArgs(map[string]interface{}{"brokerId": brokerID, "accountId": c.clientAccountID}),
		transaction.FunctionCallWithDeposit(*deposit),
	)
	if err != nil {
		return nil, fmt.Errorf("calling rpc: %v", err)
	}
	status, ok := res.GetStatus()
	if !ok || status.SuccessValue == "" {
		return nil, fmt.Errorf("didn't receive expected status success value")
	}
	bytes, err := base64.StdEncoding.DecodeString(status.SuccessValue)
	if err != nil {
		return nil, fmt.Errorf("decoding status value string: %v", err)
	}
	log.Errorf("json: %s", string(bytes))
	var depositInfo DepositInfo
	if err := json.Unmarshal(bytes, &depositInfo); err != nil {
		return nil, fmt.Errorf("decoding lock info: %v", err)
	}
	return &depositInfo, nil
}

// ReleaseDeposits unlocks all funds from expired sessions in the contract.
func (c *Client) ReleaseDeposits(ctx context.Context) error {
	_, err := c.nc.Account(c.clientAccountID).FunctionCall(ctx, c.contractAccountID, "releaseDeposits")
	if err != nil {
		return fmt.Errorf("calling rpc unlock funds: %v", err)
	}
	return nil
}
