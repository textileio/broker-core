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
	AccountID    string `json:"account_id"`
	PrefixBase64 string `json:"prefix_base64"`
}

// Value models a state key-value pair.
type Value struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// ViewStateResponse is the response when calling ViewState.
type ViewStateResponse struct {
	Values      []Value `json:"values"`
	BlockHash   string  `json:"block_hash"`
	BlockHeight int     `json:"block_height"`
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

// WithFinality specifies the finality to be used when querying the state.
func WithFinality(finalaity string) ViewStateOption {
	return func(qr *queryRequest) {
		qr.Finality = finalaity
	}
}

// WithBlockID specifies the block id to query the state for.
func WithBlockID(blockID string) ViewStateOption {
	return func(qr *queryRequest) {
		qr.BlockID = blockID
	}
}

// WithPrefix specifies the state key prefix to query for.
func WithPrefix(prefix string) ViewStateOption {
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
		return nil, fmt.Errorf("you must provide WithBlockID or WithFinality")
	}
	if req.BlockID != "" && req.Finality != "" {
		return nil, fmt.Errorf("you must provide one of WithBlockID or WithFinality, but not both")
	}
	var res ViewStateResponse
	err := c.rpcClient.CallContext(context.Background(), &res, "query", rpc.NewNamedParams(req))
	if err != nil {
		return nil, err
	}
	return &res, nil
}
