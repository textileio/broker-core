package client

import (
	"context"

	core "github.com/textileio/broker-core/auctioneer"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/auctioneerd/cast"
	pb "github.com/textileio/broker-core/gen/broker/auctioneer/v1"
	"google.golang.org/grpc"
)

// Client provides the client api.
type Client struct {
	c pb.APIServiceClient
}

var _ core.Auctioneer = (*Client)(nil)

// New returns a new client.
func New(cc *grpc.ClientConn) *Client {
	return &Client{
		c: pb.NewAPIServiceClient(cc),
	}
}

// ReadyToAuction creates an auction.
func (c *Client) ReadyToAuction(
	ctx context.Context,
	storageDealID broker.StorageDealID,
	dealSize, dealDuration uint64,
	dealReplication uint32,
	dealVerified bool,
) (broker.AuctionID, error) {
	res, err := c.c.ReadyToAuction(ctx, &pb.ReadyToAuctionRequest{
		StorageDealId:   string(storageDealID),
		DealSize:        dealSize,
		DealDuration:    dealDuration,
		DealReplication: dealReplication,
		DealVerified:    dealVerified,
	})
	if err != nil {
		return "", err
	}
	return broker.AuctionID(res.Id), nil
}

// GetAuction returns an auction by id.
func (c *Client) GetAuction(ctx context.Context, id broker.AuctionID) (broker.Auction, error) {
	res, err := c.c.GetAuction(ctx, &pb.GetAuctionRequest{
		Id: string(id),
	})
	if err != nil {
		return broker.Auction{}, err
	}
	return cast.AuctionFromPb(res.Auction)
}
