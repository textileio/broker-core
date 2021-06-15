package lockboxclient

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"github.com/textileio/broker-core/cmd/neard/nearclient"
	"github.com/textileio/broker-core/cmd/neard/nearclient/account"
	logging "github.com/textileio/go-log/v2"
)

const (
	nullStringValue = "null"
)

var (
	log = logging.Logger("lockboxclient")

	// ErrorNotFound is returned when an item isn't found.
	ErrorNotFound = errors.New("not found")
)

// State models the contract state.
type State struct {
	LockedFunds map[string]DepositInfo
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
	LockInfo *DepositInfo
}

// Client communicates with the contract API.
type Client struct {
	nc                *nearclient.Client
	contractAccountID string
	clientAccountID   string
}

// NewClient creates a new Client.
func NewClient(nc *nearclient.Client, contractAccountID, clientAccountID string) (*Client, error) {
	return &Client{
		nc:                nc,
		contractAccountID: contractAccountID,
		clientAccountID:   clientAccountID,
	}, nil
}

// GetAccount gets information about the account.
func (c *Client) GetAccount(ctx context.Context) (*account.AccountView, error) {
	return c.nc.Account(c.contractAccountID).State(ctx, account.StateWithFinality("final"))
}

// GetState returns the contract state.
func (c *Client) GetState(ctx context.Context) (*State, error) {
	res, err := c.nc.Account(c.contractAccountID).ViewState(
		ctx,
		account.ViewStateWithFinality("final"),
		account.ViewStateWithPrefix("u"),
	)
	if err != nil {
		return nil, fmt.Errorf("calling view state: %v", err)
	}

	lockedFunds := make(map[string]DepositInfo)

	for _, val := range res.Values {
		key, err := extractKey(val.Key)
		if err != nil {
			return nil, fmt.Errorf("extracting key: %v", err)
		}
		valueBytes, err := base64.StdEncoding.DecodeString(val.Value)
		if err != nil {
			return nil, fmt.Errorf("decoding value base64: %v", err)
		}
		lockInfo, err := extractDepositInfo(valueBytes)
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

// GetChanges gets the state changes for a block height.
func (c *Client) GetChanges(ctx context.Context, blockHeight int) ([]Change, string, error) {
	res, err := c.nc.DataChanges(ctx, []string{c.contractAccountID}, nearclient.DataChangesWithBlockHeight(blockHeight))
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
		var lockInfo *DepositInfo
		if t == Update {
			valueBytes, err := base64.StdEncoding.DecodeString(item.Change.ValueBase64)
			if err != nil {
				return nil, "", fmt.Errorf("decoding value base64: %v", err)
			}
			lockInfo, err = extractDepositInfo(valueBytes)
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
