package lockboxclient

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	logging "github.com/ipfs/go-log/v2"
	"github.com/textileio/broker-core/cmd/neard/nearclient"
	"github.com/textileio/broker-core/cmd/neard/nearclient/account"
	"github.com/textileio/broker-core/cmd/neard/nearclient/transaction"
)

var (
	log = logging.Logger("lockboxclient")
)

// BrokerInfo holds information about a broker.
type BrokerInfo struct {
	BrokerID  string   `json:"brokerId"`
	Addresses []string `json:"addresses"`
}

// DepositInfo holds information about a deposit.
type DepositInfo struct {
	Amount     *big.Int
	Sender     string
	Expiration uint64
}

// LockInfo models user locked funds.
type LockInfo struct {
	AccountID string
	BrokerID  string
	Deposit   DepositInfo
}

// State models the lock box contract state.
type State struct {
	LockedFunds map[string]LockInfo
	BlockHash   string
	BlockHeight int
}

// ChangeType describes the type of data change.
type ChangeType int

const (
	// Update is a data update change.
	Update ChangeType = iota
	// Delete is a data deletion change.
	Delete
)

// Change holds information about a state change.
type Change struct {
	Key      string
	Type     ChangeType
	LockInfo *LockInfo
}

// Client communicates with the lock box contract API.
type Client struct {
	nc        *nearclient.Client
	accountID string
}

// NewClient creates a new Client.
func NewClient(nc *nearclient.Client, accountID string) (*Client, error) {
	return &Client{
		nc:        nc,
		accountID: accountID,
	}, nil
}

// DeleteBroker deletes the specified broker from the state.
func (c *Client) DeleteBroker(ctx context.Context, brokerID string) error {
	if _, err := c.nc.Account(c.accountID).FunctionCall(
		ctx,
		c.accountID,
		"deleteBroker",
		transaction.FunctionCallWithArgs(map[string]interface{}{"brokerId": brokerID}),
	); err != nil {
		return fmt.Errorf("calling rpc function: %v", err)
	}
	return nil
}

// SetBroker adds a broker to the state.
func (c *Client) SetBroker(ctx context.Context, brokerID string, addrs []string) (*BrokerInfo, error) {
	args := map[string]interface{}{
		"brokerId": brokerID,
		"addrs":    addrs,
	}
	res, err := c.nc.Account(c.accountID).FunctionCall(
		ctx,
		c.accountID,
		"setBroker",
		transaction.FunctionCallWithArgs(args),
	)
	if err != nil {
		return nil, fmt.Errorf("calling rpc function: %v", err)
	}
	status, ok := res.GetStatus()
	if !ok || status.SuccessValue == "" {
		return nil, fmt.Errorf("didn't receive expected status success value")
	}
	bytes, err := base64.StdEncoding.DecodeString(status.SuccessValue)
	if err != nil {
		return nil, fmt.Errorf("decoding status value string: %v", err)
	}
	var brokerInfo BrokerInfo
	if err := json.Unmarshal(bytes, &brokerInfo); err != nil {
		return nil, fmt.Errorf("unmarshaling broker info: %v", err)
	}
	return &brokerInfo, nil
}

// GetBroker gets a broker by id.
func (c *Client) GetBroker(ctx context.Context, brokerID string) (*BrokerInfo, error) {
	res, err := c.nc.CallFunction(
		ctx,
		c.accountID,
		"getBroker",
		nearclient.CallFunctionWithFinality("final"),
		nearclient.CallFunctionWithArgs(map[string]interface{}{"brokerId": brokerID}),
	)
	if err != nil {
		return nil, fmt.Errorf("calling rpc function: %v", err)
	}
	if string(res.Result) == "null" {
		return nil, nil
	}
	var brokerInfo BrokerInfo
	if err := json.Unmarshal(res.Result, &brokerInfo); err != nil {
		return nil, fmt.Errorf("unmarshaling broker info: %v", err)
	}
	return &brokerInfo, nil
}

// ListBrokers lists all brokers.
func (c *Client) ListBrokers(ctx context.Context) ([]BrokerInfo, error) {
	res, err := c.nc.CallFunction(
		ctx,
		c.accountID,
		"listBrokers",
		nearclient.CallFunctionWithFinality("final"),
	)
	if err != nil {
		return nil, fmt.Errorf("calling rpc function: %v", err)
	}
	var brokers []BrokerInfo
	if err := json.Unmarshal(res.Result, &brokers); err != nil {
		return nil, fmt.Errorf("unmarshaling broker info: %v", err)
	}
	return brokers, nil
}

