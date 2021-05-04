package client

import (
	"context"
	"fmt"

	"github.com/textileio/broker-core/dealer"
	pb "github.com/textileio/broker-core/gen/broker/dealer/v1"
	"google.golang.org/grpc"
)

// Client provides the client api.
type Client struct {
	c    pb.APIServiceClient
	conn *grpc.ClientConn
}

var _ dealer.Dealer = (*Client)(nil)

// NewClient starts the client.
func NewClient(cc *grpc.ClientConn) *Client {
	return &Client{
		c: pb.NewAPIServiceClient(cc),
	}
}

// ReadyToCreateDeals registers deals that are ready to be executed.
func (c *Client) ReadyToCreateDeals(ctx context.Context, sdb dealer.AuctionDeals) error {
	req := &pb.ReadyToCreateDealsRequest{
		StorageDealId: string(sdb.StorageDealID),
		PayloadCid:    sdb.PayloadCid.String(),
		PieceCid:      sdb.PieceCid.String(),
		PieceSize:     sdb.PieceSize,
		Duration:      sdb.Duration,
		Targets:       make([]*pb.ReadyToCreateDealsRequest_AuctionDealsTarget, len(sdb.Targets)),
	}
	for i, t := range sdb.Targets {
		req.Targets[i] = &pb.ReadyToCreateDealsRequest_AuctionDealsTarget{
			Miner:               t.Miner,
			PricePerGibPerEpoch: t.PricePerGiBPerEpoch,
			StartEpoch:          t.StartEpoch,
			Verified:            t.Verified,
			FastRetrieval:       t.FastRetrieval,
		}
	}
	_, err := c.c.ReadyToCreateDeals(ctx, req)
	if err != nil {
		return fmt.Errorf("calling ready to create deals api: %s", err)
	}

	return nil
}

// Close closes the client's grpc connection and cancels any active requests.
func (c *Client) Close() error {
	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("closing grpc client: %s", err)
	}
	return nil
}
