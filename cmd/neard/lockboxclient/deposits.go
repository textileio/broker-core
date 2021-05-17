package lockboxclient

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"

	"github.com/textileio/broker-core/cmd/neard/nearclient"
	"github.com/textileio/broker-core/cmd/neard/nearclient/transaction"
)

// Deposit holds information about a deposit.
type Deposit struct {
	Sender     string
	Expiration uint64
	Amount     *big.Int
}

// DepositInfo models user locked funds.
type DepositInfo struct {
	AccountID string
	BrokerID  string
	Deposit   Deposit
}

// HasDeposit calls the lock box hasLocked function.
func (c *Client) HasDeposit(ctx context.Context, brokerID, accountID string) (bool, error) {
	res, err := c.nc.CallFunction(
		ctx,
		c.lockboxAccountID,
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

// AddDeposit locks funds with the lock box contract.
func (c *Client) AddDeposit(ctx context.Context, brokerID string) (*DepositInfo, error) {
	deposit, ok := (&big.Int{}).SetString("1000000000000000000000000", 10)
	if !ok {
		return nil, fmt.Errorf("creating depoist amount")
	}
	res, err := c.nc.Account(c.clientAccountID).FunctionCall(
		ctx,
		c.lockboxAccountID,
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
	depositInfo, err := extractDepositInfo(bytes)
	if err != nil {
		return nil, fmt.Errorf("decoding lock info: %v", err)
	}
	return depositInfo, nil
}

// ReleaseDeposits unlocks all funds from expired sessions in the contract.
func (c *Client) ReleaseDeposits(ctx context.Context) error {
	_, err := c.nc.Account(c.clientAccountID).FunctionCall(ctx, c.lockboxAccountID, "releaseDeposits")
	if err != nil {
		return fmt.Errorf("calling rpc unlock funds: %v", err)
	}
	return nil
}

// TODO: convert to custom JSON decoding.
func extractDepositInfo(bytes []byte) (*DepositInfo, error) {
	var j struct {
		AccountID string `json:"accountId"`
		BrokerID  string `json:"brokerId"`
		Deposit   struct {
			Amount     string `json:"amount"`
			Sender     string `json:"sender"`
			Expiration string `json:"expiration"`
		} `json:"deposit"`
	}
	err := json.Unmarshal(bytes, &j)
	if err != nil {
		return nil, err
	}

	depositAmount, ok := (&big.Int{}).SetString(j.Deposit.Amount, 10)
	if !ok {
		return nil, fmt.Errorf("error parsing string to big int: %s", j.Deposit.Amount)
	}

	expiration, err := strconv.ParseUint(j.Deposit.Expiration, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("parsing expiration: %v", err)
	}

	return &DepositInfo{
		AccountID: j.AccountID,
		BrokerID:  j.BrokerID,
		Deposit: Deposit{
			Amount:     depositAmount,
			Sender:     j.Deposit.Sender,
			Expiration: expiration,
		},
	}, nil
}
