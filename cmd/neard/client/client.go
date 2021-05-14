package client

import (
	"context"
	"fmt"

	"github.com/ipfs/go-cid"
	"github.com/textileio/broker-core/chainapi"
	pb "github.com/textileio/broker-core/gen/broker/chainapi/v1"
	"google.golang.org/grpc"
)

// Client provides the client api.
type Client struct {
	c pb.ChainApiServiceClient
}

// New returns a new client.
func New(cc *grpc.ClientConn) chainapi.ChainAPI {
	return &Client{
		c: pb.NewChainApiServiceClient(cc),
	}
}

// HasDeposit checks if an account has deposited funds for a broker.
func (c *Client) HasDeposit(ctx context.Context, brokerID, accountID string) (bool, error) {
	req := &pb.HasDepositRequest{
		BrokerId:  brokerID,
		AccountId: accountID,
	}
	res, err := c.c.HasDeposit(ctx, req)
	if err != nil {
		return false, fmt.Errorf("calling has deposit api: %v", err)
	}
	return res.HasDeposit, nil
}

// UpdatePayload creates or updates a storage payload.
func (c *Client) UpdatePayload(ctx context.Context, payloadCid cid.Cid, opts ...chainapi.UpdatePayloadOption) error {
	options := &chainapi.UpdatePayloadOptions{}
	for _, opt := range opts {
		opt(options)
	}
	dealInfos := make([]*pb.DealInfo, len(options.Deals))
	for i := range options.Deals {
		dealInfos[i] = &pb.DealInfo{
			DealId:     options.Deals[i].DealID,
			MinerId:    options.Deals[i].MinerID,
			Expiration: options.Deals[i].Expiration,
		}
	}
	dataCidsStr := make([]string, len(options.DataCids))
	for i := range options.DataCids {
		dataCidsStr[i] = options.DataCids[i].String()
	}
	pbOptions := &pb.PayloadOptions{
		Deals:    dealInfos,
		DataCids: dataCidsStr,
	}
	if options.PieceCid != nil {
		pbOptions.PieceCid = options.PieceCid.String()
	}
	req := &pb.UpdatePayloadRequest{
		PayloadCid: payloadCid.String(),
		Options:    pbOptions,
	}
	if _, err := c.c.UpdatePayload(ctx, req); err != nil {
		return fmt.Errorf("call update payload api: %s", err)
	}
	return nil
}
