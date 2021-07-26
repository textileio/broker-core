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

// Create creates a new StorageRequest.
func (c *Client) Create(ctx context.Context, dataCid cid.Cid) (broker.StorageRequest, error) {
	req := &pb.CreateStorageRequestRequest{
		Cid: dataCid.String(),
	}
	res, err := c.c.CreateStorageRequest(ctx, req)
	if err != nil {
		return broker.StorageRequest{}, fmt.Errorf("creating storage request: %s", err)
	}

	br, err := cast.FromProtoStorageRequest(res.Request)
	if err != nil {
		return broker.StorageRequest{}, fmt.Errorf("decoding proto response: %s", err)
	}

	return br, nil
}

// CreatePrepared creates a storage request for prepared data.
func (c *Client) CreatePrepared(
	ctx context.Context,
	dataCid cid.Cid,
	pc broker.PreparedCAR) (broker.StorageRequest, error) {
	req := &pb.CreatePreparedStorageRequestRequest{
		Cid: dataCid.String(),
	}
	req.PreparedCAR = &pb.CreatePreparedStorageRequestRequest_PreparedCAR{
		PieceCid:  pc.PieceCid.String(),
		PieceSize: pc.PieceSize,
		RepFactor: int64(pc.RepFactor),
		Deadline:  timestamppb.New(pc.Deadline),
	}
	if pc.Sources.CARURL != nil {
		req.PreparedCAR.CarUrl = &pb.CreatePreparedStorageRequestRequest_PreparedCAR_CARURL{
			Url: pc.Sources.CARURL.URL.String(),
		}
	}
	if pc.Sources.CARIPFS != nil {
		req.PreparedCAR.CarIpfs = &pb.CreatePreparedStorageRequestRequest_PreparedCAR_CARIPFS{
			Cid:        pc.Sources.CARIPFS.Cid.String(),
			Multiaddrs: make([]string, len(pc.Sources.CARIPFS.Multiaddrs)),
		}
		for i, ma := range pc.Sources.CARIPFS.Multiaddrs {
			req.PreparedCAR.CarIpfs.Multiaddrs[i] = ma.String()
		}
	}

	res, err := c.c.CreatePreparedStorageRequest(ctx, req)
	if err != nil {
		return broker.StorageRequest{}, fmt.Errorf("creating storage request: %s", err)
	}

	br, err := cast.FromProtoStorageRequest(res.Request)
	if err != nil {
		return broker.StorageRequest{}, fmt.Errorf("decoding proto response: %s", err)
	}

	return br, nil
}

// GetStorageRequestInfo gets a storage request information by id.
func (c *Client) GetStorageRequestInfo(
	ctx context.Context,
	id broker.StorageRequestID) (broker.StorageRequestInfo, error) {
	req := &pb.GetStorageRequestInfoRequest{
		Id: string(id),
	}
	res, err := c.c.GetStorageRequestInfo(ctx, req)
	if err != nil {
		return broker.StorageRequestInfo{}, fmt.Errorf("calling get broker api: %s", err)
	}
	br, err := cast.FromProtoStorageRequestInfo(res)
	if err != nil {
		return broker.StorageRequestInfo{}, fmt.Errorf("converting storage request response: %s", err)
	}

	return br, nil
}

// Close closes gracefully the client.
func (c *Client) Close() error {
	if err := c.conn.Close(); err != nil {
		return fmt.Errorf("closing gRPC client: %s", err)
	}
	return nil
}
