package client

import (
	"context"
	"fmt"

	"github.com/ipfs/go-cid"
	"github.com/textileio/broker-core/broker"
	pb "github.com/textileio/broker-core/gen/broker/packer/v1"
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

// ReadyToPack signals that a broker request is ready to be packed.
func (c *Client) ReadyToPack(ctx context.Context, id broker.BrokerRequestID, dataCid cid.Cid) error {
	if !dataCid.Defined() {
		return fmt.Errorf("data cid is undefined")
	}
	if len(id) == 0 {
		return fmt.Errorf("broker request id is empty")
	}
	req := &pb.ReadyToPackRequest{
		BrokerRequestId: string(id),
		DataCid:         dataCid.String(),
	}
	if _, err := c.c.ReadyToPack(ctx, req); err != nil {
		return fmt.Errorf("calling ready to pack api: %s", err)
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
