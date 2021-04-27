package account

import (
	"github.com/textileio/broker-core/cmd/neard/nearclient/internal/types"
)

// Value models a state key-value pair.
type Value struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// AccountStateView holds information about contract state.
type AccountStateView struct {
	types.QueryResponse
	Values []Value `json:"values"`
}

// AccountView holds information about an account.
type AccountView struct {
	types.QueryResponse
	Amount        string `json:"amount"`
	Locked        string `json:"locked"`
	CodeHash      string `json:"code_hash"`
	StorageUsage  int    `json:"storage_usage"`
	StoragePaidAt int    `json:"storage_paid_at"`
}

// AccessKeyView contains information about an access key.
type AccessKeyView struct {
	types.QueryResponse
	Nonce int `json:"nonce"`
	// Permission can be a string "FullAccess" or a FunctionCallPermissionView.
	Permission interface{} `json:"permission"`
}

// FunctionCall provides information about the allowed function call.
type FunctionCall struct {
	Allowance   string   `json:"allowance"`
	ReceiverID  string   `json:"receiver_id"`
	MethodNames []string `json:"method_names"`
}

// FunctionCallPermissionView contains a FunctionCall.
type FunctionCallPermissionView struct {
	FunctionCall FunctionCall `json:"FunctionCall"`
}
