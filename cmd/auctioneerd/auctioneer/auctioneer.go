package auctioneer

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/oklog/ulid/v2"
	pb "github.com/textileio/bidbot/gen/v1"
	"github.com/textileio/bidbot/lib/auction"
	core "github.com/textileio/bidbot/lib/auction"
	"github.com/textileio/bidbot/lib/cast"
	"github.com/textileio/bidbot/lib/filclient"
	"github.com/textileio/broker-core/auctioneer"
	"github.com/textileio/broker-core/broker"
	q "github.com/textileio/broker-core/cmd/auctioneerd/auctioneer/queue"
	"github.com/textileio/broker-core/metrics"
	mbroker "github.com/textileio/broker-core/msgbroker"
	rpc "github.com/textileio/go-libp2p-pubsub-rpc"
	"github.com/textileio/go-libp2p-pubsub-rpc/finalizer"
	rpcpeer "github.com/textileio/go-libp2p-pubsub-rpc/peer"
	golog "github.com/textileio/go-log/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	log = golog.Logger("auctioneer")

	// maxAuctionDuration is the max duration an auction can run for.
	maxAuctionDuration = time.Minute * 10

	// NotifyTimeout is the max duration the auctioneer will wait for a response from bidders.
	NotifyTimeout = time.Second * 30

	// ErrAuctionNotFound indicates the requested auction was not found.
	ErrAuctionNotFound = errors.New("auction not found")

	// ErrInsufficientBids indicates the auction failed due to insufficient bids.
	ErrInsufficientBids = errors.New("insufficient bids")
)

// AuctionConfig defines auction params.
type AuctionConfig struct {
	// Duration auctions will be held for.
	Duration time.Duration
}

// Auctioneer handles deal auctions for a broker.
type Auctioneer struct {
	mb          mbroker.MsgBroker
	queue       *q.Queue
	started     bool
	auctionConf AuctionConfig

	peer     *rpcpeer.Peer
	fc       filclient.FilClient
	auctions *rpc.Topic

	finalizer *finalizer.Finalizer
	lkEntropy sync.Mutex
	entropy   *ulid.MonotonicEntropy

	statLastCreatedAuction    atomic.Value // time.Time
	metricNewAuction          metric.Int64Counter
	metricNewFinalizedAuction metric.Int64Counter
	metricNewBid              metric.Int64Counter
	metricAcceptedBid         metric.Int64Counter
	metricLastCreatedAuction  metric.Int64ValueObserver
	metricPubsubPeers         metric.Int64ValueObserver

	winsTopics     map[peer.ID]*rpc.Topic
	proposalTopics map[peer.ID]*rpc.Topic
	lkTopics       sync.Mutex

	providerFailureRates         map[string]int
	providerFailureRatesLoadedAt time.Time
	lkProviderFailureRates       sync.Mutex

	providerWinningRates         map[string]int
	providerWinningRatesLoadedAt time.Time
	lkProviderWinningRates       sync.Mutex
}