// GetState returns the contract state.
func (c *Client) GetState(ctx context.Context) (*State, error) {
	res, err := c.nc.Account(c.accountID).ViewState(
		ctx,
		account.ViewStateWithFinality("final"),
		account.ViewStateWithPrefix("u"),
	)
	if err != nil {
		return nil, fmt.Errorf("calling view state: %v", err)
	}

	lockedFunds := make(map[string]LockInfo)

	for _, val := range res.Values {
		key, err := extractKey(val.Key)
		if err != nil {
			return nil, fmt.Errorf("extracting key: %v", err)
		}
		valueBytes, err := base64.StdEncoding.DecodeString(val.Value)
		if err != nil {
			return nil, fmt.Errorf("decoding value base64: %v", err)
		}
		lockInfo, err := extractLockInfo(valueBytes)
		if err != nil {
			return nil, fmt.Errorf("extracting lock info: %v", err)
		}

		lockedFunds[key] = *lockInfo
	}
	return &State{
		LockedFunds: lockedFunds,
		BlockHash:   res.BlockHash,
		BlockHeight: res.BlockHeight,
	}, nil
}

// GetAccount gets information about the lock box account.
func (c *Client) GetAccount(ctx context.Context) (*account.AccountView, error) {
	return c.nc.Account(c.accountID).State(ctx, account.StateWithFinality("final"))
}

// LockFunds locks funds with the lock box contract.
func (c *Client) LockFunds(ctx context.Context, brokerID string) (*LockInfo, error) {
	deposit, ok := (&big.Int{}).SetString("1000000000000000000000000", 10)
	if !ok {
		return nil, fmt.Errorf("creating depoist amount")
	}
	res, err := c.nc.Account(c.accountID).FunctionCall(
		ctx,
		c.accountID,
		"lockFunds",
		transaction.FunctionCallWithArgs(map[string]interface{}{"brokerId": brokerID, "accountId": c.accountID}),
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
	lockInfo, err := extractLockInfo(bytes)
	if err != nil {
		return nil, fmt.Errorf("decoding lock info: %v", err)
	}
	return lockInfo, nil
}

// HasLocked calls the lock box hasLocked function.
func (c *Client) HasLocked(ctx context.Context, brokerID, accountID string) (bool, error) {
	res, err := c.nc.CallFunction(
		ctx,
		c.accountID,
		"hasLocked",
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

// GetChanges gets the lock box state changes for a block height.
func (c *Client) GetChanges(ctx context.Context, blockHeight int) ([]Change, string, error) {
	res, err := c.nc.DataChanges(ctx, []string{c.accountID}, nearclient.DataChangesWithBlockHeight(blockHeight))
	if err != nil {
		return nil, "", fmt.Errorf("calling data changes: %v", err)
	}
	var changes []Change
	for _, item := range res.Changes {
		var t ChangeType
		switch item.Type {
		case "data_update":
			t = Update
		case "data_deletion":
			t = Delete
		default:
			return nil, "", fmt.Errorf("unknown change type: %v", item.Type)
		}
		key, err := extractKey(item.Change.KeyBase64)
		if err != nil {
			return nil, "", fmt.Errorf("extracting key: %v", err)
		}
		var lockInfo *LockInfo
		if t == Update {
			valueBytes, err := base64.StdEncoding.DecodeString(item.Change.ValueBase64)
			if err != nil {
				return nil, "", fmt.Errorf("decoding value base64: %v", err)
			}
			lockInfo, err = extractLockInfo(valueBytes)
			if err != nil {
				return nil, "", fmt.Errorf("extracting lock info: %v", err)
			}
		}
		changes = append(changes, Change{Key: key, Type: t, LockInfo: lockInfo})
	}
	return changes, res.BlockHash, nil
}

func extractKey(encoded string) (string, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	parts := strings.Split(string(keyBytes), "::")
	if len(parts) != 2 {
		return "", fmt.Errorf("unexpected key format: %s", string(keyBytes))
	}
	return parts[1], nil
}

func extractLockInfo(bytes []byte) (*LockInfo, error) {
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

	return &LockInfo{
		AccountID: j.AccountID,
		BrokerID:  j.BrokerID,
		Deposit: DepositInfo{
			Amount:     depositAmount,
			Sender:     j.Deposit.Sender,
			Expiration: expiration,
		},
	}, nil
}
