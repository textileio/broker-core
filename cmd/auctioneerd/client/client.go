package client

import (
	"context"
	"fmt"

	"github.com/ipfs/go-cid"
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
	dataURI string,
	dealSize, dealDuration, dealReplication int,
	dealVerified bool,
	excludedMiners []string,
) (broker.AuctionID, error) {
	res, err := c.c.ReadyToAuction(ctx, &pb.ReadyToAuctionRequest{
		StorageDealId:   string(storageDealID),
		DataUri:         dataURI,
		DealSize:        uint64(dealSize),
		DealDuration:    uint64(dealDuration),
		DealReplication: uint32(dealReplication),
		DealVerified:    dealVerified,
		ExcludedMiners:  excludedMiners,
	})
	if err != nil {
		return "", fmt.Errorf("calling ready to auction api: %s", err)
	}
	return broker.AuctionID(res.Id), nil
}

// GetAuction returns an auction by id.
func (c *Client) GetAuction(ctx context.Context, auctionID broker.AuctionID) (broker.Auction, error) {
	res, err := c.c.GetAuction(ctx, &pb.GetAuctionRequest{
		Id: string(auctionID),
	})
	if err != nil {
		return broker.Auction{}, err
	}
	return cast.AuctionFromPb(res.Auction)
}

// ProposalAccepted signals the auctioneer that a miner has accepted a deal proposal.
func (c *Client) ProposalAccepted(
	ctx context.Context,
	auctionID broker.AuctionID,
	bidID broker.BidID,
	proposalCid cid.Cid,
) error {
	_, err := c.c.ProposalAccepted(ctx, &pb.ProposalAcceptedRequest{
		AuctionId:   string(auctionID),
		BidId:       string(bidID),
		ProposalCid: proposalCid.String(),
	})
	if err != nil {
		return fmt.Errorf("calling proposal accepted api: %s", err)
	}
	return nil
}
