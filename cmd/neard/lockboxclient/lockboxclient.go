package lockboxclient

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/big"
	"strings"

	"github.com/textileio/broker-core/cmd/neard/nearclient"
)

type LockInfo struct {
	Deposit   *big.Int
	Sender    string
	AccountID string
}

type State struct {
	LockedFunds map[string]LockInfo
	BlockHash   string
	BlockHeight int
}

type Client struct {
	nc        *nearclient.Client
	accountID string
}

func NewClient(nc *nearclient.Client, accountID string) (*Client, error) {
	return &Client{
		nc:        nc,
		accountID: accountID,
	}, nil
}

func (c *Client) GetState(ctx context.Context) (*State, error) {
	res, err := c.nc.ViewState(ctx, c.accountID, nearclient.WithFinality("final"))
	if err != nil {
		return nil, fmt.Errorf("calling view state: %v", err)
	}

	lockedFunds := make(map[string]LockInfo)

	for _, val := range res.Values {
		keyBytes, err := base64.StdEncoding.DecodeString(val.Key)
		if err != nil {
			return nil, err
		}
		valueBytes, err := base64.StdEncoding.DecodeString(val.Value)
		if err != nil {
			return nil, err
		}

		parts := strings.Split(string(keyBytes), "::")
		if len(parts) != 2 {
			return nil, fmt.Errorf("unexpected key format: %s", string(keyBytes))
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

		lockedFunds[parts[1]] = LockInfo{
			Deposit:   deposit,
			Sender:    j.Sender,
			AccountID: j.AccountID,
		}
	}
	return &State{
		LockedFunds: lockedFunds,
		BlockHash:   res.BlockHash,
		BlockHeight: res.BlockHeight,
	}, nil
}
