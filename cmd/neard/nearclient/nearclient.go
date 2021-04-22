package nearclient

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/rpc"
)

type queryRequest struct {
	RequestType  string      `json:"request_type"`
	Finality     string      `json:"finality,omitempty"`
	BlockID      interface{} `json:"block_id,omitempty"`
	AccountID    string      `json:"account_id,omitempty"`
	PrefixBase64 string      `json:"prefix_base64"`
	MethodName   string      `json:"method_name,omitempty"`
	ArgsBase64   string      `json:"args_base64,omitempty"`
}

type changesRequest struct {
	ChangesType     string      `json:"changes_type"`
	AccountIDs      []string    `json:"account_ids"`
	KeyPrefixBase64 string      `json:"key_prefix_base64"`
	Finality        string      `json:"finality,omitempty"`
	BlockID         interface{} `json:"block_id,omitempty"`
}

// Value models a state key-value pair.
type Value struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ViewStateResponse holds information about contract state.
type ViewStateResponse struct {
	Values      []Value `json:"values"`
	BlockHash   string  `json:"block_hash"`
	BlockHeight int     `json:"block_height"`
}

// ViewAccountResponse holds information about an account.
type ViewAccountResponse struct {
	Amount        string `json:"amount"`
	Locked        string `json:"locked"`
	CodeHash      string `json:"code_hash"`
	StorageUsage  int    `json:"storage_usage"`
	StoragePaidAt int    `json:"storage_paid_at"`
	BlockHeight   int    `json:"block_height"`
	BlockHash     string `json:"block_hash"`
}

// CallFunctionResponse holds information about the result of a function call.
type CallFunctionResponse struct {
	Result      []byte   `json:"result"`
	Logs        []string `json:"logs"`
	BlockHeight int      `json:"block_height"`
	BlockHash   string   `json:"block_hash"`
}

// Change holds information about a state change of a key-value pair.
type Change struct {
	AccountID   string `json:"account_id"`
	KeyBase64   string `json:"key_base64"`
	ValueBase64 string `json:"value_base64"`
}

// Cause holds information about the cause of a state change.
type Cause struct {
	Type        string `json:"type"`
	ReceiptHash string `json:"receipt_hash"`
}

// ChangeData holds information about a state change.
type ChangeData struct {
	Cause  Cause  `json:"cause"`
	Type   string `json:"type"`
	Change Change `json:"change"`
}

// DataChangesResponse holds information about all data changes in a block.
type DataChangesResponse struct {
	BlockHash string       `json:"block_hash"`
	Changes   []ChangeData `json:"changes"`
}

// Client communicates with the NEAR API.
type Client struct {
	rpcClient *rpc.Client
}

// NewClient creates a new Client.
func NewClient(rpcClient *rpc.Client) (*Client, error) {
	return &Client{
		rpcClient: rpcClient,
	}, nil
}

// ViewStateOption controls the behavior when calling ViewState.
type ViewStateOption func(*queryRequest)

// ViewStateWithFinality specifies the finality to be used when querying the state.
func ViewStateWithFinality(finalaity string) ViewStateOption {
	return func(qr *queryRequest) {
		qr.Finality = finalaity
	}
}

// ViewStateWithBlockHeight specifies the block height to query the state for.
func ViewStateWithBlockHeight(blockHeight int) ViewStateOption {
	return func(qr *queryRequest) {
		qr.BlockID = blockHeight
	}
}

// ViewStateWithBlockHash specifies the block hash to query the state for.
func ViewStateWithBlockHash(blockHash string) ViewStateOption {
	return func(qr *queryRequest) {
		qr.BlockID = blockHash
	}
}

// ViewStateWithPrefix specifies the state key prefix to query for.
func ViewStateWithPrefix(prefix string) ViewStateOption {
	return func(qr *queryRequest) {
		qr.PrefixBase64 = base64.StdEncoding.EncodeToString([]byte(prefix))
	}
}

