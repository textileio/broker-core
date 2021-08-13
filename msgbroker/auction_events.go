package msgbroker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/textileio/bidbot/lib/auction"
	"github.com/textileio/broker-core/auctioneer"
	pb "github.com/textileio/broker-core/gen/broker/v1"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// AuctionEventsListener is a handler for auction events related topics.
// Note that the methods do not return error, since the caller can do nothing
// if something happens on the consuming side.
type AuctionEventsListener interface {
	OnAuctionStarted(context.Context, time.Time, *pb.AuctionSummary)
	OnAuctionBidReceived(context.Context, time.Time, *pb.AuctionSummary, *auctioneer.Bid)
	OnAuctionWinnerSelected(context.Context, time.Time, *pb.AuctionSummary, *auctioneer.Bid)
	OnAuctionWinnerAcked(context.Context, time.Time, *pb.AuctionSummary, *auctioneer.Bid)
	OnAuctionProposalCidDelivered(ctx context.Context, ts time.Time, auctionID,
		bidderID, bidID, proposalCid, errorCause string)
	AuctionClosedListener
}

// PublishMsgAuctionStarted .
func PublishMsgAuctionStarted(ctx context.Context, mb MsgBroker, a *pb.AuctionSummary) error {
	return marshalAndPublish(ctx, mb, AuctionStartedTopic, &pb.AuctionStarted{
		Ts:      timestamppb.Now(),
		Auction: a,
	})
}

// PublishMsgAuctionBidReceived .
func PublishMsgAuctionBidReceived(ctx context.Context, mb MsgBroker, a *pb.AuctionSummary, b *auctioneer.Bid) error {
	return marshalAndPublish(ctx, mb, AuctionBidReceivedTopic, &pb.AuctionBidReceived{
		Ts:      timestamppb.Now(),
		Auction: a,
		Bid:     bidToPb(b),
	})
}

// PublishMsgAuctionWinnerSelected .
func PublishMsgAuctionWinnerSelected(ctx context.Context, mb MsgBroker, a *pb.AuctionSummary, b *auctioneer.Bid) error {
	return marshalAndPublish(ctx, mb, AuctionWinnerSelectedTopic, &pb.AuctionWinnerSelected{
		Ts:      timestamppb.Now(),
		Auction: a,
		Bid:     bidToPb(b),
	})
}

// PublishMsgAuctionWinnerAcked .
func PublishMsgAuctionWinnerAcked(ctx context.Context, mb MsgBroker, a *pb.AuctionSummary, b *auctioneer.Bid) error {
	return marshalAndPublish(ctx, mb, AuctionWinnerAckedTopic, &pb.AuctionWinnerAcked{
		Ts:      timestamppb.Now(),
		Auction: a,
		Bid:     bidToPb(b),
	})
}

// PublishMsgAuctionProposalCidDelivered .
func PublishMsgAuctionProposalCidDelivered(ctx context.Context, mb MsgBroker, auctionID auction.ID,
	bidderID peer.ID, bidID auction.BidID, proposalCid cid.Cid, errorCause string) error {
	return marshalAndPublish(ctx, mb, AuctionProposalCidDeliveredTopic, &pb.AuctionProposalCidDelivered{
		Ts:          timestamppb.Now(),
		AuctionId:   string(auctionID),
		BidderId:    bidderID.String(),
		BidId:       string(bidID),
		ProposalCid: proposalCid.Bytes(),
		ErrorCause:  errorCause,
	})
}

func registerAuctionEventsListener(mb MsgBroker, l AuctionEventsListener) error {
	for topic, f := range map[TopicName]TopicHandler{
		AuctionStartedTopic:              onAuctionStartedTopic(l),
		AuctionBidReceivedTopic:          onAuctionBidReceivedTopic(l),
		AuctionWinnerSelectedTopic:       onAuctionWinnerSelectedTopic(l),
		AuctionWinnerAckedTopic:          onAuctionWinnerAckedTopic(l),
		AuctionProposalCidDeliveredTopic: onAuctionProposalCidDeliveredTopic(l),
		AuctionClosedTopic:               onAuctionClosedTopic(l),
	} {
		if err := mb.RegisterTopicHandler(topic, f); err != nil {
			return err
		}
	}
	return nil
}

func onAuctionStartedTopic(l AuctionEventsListener) TopicHandler {
	return func(ctx context.Context, data []byte) error {
		r := &pb.AuctionStarted{}
		if err := proto.Unmarshal(data, r); err != nil {
			return fmt.Errorf("unmarshal new batch created: %s", err)
		}
		l.OnAuctionStarted(ctx, r.Ts.AsTime(), r.Auction)
		return nil
	}
}

