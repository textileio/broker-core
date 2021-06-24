package auctioneer

import (
	"container/heap"
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/hashicorp/go-multierror"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/textileio/broker-core/broker"
	core "github.com/textileio/broker-core/broker"
	q "github.com/textileio/broker-core/cmd/auctioneerd/auctioneer/queue"
	"github.com/textileio/broker-core/cmd/auctioneerd/cast"
	"github.com/textileio/broker-core/dshelper/txndswrap"
	"github.com/textileio/broker-core/finalizer"
	pb "github.com/textileio/broker-core/gen/broker/auctioneer/v1/message"
	"github.com/textileio/broker-core/marketpeer"
	"github.com/textileio/broker-core/metrics"
	"github.com/textileio/broker-core/pubsub"
	golog "github.com/textileio/go-log/v2"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	log = golog.Logger("auctioneer")

	// maxAuctionDuration is the max duration an auction can run for.
	maxAuctionDuration = time.Minute * 10

	// notifyTimeout is the max duration the auctioneer will wait for a response from bidders.
	notifyTimeout = time.Second * 10

	// ErrAuctionNotFound indicates the requested auction was not found.
	ErrAuctionNotFound = errors.New("auction not found")

	// ErrBidNotFound indicates the requested bid was not found.
	ErrBidNotFound = errors.New("bid not found")

	// ErrInsufficientBids indicates the auction failed due to insufficient bids.
	ErrInsufficientBids = errors.New("auction failed; insufficient bids")
)

// AuctionConfig defines auction params.
type AuctionConfig struct {
	// Duration auctions will be held for.
	Duration time.Duration

	// Attempts that an auction will run before signaling to the broker that it failed.
	// Auctions will continue to run under the following conditions:
	// 1. While deal replication is greater than the number of winning bids.
	// 2. While at least one owner of a winning bid is unreachable when notifying they won or delivery proposal cid.
	// 3. While signaling the broker results in an error.
	Attempts uint32
}

// Auctioneer handles deal auctions for a broker.
type Auctioneer struct {
	queue       *q.Queue
	started     bool
	auctionConf AuctionConfig

	peer     *marketpeer.Peer
	fc       FilClient
	auctions *pubsub.Topic

	broker core.Broker

	finalizer *finalizer.Finalizer
	lk        sync.Mutex

	statLastCreatedAuction    time.Time
	metricNewAuction          metric.Int64Counter
	metricNewFinalizedAuction metric.Int64Counter
	metricNewBid              metric.Int64Counter
	metricAcceptedBid         metric.Int64Counter
	metricLastCreatedAuction  metric.Int64ValueObserver
}

// New returns a new Auctioneer.
func New(
	peer *marketpeer.Peer,
	store txndswrap.TxnDatastore,
	broker core.Broker,
	fc FilClient,
	auctionConf AuctionConfig,
) (*Auctioneer, error) {
	if err := validateConfig(auctionConf); err != nil {
		return nil, fmt.Errorf("validating config: %v", err)
	}

	a := &Auctioneer{
		peer:        peer,
		fc:          fc,
		broker:      broker,
		auctionConf: auctionConf,
		finalizer:   finalizer.NewFinalizer(),
	}
	a.initMetrics()

	queue := q.NewQueue(store, a.processAuction, a.finalizeAuction, auctionConf.Attempts)
	a.finalizer.Add(queue)
	a.queue = queue

	return a, nil
}

func validateConfig(c AuctionConfig) error {
	if c.Duration <= 0 {
		return fmt.Errorf("duration must be greater than zero")
	} else if c.Duration > maxAuctionDuration {
		return fmt.Errorf("duration must be less than or equal to %v", maxAuctionDuration)
	}
	if c.Attempts == 0 {
		return fmt.Errorf("max attempts must be greater than zero")
	}
	return nil
}

// Close the auctioneer.
func (a *Auctioneer) Close() error {
	log.Info("closing auctioneer...")
	return a.finalizer.Cleanup(nil)
}

