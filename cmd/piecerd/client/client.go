package client

import (
	"context"
	"fmt"

	"github.com/ipfs/go-cid"
	"github.com/textileio/broker-core/broker"
	pb "github.com/textileio/broker-core/gen/broker/piecer/v1"
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

// ReadyToPrepare signals the piecer that a new StorageDeal is ready to be prepared.
// Piecer will call thebroker async with the end result.
func (c *Client) ReadyToPrepare(ctx context.Context, id broker.StorageDealID, dataCid cid.Cid) error {
	if !dataCid.Defined() {
		return fmt.Errorf("data cid is undefined")
	}
	if len(id) == 0 {
		return fmt.Errorf("broker request id is empty")
	}
	req := &pb.ReadyToPrepareRequest{
		StorageDealId: string(id),
		DataCid:       dataCid.String(),
	}
	if _, err := c.c.ReadyToPrepare(ctx, req); err != nil {
		return fmt.Errorf("calling ready to prepare api: %s", err)
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
