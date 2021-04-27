package account

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/rpc"
	itypes "github.com/textileio/broker-core/cmd/neard/nearclient/internal/types"
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
) (*AccessKeyView, error) {
	// TODO: Find matching access key based on transaction (i.e. receiverId and actions)
	_ = receiverID
	_ = actions

	pubKey := a.config.Signer.GetPublicKey()

	// TODO: Lookup answer in a cache first.

	pubKeyStr, err := pubKey.ToString()
	if err != nil {
		return nil, fmt.Errorf("converting public key to string: %v", err)
	}

	req := &itypes.QueryRequest{
		RequestType: "view_access_key",
		AccountID:   a.accountID,
		PublicKey:   pubKeyStr,
		Finality:    "optimistic",
	}

	var res AccessKeyView
	if err := a.config.RPCClient.CallContext(ctx, &res, "query", rpc.NewNamedParams(req)); err != nil {
		return nil, fmt.Errorf("calling rpc: %v", err)
	}

	return &res, nil
}