// Start the deal auction feed.
// If bootstrap is true, the peer will dial the configured bootstrap addresses
// before creating the deal auction feed.
func (a *Auctioneer) Start(bootstrap bool) error {
	a.lk.Lock()
	defer a.lk.Unlock()
	if a.started {
		return nil
	}

	// Bootstrap against configured addresses
	if bootstrap {
		a.peer.Bootstrap()
	}

	ctx, cancel := context.WithCancel(context.Background())
	a.finalizer.Add(finalizer.NewContextCloser(cancel))

	// Create the global auctions topic
	auctions, err := a.peer.NewTopic(ctx, core.AuctionTopic, false)
	if err != nil {
		return fmt.Errorf("creating auctions topic: %v", err)
	}
	auctions.SetEventHandler(a.eventHandler)
	a.auctions = auctions
	a.finalizer.Add(auctions)

	log.Info("created the deal auction feed")

	a.started = true
	return nil
}

// CreateAuction creates a new auction.
// New auctions are queud if the auctioneer is busy.
func (a *Auctioneer) CreateAuction(auction core.Auction) (core.AuctionID, error) {
	auction.Status = broker.AuctionStatusUnspecified
	auction.Duration = a.auctionConf.Duration
	id, err := a.queue.CreateAuction(auction)
	if err != nil {
		return "", fmt.Errorf("creating auction: %v", err)
	}

	log.Debugf("created auction %s", id)
	a.metricNewAuction.Add(context.Background(), 1)
	a.statLastCreatedAuction = time.Now()

	return id, nil
}

// GetAuction returns an auction by id.
// If an auction is not found for id, ErrAuctionNotFound is returned.
func (a *Auctioneer) GetAuction(id core.AuctionID) (*core.Auction, error) {
	auction, err := a.queue.GetAuction(id)
	if errors.Is(q.ErrAuctionNotFound, err) {
		return nil, ErrAuctionNotFound
	} else if err != nil {
		return nil, fmt.Errorf("getting auction: %v", err)
	}
	return auction, nil
}

// DeliverProposal delivers the proposal Cid for an accepted deal to the winning bidder.
// If an auction is not found for id, ErrAuctionNotFound is returned.
// If a bid is not found for id, ErrBidNotFound is returned.
func (a *Auctioneer) DeliverProposal(id broker.AuctionID, bid broker.BidID, pcid cid.Cid) error {
	if err := a.queue.SetWinningBidProposalCid(id, bid, pcid); errors.Is(q.ErrAuctionNotFound, err) {
		return ErrAuctionNotFound
	} else if errors.Is(q.ErrBidNotFound, err) {
		return ErrBidNotFound
	} else if err != nil {
		return fmt.Errorf("setting winning bid proposal cid: %v", err)
	}
	return nil
}

