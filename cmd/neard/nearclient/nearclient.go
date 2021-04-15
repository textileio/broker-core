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

type Value struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type ViewStateResponse struct {
	Values      []Value `json:"values"`
	BlockHash   string  `json:"block_hash"`
	BlockHeight int     `json:"block_height"`
}

type Client struct {
	rpcClient *rpc.Client
}

func NewClient(rpcClient *rpc.Client) (*Client, error) {
	return &Client{
		rpcClient: rpcClient,
	}, nil
}

type ViewStateOption func(*queryRequest)

func WithFinality(finalaity string) ViewStateOption {
	return func(qr *queryRequest) {
		qr.Finality = finalaity
	}
}

func WithBlockID(blockID string) ViewStateOption {
	return func(qr *queryRequest) {
		qr.BlockID = blockID
	}
}

func WithPrefix(prefix string) ViewStateOption {
	return func(qr *queryRequest) {
		qr.PrefixBase64 = base64.StdEncoding.EncodeToString([]byte(prefix))
	}
}

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