// ViewState queries the contract state.
func (c *Client) ViewState(ctx context.Context, accountID string, opts ...ViewStateOption) (*ViewStateResponse, error) {
	req := &queryRequest{
		RequestType:  "view_state",
		AccountID:    accountID,
		PrefixBase64: "",
	}
	for _, opt := range opts {
		opt(req)
	}
	if req.BlockID == nil && req.Finality == "" {
		return nil, fmt.Errorf("you must provide ViewStateWithBlockHeight, ViewStateWithBlockHash or ViewStateWithFinality")
	}
	if req.BlockID != nil && req.Finality != "" {
		return nil, fmt.Errorf(
			"you must provide one of ViewStateWithBlockHeight, ViewStateWithBlockHash or ViewStateWithFinality",
		)
	}
	var res ViewStateResponse
	err := c.rpcClient.CallContext(ctx, &res, "query", rpc.NewNamedParams(req))
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// ViewAccountOption controls the behavior when calling ViewAccount.
type ViewAccountOption func(*queryRequest)

// ViewAccountWithFinality specifies the finality to be used when querying the account.
func ViewAccountWithFinality(finalaity string) ViewAccountOption {
	return func(qr *queryRequest) {
		qr.Finality = finalaity
	}
}

// ViewAccountWithBlockHeight specifies the block height to query the account for.
func ViewAccountWithBlockHeight(blockHeight int) ViewAccountOption {
	return func(qr *queryRequest) {
		qr.BlockID = blockHeight
	}
}

// ViewAccountWithBlockHash specifies the block hash to query the account for.
func ViewAccountWithBlockHash(blockHash string) ViewAccountOption {
	return func(qr *queryRequest) {
		qr.BlockID = blockHash
	}
}

// ViewAccount queries information about an account.
func (c *Client) ViewAccount(
	ctx context.Context,
	accountID string,
	opts ...ViewAccountOption,
) (*ViewAccountResponse, error) {
	req := &queryRequest{
		RequestType: "view_account",
		AccountID:   accountID,
	}
	for _, opt := range opts {
		opt(req)
	}
	if req.BlockID == nil && req.Finality == "" {
		return nil, fmt.Errorf(
			"you must provide ViewAccountWithBlockHeight, ViewAccountWithBlockHash or ViewAccountWithFinality",
		)
	}
	if req.BlockID != nil && req.Finality != "" {
		return nil, fmt.Errorf(
			"you must provide one of ViewAccountWithBlockHeight, ViewAccountWithBlockHash or ViewAccountWithFinality",
		)
	}
	var res ViewAccountResponse
	err := c.rpcClient.CallContext(ctx, &res, "query", rpc.NewNamedParams(req))
	if err != nil {
		return nil, err
	}
	return &res, nil
}

// CallFunctionOption controls the behavior when calling CallFunction.
type CallFunctionOption func(*queryRequest) error

// CallFunctionWithFinality specifies the finality to be used when calling the function.
func CallFunctionWithFinality(finalaity string) CallFunctionOption {
	return func(qr *queryRequest) error {
		qr.Finality = finalaity
		return nil
	}
}

// CallFunctionWithBlockHeight specifies the block height to call the function for.
func CallFunctionWithBlockHeight(blockHeight int) CallFunctionOption {
	return func(qr *queryRequest) error {
		qr.BlockID = blockHeight
		return nil
	}
}

// CallFunctionWithBlockHash specifies the block hash to call the function for.
func CallFunctionWithBlockHash(blockHash string) CallFunctionOption {
	return func(qr *queryRequest) error {
		qr.BlockID = blockHash
		return nil
	}
}

// CallFunctionWithArgs specified the args to call the function with.
// Should be a JSON encodable object.
func CallFunctionWithArgs(args interface{}) CallFunctionOption {
	return func(qr *queryRequest) error {
		if args == nil {
			args = make(map[string]interface{})
		}
		bytes, err := json.Marshal(args)
		if err != nil {
			return err
		}
		qr.ArgsBase64 = base64.StdEncoding.EncodeToString(bytes)
		return nil
	}
}

// CallFunction calls a function on a contract.
func (c *Client) CallFunction(
	ctx context.Context,
	accountID string,
	methodName string,
	opts ...CallFunctionOption,
) (*CallFunctionResponse, error) {
	req := &queryRequest{
		RequestType: "call_function",
		AccountID:   accountID,
		MethodName:  methodName,
	}
	for _, opt := range opts {
		if err := opt(req); err != nil {
			return nil, err
		}
	}
	if req.BlockID == nil && req.Finality == "" {
		return nil, fmt.Errorf(
			"you must provide ViewAccountWithBlockHeight, ViewAccountWithBlockHash or ViewAccountWithFinality",
		)
	}
	if req.BlockID != nil && req.Finality != "" {
		return nil, fmt.Errorf(
			"you must provide one of ViewAccountWithBlockHeight, ViewAccountWithBlockHash or ViewAccountWithFinality",
		)
	}
	var res CallFunctionResponse
	if err := c.rpcClient.CallContext(ctx, &res, "query", rpc.NewNamedParams(req)); err != nil {
		return nil, err
	}
	return &res, nil
}

// DataChangesOption controls behavior when calling DataChanges.
type DataChangesOption func(*changesRequest)

// DataChangesWithPrefix sets the data key prefix to query for.
func DataChangesWithPrefix(prefix string) DataChangesOption {
	return func(cr *changesRequest) {
		cr.KeyPrefixBase64 = base64.StdEncoding.EncodeToString([]byte(prefix))
	}
}

// DataChangesWithFinality specifies the finality to be used when querying data changes.
func DataChangesWithFinality(finalaity string) DataChangesOption {
	return func(qr *changesRequest) {
		qr.Finality = finalaity
	}
}

// DataChangesWithBlockHeight specifies the block id to query data changes for.
func DataChangesWithBlockHeight(blockHeight int) DataChangesOption {
	return func(qr *changesRequest) {
		qr.BlockID = blockHeight
	}
}

// DataChangesWithBlockHash specifies the block id to query data changes for.
func DataChangesWithBlockHash(blockHash string) DataChangesOption {
	return func(qr *changesRequest) {
		qr.BlockID = blockHash
	}
}

// DataChanges queries changes to contract data changes.
func (c *Client) DataChanges(
	ctx context.Context,
	accountIDs []string,
	opts ...DataChangesOption,
) (*DataChangesResponse, error) {
	req := &changesRequest{
		ChangesType: "data_changes",
		AccountIDs:  accountIDs,
	}
	for _, opt := range opts {
		opt(req)
	}
	if req.BlockID == nil && req.Finality == "" {
		return nil, fmt.Errorf(
			"you must provide DataChangesWithBlockHeight, DataChangesWithBlockHash or DataChangesWithFinality",
		)
	}
	if req.BlockID != nil && req.Finality != "" {
		return nil, fmt.Errorf(
			"you must provide one of DataChangesWithBlockHeight, DataChangesWithBlockHash or DataChangesWithFinality",
		)
	}
	var res DataChangesResponse
	if err := c.rpcClient.CallContext(ctx, &res, "EXPERIMENTAL_changes", rpc.NewNamedParams(req)); err != nil {
		return nil, err
	}
	return &res, nil
}