// processAuction handles the next auction in the queue.
// Processing may involve the following tasks:
// - Running the actual auction. When selecting winners, only the first bid from a bidbot will be considered.
// - Notifying winners that they have won.
// - Notifying winners of a proposal cid that was from an agreed upon deal.
func (a *Auctioneer) processAuction(
	ctx context.Context,
	auction core.Auction,
	addBid func(bid core.Bid) (core.BidID, error),
) (map[core.BidID]core.WinningBid, error) {
	log.Debugf("auction %s started", auction.ID)

	// No need to re-auction if we have enough bids; just notify winners.
	// This case can happen if there was an error during winner selection on
	// a prior attempt, or if an auction was re-queued after receiving
	// a proposal Cid from the broker.
	if len(auction.WinningBids) == int(auction.DealReplication) {
		log.Debugf(
			"auction %s completed (attempt=%d/%d); total bids: %d/%d",
			auction.ID,
			auction.Attempts,
			a.auctionConf.Attempts,
			len(auction.Bids),
			auction.DealReplication,
		)
		winners, err := a.notifyWinners(ctx, auction, nil)
		if err != nil {
			return nil, fmt.Errorf("notifying winners: %v", err)
		}
		return winners, nil
	}

	// Subscribe to bids topic
	topic, err := a.peer.NewTopic(ctx, core.BidsTopic(auction.ID), true)
	if err != nil {
		return nil, fmt.Errorf("creating bids topic: %v", err)
	}
	defer func() {
		if err := topic.Close(); err != nil {
			log.Errorf("closing bids topic: %v", err)
		}
	}()
	topic.SetEventHandler(a.eventHandler)

	bids := make(map[core.BidID]core.Bid)
	bidders := make(map[peer.ID]struct{})
	for _, b := range auction.WinningBids {
		bidders[b.BidderID] = struct{}{}
	}
	bidsHandler := func(from peer.ID, _ string, msg []byte) ([]byte, error) {
		if err := from.Validate(); err != nil {
			return nil, fmt.Errorf("invalid bidder: %v", err)
		}
		if _, ok := bidders[from]; ok {
			return nil, fmt.Errorf("bid was already received")
		}

		pbid := &pb.Bid{}
		if err := proto.Unmarshal(msg, pbid); err != nil {
			return nil, fmt.Errorf("unmarshaling message: %v", err)
		}

		bid := core.Bid{
			MinerAddr:        pbid.MinerAddr,
			WalletAddrSig:    pbid.WalletAddrSig,
			BidderID:         from,
			AskPrice:         pbid.AskPrice,
			VerifiedAskPrice: pbid.VerifiedAskPrice,
			StartEpoch:       pbid.StartEpoch,
			FastRetrieval:    pbid.FastRetrieval,
			ReceivedAt:       time.Now(),
		}
		if err := a.validateBid(bid); err != nil {
			return nil, fmt.Errorf("invalid bid: %v", err)
		}

		var price int64
		if auction.DealVerified {
			price = bid.VerifiedAskPrice
		} else {
			price = bid.AskPrice
		}
		log.Debugf("auction %s received bid from %s: %d", auction.ID, bid.BidderID, price)
		a.metricNewBid.Add(ctx, 1)

		if !a.acceptBid(&auction, &bid) {
			return nil, errors.New("bid rejected")
		}
		id, err := addBid(bid)
		if err != nil {
			return nil, fmt.Errorf("adding bid to auction %s: %v", auction.ID, err)
		}
		bidders[bid.BidderID] = struct{}{}
		bids[id] = bid
		a.metricAcceptedBid.Add(ctx, 1)

		return []byte(id), nil
	}
	topic.SetMessageHandler(bidsHandler)

	// Set deadline
	deadline := auction.StartedAt.Add(auction.Duration)

	// Publish the auction
	msg, err := proto.Marshal(&pb.Auction{
		Id:           string(auction.ID),
		PayloadCid:   auction.PayloadCid.String(),
		DealSize:     auction.DealSize,
		DealDuration: auction.DealDuration,
		Sources:      cast.SourcesToPb(auction.Sources),
		EndsAt:       timestamppb.New(deadline),
	})
	if err != nil {
		return nil, fmt.Errorf("marshaling message: %v", err)
	}
	if _, err := a.auctions.Publish(ctx, msg, pubsub.WithIgnoreResponse(true)); err != nil {
		return nil, fmt.Errorf("publishing auction: %v", err)
	}

	actx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	<-actx.Done()
	log.Debugf(
		"auction %s completed (attempt=%d/%d); total bids: %d/%d",
		auction.ID,
		auction.Attempts,
		a.auctionConf.Attempts,
		len(bids),
		auction.DealReplication,
	)
	roundWinners, err := a.selectNewWinners(auction, bids)
	if err != nil {
		return nil, fmt.Errorf("selecting new winners: %v", err)
	}
	notifiedWinners, err := a.notifyWinners(ctx, auction, roundWinners)
	if err != nil {
		return nil, fmt.Errorf("notifying winners: %v", err)
	}
	return notifiedWinners, nil
}

func (a *Auctioneer) validateBid(b core.Bid) error {
	if b.MinerAddr == "" {
		return errors.New("miner address must not be empty")
	}
	if b.WalletAddrSig == nil {
		return errors.New("wallet address signature must not be empty")
	}
	if err := b.BidderID.Validate(); err != nil {
		return fmt.Errorf("bidder id is not valid: %v", err)
	}
	if b.AskPrice < 0 {
		return errors.New("ask price must be greater than or equal to zero")
	}
	if b.VerifiedAskPrice < 0 {
		return errors.New("verified ask price must be greater than or equal to zero")
	}
	if b.StartEpoch <= 0 {
		return errors.New("start epoch must be greater than zero")
	}

	ok, err := a.fc.VerifyBidder(b.WalletAddrSig, b.BidderID, b.MinerAddr)
	if err != nil {
		return fmt.Errorf("verifying miner address: %v", err)
	}
	if !ok {
		return fmt.Errorf("invalid miner address or signature")
	}
	return nil
}

