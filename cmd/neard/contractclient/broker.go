package contractclient

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/textileio/broker-core/cmd/neard/nearclient"
	"github.com/textileio/broker-core/cmd/neard/nearclient/transaction"
)

// BrokerInfo holds information about a broker.
type BrokerInfo struct {
	BrokerID  string   `json:"brokerId"`
	Addresses []string `json:"addresses"`
}

// DeleteBroker deletes the specified broker from the state.
func (c *Client) DeleteBroker(ctx context.Context, brokerID string) error {
	if _, err := c.NearClient.Account(c.clientAccountID).FunctionCall(
		ctx,
		c.contractAccountID,
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
	res, err := c.NearClient.Account(c.clientAccountID).FunctionCall(
		ctx,
		c.contractAccountID,
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
	res, err := c.NearClient.CallFunction(
		ctx,
		c.contractAccountID,
		"getBroker",
		nearclient.CallFunctionWithFinality("final"),
		nearclient.CallFunctionWithArgs(map[string]interface{}{"brokerId": brokerID}),
	)
	if err != nil {
		return nil, fmt.Errorf("calling rpc function: %v", err)
	}
	if string(res.Result) == nullStringValue {
		return nil, ErrorNotFound
	}
	var brokerInfo BrokerInfo
	if err := json.Unmarshal(res.Result, &brokerInfo); err != nil {
		return nil, fmt.Errorf("unmarshaling broker info: %v", err)
	}
	return &brokerInfo, nil
}

// ListBrokers lists all brokers.
func (c *Client) ListBrokers(ctx context.Context) ([]BrokerInfo, error) {
	res, err := c.NearClient.CallFunction(
		ctx,
		c.contractAccountID,
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
