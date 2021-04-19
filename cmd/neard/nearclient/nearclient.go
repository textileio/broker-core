package nearclient

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/ethereum/go-ethereum/rpc"
)

type queryRequest struct {
	RequestType  string `json:"request_type"`
	Finality     string `json:"finality,omitempty"`
	BlockID      string `json:"block_id,omitempty"`
	AccountID    string `json:"account_id,omitempty"`
	PrefixBase64 string `json:"prefix_base64,omitempty"`
}

type changesRequest struct {
	ChangesType     string   `json:"changes_type"`
	AccountIDs      []string `json:"account_ids"`
	KeyPrefixBase64 string   `json:"key_prefix_base64"`
	Finality        string   `json:"finality,omitempty"`
	BlockID         string   `json:"block_id,omitempty"`
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

// ViewStateWithBlockID specifies the block id to query the state for.
func ViewStateWithBlockID(blockID string) ViewStateOption {
	return func(qr *queryRequest) {
		qr.BlockID = blockID
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
		RequestType: "view_state",
		AccountID:   accountID,
	}
	for _, opt := range opts {
		opt(req)
	}
	if req.BlockID == "" && req.Finality == "" {
		return nil, fmt.Errorf("you must provide ViewStateWithBlockID or ViewStateWithFinality")
	}
	if req.BlockID != "" && req.Finality != "" {
		return nil, fmt.Errorf("you must provide one of ViewStateWithBlockID or ViewStateWithFinality, but not both")
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

// ViewAccountWithBlockID specifies the block id to query the account for.
func ViewAccountWithBlockID(blockID string) ViewAccountOption {
	return func(qr *queryRequest) {
		qr.BlockID = blockID
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
	if req.BlockID == "" && req.Finality == "" {
		return nil, fmt.Errorf("you must provide ViewAccountWithBlockID or ViewAccountWithFinality")
	}
	if req.BlockID != "" && req.Finality != "" {
		return nil, fmt.Errorf("you must provide one of ViewAccountWithBlockID or ViewAccountWithFinality, but not both")
	}
	var res ViewAccountResponse
	err := c.rpcClient.CallContext(ctx, &res, "query", rpc.NewNamedParams(req))
	if err != nil {
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

// DataChangesWithBlockID specifies the block id to query data changes for.
func DataChangesWithBlockID(blockID string) DataChangesOption {
	return func(qr *changesRequest) {
		qr.BlockID = blockID
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
	if req.BlockID == "" && req.Finality == "" {
		return nil, fmt.Errorf("you must provide DataChangesWithBlockID or DataChangesWithFinality")
	}
	if req.BlockID != "" && req.Finality != "" {
		return nil, fmt.Errorf("you must provide one of DataChangesWithBlockID or DataChangesWithFinality, but not both")
	}
	var res DataChangesResponse
	err := c.rpcClient.CallContext(ctx, &res, "EXPERIMENTAL_changes", rpc.NewNamedParams(req))
	if err != nil {
		return nil, err
	}
	return &res, nil
}