func (a *Auctioneer) acceptBid(auction *core.Auction, bid *core.Bid) bool {
	if auction.FilEpochDeadline > 0 && auction.FilEpochDeadline < bid.StartEpoch {
		log.Debugf("miner %s start epoch %d doesn't meet the deadline %d of auction %s",
			bid.MinerAddr, bid.StartEpoch, auction.FilEpochDeadline, auction.ID)
		return false
	}
	for _, addr := range auction.ExcludedMiners {
		if bid.MinerAddr == addr {
			log.Debugf("miner %s is explicitly excluded from auction %s", bid.MinerAddr, auction.ID)
			return false
		}
	}
	return true
}

func (a *Auctioneer) finalizeAuction(ctx context.Context, auction core.Auction) error {
	switch auction.Status {
	case broker.AuctionStatusFinalized:
		if auction.ErrorCause != "" {
			a.metricNewFinalizedAuction.Add(ctx, 1, metrics.AttrError)
		} else {
			a.metricNewFinalizedAuction.Add(ctx, 1, metrics.AttrOK)
		}
	default:
		return fmt.Errorf("invalid final status: %s", auction.Status)
	}
	if err := a.broker.StorageDealAuctioned(ctx, auction); err != nil {
		return fmt.Errorf("signaling broker: %v", err)
	}
	return nil
}

func (a *Auctioneer) eventHandler(from peer.ID, topic string, msg []byte) {
	log.Debugf("%s peer event: %s %s", topic, from, msg)
	if topic == core.AuctionTopic && string(msg) == "JOINED" {
		a.peer.Host().ConnManager().Protect(from, "auctioneer:<bidder>")
	}
}

type rankedBid struct {
	ID  core.BidID
	Bid core.Bid
}

func heapifyBids(bids map[core.BidID]core.Bid, dealVerified bool) *BidHeap {
	h := &BidHeap{dealVerified: dealVerified}
	heap.Init(h)
	for id, b := range bids {
		heap.Push(h, rankedBid{ID: id, Bid: b})
	}
	return h
}

// BidHeap is used to efficiently select auction winners.
type BidHeap struct {
	h            []rankedBid
	dealVerified bool
}

// Len returns the length of h.
func (bh *BidHeap) Len() int {
	return len(bh.h)
}

// Less returns true if the value at j is less than the value at i.
func (bh *BidHeap) Less(i, j int) bool {
	if bh.dealVerified {
		return bh.h[i].Bid.VerifiedAskPrice > bh.h[j].Bid.VerifiedAskPrice
	}
	return bh.h[i].Bid.AskPrice > bh.h[j].Bid.AskPrice
}

// Swap index i and j.
func (bh *BidHeap) Swap(i, j int) {
	bh.h[i], bh.h[j] = bh.h[j], bh.h[i]
}

// Push adds x to h.
func (bh *BidHeap) Push(x interface{}) {
	bh.h = append(bh.h, x.(rankedBid))
}

// Pop removes and returns the last element in h.
func (bh *BidHeap) Pop() (x interface{}) {
	x, bh.h = bh.h[len(bh.h)-1], bh.h[:len(bh.h)-1]
	return x
}

func (a *Auctioneer) selectNewWinners(
	auction core.Auction,
	bids map[core.BidID]core.Bid,
) (map[core.BidID]core.WinningBid, error) {
	selectCount := int(auction.DealReplication) - len(auction.WinningBids)
	if selectCount == 0 {
		return nil, nil
	}
	if len(bids) == 0 {
		return nil, ErrInsufficientBids
	}

	var (
		bh      = heapifyBids(bids, auction.DealVerified)
		winners = make(map[core.BidID]core.WinningBid)
	)

	// Select lowest bids until deal replication is met
	for i := 0; i < selectCount; i++ {
		if bh.Len() == 0 {
			break
		}
		b := heap.Pop(bh).(rankedBid)
		winners[b.ID] = core.WinningBid{
			BidderID: b.Bid.BidderID,
		}
	}
	if len(winners) < selectCount {
		return winners, ErrInsufficientBids
	}

	return winners, nil
}

type notifyResult struct {
	id  core.BidID
	wb  core.WinningBid
	err error
}

