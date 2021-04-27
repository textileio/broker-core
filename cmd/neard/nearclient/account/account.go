package account

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/mr-tron/base58/base58"
	itypes "github.com/textileio/broker-core/cmd/neard/nearclient/internal/types"
	"github.com/textileio/broker-core/cmd/neard/nearclient/keys"
	"github.com/textileio/broker-core/cmd/neard/nearclient/transaction"
	"github.com/textileio/broker-core/cmd/neard/nearclient/types"
)

// Account provides functions for a single account.
type Account struct {
	config    *types.Config
	accountID string
}

// NewAccount creates a new account.
func NewAccount(config *types.Config, accountID string) *Account {
	return &Account{
		config:    config,
		accountID: accountID,
	}
}

// ViewState queries the contract state.
func (a *Account) ViewState(ctx context.Context, opts ...ViewStateOption) (*AccountStateView, error) {
	req := &itypes.QueryRequest{
		RequestType:  "view_state",
		AccountID:    a.accountID,
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
	var res AccountStateView
	if err := a.config.RPCClient.CallContext(ctx, &res, "query", rpc.NewNamedParams(req)); err != nil {
		return nil, fmt.Errorf("calling rpc: %v", err)
	}
	return &res, nil
}

// State queries information about the account state.
func (a *Account) State(
	ctx context.Context,
	opts ...StateOption,
) (*AccountView, error) {
	req := &itypes.QueryRequest{
		RequestType: "view_account",
		AccountID:   a.accountID,
		Finality:    "optimistic",
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
	var res AccountView
	if err := a.config.RPCClient.CallContext(ctx, &res, "query", rpc.NewNamedParams(req)); err != nil {
		return nil, fmt.Errorf("calling rpc: %v", err)
	}
	return &res, nil
}

// FindAccessKey finds the AccessKeyView associated with the account's PublicKey stored in the KeyStore.
func (a *Account) FindAccessKey(
	ctx context.Context,
	receiverID string,
	actions []transaction.Action,
) (*keys.PublicKey, *AccessKeyView, error) {
	// TODO: Find matching access key based on transaction (i.e. receiverId and actions)
	_ = receiverID
	_ = actions

	pubKey := a.config.Signer.GetPublicKey()

	// TODO: Lookup answer in a cache first.

	pubKeyStr, err := pubKey.ToString()
	if err != nil {
		return nil, nil, fmt.Errorf("converting public key to string: %v", err)
	}

	req := &itypes.QueryRequest{
		RequestType: "view_access_key",
		AccountID:   a.accountID,
		PublicKey:   pubKeyStr,
		Finality:    "optimistic",
	}

	type viewAccessKeyResp struct {
		itypes.QueryResponse
		Nonce      uint64           `json:"nonce"`
		Permission *json.RawMessage `json:"permission"`
	}

	var raw json.RawMessage
	resp := &viewAccessKeyResp{Permission: &raw}

	// var res AccessKeyView
	if err := a.config.RPCClient.CallContext(ctx, &resp, "query", rpc.NewNamedParams(req)); err != nil {
		return nil, nil, fmt.Errorf("calling rpc: %v", err)
	}

	ret := &AccessKeyView{
		QueryResponse: itypes.QueryResponse{
			BlockHash:   resp.BlockHash,
			BlockHeight: resp.BlockHeight,
		},
		Nonce: resp.Nonce,
	}

	if string(raw) == "\"FullAccess\"" {
		ret.PermissionType = FullAccessPermissionType
	} else {
		var view FunctionCallPermissionView
		if err := json.Unmarshal(raw, &view); err != nil {
			return nil, nil, fmt.Errorf("unmarshaling permission: %v", err)
		}
		ret.FunctionCallPermissionView = &view
		ret.PermissionType = FunctionCallPermissionType
	}

	return &pubKey, ret, nil
}

func (a *Account) SignTransaction(ctx context.Context, receiverID string, actions ...transaction.Action) ([]byte, *transaction.SignedTransaction, error) {
	_, accessKeyView, err := a.FindAccessKey(ctx, receiverID, actions)
	if err != nil {
		return nil, nil, fmt.Errorf("finding access key: %v", err)
	}
	if accessKeyView == nil {
		return nil, nil, fmt.Errorf("no access key view founf") // TODO: Better error message.
	}
	var res itypes.BlockResult
	if err := a.config.RPCClient.CallContext(ctx, &res, "block", rpc.NewNamedParams(itypes.BlockRequest{Finality: "final"})); err != nil {
		return nil, nil, fmt.Errorf("calling block rpc: %v", err)
	}
	blockHash, err := base58.Decode(res.Header.Hash)
	if err != nil {
		return nil, nil, fmt.Errorf("decoding hash: %v", err)
	}
	nonce := accessKeyView.Nonce + 1

	pk := a.config.Signer.GetPublicKey()

	t := transaction.Transaction{
		SignerID: a.accountID,
		PublicKey: transaction.PublicKey{
			KeyType: uint8(pk.Type),
			Data:    pk.Data,
		},
		Nonce:      nonce,
		ReceiverID: receiverID,
		BlockHash:  blockHash,
		Actions:    actions,
	}
	hash, signedTransaction, err := transaction.SignTransaction(t, a.config.Signer, a.accountID, a.config.NetworkID)
	if err != nil {
		return nil, nil, fmt.Errorf("signing transaction: %v", err)
	}
	return hash, signedTransaction, nil
}
