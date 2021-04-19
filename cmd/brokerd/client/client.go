package client

import (
	"context"
	"fmt"

	"google.golang.org/grpc"

	"github.com/ipfs/go-cid"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/brokerd/cast"
	pb "github.com/textileio/broker-core/gen/broker/v1"
	"github.com/textileio/broker-core/rpc"
)

// Client is a brokerd client.
type Client struct {
	c    pb.APIServiceClient
	conn *grpc.ClientConn
}

// New returns a new *Client.
func New(brokerAPIAddr string, opts ...grpc.DialOption) (*Client, error) {
	conn, err := grpc.Dial(brokerAPIAddr, rpc.GetClientOpts(brokerAPIAddr)...)
	if err != nil {
		return nil, err
	}
	c := &Client{
		c:    pb.NewAPIServiceClient(conn),
		conn: conn,
	}

	return c, nil
}

// Create creates a new BrokerRequest.
func (c *Client) Create(ctx context.Context, dataCid cid.Cid, meta broker.Metadata) (broker.BrokerRequest, error) {
	req := &pb.CreateBrokerRequestRequest{
		Cid: dataCid.String(),
		Meta: &pb.BrokerRequest_Metadata{
			Region: meta.Region,
		},
	}
	res, err := c.c.CreateBrokerRequest(ctx, req)
	if err != nil {
		return broker.BrokerRequest{}, fmt.Errorf("creating broker request: %s", err)
	}

	br, err := cast.FromProtoBrokerRequest(res.Request)
	if err != nil {
		return broker.BrokerRequest{}, fmt.Errorf("decoding proto response: %s", err)
	}

	return br, nil
}

// Get gets a broker request from its ID.
func (c *Client) Get(ctx context.Context, id broker.BrokerRequestID) (broker.BrokerRequest, error) {
	req := &pb.GetBrokerRequestRequest{
		Id: string(id),
	}
	res, err := c.c.GetBrokerRequest(ctx, req)
	if err != nil {
		return broker.BrokerRequest{}, fmt.Errorf("calling get broker api: %s", err)
	}
	br, err := cast.FromProtoBrokerRequest(res.BrokerRequest)
	if err != nil {
		return broker.BrokerRequest{}, fmt.Errorf("converting broker request response: %s", err)
	}

	return br, nil
}

// CreateStorageDeal deal creates a storage deal.
func (c *Client) CreateStorageDeal(
	ctx context.Context,
	batchCid cid.Cid,
	ids []broker.BrokerRequestID) (broker.StorageDealID, error) {
	if !batchCid.Defined() {
		return "", fmt.Errorf("batch cid is undefined")
	}
	if len(ids) == 0 {
		return "", fmt.Errorf("grouped broker requests list is empty")
	}

	brids := make([]string, len(ids))
	for i, brID := range ids {
		brids[i] = string(brID)
	}

	req := &pb.CreateStorageDealRequest{
		BatchCid:         batchCid.String(),
		BrokerRequestIds: brids,
	}
	res, err := c.c.CreateStorageDeal(ctx, req)
	if err != nil {
		return "", fmt.Errorf("calling create storage deal api: %s", err)
	}

	return broker.StorageDealID(res.Id), nil
}

// Close closes gracefully the client.
func (c *Client) Close() error {
	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("closing gRPC client: %s", err)
	}
	return nil
}
