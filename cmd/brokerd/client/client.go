package client

import (
	"context"
	"fmt"

	"github.com/ipfs/go-cid"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/brokerd/cast"
	"github.com/textileio/broker-core/common"
	pb "github.com/textileio/broker-core/gen/broker/v1"
	"github.com/textileio/broker-core/rpc"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/durationpb"
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
func (c *Client) Create(ctx context.Context, dataCid cid.Cid, origin string) (broker.StorageRequest, error) {
	req := &pb.CreateStorageRequestRequest{
		Cid:    dataCid.String(),
		Origin: origin,
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
	pc broker.PreparedCAR,
	meta broker.BatchMetadata,
	rw *broker.RemoteWallet) (broker.StorageRequest, error) {
	var carURL *pb.CreatePreparedStorageRequestRequest_PreparedCAR_CARURL
	if pc.Sources.CARURL != nil {
		carURL = &pb.CreatePreparedStorageRequestRequest_PreparedCAR_CARURL{
			Url: pc.Sources.CARURL.URL.String(),
		}
	}
	var carIpfs *pb.CreatePreparedStorageRequestRequest_PreparedCAR_CARIPFS
	if pc.Sources.CARIPFS != nil {
		carIpfs = &pb.CreatePreparedStorageRequestRequest_PreparedCAR_CARIPFS{
			Cid:        pc.Sources.CARIPFS.Cid.String(),
			Multiaddrs: make([]string, len(pc.Sources.CARIPFS.Multiaddrs)),
		}
		for i, ma := range pc.Sources.CARIPFS.Multiaddrs {
			carIpfs.Multiaddrs[i] = ma.String()
		}
	}

	req := &pb.CreatePreparedStorageRequestRequest{
		Cid: dataCid.String(),
		PreparedCar: &pb.CreatePreparedStorageRequestRequest_PreparedCAR{
			PieceCid:         pc.PieceCid.String(),
			PieceSize:        pc.PieceSize,
			RepFactor:        int64(pc.RepFactor),
			Deadline:         timestamppb.New(pc.Deadline),
			ProposalDuration: durationpb.New(pc.ProposalDuration),
			CarUrl:           carURL,
			CarIpfs:          carIpfs,
		},
		Metadata: &pb.CreatePreparedStorageRequestRequest_Metadata{
			Origin:    meta.Origin,
			Tags:      meta.Tags,
			Providers: common.StringifyAddrs(meta.Providers...),
		},
	}
	if rw != nil {
		rwStrMultiaddrs := make([]string, 0, len(rw.Multiaddrs))
		for _, maddr := range rw.Multiaddrs {
			rwStrMultiaddrs = append(rwStrMultiaddrs, maddr.String())
		}
		walletAddr := common.StringifyAddr(rw.WalletAddr)
		req.RemoteWallet = &pb.RemoteWallet{
			PeerId:     rw.PeerID.String(),
			AuthToken:  rw.AuthToken,
			WalletAddr: walletAddr,
			Multiaddrs: rwStrMultiaddrs,
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
