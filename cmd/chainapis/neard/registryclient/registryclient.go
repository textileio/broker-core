package registryclient

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	api "github.com/textileio/near-api-go"
	"github.com/textileio/near-api-go/account"
)

const (
	providersPrefix = "_vectorp"
)

// Provider is a storage provider address.
type Provider string

// State models the contract state.
type State struct {
	Providers   []Provider
	BlockHash   string
	BlockHeight int
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
	contractAccountID string,
	clientAccountID string,
) (*Client, error) {
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

	providers := []Provider{}

	for _, val := range res.Values {
		if err := extractNewEntry(
			val.Key,
			val.Value,
			func(v Provider) { providers = append(providers, v) },
		); err != nil {
			return nil, fmt.Errorf("extracting entry: %v", err)
		}
	}
	return &State{
		Providers: providers,
	}, nil
}

func extractNewEntry(
	keyBase64 string,
	valueBase64 string,
	providerHandler func(Provider),
) error {
	prefix, meta, index, err := parseKey(keyBase64)
	if err != nil {
		return fmt.Errorf("parsing key: %v", err)
	}
	if len(meta) > 0 || index == "" {
		return nil
	}
	valueBytes, err := base64.StdEncoding.DecodeString(valueBase64)
	if err != nil {
		return fmt.Errorf("decoding value base64: %v", err)
	}
	switch prefix {
	case providersPrefix:
		providerHandler(Provider(valueBytes))
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
