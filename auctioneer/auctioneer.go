package auctioneer

import (
	"context"
	"path"
	"time"

	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/textileio/broker-core/broker"
)

// AuctionTopic is used by brokers to publish and by miners to subscribe to deal auctions.
const AuctionTopic string = "/textile/auction/0.0.1"

// BidsTopic is used by miners to submit deal auction bids.
// "/textile/auction/0.0.1/<auction_id>/bids".
func BidsTopic(auctionID string) string {
	return path.Join(AuctionTopic, auctionID, "bids")
}

// WinsTopic is used by brokers to notify a miner thay have won the deal auction.
// "/textile/auction/0.0.1/<peer_id>/wins".
func WinsTopic(pid peer.ID) string {
	return path.Join(AuctionTopic, pid.String(), "wins")
}

// Auction defines the core auction model.
type Auction struct {
	ID           string
	DealID       string
	DealSize     uint64
	DealDuration uint64
	Status       AuctionStatus
	Bids         map[string]Bid
	WinningBid   string
	StartedAt    time.Time
	Duration     time.Duration
	Error        string
}

// AuctionStatus is the status of an auction.
type AuctionStatus int

const (
	// AuctionStatusUnspecified indicates the initial or invalid status of an auction.
	AuctionStatusUnspecified AuctionStatus = iota
	// AuctionStatusQueued indicates the auction is currently queued.
	AuctionStatusQueued
	// AuctionStatusStarted indicates the auction has started.
	AuctionStatusStarted
	// AuctionStatusEnded indicates the auction has ended.
	AuctionStatusEnded
	// AuctionStatusError indicates the auction resulted in an error.
	AuctionStatusError
)

// String returns a string-encoded status.
func (as AuctionStatus) String() string {
	switch as {
	case AuctionStatusUnspecified:
		return "unspecified"
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

// Bid defines the core bid model.
type Bid struct {
	Broker     peer.ID
	From       peer.ID
	NanoFil    int64
	ReceivedAt time.Time
}

// Auctioneer creates auctions and decides on winning bids.
type Auctioneer interface {
	// ReadyToAuction signals the auctioneer that this storage deal is ready to be included in an auction.
	// dealDuration is in units of Filecoin epochs (~30s).
	ReadyToAuction(ctx context.Context, id broker.StorageDealID, dealSize, dealDuration uint64) error
}
