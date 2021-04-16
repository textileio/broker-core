package cast

import (
	core "github.com/textileio/broker-core/auctioneer"
	"github.com/textileio/broker-core/cmd/auctioneerd/pb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func AuctionToPb(a *core.Auction) *pb.Auction {
	pba := &pb.Auction{
		Id:        a.ID,
		Status:    AuctionStatusToPb(a.Status),
		Bids:      AuctionBidsToPb(a.Bids),
		Winner:    a.Winner,
		StartedAt: timestamppb.New(a.StartedAt),
		Duration:  a.Duration,
		Error:     a.Error,
	}
	return pba
}

func AuctionStatusToPb(s core.AuctionStatus) pb.Auction_Status {
	switch s {
	case core.AuctionStatusNew:
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

func AuctionBidsToPb(bids map[string]core.Bid) map[string]*pb.Auction_Bid {
	pbbids := make(map[string]*pb.Auction_Bid)
	for k, v := range bids {
		pbbids[k] = &pb.Auction_Bid{
			From:       v.From.String(),
			Amount:     v.Amount,
			ReceivedAt: timestamppb.New(v.ReceivedAt),
		}
	}
	return pbbids
}