// New returns a new Auctioneer.
func New(
	p *rpcpeer.Peer,
	postgresURI string,
	mb mbroker.MsgBroker,
	fc filclient.FilClient,
	auctionConf AuctionConfig,
) (*Auctioneer, error) {
	if err := validateConfig(auctionConf); err != nil {
		return nil, fmt.Errorf("validating config: %v", err)
	}

	a := &Auctioneer{
		mb:             mb,
		peer:           p,
		fc:             fc,
		auctionConf:    auctionConf,
		finalizer:      finalizer.NewFinalizer(),
		winsTopics:     make(map[peer.ID]*rpc.Topic),
		proposalTopics: make(map[peer.ID]*rpc.Topic),
	}
	a.initMetrics()

	queue, err := q.NewQueue(postgresURI, a.processAuction, a.finalizeAuction)
	if err != nil {
		return nil, fmt.Errorf("creating queue: %v", err)
	}
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
	auctions, err := a.peer.NewTopic(ctx, core.Topic, false)
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
// New auctions are queued if the auctioneer is busy.
func (a *Auctioneer) CreateAuction(ctx context.Context, auction auctioneer.Auction) error {
	auction.Status = broker.AuctionStatusUnspecified
	auction.Duration = a.auctionConf.Duration
	if err := a.queue.CreateAuction(ctx, auction); err != nil {
		return fmt.Errorf("creating auction: %v", err)
	}

	log.Infof("created auction %s", auction.ID)

	labels := []attribute.KeyValue{
		attribute.Int("replication", int(auction.DealReplication)),
		attribute.Bool("verified", auction.DealVerified),
	}
	a.metricNewAuction.Add(context.Background(), 1, labels...)
	a.statLastCreatedAuction.Store(time.Now())

	return nil
}

// GetAuction returns an auction by id.
// If an auction is not found for id, ErrAuctionNotFound is returned.
func (a *Auctioneer) GetAuction(ctx context.Context, id core.ID) (*auctioneer.Auction, error) {
	auc, err := a.queue.GetAuction(ctx, id)
	if errors.Is(q.ErrAuctionNotFound, err) {
		return nil, ErrAuctionNotFound
	} else if err != nil {
		return nil, fmt.Errorf("getting auction: %v", err)
	}
	return auc, nil
}

// DeliverProposal delivers the proposal Cid for an accepted deal to the winning bidder.
// This may be called multiple times by the broker in the event delivery fails.
func (a *Auctioneer) DeliverProposal(ctx context.Context, auctionID core.ID, bidID core.BidID, pcid cid.Cid) error {
	if !pcid.Defined() {
		return errors.New("proposal cid is not defined")
	}

	bid, err := a.queue.GetFinalizedAuctionBid(ctx, auctionID, bidID)
	if err != nil {
		return fmt.Errorf("getting bid %s for auction %s: %v", bidID, auctionID, err)
	}
	if bid.ProposalCid.Defined() {
		log.Warnf("proposal cid %s is already published, duplicated message?", pcid)
		return nil
	}
	var errCause string
	publishErr := a.publishProposal(ctx, auctionID, bidID, bid.BidderID, pcid)
	if publishErr != nil {
		errCause = publishErr.Error()
		if err := a.queue.SetProposalCidDeliveryError(ctx, auctionID, bidID, errCause); err != nil {
			log.Errorf("setting proposal cid delivery error: %v", err)
		}
		publishErr = fmt.Errorf("publishing proposal to %s in auction %s: %v", bid.StorageProviderID, auctionID, publishErr)
	} else {
		log.Infof("delivered proposal %s for bid %s in auction %s to %s", pcid, bidID, auctionID, bid.StorageProviderID)
		if err := a.queue.SetProposalCidDelivered(ctx, auctionID, bidID, pcid); err != nil {
			log.Errorf("saving proposal cid: %v", err)
		}
	}
	if err := mbroker.PublishMsgAuctionProposalCidDelivered(ctx, a.mb, auctionID,
		bid.BidderID, bidID, pcid, errCause); err != nil {
		log.Warn(err) // error is annotated
	}
	return publishErr
}

// processAuction handles the next auction in the queue.
// An auction involves the following steps:
// 1. Publish the auction to the deal feed.
// 2. Wait for bids to come in from bidders.
// 3. Close the auction after the configured duration has passed.
// 4. Select winners, during which winners are notified. If this notification fails, the winner is
//    removed from the winner pool and the next best bid is selected. This process is continued
//    until the number of winners equals deal replication.
//    The auction fails if there are not enough bids to complete this process.
func (a *Auctioneer) processAuction(
	ctx context.Context,
	auction auctioneer.Auction,
	addBid func(bid auctioneer.Bid) error,
) (map[core.BidID]auctioneer.WinningBid, error) {
	if err := mbroker.PublishMsgAuctionStarted(ctx, a.mb, mbroker.AuctionToPbSummary(&auction)); err != nil {
		log.Warn(err) // error is annotated
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

	var (
		bids    []auctioneer.Bid
		bidders = make(map[peer.ID]struct{})
		mu      sync.Mutex
	)

	bidsHandler := func(from peer.ID, _ string, msg []byte) ([]byte, error) {
		if err := from.Validate(); err != nil {
			return nil, fmt.Errorf("invalid bidder: %v", err)
		}
		pbid := &pb.Bid{}
		if err := proto.Unmarshal(msg, pbid); err != nil {
			return nil, fmt.Errorf("unmarshaling message: %v", err)
		}
		id, err := a.newID()
		if err != nil {
			return nil, fmt.Errorf("generating bid id: %v", err)
		}

		bid := auctioneer.Bid{
			ID:                id,
			StorageProviderID: pbid.StorageProviderId,
			WalletAddrSig:     pbid.WalletAddrSig,
			BidderID:          from,
			AskPrice:          pbid.AskPrice,
			VerifiedAskPrice:  pbid.VerifiedAskPrice,
			StartEpoch:        pbid.StartEpoch,
			FastRetrieval:     pbid.FastRetrieval,
			ReceivedAt:        time.Now(),
		}
		if err := a.validateBid(&bid); err != nil {
			return nil, fmt.Errorf("invalid bid: %v", err)
		}
		if err := mbroker.PublishMsgAuctionBidReceived(ctx, a.mb, mbroker.AuctionToPbSummary(&auction), &bid); err != nil {
			log.Warn(err) // error is annotated
		}

		var price int64
		if auction.DealVerified {
			price = bid.VerifiedAskPrice
		} else {
			price = bid.AskPrice
		}
		log.Infof("auction %s received bid from %s: %d", auction.ID, bid.BidderID, price)
		label := attribute.String("storage-provider-id", bid.StorageProviderID)
		a.metricNewBid.Add(ctx, 1, label)

		if !acceptBid(&auction, &bid) {
			return nil, errors.New("bid rejected")
		}

		mu.Lock()
		_, exists := bidders[from]
		if exists {
			mu.Unlock()
			return nil, fmt.Errorf("bid was already received")
		}
		bidders[bid.BidderID] = struct{}{}
		mu.Unlock()

		err = addBid(bid)
		if err != nil {
			return nil, fmt.Errorf("adding bid to auction %s: %v", auction.ID, err)
		}
		mu.Lock()
		bids = append(bids, bid)
		mu.Unlock()
		a.metricAcceptedBid.Add(ctx, 1, label)

		return []byte(bid.ID), nil
	}
	topic.SetMessageHandler(bidsHandler)

	deadline := time.Now().Add(auction.Duration)
	// Publish the auction
	msg, err := proto.Marshal(&pb.Auction{
		Id:               string(auction.ID),
		PayloadCid:       auction.PayloadCid.String(),
		DealSize:         auction.DealSize,
		DealDuration:     auction.DealDuration,
		FilEpochDeadline: auction.FilEpochDeadline,
		Sources:          cast.SourcesToPb(auction.Sources),
		EndsAt:           timestamppb.New(deadline),
	})
	if err != nil {
		return nil, fmt.Errorf("marshaling message: %v", err)
	}
	actx, cancel := context.WithDeadline(ctx, deadline)
	defer cancel()
	if _, err := a.auctions.Publish(actx, msg, rpc.WithRepublishing(true), rpc.WithIgnoreResponse(true)); err != nil {
		return nil, fmt.Errorf("publishing auction: %v", err)
	}
	<-actx.Done()
	topic.SetMessageHandler(nil)

	log.Infof(
		"auction %s ended; total bids: %d; num required: %d",
		auction.ID,
		len(bids),
		auction.DealReplication,
	)

	winners, err := a.selectWinners(ctx, auction, bids)
	if err != nil {
		log.Warnf("auction %s failed: %v", auction.ID, err)
		return nil, fmt.Errorf("selecting winners: %v", err)
	}

	var info []string
	for id, wb := range winners {
		info = append(info, fmt.Sprintf("bid: %s; bidder: %s", id, wb.BidderID))
	}
	log.Infof("auction %s succeeded; winning bids: [%s]", auction.ID, strings.Join(info, ", "))
	return winners, nil
}

func (a *Auctioneer) validateBid(b *auctioneer.Bid) error {
	if b.StorageProviderID == "" {
		return errors.New("storage provider ID must not be empty")
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

	ok, err := a.fc.VerifyBidder(b.WalletAddrSig, b.BidderID, b.StorageProviderID)
	if err != nil {
		return fmt.Errorf("verifying miner address: %v", err)
	}
	if !ok {
		return fmt.Errorf("invalid miner address or signature")
	}
	return nil
}

func (a *Auctioneer) finalizeAuction(ctx context.Context, auction *auctioneer.Auction) error {
	labels := []attribute.KeyValue{
		attribute.Int("replication", int(auction.DealReplication)),
		attribute.Bool("verified", auction.DealVerified),
	}
	switch auction.Status {
	case broker.AuctionStatusFinalized:
		if auction.ErrorCause != "" {
			labels = append(labels, metrics.AttrError)
			labels = append(labels, attribute.String("error-cause", auction.ErrorCause))
		} else {
			labels = append(labels, metrics.AttrOK)
		}
	default:
		return fmt.Errorf("invalid final status: %s", auction.Status)
	}
	a.metricNewFinalizedAuction.Add(ctx, 1, labels...)
	if err := mbroker.PublishMsgAuctionClosed(ctx, a.mb, toClosedAuction(auction)); err != nil {
		log.Warn(err) // error is annotated
	}
	return nil
}

func toClosedAuction(a *auctioneer.Auction) broker.ClosedAuction {
	wbids := make(map[auction.BidID]broker.WinningBid)
	for wbid := range a.WinningBids {
		bid, ok := a.Bids[wbid]
		if !ok {
			log.Errorf("winning bid %s wasn't found in bid map", wbid)
			continue
		}
		var price int64
		if a.DealVerified {
			price = bid.VerifiedAskPrice
		} else {
			price = bid.AskPrice
		}
		wbids[wbid] = broker.WinningBid{
			StorageProviderID: bid.StorageProviderID,
			Price:             price,
			StartEpoch:        bid.StartEpoch,
			FastRetrieval:     bid.FastRetrieval,
		}
	}
	return broker.ClosedAuction{
		ID:              a.ID,
		BatchID:         a.BatchID,
		DealDuration:    a.DealDuration,
		DealReplication: a.DealReplication,
		DealVerified:    a.DealVerified,
		Status:          a.Status,
		WinningBids:     wbids,
		ErrorCause:      a.ErrorCause,
	}
}

func (a *Auctioneer) eventHandler(from peer.ID, topic string, msg []byte) {
	log.Debugf("%s peer event: %s %s", topic, from, msg)
	if topic == core.Topic && string(msg) == "JOINED" {
		a.peer.Host().ConnManager().Protect(from, "auctioneer:<bidder>")
	}
}

func acceptBid(auction *auctioneer.Auction, bid *auctioneer.Bid) bool {
	if auction.FilEpochDeadline > 0 && (bid.StartEpoch <= 0 || auction.FilEpochDeadline < bid.StartEpoch) {
		log.Debugf("miner %s start epoch %d doesn't meet the deadline %d of auction %s",
			bid.StorageProviderID, bid.StartEpoch, auction.FilEpochDeadline, auction.ID)
		return false
	}
	for _, addr := range auction.ExcludedStorageProviders {
		if bid.StorageProviderID == addr {
			log.Debugf("miner %s is explicitly excluded from auction %s", bid.StorageProviderID, auction.ID)
			return false
		}
	}
	return true
}

func (a *Auctioneer) selectWinners(
	ctx context.Context,
	auction auctioneer.Auction,
	bids []auctioneer.Bid,
) (map[core.BidID]auctioneer.WinningBid, error) {
	winners := make(map[core.BidID]auctioneer.WinningBid)
	tctx, cancel := context.WithCancel(ctx)
	defer cancel()
	sortStrategy := Ordered(
		ByPrice(),      // sort by lower price
		ByStartEpoch(), // then by earlier start epoch
		Weighed{}.Add( // if both are the same, below factors are equally considered. Note that the rates are in the scale of [0, 1e6].
			ByProviderRate(a.getProviderFailureRates(ctx)), 1, // lower recent failures are preferred - they are more stable
		).Add(ByProviderRate(a.getProviderWinningRates(ctx)), 1)) // lower recent winnings are also preferred - for fairness
	chCandidates := BidsSorter(sortStrategy).Sort(tctx, &auction, bids)
	for b := range chCandidates {
		if err := mbroker.PublishMsgAuctionWinnerSelected(ctx, a.mb,
			mbroker.AuctionToPbSummary(&auction), &b); err != nil {
			log.Warn(err) // error is annotated
		}

		if err := a.publishWin(ctx, auction.ID, b.ID, b.BidderID); err != nil {
			// skip this intended error from bidder which signals auctioneer to silently move on
			if !strings.Contains(err.Error(), core.ErrStringWouldExceedRunningBytesLimit) {
				log.Warn(err) // error is annotated in publishWin
			}
			continue
		}
		winners[b.ID] = auctioneer.WinningBid{
			BidderID: b.BidderID,
		}
		if err := mbroker.PublishMsgAuctionWinnerAcked(ctx, a.mb, mbroker.AuctionToPbSummary(&auction), &b); err != nil {
			log.Warn(err) // error is annotated
		}
		if len(winners) == int(auction.DealReplication) {
			return winners, nil
		}
	}
	return winners, ErrInsufficientBids
}

func (a *Auctioneer) getProviderFailureRates(ctx context.Context) map[string]int {
	a.lkProviderFailureRates.Lock()
	defer a.lkProviderFailureRates.Unlock()
	if time.Since(a.providerFailureRatesLoadedAt) > time.Hour {
		if rates, err := a.queue.GetProviderFailureRates(ctx); err != nil {
			log.Warnf("getting storage provider recent failure rates, use cached rates: %v", err)
		} else {
			a.providerFailureRates = rates
			a.providerFailureRatesLoadedAt = time.Now()
		}
	}
	return a.providerFailureRates
}

func (a *Auctioneer) getProviderWinningRates(ctx context.Context) map[string]int {
	a.lkProviderWinningRates.Lock()
	defer a.lkProviderWinningRates.Unlock()
	if time.Since(a.providerWinningRatesLoadedAt) > time.Hour {
		if rates, err := a.queue.GetProviderWinningRates(ctx); err != nil {
			log.Warnf("getting storage provider recent failure rates, use cached rates: %v", err)
		} else {
			a.providerWinningRates = rates
			a.providerWinningRatesLoadedAt = time.Now()
		}
	}
	return a.providerWinningRates
}

func (a *Auctioneer) publishWin(ctx context.Context, id core.ID, bid core.BidID, bidder peer.ID) error {
	topic, err := a.winsTopicFor(ctx, bidder)
	if err != nil {
		return fmt.Errorf("creating win topic: %v", err)
	}
	msg, err := proto.Marshal(&pb.WinningBid{
		AuctionId: string(id),
		BidId:     string(bid),
	})
	if err != nil {
		return fmt.Errorf("marshaling message: %v", err)
	}
	tctx, cancel := context.WithTimeout(ctx, NotifyTimeout)
	defer cancel()
	res, err := topic.Publish(tctx, msg)
	if err != nil {
		return fmt.Errorf("publishing win to %s in auction %s: %v", bidder, id, err)
	}
	r := <-res
	if errors.Is(r.Err, rpc.ErrResponseNotReceived) {
		return fmt.Errorf("publishing win to %s in auction %s: %v", bidder, id, r.Err)
	} else if r.Err != nil {
		return fmt.Errorf("publishing win in auction %s; bidder %s returned error: %v", id, bidder, r.Err)
	}
	return nil
}

func (a *Auctioneer) publishProposal(
	ctx context.Context,
	id core.ID,
	bid core.BidID,
	bidder peer.ID,
	pcid cid.Cid,
) error {
	topic, err := a.proposalTopicFor(ctx, bidder)
	if err != nil {
		return fmt.Errorf("creating proposals topic: %v", err)
	}
	msg, err := proto.Marshal(&pb.WinningBidProposal{
		AuctionId:   string(id),
		BidId:       string(bid),
		ProposalCid: pcid.String(),
	})
	if err != nil {
		return fmt.Errorf("marshaling message: %v", err)
	}
	tctx, cancel := context.WithTimeout(ctx, NotifyTimeout)
	defer cancel()
	res, err := topic.Publish(tctx, msg, rpc.WithRepublishing(true))
	if err != nil {
		return err
	}
	r := <-res
	if errors.Is(r.Err, rpc.ErrResponseNotReceived) {
		return err
	} else if r.Err != nil {
		return fmt.Errorf("bidder returned error: %v", r.Err)
	}
	return nil
}

func (a *Auctioneer) winsTopicFor(ctx context.Context, peer peer.ID) (*rpc.Topic, error) {
	a.lkTopics.Lock()
	topic, exists := a.winsTopics[peer]
	if exists {
		a.lkTopics.Unlock()
		return topic, nil
	}
	defer a.lkTopics.Unlock()
	topic, err := a.peer.NewTopic(ctx, core.WinsTopic(peer), false)
	if err != nil {
		return nil, err
	}
	topic.SetEventHandler(a.eventHandler)
	a.winsTopics[peer] = topic
	a.finalizer.Add(topic)
	return topic, nil
}

func (a *Auctioneer) proposalTopicFor(ctx context.Context, peer peer.ID) (*rpc.Topic, error) {
	a.lkTopics.Lock()
	topic, exists := a.proposalTopics[peer]
	if exists {
		a.lkTopics.Unlock()
		return topic, nil
	}
	defer a.lkTopics.Unlock()
	topic, err := a.peer.NewTopic(ctx, core.ProposalsTopic(peer), false)
	if err != nil {
		return nil, err
	}
	topic.SetEventHandler(a.eventHandler)
	a.proposalTopics[peer] = topic
	a.finalizer.Add(topic)
	return topic, nil
}

// newID returns new monotonically increasing bid ids.
func (a *Auctioneer) newID() (auction.BidID, error) {
	a.lkEntropy.Lock() // entropy is not safe for concurrent use

	if a.entropy == nil {
		a.entropy = ulid.Monotonic(rand.Reader, 0)
	}
	id, err := ulid.New(ulid.Timestamp(time.Now().UTC()), a.entropy)
	if errors.Is(err, ulid.ErrMonotonicOverflow) {
		a.entropy = nil
		a.lkEntropy.Unlock()
		return a.newID()
	} else if err != nil {
		a.lkEntropy.Unlock()
		return "", fmt.Errorf("generating id: %v", err)
	}
	a.lkEntropy.Unlock()
	return auction.BidID(strings.ToLower(id.String())), nil
}
