package client

import (
	"context"
	"fmt"
	"strconv"

	"github.com/ipfs/go-cid"
	pb "github.com/textileio/broker-core/gen/broker/chainapi/v1"
	"github.com/textileio/broker-core/reporter"
	"google.golang.org/grpc"
)

// Client provides the client api.
type Client struct {
	c pb.ChainApiServiceClient
}

// New returns a new client.
func New(cc *grpc.ClientConn) *Client {
	return &Client{
		c: pb.NewChainApiServiceClient(cc),
	}
}

// ReportStorageInfo reports deal information on-chain.
func (c *Client) ReportStorageInfo(
	ctx context.Context,
	payloadCid cid.Cid,
	pieceCid cid.Cid,
	deals []reporter.DealInfo,
	dataCids []cid.Cid,
) error {
	dealInfos := make([]*pb.DealInfo, len(deals))
	for i := range deals {
		dealInfos[i] = &pb.DealInfo{
			DealId:     strconv.FormatInt(deals[i].DealID, 10),
			MinerId:    deals[i].MinerID,
			Expiration: deals[i].Expiration,
		}
	}
	dataCidsStr := make([]string, len(dataCids))
	for i := range dataCids {
		dataCidsStr[i] = dataCids[i].String()
	}
	req := pb.ReportStorageInfoRequest{
		StorageInfo: &pb.StorageInfo{
			Cid:      payloadCid.String(),
			PieceCid: pieceCid.String(),
			Deals:    dealInfos,
		},
		DataCids: dataCidsStr,
	}
	if _, err := c.c.ReportStorageInfo(ctx, &req); err != nil {
		return fmt.Errorf("call report storage info api: %s", err)
	}

	return nil
}
