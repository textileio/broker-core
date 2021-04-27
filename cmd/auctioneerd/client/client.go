package client

import (
	"context"

	"github.com/textileio/broker-core/broker"
	pb "github.com/textileio/broker-core/gen/broker/auctioneer/v1"
	"google.golang.org/grpc"
)

// Client provides the client api.
type Client struct {
	c pb.APIServiceClient
}

// NewClient starts the client.
func NewClient(cc *grpc.ClientConn) *Client {
	return &Client{
		c: pb.NewAPIServiceClient(cc),
	}
}

// CreateAuction creates an auction.
func (c *Client) CreateAuction(
	ctx context.Context,
	dealID broker.StorageDealID,
	dealSize, dealDuration uint64,
) (*pb.CreateAuctionResponse, error) {
	return c.c.CreateAuction(ctx, &pb.CreateAuctionRequest{
		DealId:       string(dealID),
		DealSize:     dealSize,
		DealDuration: dealDuration,
	})
}

// GetAuction returns an auction by id.
func (c *Client) GetAuction(ctx context.Context, id string) (*pb.GetAuctionResponse, error) {
	return c.c.GetAuction(ctx, &pb.GetAuctionRequest{
		Id: id,
	})
}
