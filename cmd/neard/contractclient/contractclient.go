package contractclient

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/textileio/broker-core/cmd/neard/nearclient"
	"github.com/textileio/broker-core/cmd/neard/nearclient/account"
	logging "github.com/textileio/go-log/v2"
)

const (
	nullStringValue = "null"
	depositsPrefix  = "a"
	brokersPrefix   = "b"
)

var (
	log = logging.Logger("contractclient")

	// ErrorNotFound is returned when an item isn't found.
	ErrorNotFound = errors.New("not found")
)

// State models the contract state.
type State struct {
	DepositMap  map[string]DepositInfo
	BrokerMap   map[string]BrokerInfo
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
	Type         ChangeType
	BrokerValue  *BrokerValue
	DepositValue *DepositValue
}

// DepositValue holds information about a DepositInfo.
type DepositValue struct {
	Key   string
	Value *DepositInfo
}

// BrokerValue holds information about a BrokerInfo.
type BrokerValue struct {
	Key   string
	Value *BrokerInfo
}

// Client communicates with the contract API.
type Client struct {
	NearClient        *nearclient.Client
	contractAccountID string
	clientAccountID   string
}

// NewClient creates a new Client.
func NewClient(nc *nearclient.Client, contractAccountID, clientAccountID string) (*Client, error) {
	return &Client{
		NearClient:        nc,
		contractAccountID: contractAccountID,
		clientAccountID:   clientAccountID,
	}, nil
}

// GetAccount gets information about the account.
func (c *Client) GetAccount(ctx context.Context) (*account.AccountView, error) {
	return c.NearClient.Account(c.contractAccountID).State(ctx, account.StateWithFinality("final"))
}

// GetState returns the contract state.
func (c *Client) GetState(ctx context.Context) (*State, error) {
	res, err := c.NearClient.Account(c.contractAccountID).ViewState(
		ctx,
		account.ViewStateWithFinality("final"),
	)
	if err != nil {
		return nil, fmt.Errorf("calling view state: %v", err)
	}

	deposits := make(map[string]DepositInfo)
	brokers := make(map[string]BrokerInfo)

	for _, val := range res.Values {
		if err := extractNewEntry(
			val.Key,
			val.Value,
			func(v BrokerValue) { brokers[v.Key] = *v.Value },
			func(v DepositValue) { deposits[v.Key] = *v.Value },
		); err != nil {
			return nil, fmt.Errorf("extracting entry: %v", err)
		}
	}
	return &State{
		BrokerMap:   brokers,
		DepositMap:  deposits,
		BlockHash:   res.BlockHash,
		BlockHeight: res.BlockHeight,
	}, nil
}

// GetChanges gets the state changes for a block height.
func (c *Client) GetChanges(ctx context.Context, blockHeight int) ([]Change, string, error) {
	res, err := c.NearClient.DataChanges(ctx, []string{c.contractAccountID}, nearclient.DataChangesWithBlockHeight(blockHeight))
	if err != nil {
		return nil, "", fmt.Errorf("calling data changes: %v", err)
	}
	var changes []Change
	for _, item := range res.Changes {
		c := Change{}
		switch item.Type {
		case "data_update":
			c.Type = Update
			if err := extractNewEntry(
				item.Change.KeyBase64,
				item.Change.ValueBase64,
				func(v BrokerValue) {
					c.BrokerValue = &v
				},
				func(v DepositValue) {
					c.DepositValue = &v
				},
			); err != nil {
				return nil, "", fmt.Errorf("extracting entry: %v", err)
			}
		case "data_deletion":
			c.Type = Delete
			if err := extractDeletion(
				item.Change.KeyBase64,
				func(v BrokerValue) {
					c.BrokerValue = &v
				},
				func(v DepositValue) {
					c.DepositValue = &v
				},
			); err != nil {
				return nil, "", fmt.Errorf("extracting deletion: %v", err)
			}
		default:
			return nil, "", fmt.Errorf("unknown change type: %v", item.Type)
		}
		changes = append(changes, c)
	}
	return changes, res.BlockHash, nil
}

func extractNewEntry(
	keyBase64 string,
	valueBase64 string,
	brokerHandler func(BrokerValue),
	depositHandler func(DepositValue),
) error {
	prefix, meta, _, err := parseKey(keyBase64)
	if err != nil {
		return fmt.Errorf("parsing key: %v", err)
	}
	if len(meta) != 1 || meta[0] != "entries" {
		return nil
	}
	valueBytes, err := base64.StdEncoding.DecodeString(valueBase64)
	if err != nil {
		return fmt.Errorf("decoding value base64: %v", err)
	}
	switch prefix {
	case brokersPrefix:
		var v BrokerValue
		if err := json.Unmarshal(valueBytes, &v); err != nil {
			return fmt.Errorf("unmarshaling value: %v", err)
		}
		brokerHandler(v)
	case depositsPrefix:
		var v DepositValue
		if err := json.Unmarshal(valueBytes, &v); err != nil {
			return fmt.Errorf("unmarshaling value: %v", err)
		}
		depositHandler(v)
	}
	return nil
}

func extractDeletion(keyBase64 string, brokerHandler func(BrokerValue), depositHandler func(DepositValue)) error {
	prefix, meta, index, err := parseKey(keyBase64)
	if err != nil {
		return fmt.Errorf("parsing key: %v", err)
	}
	if len(meta) != 1 || meta[0] != "map" {
		return nil
	}
	switch prefix {
	case brokersPrefix:
		brokerHandler(BrokerValue{Key: index})
	case depositsPrefix:
		depositHandler(DepositValue{Key: index})
	}
	return nil
}

func parseKey(encoded string) (string, []string, string, error) {
	keyBytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", nil, "", err
	}
	full := string(keyBytes)
	parts := strings.Split(full, "::")
	if len(parts) != 1 && len(parts) != 2 {
		return "", nil, "", fmt.Errorf("unexpected key format: %s", full)
	}
	index := ""
	if len(parts) == 2 {
		index = parts[1]
	}
	prefixParts := strings.Split(parts[0], ":")
	if len(prefixParts) == 0 {
		return "", nil, "", fmt.Errorf("unexpected prefix parts: %s", parts[0])
	}
	return prefixParts[0], prefixParts[1:], index, nil
}
