package cast

import (
	core "github.com/textileio/broker-core/auctioneer"
	pb "github.com/textileio/broker-core/gen/broker/auctioneer/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AuctionToPb returns pb.Auction from auctioneer.Auction.
func AuctionToPb(a *core.Auction) *pb.Auction {
	pba := &pb.Auction{
		Id:           a.ID,
		DealId:       a.DealID,
		DealSize:     a.DealSize,
		DealDuration: a.DealDuration,
		Status:       AuctionStatusToPb(a.Status),
		Bids:         AuctionBidsToPb(a.Bids),
		WinningBid:   a.WinningBid,
		StartedAt:    timestamppb.New(a.StartedAt),
		Duration:     int64(a.Duration),
		Error:        a.Error,
	}
	return pba
}

// AuctionStatusToPb returns pb.Auction_Status from auctioneer.AuctionStatus.
func AuctionStatusToPb(s core.AuctionStatus) pb.Auction_Status {
	switch s {
	case core.AuctionStatusUnspecified:
		return pb.Auction_STATUS_UNSPECIFIED
	case core.AuctionStatusQueued:
		return pb.Auction_STATUS_QUEUED
	case core.AuctionStatusStarted:
		return pb.Auction_STATUS_STARTED
	case core.AuctionStatusEnded:
		return pb.Auction_STATUS_ENDED
	case core.AuctionStatusError:
		return pb.Auction_STATUS_ERROR
	default:
		return pb.Auction_STATUS_UNSPECIFIED
	}
}

// AuctionBidsToPb returns a map of pb.Auction_bid from a map of auctioneer.Bid.
func AuctionBidsToPb(bids map[string]core.Bid) map[string]*pb.Auction_Bid {
	pbbids := make(map[string]*pb.Auction_Bid)
	for k, v := range bids {
		pbbids[k] = &pb.Auction_Bid{
			From:       v.From.String(),
			AttoFil:    v.AttoFil,
			ReceivedAt: timestamppb.New(v.ReceivedAt),
		}
	}
	return pbbids
}
