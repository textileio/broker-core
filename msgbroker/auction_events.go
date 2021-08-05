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
type AuctionEventsListener interface {
	OnAuctionStarted(context.Context, time.Time, *pb.AuctionSummary) error
	OnAuctionBidReceived(context.Context, time.Time, *pb.AuctionSummary, *auctioneer.Bid) error
	OnAuctionWinnerSelected(context.Context, time.Time, *pb.AuctionSummary, *auctioneer.Bid) error
	OnAuctionWinnerAcked(context.Context, time.Time, *pb.AuctionSummary, *auctioneer.Bid) error
	OnAuctionProposalCidDelivered(ctx context.Context, ts time.Time, auctionID auction.ID,
		bidID auction.BidID, proposalCid cid.Cid, errorCause string) error
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
	bidID auction.BidID, proposalCid cid.Cid, errorCause string) error {
	return marshalAndPublish(ctx, mb, AuctionProposalCidDeliveredTopic, &pb.AuctionProposalCidDelivered{
		Ts:          timestamppb.Now(),
		AuctionId:   string(auctionID),
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
		if err := l.OnAuctionStarted(ctx, r.Ts.AsTime(), r.Auction); err != nil {
			return fmt.Errorf("calling auction-closed handler: %s", err)
		}
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
		if err := l.OnAuctionBidReceived(ctx, r.Ts.AsTime(), r.Auction, bid); err != nil {
			return fmt.Errorf("calling auction-closed handler: %s", err)
		}
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
		if err := l.OnAuctionWinnerSelected(ctx, r.Ts.AsTime(), r.Auction, bid); err != nil {
			return fmt.Errorf("calling auction-closed handler: %s", err)
		}
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
		if err := l.OnAuctionWinnerAcked(ctx, r.Ts.AsTime(), r.Auction, bid); err != nil {
			return fmt.Errorf("calling auction-closed handler: %s", err)
		}
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
		if err := l.OnAuctionProposalCidDelivered(ctx,
			r.Ts.AsTime(),
			auction.ID(r.AuctionId),
			auction.BidID(r.BidId),
			proposalCid,
			r.ErrorCause,
		); err != nil {
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
		MinerAddr:        pbid.MinerAddr,
		WalletAddrSig:    pbid.WalletAddrSig,
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
		MinerAddr:        bid.MinerAddr,
		WalletAddrSig:    bid.WalletAddrSig,
		BidderId:         peer.Encode(bid.BidderID),
		AskPrice:         bid.AskPrice,
		VerifiedAskPrice: bid.VerifiedAskPrice,
		StartEpoch:       bid.StartEpoch,
		FastRetrieval:    bid.FastRetrieval,
		ReceivedAt:       timestamppb.New(bid.ReceivedAt),
	}
}
