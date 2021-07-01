package cast

import (
	"fmt"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/textileio/bidbot/lib/broker"
	pb "github.com/textileio/broker-core/gen/broker/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// FromProtoBrokerRequest transforms a pb.BrokerRequest to broker.BrokerRequest.
func FromProtoBrokerRequest(brproto *pb.BrokerRequest) (broker.BrokerRequest, error) {
	c, err := cid.Decode(brproto.DataCid)
	if err != nil {
		return broker.BrokerRequest{}, fmt.Errorf("decoding cid: %s", err)
	}

	var status broker.BrokerRequestStatus
	switch brproto.Status {
	case pb.BrokerRequest_UNSPECIFIED:
		status = broker.RequestUnknown
	case pb.BrokerRequest_BATCHING:
		status = broker.RequestBatching
	case pb.BrokerRequest_PREPARING:
		status = broker.RequestPreparing
	case pb.BrokerRequest_AUCTIONING:
		status = broker.RequestAuctioning
	case pb.BrokerRequest_DEALMAKING:
		status = broker.RequestDealMaking
	case pb.BrokerRequest_SUCCESS:
		status = broker.RequestSuccess
	default:
		return broker.BrokerRequest{}, fmt.Errorf("unknown status: %s", brproto.Status)
	}

	return broker.BrokerRequest{
		ID:            broker.BrokerRequestID(brproto.Id),
		DataCid:       c,
		Status:        status,
		StorageDealID: broker.StorageDealID(brproto.StorageDealId),
		CreatedAt:     brproto.CreatedAt.AsTime(),
		UpdatedAt:     brproto.UpdatedAt.AsTime(),
	}, nil
}

// FromProtoBrokerRequestInfo transforms a pb.BrokerRequest to broker.BrokerRequest.
func FromProtoBrokerRequestInfo(brproto *pb.GetBrokerRequestInfoResponse) (broker.BrokerRequestInfo, error) {
	br, err := FromProtoBrokerRequest(brproto.BrokerRequest)
	if err != nil {
		return broker.BrokerRequestInfo{}, nil
	}

	bri := broker.BrokerRequestInfo{
		BrokerRequest: br,
		Deals:         make([]broker.BrokerRequestDeal, len(brproto.Deals)),
	}
	for i, d := range brproto.Deals {
		deal := broker.BrokerRequestDeal{
			Miner:      d.Miner,
			DealID:     d.DealId,
			Expiration: d.Expiration,
		}
		bri.Deals[i] = deal
	}

	return bri, nil
}

// BrokerRequestToProto maps a broker.BrokerRequest to pb.BrokerRequest.
func BrokerRequestToProto(br broker.BrokerRequest) (*pb.BrokerRequest, error) {
	var pbStatus pb.BrokerRequest_Status
	switch br.Status {
	case broker.RequestUnknown:
		pbStatus = pb.BrokerRequest_UNSPECIFIED
	case broker.RequestBatching:
		pbStatus = pb.BrokerRequest_BATCHING
	case broker.RequestPreparing:
		pbStatus = pb.BrokerRequest_PREPARING
	case broker.RequestAuctioning:
		pbStatus = pb.BrokerRequest_AUCTIONING
	case broker.RequestDealMaking:
		pbStatus = pb.BrokerRequest_DEALMAKING
	case broker.RequestSuccess:
		pbStatus = pb.BrokerRequest_SUCCESS
	default:
		return nil, fmt.Errorf("unknown status: %d", br.Status)
	}

	return &pb.BrokerRequest{
		Id:            string(br.ID),
		DataCid:       br.DataCid.String(),
		Status:        pbStatus,
		StorageDealId: string(br.StorageDealID),
		CreatedAt:     timestamppb.New(br.CreatedAt),
		UpdatedAt:     timestamppb.New(br.UpdatedAt),
	}, nil
}

// BrokerRequestInfoToProto maps a broker.
func BrokerRequestInfoToProto(br broker.BrokerRequestInfo) (*pb.GetBrokerRequestInfoResponse, error) {
	protobr, err := BrokerRequestToProto(br.BrokerRequest)
	if err != nil {
		return nil, fmt.Errorf("creating proto for broker request: %s", err)
	}

	deals := make([]*pb.GetBrokerRequestInfoResponse_BrokerRequestDeal, len(br.Deals))
	for i, d := range br.Deals {
		deal := &pb.GetBrokerRequestInfoResponse_BrokerRequestDeal{
			Miner:      d.Miner,
			DealId:     d.DealID,
			Expiration: d.Expiration,
		}
		deals[i] = deal
	}

	return &pb.GetBrokerRequestInfoResponse{
		BrokerRequest: protobr,
		Deals:         deals,
	}, nil
}

// AuctionToPb returns pb.Auction from broker.Auction.
func AuctionToPb(a broker.Auction) *pb.StorageDealAuctionedRequest {
	pba := &pb.StorageDealAuctionedRequest{
		Id:              string(a.ID),
		StorageDealId:   string(a.StorageDealID),
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
func AuctionStatusToPb(s broker.AuctionStatus) pb.StorageDealAuctionedRequest_Status {
	switch s {
	case broker.AuctionStatusUnspecified:
		return pb.StorageDealAuctionedRequest_STATUS_UNSPECIFIED
	case broker.AuctionStatusQueued:
		return pb.StorageDealAuctionedRequest_STATUS_QUEUED
	case broker.AuctionStatusStarted:
		return pb.StorageDealAuctionedRequest_STATUS_STARTED
	case broker.AuctionStatusFinalized:
		return pb.StorageDealAuctionedRequest_STATUS_FINALIZED
	default:
		return pb.StorageDealAuctionedRequest_STATUS_UNSPECIFIED
	}
}

// AuctionBidsToPb returns a map of pb.Auction_Bid from a map of broker.Bid.
func AuctionBidsToPb(bids map[broker.BidID]broker.Bid) map[string]*pb.StorageDealAuctionedRequest_Bid {
	pbbids := make(map[string]*pb.StorageDealAuctionedRequest_Bid)
	for k, v := range bids {
		pbbids[string(k)] = &pb.StorageDealAuctionedRequest_Bid{
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

// AuctionWinningBidsToPb returns a map of pb.StorageDealAuctionedRequest_WinningBid from a map of broker.WinningBid.
func AuctionWinningBidsToPb(
	bids map[broker.BidID]broker.WinningBid,
) map[string]*pb.StorageDealAuctionedRequest_WinningBid {
	pbbids := make(map[string]*pb.StorageDealAuctionedRequest_WinningBid)
	for k, v := range bids {
		var pcid string
		if v.ProposalCid.Defined() {
			pcid = v.ProposalCid.String()
		}
		pbbids[string(k)] = &pb.StorageDealAuctionedRequest_WinningBid{
			BidderId:                v.BidderID.String(),
			Acknowledged:            v.Acknowledged,
			ProposalCid:             pcid,
			ProposalCidAcknowledged: v.ProposalCidAcknowledged,
		}
	}
	return pbbids
}

// AuctionFromPb returns broker.Auction from pb.Auction.
func AuctionFromPb(pba *pb.StorageDealAuctionedRequest) (broker.Auction, error) {
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

// AuctionStatusFromPb returns broker.AuctionStatus from pb.StorageDealAuctionedRequest_Status.
func AuctionStatusFromPb(pbs pb.StorageDealAuctionedRequest_Status) broker.AuctionStatus {
	switch pbs {
	case pb.StorageDealAuctionedRequest_STATUS_UNSPECIFIED:
		return broker.AuctionStatusUnspecified
	case pb.StorageDealAuctionedRequest_STATUS_QUEUED:
		return broker.AuctionStatusQueued
	case pb.StorageDealAuctionedRequest_STATUS_STARTED:
		return broker.AuctionStatusStarted
	case pb.StorageDealAuctionedRequest_STATUS_FINALIZED:
		return broker.AuctionStatusFinalized
	default:
		return broker.AuctionStatusUnspecified
	}
}

// AuctionBidsFromPb returns a map of broker.Bid from a map of pb.StorageDealAuctionedRequest_Bid.
func AuctionBidsFromPb(pbbids map[string]*pb.StorageDealAuctionedRequest_Bid) (map[broker.BidID]broker.Bid, error) {
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

// AuctionWinningBidsFromPb returns a map of broker.WinningBid from a map of pb.StorageDealAuctionedRequest_WinningBid.
func AuctionWinningBidsFromPb(
	pbbids map[string]*pb.StorageDealAuctionedRequest_WinningBid,
) (map[broker.BidID]broker.WinningBid, error) {
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
