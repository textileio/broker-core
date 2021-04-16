package client

import (
	"context"

	pb "github.com/textileio/broker-core/gen/broker/auctioneer/v1"
	"google.golang.org/grpc"
)

// Client provides the client api.
type Client struct {
	c    pb.APIServiceClient
	conn *grpc.ClientConn
}

// NewClient starts the client.
func NewClient(addr string, opts ...grpc.DialOption) (*Client, error) {
	conn, err := grpc.Dial(addr, opts...)
	if err != nil {
		return nil, err
	}
	return &Client{
		c:    pb.NewAPIServiceClient(conn),
		conn: conn,
	}, nil
}

// Close closes the client's grpc connection and cancels any active requests.
func (c *Client) Close() error {
	return c.conn.Close()
}

// CreateAuction creates an auction.
func (c *Client) CreateAuction(ctx context.Context) (*pb.CreateAuctionResponse, error) {
	return c.c.CreateAuction(ctx, &pb.CreateAuctionRequest{})
}

// GetAuction returns an auction by id.
func (c *Client) GetAuction(ctx context.Context, id string) (*pb.GetAuctionResponse, error) {
	return c.c.GetAuction(ctx, &pb.GetAuctionRequest{
		Id: id,
	})
}
