package client

import (
	"context"

	pb "github.com/textileio/broker-core/gen/broker/auth/v1"
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

// Auth authenticates the JWT.
func (c *Client) Auth(ctx context.Context, req *pb.AuthRequest) (*pb.AuthResponse, error) {
	return c.c.Auth(ctx, req)
}