func onAuctionBidReceivedTopic(l AuctionEventsListener) TopicHandler {
	return func(ctx context.Context, data []byte) error {
		r := &pb.AuctionBidReceived{}
		if err := proto.Unmarshal(data, r); err != nil {
			return fmt.Errorf("unmarshal new batch created: %s", err)
		}
		bid, err := bidFromPb(r.Bid)
		if err != nil {
			return fmt.Errorf("converting bid from pb: %v", err)
		}
		l.OnAuctionBidReceived(ctx, r.Ts.AsTime(), r.Auction, bid)
		return nil
	}
}

func onAuctionWinnerSelectedTopic(l AuctionEventsListener) TopicHandler {
	return func(ctx context.Context, data []byte) error {
		r := &pb.AuctionWinnerSelected{}
		if err := proto.Unmarshal(data, r); err != nil {
			return fmt.Errorf("unmarshal new batch created: %s", err)
		}
		bid, err := bidFromPb(r.Bid)
		if err != nil {
			return fmt.Errorf("converting bid from pb: %v", err)
		}
		l.OnAuctionWinnerSelected(ctx, r.Ts.AsTime(), r.Auction, bid)
		return nil
	}
}

func onAuctionWinnerAckedTopic(l AuctionEventsListener) TopicHandler {
	return func(ctx context.Context, data []byte) error {
		r := &pb.AuctionWinnerAcked{}
		if err := proto.Unmarshal(data, r); err != nil {
			return fmt.Errorf("unmarshal new batch created: %s", err)
		}
		bid, err := bidFromPb(r.Bid)
		if err != nil {
			return fmt.Errorf("converting bid from pb: %v", err)
		}
		l.OnAuctionWinnerAcked(ctx, r.Ts.AsTime(), r.Auction, bid)
		return nil
	}
}

func onAuctionProposalCidDeliveredTopic(l AuctionEventsListener) TopicHandler {
	return func(ctx context.Context, data []byte) error {
		r := &pb.AuctionProposalCidDelivered{}
		if err := proto.Unmarshal(data, r); err != nil {
			return fmt.Errorf("unmarshal new batch created: %s", err)
		}
		proposalCid, err := cid.Cast(r.ProposalCid)
		if err != nil {
			return errors.New("proposal cid invalid")
		}
		l.OnAuctionProposalCidDelivered(ctx,
			r.Ts.AsTime(),
			r.AuctionId,
			r.BidderId,
			r.BidId,
			proposalCid.String(),
			r.ErrorCause,
		)
		return nil
	}
}

func onAuctionClosedTopic(l AuctionClosedListener) TopicHandler {
	return func(ctx context.Context, data []byte) error {
		r := &pb.AuctionClosed{}
		if err := proto.Unmarshal(data, r); err != nil {
			return fmt.Errorf("unmarshal auction closed: %s", err)
		}
		if r.OperationId == "" {
			return errors.New("operation-id is empty")
		}
		auction, err := closedAuctionFromPb(r)
		if err != nil {
			return fmt.Errorf("invalid auction closed: %s", err)
		}
		if err := l.OnAuctionClosed(ctx, OperationID(r.OperationId), auction); err != nil {
			return fmt.Errorf("calling auction-closed handler: %s", err)
		}
		return nil
	}
}

// AuctionToPbSummary converts auctioneer.Auction to pb.
func AuctionToPbSummary(a *auctioneer.Auction) *pb.AuctionSummary {
	return &pb.AuctionSummary{
		Id:                       string(a.ID),
		BatchId:                  string(a.BatchID),
		DealVerified:             a.DealVerified,
		ExcludedStorageProviders: a.ExcludedStorageProviders,
		Status:                   a.Status.String(),
		StartedAt:                timestamppb.New(a.StartedAt),
		UpdatedAt:                timestamppb.New(a.UpdatedAt),
		Duration:                 uint64(a.Duration),
	}
}

func bidFromPb(pbid *pb.Bid) (*auctioneer.Bid, error) {
	bidderID, err := peer.Decode(pbid.BidderId)
	if err != nil {
		return nil, fmt.Errorf("invalid bidder ID: %v", err)
	}
	return &auctioneer.Bid{
		MinerAddr:        pbid.StorageProviderId,
		BidderID:         bidderID,
		AskPrice:         pbid.AskPrice,
		VerifiedAskPrice: pbid.VerifiedAskPrice,
		StartEpoch:       pbid.StartEpoch,
		FastRetrieval:    pbid.FastRetrieval,
		ReceivedAt:       pbid.ReceivedAt.AsTime(),
	}, nil
}

func bidToPb(bid *auctioneer.Bid) *pb.Bid {
	return &pb.Bid{
		StorageProviderId: bid.MinerAddr,
		BidderId:          peer.Encode(bid.BidderID),
		AskPrice:          bid.AskPrice,
		VerifiedAskPrice:  bid.VerifiedAskPrice,
		StartEpoch:        bid.StartEpoch,
		FastRetrieval:     bid.FastRetrieval,
		ReceivedAt:        timestamppb.New(bid.ReceivedAt),
	}
}
