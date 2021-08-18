package providerclient

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	api "github.com/textileio/near-api-go"
	"github.com/textileio/near-api-go/account"
)

const (
	depositsPrefix = "d"
)

// Deposit holds information about a deposit.
type Deposit struct {
	Timestamp uint64   `json:"timestamp"`
	Depositor string   `json:"depositor"`
	Value     *big.Int `json:"value"`
}

// MarshalJSON implements MarshalJSON.
func (d *Deposit) MarshalJSON() ([]byte, error) {
	type Alias Deposit
	return json.Marshal(&struct {
		Timestamp string `json:"timestamp"`
		Value     string `json:"value"`
		*Alias
	}{
		Timestamp: strconv.FormatUint(d.Timestamp, 10),
		Value:     d.Value.String(),
		Alias:     (*Alias)(d),
	})
}

// UnmarshalJSON implements UnmarshalJSON.
func (d *Deposit) UnmarshalJSON(data []byte) error {
	type Alias Deposit
	aux := &struct {
		Timestamp string `json:"timestamp"`
		Value     string `json:"value"`
		*Alias
	}{
		Alias: (*Alias)(d),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	timestamp, err := strconv.ParseUint(aux.Timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("parsing expiration: %v", err)
	}
	d.Timestamp = timestamp
	v, success := (&big.Int{}).SetString(aux.Value, 10)
	if !success {
		return fmt.Errorf("failed to create big int from value %s", aux.Value)
	}
	d.Value = v
	return nil
}

// State models the contract state.
type State struct {
	Initiated          bool
	APIEndpoint        string
	ProviderProportion float64
	SessionDivisor     *big.Int
	DepositMap         map[string]Deposit
	BlockHash          string
	BlockHeight        int
}

// DepositValue holds information about a DepositInfo.
type DepositValue struct {
	Key   string
	Value *Deposit
}

// Client communicates with the contract API.
type Client struct {
	NearClient        *api.Client
	contractAccountID string
	clientAccountID   string
}

// NewClient creates a new Client.
func NewClient(
	nc *api.Client,
	registryContractAccountID string,
	clientAccountID string,
) (*Client, error) {
	return &Client{
		NearClient:        nc,
		contractAccountID: registryContractAccountID,
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

	deposits := make(map[string]Deposit)

	for _, val := range res.Values {
		if err := extractNewEntry(
			val.Key,
			val.Value,
			func(v DepositValue) { deposits[v.Key] = *v.Value },
		); err != nil {
			return nil, fmt.Errorf("extracting entry: %v", err)
		}
	}
	return &State{
		DepositMap:  deposits,
		BlockHash:   res.BlockHash,
		BlockHeight: res.BlockHeight,
	}, nil
}

func extractNewEntry(
	keyBase64 string,
	valueBase64 string,
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
	case depositsPrefix:
		var v DepositValue
		if err := json.Unmarshal(valueBytes, &v); err != nil {
			return fmt.Errorf("unmarshaling value: %v", err)
		}
		depositHandler(v)
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
