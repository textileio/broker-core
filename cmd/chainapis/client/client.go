package client

import (
	"context"
	"fmt"

	"github.com/textileio/broker-core/chainapi"
	pb "github.com/textileio/broker-core/gen/broker/chainapi/v1"
	"google.golang.org/grpc"
)

// Client provides the client api.
type Client struct {
	c pb.ChainApiServiceClient
}

// New returns a new client.
func New(cc *grpc.ClientConn) chainapi.ChainAPI {
	return &Client{
		c: pb.NewChainApiServiceClient(cc),
	}
}

// HasDeposit checks if an account has deposited funds for a broker.
func (c *Client) HasDeposit(ctx context.Context, depositee string, chainID string) (bool, error) {
	req := &pb.HasDepositRequest{
		Depositee: depositee,
		ChainId:   chainID,
	}
	res, err := c.c.HasDeposit(ctx, req)
	if err != nil {
		return false, fmt.Errorf("calling has deposit api: %v", err)
	}
	return res.HasDeposit, nil
}

// OwnsPublicKey checks if the provided accountID owns the specified publicKey.
func (c *Client) OwnsPublicKey(ctx context.Context, accountID, publicKey, chainID string) (bool, error) {
	req := &pb.OwnsPublicKeyRequest{
		AccountId: accountID,
		PublicKey: publicKey,
		ChainId:   chainID,
	}
	res, err := c.c.OwnsPublicKey(ctx, req)
	if err != nil {
		return false, fmt.Errorf("calling owns public key api: %v", err)
	}
	return res.OwnsPublicKey, nil
}
