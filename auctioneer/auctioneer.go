package auctioneer

import (
	"context"
	"path"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/textileio/broker-core/broker"
)

const AuctionTopic string = "/textile/auction/0.0.1"

// BidsTopic is used by miners to submit deal auction bids.
// "/textile/auction/0.0.1/<auction_id>/bids"
func BidsTopic(auctionID string) string {
	return path.Join(AuctionTopic, auctionID, "bids")
}

// WinsTopic is used by brokers to notify a miner thay have won the deal auction.
// "/textile/auction/0.0.1/<peer_id>/wins"
func WinsTopic(pid peer.ID) string {
	return path.Join(AuctionTopic, pid.String(), "wins")
}

type Auction struct {
	ID        string
	Status    AuctionStatus
	Bids      map[string]Bid
	Winner    string
	StartedAt time.Time
	Duration  int64
	Error     string
}

type AuctionStatus int

const (
	AuctionStatusNew AuctionStatus = iota
	AuctionStatusQueued
	AuctionStatusStarted
	AuctionStatusEnded
	AuctionStatusError
)

func (as AuctionStatus) String() string {
	switch as {
	case AuctionStatusNew:
		return "new"
	case AuctionStatusQueued:
		return "queued"
	case AuctionStatusStarted:
		return "started"
	case AuctionStatusEnded:
		return "ended"
	case AuctionStatusError:
		return "error"
	default:
		return "invalid"
	}
}

type Bid struct {
	From       peer.ID
	Amount     int64
	ReceivedAt time.Time
}

type Auctioneer interface {
	ReadyToAuction(ctx context.Context, sd broker.StorageDeal) error
}