func (a *Auctioneer) notifyWinners(
	ctx context.Context,
	auction core.Auction,
	winners map[core.BidID]core.WinningBid,
) (map[core.BidID]core.WinningBid, error) {
	if winners == nil {
		winners = make(map[core.BidID]core.WinningBid)
	}

	// Add non-acked winners from prior attempts
	for id, wb := range auction.WinningBids {
		if !wb.Acknowledged || (wb.ProposalCid.Defined() && !wb.ProposalCidAcknowledged) {
			winners[id] = wb
		}
	}

	// Bail if nothing to do
	if len(winners) == 0 {
		return nil, nil
	}

	var wg sync.WaitGroup
	resCh := make(chan notifyResult, len(winners))
	aid := auction.ID
	for id, wb := range winners {
		wg.Add(1)
		go func(id core.BidID, wb core.WinningBid) {
			defer wg.Done()
			res := notifyResult{id: id, wb: wb}

			if !wb.Acknowledged {
				if err := a.publishWin(ctx, aid, id, wb.BidderID); err != nil {
					res.err = fmt.Errorf("publishing win: %v", err)
				} else {
					res.wb.Acknowledged = true
				}
			} else if !wb.ProposalCidAcknowledged {
				if err := a.publishProposal(ctx, aid, id, wb.BidderID, wb.ProposalCid); err != nil {
					res.err = fmt.Errorf("publishing win: %v", err)
				} else {
					res.wb.ProposalCidAcknowledged = true
				}
			} else {
				res.err = fmt.Errorf("nothing to do for winning bid %s", id)
			}
			resCh <- res
		}(id, wb)
	}
	wg.Wait()
	close(resCh)

	merr := &multierror.Error{}
	for r := range resCh {
		merr = multierror.Append(merr, r.err)
		winners[r.id] = r.wb
	}
	return winners, merr.ErrorOrNil()
}

func (a *Auctioneer) publishWin(ctx context.Context, id core.AuctionID, bid core.BidID, bidder peer.ID) error {
	topic, err := a.peer.NewTopic(ctx, core.WinsTopic(bidder), false)
	if err != nil {
		return fmt.Errorf("creating win topic: %v", err)
	}
	defer func() {
		if err := topic.Close(); err != nil {
			log.Errorf("closing wins topic: %v", err)
		}
	}()
	topic.SetEventHandler(a.eventHandler)

	msg, err := proto.Marshal(&pb.WinningBid{
		AuctionId: string(id),
		BidId:     string(bid),
	})
	if err != nil {
		return fmt.Errorf("marshaling message: %v", err)
	}
	tctx, cancel := context.WithTimeout(ctx, notifyTimeout)
	defer cancel()
	res, err := topic.Publish(tctx, msg)
	if err != nil {
		return fmt.Errorf("publishing win to %s: %v", bidder, err)
	}
	r := <-res
	if errors.Is(r.Err, pubsub.ErrResponseNotReceived) {
		return fmt.Errorf("publishing win to %s: %v", bidder, r.Err)
	} else if r.Err != nil {
		return fmt.Errorf("publishing win; bidder %s returned error: %v", bidder, r.Err)
	}
	return nil
}

func (a *Auctioneer) publishProposal(
	ctx context.Context,
	id core.AuctionID,
	bid core.BidID,
	bidder peer.ID,
	pcid cid.Cid,
) error {
	topic, err := a.peer.NewTopic(ctx, core.ProposalsTopic(bidder), false)
	if err != nil {
		return fmt.Errorf("creating proposals topic: %v", err)
	}
	defer func() {
		if err := topic.Close(); err != nil {
			log.Errorf("closing proposals topic: %v", err)
		}
	}()
	topic.SetEventHandler(a.eventHandler)

	msg, err := proto.Marshal(&pb.WinningBidProposal{
		AuctionId:   string(id),
		BidId:       string(bid),
		ProposalCid: pcid.String(),
	})
	if err != nil {
		return fmt.Errorf("marshaling message: %v", err)
	}
	tctx, cancel := context.WithTimeout(ctx, notifyTimeout)
	defer cancel()
	res, err := topic.Publish(tctx, msg)
	if err != nil {
		return fmt.Errorf("publishing proposal to %s: %v", bidder, err)
	}
	r := <-res
	if errors.Is(r.Err, pubsub.ErrResponseNotReceived) {
		return fmt.Errorf("publishing proposal to %s: %v", bidder, r.Err)
	} else if r.Err != nil {
		return fmt.Errorf("publishing proposal; bidder %s returned error: %v", bidder, r.Err)
	}
	return nil
}
