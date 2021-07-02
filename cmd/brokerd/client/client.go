package client

import (
	"context"
	"fmt"

	"github.com/ipfs/go-cid"
	"github.com/textileio/bidbot/lib/broker"
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

// StorageDealPrepared indicates the preparing output for a storage deal.
func (c *Client) StorageDealPrepared(
	ctx context.Context,
	id broker.StorageDealID,
	pr broker.DataPreparationResult) error {
	req := &pb.StorageDealPreparedRequest{
		StorageDealId: string(id),
		PieceCid:      pr.PieceCid.String(),
		PieceSize:     pr.PieceSize,
	}
	if _, err := c.c.StorageDealPrepared(ctx, req); err != nil {
		return fmt.Errorf("calling storage deal prepared api: %s", err)
	}
	return nil
}

// StorageDealAuctioned indicates the storage deal auction has completed.
func (c *Client) StorageDealAuctioned(ctx context.Context, auction broker.Auction) error {
	req := cast.AuctionToPb(auction)
	if _, err := c.c.StorageDealAuctioned(ctx, req); err != nil {
		return fmt.Errorf("calling storage deal winners api: %s", err)
	}
	return nil
}

// StorageDealProposalAccepted notifies that a proposal has been accepted by a miner.
func (c *Client) StorageDealProposalAccepted(
	ctx context.Context,
	sdID broker.StorageDealID,
	miner string,
	proposalCid cid.Cid) error {
	req := &pb.StorageDealProposalAcceptedRequest{
		StorageDealId: string(sdID),
		Miner:         miner,
		ProposalCid:   proposalCid.String(),
	}
	if _, err := c.c.StorageDealProposalAccepted(ctx, req); err != nil {
		return fmt.Errorf("calling proposal accepted deals api: %s", err)
	}
	return nil
}

// StorageDealFinalizedDeal report a finalized deal to the broker.
func (c *Client) StorageDealFinalizedDeal(ctx context.Context, fad broker.FinalizedAuctionDeal) error {
	req := &pb.StorageDealFinalizedDealRequest{
		StorageDealId:  string(fad.StorageDealID),
		MinerId:        fad.Miner,
		DealId:         fad.DealID,
		DealExpiration: fad.DealExpiration,
		ErrorCause:     fad.ErrorCause,
	}
	if _, err := c.c.StorageDealFinalizedDeal(ctx, req); err != nil {
		return fmt.Errorf("calling storage finalized deals api: %s", err)
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
