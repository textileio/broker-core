package types

// QueryRequest is used for RPC query requests.
type QueryRequest struct {
	RequestType  string      `json:"request_type"`
	Finality     string      `json:"finality,omitempty"`
	BlockID      interface{} `json:"block_id,omitempty"`
	AccountID    string      `json:"account_id,omitempty"`
	PrefixBase64 string      `json:"prefix_base64"`
	MethodName   string      `json:"method_name,omitempty"`
	ArgsBase64   string      `json:"args_base64,omitempty"`
	PublicKey    string      `json:"public_key,omitempty"`
}

// QueryResponse is a base type used for responses to query requests.
type QueryResponse struct {
	BlockHash   string `json:"block_hash"`
	BlockHeight int    `json:"block_height"`
}

// ChangesRequest is used for RPC changes requests.
type ChangesRequest struct {
	ChangesType     string      `json:"changes_type"`
	AccountIDs      []string    `json:"account_ids"`
	KeyPrefixBase64 string      `json:"key_prefix_base64"`
	Finality        string      `json:"finality,omitempty"`
	BlockID         interface{} `json:"block_id,omitempty"`
}
