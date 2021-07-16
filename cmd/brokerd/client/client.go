package client

import (
	"context"
	"fmt"

	"github.com/ipfs/go-cid"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/brokerd/cast"
	pb "github.com/textileio/broker-core/gen/broker/v1"
	"github.com/textileio/broker-core/rpc"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// Client is a brokerd client.
type Client struct {
	c    pb.APIServiceClient
	conn *grpc.ClientConn
}

var _ broker.Broker = (*Client)(nil)

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
func (c *Client) Create(ctx context.Context, dataCid cid.Cid) (broker.BrokerRequest, error) {
	req := &pb.CreateBrokerRequestRequest{
		Cid: dataCid.String(),
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

// CreatePrepared creates a broker request for prepared data.
func (c *Client) CreatePrepared(
	ctx context.Context,
	dataCid cid.Cid,
	pc broker.PreparedCAR) (broker.BrokerRequest, error) {
	req := &pb.CreatePreparedBrokerRequestRequest{
		Cid: dataCid.String(),
	}
	req.PreparedCAR = &pb.CreatePreparedBrokerRequestRequest_PreparedCAR{
		PieceCid:  pc.PieceCid.String(),
		PieceSize: pc.PieceSize,
		RepFactor: int64(pc.RepFactor),
		Deadline:  timestamppb.New(pc.Deadline),
	}
	if pc.Sources.CARURL != nil {
		req.PreparedCAR.CarUrl = &pb.CreatePreparedBrokerRequestRequest_PreparedCAR_CARURL{
			Url: pc.Sources.CARURL.URL.String(),
		}
	}
	if pc.Sources.CARIPFS != nil {
		req.PreparedCAR.CarIpfs = &pb.CreatePreparedBrokerRequestRequest_PreparedCAR_CARIPFS{
			Cid:        pc.Sources.CARIPFS.Cid.String(),
			Multiaddrs: make([]string, len(pc.Sources.CARIPFS.Multiaddrs)),
		}
		for i, ma := range pc.Sources.CARIPFS.Multiaddrs {
			req.PreparedCAR.CarIpfs.Multiaddrs[i] = ma.String()
		}
	}

	res, err := c.c.CreatePreparedBrokerRequest(ctx, req)
	if err != nil {
		return broker.BrokerRequest{}, fmt.Errorf("creating broker request: %s", err)
	}

	br, err := cast.FromProtoBrokerRequest(res.Request)
	if err != nil {
		return broker.BrokerRequest{}, fmt.Errorf("decoding proto response: %s", err)
	}

	return br, nil
}

// GetBrokerRequestInfo gets a broker request information by id.
func (c *Client) GetBrokerRequestInfo(
	ctx context.Context,
	id broker.BrokerRequestID) (broker.BrokerRequestInfo, error) {
	req := &pb.GetBrokerRequestInfoRequest{
		Id: string(id),
	}
	res, err := c.c.GetBrokerRequestInfo(ctx, req)
	if err != nil {
		return broker.BrokerRequestInfo{}, fmt.Errorf("calling get broker api: %s", err)
	}
	br, err := cast.FromProtoBrokerRequestInfo(res)
	if err != nil {
		return broker.BrokerRequestInfo{}, fmt.Errorf("converting broker request response: %s", err)
	}

	return br, nil
}

// StorageDealAuctioned indicates the storage deal auction has completed.
func (c *Client) StorageDealAuctioned(ctx context.Context, auction broker.ClosedAuction) error {
	req := cast.ClosedAuctionToPb(auction)
	if _, err := c.c.StorageDealAuctioned(ctx, req); err != nil {
		return fmt.Errorf("calling storage deal winners api: %s", err)
	}
	return nil
}

// Close closes gracefully the client.
func (c *Client) Close() error {
	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("closing gRPC client: %s", err)
	}
	return nil
}
