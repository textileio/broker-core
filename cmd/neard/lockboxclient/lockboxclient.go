package lockboxclient

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	logging "github.com/ipfs/go-log/v2"
	"github.com/textileio/broker-core/cmd/neard/nearclient"
	"github.com/textileio/broker-core/cmd/neard/nearclient/account"
	"github.com/textileio/broker-core/cmd/neard/nearclient/transaction"
)

const (
	nullStringValue = "null"
)

var (
	log = logging.Logger("lockboxclient")

	// ErrorNotFound is returned when an item isn't found.
	ErrorNotFound = errors.New("not found")
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

// DealInfo desscribes a single storage deal.
type DealInfo struct {
	DealID     string `json:"dealId"`
	MinerID    string `json:"minerId"`
	Expiration uint64 `json:"expiration"`
}

// MarshalJSON implements MarshalJSON.
func (d *DealInfo) MarshalJSON() ([]byte, error) {
	type Alias DealInfo
	return json.Marshal(&struct {
		Expiration string `json:"expiration"`
		*Alias
	}{
		Expiration: strconv.FormatUint(d.Expiration, 10),
		Alias:      (*Alias)(d),
	})
}

// UnmarshalJSON implements UnmarshalJSON.
func (d *DealInfo) UnmarshalJSON(data []byte) error {
	type Alias DealInfo
	aux := &struct {
		Expiration string `json:"expiration"`
		*Alias
	}{
		Alias: (*Alias)(d),
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	exp, err := strconv.ParseUint(aux.Expiration, 10, 64)
	if err != nil {
		return fmt.Errorf("parsing expiration: %v", err)
	}
	d.Expiration = exp
	return nil
}

// PayloadInfo describes something.
type PayloadInfo struct {
	PayloadCid string     `json:"payloadCid"`
	PieceCid   string     `json:"pieceCid"`
	Deals      []DealInfo `json:"deals"`
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
	nc               *nearclient.Client
	lockboxAccountID string
	clientAccountID  string
}

// NewClient creates a new Client.
func NewClient(nc *nearclient.Client, lockboxAccountID, clientAccountID string) (*Client, error) {
	return &Client{
		nc:               nc,
		lockboxAccountID: lockboxAccountID,
		clientAccountID:  clientAccountID,
	}, nil
}

// LOCKED FUNDS

// HasLocked calls the lock box hasLocked function.
func (c *Client) HasLocked(ctx context.Context, brokerID, accountID string) (bool, error) {
	res, err := c.nc.CallFunction(
		ctx,
		c.lockboxAccountID,
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

// LockFunds locks funds with the lock box contract.
func (c *Client) LockFunds(ctx context.Context, brokerID string) (*LockInfo, error) {
	deposit, ok := (&big.Int{}).SetString("1000000000000000000000000", 10)
	if !ok {
		return nil, fmt.Errorf("creating depoist amount")
	}
	res, err := c.nc.Account(c.clientAccountID).FunctionCall(
		ctx,
		c.lockboxAccountID,
		"lockFunds",
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
	lockInfo, err := extractLockInfo(bytes)
	if err != nil {
		return nil, fmt.Errorf("decoding lock info: %v", err)
	}
	return lockInfo, nil
}

// UnlockFunds unlocks all funds from expired sessions in the contract.
func (c *Client) UnlockFunds(ctx context.Context) error {
	_, err := c.nc.Account(c.clientAccountID).FunctionCall(ctx, c.lockboxAccountID, "unlockFunds")
	if err != nil {
		return fmt.Errorf("calling rpc unlock funds: %v", err)
	}
	return nil
}

// BROKER

// DeleteBroker deletes the specified broker from the state.
func (c *Client) DeleteBroker(ctx context.Context, brokerID string) error {
	if _, err := c.nc.Account(c.clientAccountID).FunctionCall(
		ctx,
		c.lockboxAccountID,
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
	res, err := c.nc.Account(c.clientAccountID).FunctionCall(
		ctx,
		c.lockboxAccountID,
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
		c.lockboxAccountID,
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
	res, err := c.nc.CallFunction(
		ctx,
		c.lockboxAccountID,
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

// REPORTING

// ListPayloads lists payload records.
func (c *Client) ListPayloads(ctx context.Context, offset, maxLength int) ([]PayloadInfo, error) {
	res, err := c.nc.CallFunction(
		ctx,
		c.lockboxAccountID,
		"listPayloads",
		nearclient.CallFunctionWithFinality("final"),
		nearclient.CallFunctionWithArgs(map[string]interface{}{"offset": offset, "maxLength": maxLength}),
	)
	if err != nil {
		return nil, fmt.Errorf("calling rpc function: %v", err)
	}
	var infos []PayloadInfo
	if err := json.Unmarshal(res.Result, &infos); err != nil {
		return nil, fmt.Errorf("unmarshaling payload info: %v", err)
	}
	return infos, nil
}

// GetByPayload gets a payload record by payload cid.
func (c *Client) GetByPayload(ctx context.Context, payloadCid string) (*PayloadInfo, error) {
	res, err := c.nc.CallFunction(
		ctx,
		c.lockboxAccountID,
		"getByPayload",
		nearclient.CallFunctionWithFinality("final"),
		nearclient.CallFunctionWithArgs(map[string]interface{}{"payloadCid": payloadCid}),
	)
	if err != nil {
		return nil, fmt.Errorf("calling rpc function: %v", err)
	}
	if string(res.Result) == nullStringValue {
		return nil, ErrorNotFound
	}
	var payloadInfo PayloadInfo
	if err := json.Unmarshal(res.Result, &payloadInfo); err != nil {
		return nil, fmt.Errorf("unmarshaling payload info: %v", err)
	}
	return &payloadInfo, nil
}

// GetByCid get a payload record by data cid.
func (c *Client) GetByCid(ctx context.Context, dataCid string) (*PayloadInfo, error) {
	res, err := c.nc.CallFunction(
		ctx,
		c.lockboxAccountID,
		"getByCid",
		nearclient.CallFunctionWithFinality("final"),
		nearclient.CallFunctionWithArgs(map[string]interface{}{"dataCid": dataCid}),
	)
	if err != nil {
		return nil, fmt.Errorf("calling rpc function: %v", err)
	}
	if string(res.Result) == nullStringValue {
		return nil, ErrorNotFound
	}
	var payloadInfo PayloadInfo
	if err := json.Unmarshal(res.Result, &payloadInfo); err != nil {
		return nil, fmt.Errorf("unmarshaling payload info: %v", err)
	}
	return &payloadInfo, nil
}

// PushPayload pushes a new payload record and optionally updates cid mappings.
func (c *Client) PushPayload(ctx context.Context, payloadInfo PayloadInfo, dataCids []string) error {
	_, err := c.nc.Account(c.clientAccountID).FunctionCall(
		ctx,
		c.lockboxAccountID,
		"pushPayload",
		transaction.FunctionCallWithArgs(map[string]interface{}{
			"payload":  payloadInfo,
			"dataCids": dataCids,
		}),
	)
	if err != nil {
		return fmt.Errorf("calling rpc function: %v", err)
	}
	return nil
}

// STATE

// GetState returns the contract state.
func (c *Client) GetState(ctx context.Context) (*State, error) {
	res, err := c.nc.Account(c.lockboxAccountID).ViewState(
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
	return c.nc.Account(c.lockboxAccountID).State(ctx, account.StateWithFinality("final"))
}

// GetChanges gets the lock box state changes for a block height.
func (c *Client) GetChanges(ctx context.Context, blockHeight int) ([]Change, string, error) {
	res, err := c.nc.DataChanges(ctx, []string{c.lockboxAccountID}, nearclient.DataChangesWithBlockHeight(blockHeight))
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
