package cast

import (
	"fmt"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/textileio/broker-core/broker"
	pb "github.com/textileio/broker-core/gen/broker/auctioneer/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AuctionToPb returns pb.Auction from broker.Auction.
func AuctionToPb(a broker.Auction) *pb.Auction {
	pba := &pb.Auction{
		Id:              string(a.ID),
		StorageDealId:   string(a.StorageDealID),
		DataUri:         a.DataUri,
		DealSize:        a.DealSize,
		DealDuration:    a.DealDuration,
		DealReplication: a.DealReplication,
		DealVerified:    a.DealVerified,
		Status:          AuctionStatusToPb(a.Status),
		Bids:            AuctionBidsToPb(a.Bids),
		WinningBids:     AuctionWinningBidsToPb(a.WinningBids),
		StartedAt:       timestamppb.New(a.StartedAt),
		UpdatedAt:       timestamppb.New(a.UpdatedAt),
		Duration:        int64(a.Duration),
		Attempts:        a.Attempts,
		Error:           a.ErrorCause,
	}
	return pba
}

// AuctionStatusToPb returns pb.Auction_Status from broker.AuctionStatus.
func AuctionStatusToPb(s broker.AuctionStatus) pb.Auction_Status {
	switch s {
	case broker.AuctionStatusUnspecified:
		return pb.Auction_STATUS_UNSPECIFIED
	case broker.AuctionStatusQueued:
		return pb.Auction_STATUS_QUEUED
	case broker.AuctionStatusStarted:
		return pb.Auction_STATUS_STARTED
	case broker.AuctionStatusFinalized:
		return pb.Auction_STATUS_FINALIZED
	default:
		return pb.Auction_STATUS_UNSPECIFIED
	}
}

// AuctionBidsToPb returns a map of pb.Auction_Bid from a map of broker.Bid.
func AuctionBidsToPb(bids map[broker.BidID]broker.Bid) map[string]*pb.Auction_Bid {
	pbbids := make(map[string]*pb.Auction_Bid)
	for k, v := range bids {
		pbbids[string(k)] = &pb.Auction_Bid{
			MinerAddr:        v.MinerAddr,
			WalletAddrSig:    v.WalletAddrSig,
			BidderId:         v.BidderID.String(),
			AskPrice:         v.AskPrice,
			VerifiedAskPrice: v.VerifiedAskPrice,
			StartEpoch:       v.StartEpoch,
			FastRetrieval:    v.FastRetrieval,
			ReceivedAt:       timestamppb.New(v.ReceivedAt),
		}
	}
	return pbbids
}

// AuctionWinningBidsToPb returns a map of pb.Auction_WinningBid from a map of broker.WinningBid.
func AuctionWinningBidsToPb(bids map[broker.BidID]broker.WinningBid) map[string]*pb.Auction_WinningBid {
	pbbids := make(map[string]*pb.Auction_WinningBid)
	for k, v := range bids {
		var pcid string
		if v.ProposalCid.Defined() {
			pcid = v.ProposalCid.String()
		}
		pbbids[string(k)] = &pb.Auction_WinningBid{
			BidderId:                v.BidderID.String(),
			Acknowledged:            v.Acknowledged,
			ProposalCid:             pcid,
			ProposalCidAcknowledged: v.ProposalCidAcknowledged,
		}
	}
	return pbbids
}

// AuctionFromPb returns broker.Auction from pb.Auction.
func AuctionFromPb(pba *pb.Auction) (broker.Auction, error) {
	bids, err := AuctionBidsFromPb(pba.Bids)
	if err != nil {
		return broker.Auction{}, fmt.Errorf("decoding bids: %v", err)
	}
	wbids, err := AuctionWinningBidsFromPb(pba.WinningBids)
	if err != nil {
		return broker.Auction{}, fmt.Errorf("decoding bids: %v", err)
	}
	a := broker.Auction{
		ID:              broker.AuctionID(pba.Id),
		StorageDealID:   broker.StorageDealID(pba.StorageDealId),
		DataUri:         pba.DataUri,
		DealSize:        pba.DealSize,
		DealDuration:    pba.DealDuration,
		DealReplication: pba.DealReplication,
		DealVerified:    pba.DealVerified,
		Status:          AuctionStatusFromPb(pba.Status),
		Bids:            bids,
		WinningBids:     wbids,
		StartedAt:       pba.StartedAt.AsTime(),
		UpdatedAt:       pba.UpdatedAt.AsTime(),
		Duration:        time.Duration(pba.Duration),
		Attempts:        pba.Attempts,
		ErrorCause:      pba.Error,
	}
	return a, nil
}

// AuctionStatusFromPb returns broker.AuctionStatus from pb.Auction_Status.
func AuctionStatusFromPb(pbs pb.Auction_Status) broker.AuctionStatus {
	switch pbs {
	case pb.Auction_STATUS_UNSPECIFIED:
		return broker.AuctionStatusUnspecified
	case pb.Auction_STATUS_QUEUED:
		return broker.AuctionStatusQueued
	case pb.Auction_STATUS_STARTED:
		return broker.AuctionStatusStarted
	case pb.Auction_STATUS_FINALIZED:
		return broker.AuctionStatusFinalized
	default:
		return broker.AuctionStatusUnspecified
	}
}

// AuctionBidsFromPb returns a map of broker.Bid from a map of pb.Auction_Bid.
func AuctionBidsFromPb(pbbids map[string]*pb.Auction_Bid) (map[broker.BidID]broker.Bid, error) {
	bids := make(map[broker.BidID]broker.Bid)
	for k, v := range pbbids {
		bidder, err := peer.Decode(v.BidderId)
		if err != nil {
			return nil, fmt.Errorf("decoding bidder: %v", err)
		}
		bids[broker.BidID(k)] = broker.Bid{
			MinerAddr:        v.MinerAddr,
			WalletAddrSig:    v.WalletAddrSig,
			BidderID:         bidder,
			AskPrice:         v.AskPrice,
			VerifiedAskPrice: v.VerifiedAskPrice,
			StartEpoch:       v.StartEpoch,
			FastRetrieval:    v.FastRetrieval,
			ReceivedAt:       v.ReceivedAt.AsTime(),
		}
	}
	return bids, nil
}

// AuctionWinningBidsFromPb returns a map of broker.WinningBid from a map of pb.Auction_WinningBid.
func AuctionWinningBidsFromPb(pbbids map[string]*pb.Auction_WinningBid) (map[broker.BidID]broker.WinningBid, error) {
	wbids := make(map[broker.BidID]broker.WinningBid)
	for k, v := range pbbids {
		bidder, err := peer.Decode(v.BidderId)
		if err != nil {
			return nil, fmt.Errorf("decoding bidder id: %v", err)
		}
		pcid := cid.Undef
		if v.ProposalCid != "" {
			var err error
			pcid, err = cid.Decode(v.ProposalCid)
			if err != nil {
				return nil, fmt.Errorf("decoding proposal cid: %v", err)
			}
		}
		wbids[broker.BidID(k)] = broker.WinningBid{
			BidderID:                bidder,
			Acknowledged:            v.Acknowledged,
			ProposalCid:             pcid,
			ProposalCidAcknowledged: v.ProposalCidAcknowledged,
		}
	}
	return wbids, nil
}
