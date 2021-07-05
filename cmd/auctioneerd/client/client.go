package client

import (
	"context"
	"fmt"

	"github.com/ipfs/go-cid"
	"google.golang.org/grpc"

	"github.com/textileio/bidbot/lib/auction"
	core "github.com/textileio/broker-core/auctioneer"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/auctioneerd/cast"
	pb "github.com/textileio/broker-core/gen/broker/auctioneer/v1"
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
	payloadCid cid.Cid,
	dealSize, dealDuration, dealReplication int,
	dealVerified bool,
	excludedMiners []string,
	filEpochDeadline uint64,
	sources auction.Sources,
) (auction.AuctionID, error) {
	res, err := c.c.ReadyToAuction(ctx, &pb.ReadyToAuctionRequest{
		StorageDealId:    string(storageDealID),
		PayloadCid:       payloadCid.String(),
		DealSize:         uint64(dealSize),
		DealDuration:     uint64(dealDuration),
		DealReplication:  uint32(dealReplication),
		DealVerified:     dealVerified,
		ExcludedMiners:   excludedMiners,
		FilEpochDeadline: filEpochDeadline,
		Sources:          cast.SourcesToPb(sources),
	})
	if err != nil {
		return "", fmt.Errorf("calling ready to auction api: %s", err)
	}
	return auction.AuctionID(res.Id), nil
}

// ProposalAccepted signals the auctioneer that a miner has accepted a deal proposal.
func (c *Client) ProposalAccepted(
	ctx context.Context,
	auctionID auction.AuctionID,
	bidID auction.BidID,
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
