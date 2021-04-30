package cast

import (
	"fmt"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/textileio/broker-core/broker"
	pb "github.com/textileio/broker-core/gen/broker/auctioneer/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AuctionToPb returns pb.Auction from auctioneer.Auction.
func AuctionToPb(a broker.Auction) *pb.Auction {
	wbids := make([]string, len(a.WinningBids))
	for i, id := range a.WinningBids {
		wbids[i] = string(id)
	}
	pba := &pb.Auction{
		Id:            string(a.ID),
		StorageDealId: string(a.StorageDealID),
		DealSize:      a.DealSize,
		DealDuration:  a.DealDuration,
		Status:        AuctionStatusToPb(a.Status),
		Bids:          AuctionBidsToPb(a.Bids),
		WinningBids:   wbids,
		StartedAt:     timestamppb.New(a.StartedAt),
		Duration:      int64(a.Duration),
		Error:         a.Error,
	}
	return pba
}

// AuctionStatusToPb returns pb.Auction_Status from auctioneer.AuctionStatus.
func AuctionStatusToPb(s broker.AuctionStatus) pb.Auction_Status {
	switch s {
	case broker.AuctionStatusUnspecified:
		return pb.Auction_STATUS_UNSPECIFIED
	case broker.AuctionStatusQueued:
		return pb.Auction_STATUS_QUEUED
	case broker.AuctionStatusStarted:
		return pb.Auction_STATUS_STARTED
	case broker.AuctionStatusEnded:
		return pb.Auction_STATUS_ENDED
	case broker.AuctionStatusError:
		return pb.Auction_STATUS_ERROR
	default:
		return pb.Auction_STATUS_UNSPECIFIED
	}
}

// AuctionBidsToPb returns a map of pb.Auction_bid from a map of auctioneer.Bid.
func AuctionBidsToPb(bids map[broker.BidID]broker.Bid) map[string]*pb.Auction_Bid {
	pbbids := make(map[string]*pb.Auction_Bid)
	for k, v := range bids {
		pbbids[string(k)] = &pb.Auction_Bid{
			MinerId:          v.MinerID,
			MinerPeerId:      v.MinerPeerID.String(),
			AskPrice:         v.AskPrice,
			VerifiedAskPrice: v.VerifiedAskPrice,
			StartEpoch:       v.StartEpoch,
			FastRetrieval:    v.FastRetrieval,
			ReceivedAt:       timestamppb.New(v.ReceivedAt),
		}
	}
	return pbbids
}

// AuctionFromPb returns auctioneer.Auction from pb.Auction.
func AuctionFromPb(pba *pb.Auction) (broker.Auction, error) {
	bids, err := AuctionBidsFromPb(pba.Bids)
	if err != nil {
		return broker.Auction{}, fmt.Errorf("decoding bids: %v", err)
	}
	wbids := make([]broker.BidID, len(pba.WinningBids))
	for i, id := range pba.WinningBids {
		wbids[i] = broker.BidID(id)
	}
	a := broker.Auction{
		ID:            broker.AuctionID(pba.Id),
		StorageDealID: broker.StorageDealID(pba.StorageDealId),
		DealSize:      pba.DealSize,
		DealDuration:  pba.DealDuration,
		Status:        AuctionStatusFromPb(pba.Status),
		Bids:          bids,
		WinningBids:   wbids,
		StartedAt:     pba.StartedAt.AsTime(),
		Duration:      time.Duration(pba.Duration),
		Error:         pba.Error,
	}
	return a, nil
}

// AuctionStatusFromPb returns auctioneer.AuctionStatus from pb.Auction_Status.
func AuctionStatusFromPb(pbs pb.Auction_Status) broker.AuctionStatus {
	switch pbs {
	case pb.Auction_STATUS_UNSPECIFIED:
		return broker.AuctionStatusUnspecified
	case pb.Auction_STATUS_QUEUED:
		return broker.AuctionStatusQueued
	case pb.Auction_STATUS_STARTED:
		return broker.AuctionStatusStarted
	case pb.Auction_STATUS_ENDED:
		return broker.AuctionStatusEnded
	case pb.Auction_STATUS_ERROR:
		return broker.AuctionStatusError
	default:
		return broker.AuctionStatusUnspecified
	}
}

// AuctionBidsFromPb returns a map of auctioneer.Bid from a map of pb.Auction_bid.
func AuctionBidsFromPb(pbbids map[string]*pb.Auction_Bid) (map[broker.BidID]broker.Bid, error) {
	bids := make(map[broker.BidID]broker.Bid)
	for k, v := range pbbids {
		from, err := peer.Decode(v.MinerPeerId)
		if err != nil {
			return nil, fmt.Errorf("decoding peer: %v", err)
		}
		bids[broker.BidID(k)] = broker.Bid{
			MinerID:          v.MinerId,
			MinerPeerID:      from,
			AskPrice:         v.AskPrice,
			VerifiedAskPrice: v.VerifiedAskPrice,
			StartEpoch:       v.StartEpoch,
			FastRetrieval:    v.FastRetrieval,
			ReceivedAt:       v.ReceivedAt.AsTime(),
		}
	}
	return bids, nil
}
