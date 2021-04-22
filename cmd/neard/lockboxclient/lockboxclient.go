package lockboxclient

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/textileio/broker-core/cmd/neard/nearclient"
)

// LockInfo models user locked funds.
type LockInfo struct {
	Deposit   *big.Int
	Sender    string
	AccountID string
}

// State models the lock box contract state.
type State struct {
	LockedFunds map[string]LockInfo
	BlockHash   string
	BlockHeight int
}

type ChangeType int

const (
	Update ChangeType = iota
	Delete
)

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

// GetState returns the contract state.
func (c *Client) GetState(ctx context.Context) (*State, error) {
	res, err := c.nc.ViewState(ctx, c.accountID, nearclient.ViewStateWithFinality("final"))
	if err != nil {
		return nil, fmt.Errorf("calling view state: %v", err)
	}

	lockedFunds := make(map[string]LockInfo)

	for _, val := range res.Values {
		key, err := extractKey(val.Key)
		if err != nil {
			return nil, fmt.Errorf("extracting key: %v", err)
		}

		lockInfo, err := extractLockInfo(val.Value)
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

func (c *Client) GetAccount(ctx context.Context) (*nearclient.ViewAccountResponse, error) {
	return c.nc.ViewAccount(ctx, c.accountID, nearclient.ViewAccountWithFinality("final"))
}

func (c *Client) HasLocked(ctx context.Context, accountID string) (bool, error) {
	res, err := c.nc.CallFunction(
		ctx,
		c.accountID,
		"hasLocked",
		nearclient.CallFunctionWithFinality("final"),
		nearclient.CallFunctionWithArgs(map[string]interface{}{
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
			lockInfo, err = extractLockInfo(item.Change.ValueBase64)
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

func extractLockInfo(encoded string) (*LockInfo, error) {
	valueBytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	var j struct {
		Deposit   string `json:"deposit"`
		Sender    string `json:"sender"`
		AccountID string `json:"accountId"`
	}
	err = json.Unmarshal(valueBytes, &j)
	if err != nil {
		return nil, err
	}

	deposit, ok := (&big.Int{}).SetString(j.Deposit, 10)
	if !ok {
		return nil, fmt.Errorf("error parsing string to big int: %s", j.Deposit)
	}

	return &LockInfo{
		Deposit:   deposit,
		Sender:    j.Sender,
		AccountID: j.AccountID,
	}, nil
}
