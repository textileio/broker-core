package broker

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	logger "github.com/textileio/go-log/v2"
	"go.opentelemetry.io/otel/metric"

	"github.com/textileio/bidbot/lib/auction"
	"github.com/textileio/broker-core/auctioneer"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/chainapi"
	"github.com/textileio/broker-core/cmd/brokerd/store"
	"github.com/textileio/broker-core/dealer"
	"github.com/textileio/broker-core/packer"
)

const (
	filecoinGenesisUnixEpoch = 1598306400
	causeMaxAuctionRetries   = "reached max number of retries"
)

var (
	// ErrNotFound is returned when the broker request doesn't exist.
	ErrNotFound = fmt.Errorf("broker request not found")
	// ErrInvalidCid is returned when the Cid is undefined.
	ErrInvalidCid = fmt.Errorf("the cid can't be undefined")
	// ErrEmptyGroup is returned when an empty storage deal group
	// is received.
	ErrEmptyGroup = fmt.Errorf("the storage deal group is empty")

	log = logger.Logger("broker")
)

// Broker creates and tracks request to store Cids in
// the Filecoin network.
type Broker struct {
	store      *store.Store
	packer     packer.Packer
	auctioneer auctioneer.Auctioneer
	dealer     dealer.Dealer
	chainAPI   chainapi.ChainAPI
	ipfsClient *httpapi.HttpApi

	conf config

	onceClose       sync.Once
	daemonCtx       context.Context
	daemonCancelCtx context.CancelFunc
	daemonClosed    chan struct{}

	metricUnpinTotal        metric.Int64Counter
	statTotalRecursivePins  int64
	metricRecursivePinCount metric.Int64ValueObserver
}

// New creates a Broker backed by the provided `ds`.
func New(
	postgresURI string,
	packer packer.Packer,
	auctioneer auctioneer.Auctioneer,
	dealer dealer.Dealer,
	chainAPI chainapi.ChainAPI,
	ipfsClient *httpapi.HttpApi,
	opts ...Option,
) (*Broker, error) {
	s, err := store.New(postgresURI)
	if err != nil {
		return nil, fmt.Errorf("initializing broker request store: %s", err)
	}

	conf := defaultConfig
	for _, op := range opts {
		if err := op(&conf); err != nil {
			return nil, fmt.Errorf("applying config: %s", err)
		}
	}
	if err := conf.validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %s", err)
	}

	ctx, cls := context.WithCancel(context.Background())
	b := &Broker{
		store:      s,
		packer:     packer,
		dealer:     dealer,
		auctioneer: auctioneer,
		chainAPI:   chainAPI,
		ipfsClient: ipfsClient,

		conf: conf,

		daemonCtx:       ctx,
		daemonCancelCtx: cls,
		daemonClosed:    make(chan struct{}),
	}
	b.initMetrics()

	go b.daemonUnpinner()

	return b, nil
}

var _ broker.Broker = (*Broker)(nil)

// Create creates a new BrokerRequest with the provided Cid and
// Metadata configuration.
func (b *Broker) Create(ctx context.Context, c cid.Cid) (broker.BrokerRequest, error) {
	if !c.Defined() {
		return broker.BrokerRequest{}, ErrInvalidCid
	}

	now := time.Now()
	br := broker.BrokerRequest{
		ID:        broker.BrokerRequestID(uuid.New().String()),
		DataCid:   c,
		Status:    broker.RequestBatching,
		CreatedAt: now,
		UpdatedAt: now,
	}
	log.Debugf("saving broker request in store")
	if err := b.store.CreateBrokerRequest(ctx, br); err != nil {
		return broker.BrokerRequest{}, fmt.Errorf("saving broker request in store: %s", err)
	}

	// We notify the Packer that this BrokerRequest is ready to be considered.
	// We'll receive a call to `(*Broker).CreateStorageDeal(...)` which will contain
	// this BrokerRequest, and continue with the bidding process..
	log.Debugf("signaling packer")
	if err := b.packer.ReadyToPack(ctx, br.ID, br.DataCid); err != nil {
		return broker.BrokerRequest{}, fmt.Errorf("notifying packer of ready broker request: %s", err)
	}

	return br, nil
}

// CreatePrepared creates a broker request for prepared data.
func (b *Broker) CreatePrepared(
	ctx context.Context,
	payloadCid cid.Cid,
	pc broker.PreparedCAR) (broker.BrokerRequest, error) {
	log.Debugf("creating prepared car broker request")
	if !payloadCid.Defined() {
		return broker.BrokerRequest{}, ErrInvalidCid
	}

	now := time.Now()
	br := broker.BrokerRequest{
		ID:        broker.BrokerRequestID(uuid.New().String()),
		DataCid:   payloadCid,
		Status:    broker.RequestAuctioning,
		CreatedAt: now,
		UpdatedAt: now,
	}

	log.Debugf("creating prepared broker request")
	if err := b.store.CreateBrokerRequest(ctx, br); err != nil {
		return broker.BrokerRequest{}, fmt.Errorf("saving broker request in store: %s", err)
	}

	if pc.RepFactor == 0 {
		pc.RepFactor = int(b.conf.dealReplication)
	}

	filEpochDeadline, err := timeToFilEpoch(pc.Deadline)
	if err != nil {
		return broker.BrokerRequest{}, fmt.Errorf("calculating FIL epoch deadline: %s", err)
	}
	sd := broker.StorageDeal{
		ID:                 broker.StorageDealID(uuid.New().String()), // TODO(jsign): this should change to storaged providing the id.
		RepFactor:          pc.RepFactor,
		DealDuration:       int(b.conf.dealDuration),
		Status:             broker.StorageDealAuctioning,
		Sources:            pc.Sources,
		DisallowRebatching: true,
		FilEpochDeadline:   filEpochDeadline,

		// We fill what packer+piecer usually do.
		PayloadCid: payloadCid,
		PieceCid:   pc.PieceCid,
		PieceSize:  pc.PieceSize,
	}

	log.Debugf("creating prepared storage deal")
	if err := b.store.CreateStorageDeal(ctx, &sd, []broker.BrokerRequestID{br.ID}); err != nil {
		return broker.BrokerRequest{}, fmt.Errorf("creating storage deal: %w", err)
	}

	auctionID, err := b.auctioneer.ReadyToAuction(
		ctx,
		sd.ID,
		sd.PayloadCid,
		int(sd.PieceSize),
		sd.DealDuration,
		sd.RepFactor,
		b.conf.verifiedDeals,
		nil,
		sd.FilEpochDeadline,
		sd.Sources,
	)
	if err != nil {
		return broker.BrokerRequest{}, fmt.Errorf("signaling auctioneer to create auction: %s", err)
	}
	log.Debugf("created prepared auction %s", auctionID)

	return br, nil
}

// GetBrokerRequestInfo gets a BrokerRequest by id. If doesn't exist, it returns ErrNotFound.
func (b *Broker) GetBrokerRequestInfo(
	ctx context.Context,
	ID broker.BrokerRequestID) (broker.BrokerRequestInfo, error) {
	br, err := b.store.GetBrokerRequest(ctx, ID)
	if err == store.ErrNotFound {
		return broker.BrokerRequestInfo{}, ErrNotFound
	}
	if err != nil {
		return broker.BrokerRequestInfo{}, fmt.Errorf("get broker request from store: %s", err)
	}

	bri := broker.BrokerRequestInfo{
		BrokerRequest: br,
	}

	if br.StorageDealID != "" {
		deals, err := b.store.GetMinerDeals(ctx, br.StorageDealID)
		if err != nil {
			return broker.BrokerRequestInfo{}, fmt.Errorf("get storage-deal: %s", err)
		}
		for _, deal := range deals {
			if deal.DealID == 0 {
				continue
			}
			di := broker.BrokerRequestDeal{
				Miner:      deal.MinerAddr,
				DealID:     deal.DealID,
				Expiration: deal.DealExpiration,
			}
			bri.Deals = append(bri.Deals, di)
		}
	}

	return bri, nil
}

// CreateNewBatch creates a StorageDeal that contains multiple BrokerRequest.
func (b *Broker) CreateNewBatch(
	ctx context.Context,
	batchID broker.StorageDealID,
	batchCid cid.Cid,
	brids []broker.BrokerRequestID) (broker.StorageDealID, error) {
	if !batchCid.Defined() {
		return "", ErrInvalidCid
	}
	if len(brids) == 0 {
		return "", ErrEmptyGroup
	}
	for i := range brids {
		if len(brids[i]) == 0 {
			return "", fmt.Errorf("storage requests id can't be empty")
		}
	}

	cidURL, err := url.Parse(batchCid.String())
	if err != nil {
		return "", fmt.Errorf("creating cid url fragment: %s", err)
	}
	sd := broker.StorageDeal{
		ID:                 batchID,
		PayloadCid:         batchCid,
		RepFactor:          int(b.conf.dealReplication),
		DealDuration:       int(b.conf.dealDuration),
		Status:             broker.StorageDealPreparing,
		DisallowRebatching: false,
		FilEpochDeadline:   0,
		Sources: auction.Sources{
			CARURL: &auction.CARURL{
				URL: *b.conf.carExportURL.ResolveReference(cidURL),
			},
		},
	}

	if err := b.store.CreateStorageDeal(ctx, &sd, brids); err != nil {
		return "", fmt.Errorf("creating storage deal: %w", err)
	}
	log.Debugf("new storage-deal created with id %s", sd.ID)
	return sd.ID, nil
}

// NewBatchPrepared contains information of a prepared batch.
func (b *Broker) NewBatchPrepared(
	ctx context.Context,
	id broker.StorageDealID,
	dpr broker.DataPreparationResult,
) error {
	if id == "" {
		return fmt.Errorf("the storage deal id is empty")
	}
	if err := dpr.Validate(); err != nil {
		return fmt.Errorf("the data preparation result is invalid: %s", err)
	}

	sd, err := b.store.GetStorageDeal(ctx, id)
	if err != nil {
		return fmt.Errorf("get stoarge deal: %s", err)
	}

	log.Debugf("storage deal %s was prepared, signaling auctioneer...", id)
	// Signal the Auctioneer to create an auction. It will eventually call StorageDealAuctioned(..) to tell
	// us about who won things.
	auctionID, err := b.auctioneer.ReadyToAuction(
		ctx,
		id,
		sd.PayloadCid,
		int(dpr.PieceSize),
		sd.DealDuration,
		sd.RepFactor,
		b.conf.verifiedDeals,
		nil,
		sd.FilEpochDeadline,
		sd.Sources,
	)
	if err != nil {
		return fmt.Errorf("signaling auctioneer to create auction: %s", err)
	}

	if err := b.store.StorageDealToAuctioning(ctx, id, dpr.PieceCid, dpr.PieceSize); err != nil {
		return fmt.Errorf("saving piecer output in storage deal: %s", err)
	}

	log.Debugf("created auction %s", auctionID)
	return nil
}

// StorageDealProposalAccepted indicates that a miner has accepted a proposed deal.
func (b *Broker) StorageDealProposalAccepted(
	ctx context.Context,
	sdID broker.StorageDealID,
	miner string,
	proposal cid.Cid) error {
	log.Debugf("accepted proposal %s from miner %s, signaling auctioneer to start download for bidbot", proposal, miner)

	deals, err := b.store.GetMinerDeals(ctx, sdID)
	if err != nil {
		return fmt.Errorf("miner deals not found: %s", err)
	}

	var auctionID auction.AuctionID
	var bidID auction.BidID
	for _, deal := range deals {
		if deal.MinerAddr == miner {
			auctionID = deal.AuctionID
			bidID = deal.BidID
			break
		}
	}

	if bidID == "" {
		return fmt.Errorf("coudn't find mienr deal in storage-deal: %s", err)
	}

	log.Debugf("proposal accepted: %s %s %s", auctionID, bidID, proposal)
	if err := b.auctioneer.ProposalAccepted(ctx, auctionID, bidID, proposal); err != nil {
		return fmt.Errorf("signaling auctioneer about accepted proposal: %s", err)
	}

	return nil
}

// StorageDealAuctioned is called by the Auctioneer with the result of the StorageDeal auction.
func (b *Broker) StorageDealAuctioned(ctx context.Context, au broker.ClosedAuction) error {
	log.Debugf("storage deal %s was auctioned with %d winning bids", au.StorageDealID, len(au.WinningBids))

	if au.Status != broker.AuctionStatusFinalized {
		return errors.New("auction status should be final")
	}

	sd, err := b.store.GetStorageDeal(ctx, au.StorageDealID)
	if err != nil {
		return fmt.Errorf("storage deal not found: %s", err)
	}

	// If the auction failed, we didn't have at least 1 bid.
	if au.ErrorCause != "" {
		switch sd.DisallowRebatching {
		case false:
			// The batch can be rebatched. We switch the storage deal to error status,
			// and also signal the store to liberate the underlying broker requests to Pending.
			// This way they can be signaled to be re-batched.
			log.Debugf("the auction %s finalized with error %s, rebatching...", au.ID, au.ErrorCause)
			if err := b.errorStorageDealAndRebatch(ctx, au.StorageDealID, au.ErrorCause); err != nil {
				return fmt.Errorf("erroring storage deal and rebatching: %s", err)
			}
		case true:
			if sd.AuctionRetries >= b.conf.auctionMaxRetries {
				log.Warnf("stop prepared SD %s re-auctioning after %d retries", sd.ID, sd.AuctionRetries)
				_, err := b.store.StorageDealError(ctx, sd.ID, causeMaxAuctionRetries, false)
				if err != nil {
					return fmt.Errorf("moving storage deal to error status: %s", err)
				}
				return nil
			}

			// The batch can't be rebatched, since it's a prepared CAR file.
			// We simply foce to create a new auction.
			auctionID, err := b.auctioneer.ReadyToAuction(
				ctx,
				sd.ID,
				sd.PayloadCid,
				int(sd.PieceSize),
				sd.DealDuration,
				sd.RepFactor,
				b.conf.verifiedDeals,
				nil,
				sd.FilEpochDeadline,
				sd.Sources,
			)
			if err != nil {
				return fmt.Errorf("signaling auctioneer to re-create auction for prepared data: %s", err)
			}
			log.Debugf("re-created new auction for prepared prepared data: %s", auctionID)
			if err := b.store.CountAuctionRetry(ctx, sd.ID); err != nil {
				return fmt.Errorf("increasing auction retry: %s", err)
			}
		}
		return nil
	}

	if len(au.WinningBids) == 0 {
		return fmt.Errorf("winning bids list is empty")
	}

	if len(au.WinningBids) > sd.RepFactor {
		return fmt.Errorf("%d winning-bids when the rep. factor is %d", len(au.WinningBids), sd.RepFactor)
	}

	if sd.Status != broker.StorageDealAuctioning && sd.Status != broker.StorageDealDealMaking {
		log.Errorf("auction finished for a storage-deal in an unexpected status %s", sd.Status)
		return fmt.Errorf("storage-deal isn't in expected status: %s", sd.Status)
	}

	deltaRepFactor := sd.RepFactor - len(au.WinningBids)
	// If we have less than expected winning bids, and we already auctioned max amount of times
	// we error.
	if deltaRepFactor > 0 && sd.AuctionRetries >= b.conf.auctionMaxRetries {
		log.Warnf("stop partial winning auction for %s re-auctioning after %d retries", sd.ID, sd.AuctionRetries)
		_, err := b.store.StorageDealError(ctx, sd.ID, causeMaxAuctionRetries, false)
		if err != nil {
			return fmt.Errorf("moving storage deal to error status: %s", err)
		}
		return nil
	}
	// 1. We tell dealerd to start making deals with the miners from winning bids.
	ads := dealer.AuctionDeals{
		StorageDealID: sd.ID,
		PayloadCid:    sd.PayloadCid,
		PieceCid:      sd.PieceCid,
		PieceSize:     sd.PieceSize,
		Duration:      au.DealDuration,
		Targets:       make([]dealer.AuctionDealsTarget, len(au.WinningBids)),
	}

	var i int
	for _, bid := range au.WinningBids {
		ads.Targets[i] = dealer.AuctionDealsTarget{
			Miner:               bid.MinerAddr,
			PricePerGiBPerEpoch: bid.Price,
			StartEpoch:          bid.StartEpoch,
			Verified:            au.DealVerified,
			FastRetrieval:       bid.FastRetrieval,
		}
		i++
	}

	log.Debug("signaling dealer...")
	if err := b.dealer.ReadyToCreateDeals(ctx, ads); err != nil {
		return fmt.Errorf("signaling dealer to execute winning bids: %s", err)
	}

	if err := b.store.AddMinerDeals(ctx, au); err != nil {
		return fmt.Errorf("adding miner deals: %s", err)
	}

	// 2. There's a chance that the winning bids from the auction are less than what we specified
	//    in the replication factor.
	if deltaRepFactor > 0 {
		// We exclude all previous/currently miners involved with this data, this includes:
		// - Miners that already confirmed a deal on-chain.
		// - Miners that errored making the deal.
		// - Miners that are in progress of making the deal.
		// Saying it differently: we want to create an auction and expect *new* miners to jump
		// in to satisfy the missing rep factor.
		var excludedMiners []string
		deals, err := b.store.GetMinerDeals(ctx, sd.ID)
		if err != nil {
			return fmt.Errorf("getting miner deals: %s", err)
		}
		for _, deal := range deals {
			excludedMiners = append(excludedMiners, deal.MinerAddr)
		}
		for _, deal := range ads.Targets {
			excludedMiners = append(excludedMiners, deal.Miner)
		}
		log.Infof("creating new auction for %d/%d missing bids", deltaRepFactor, len(au.WinningBids))
		_, err = b.auctioneer.ReadyToAuction(
			ctx,
			sd.ID,
			sd.PayloadCid,
			int(sd.PieceSize),
			sd.DealDuration,
			deltaRepFactor,
			b.conf.verifiedDeals,
			excludedMiners,
			sd.FilEpochDeadline,
			sd.Sources,
		)
		if err != nil {
			return fmt.Errorf("creating new auction for missing bids %d: %s", deltaRepFactor, err)
		}

		if err := b.store.CountAuctionRetry(ctx, sd.ID); err != nil {
			return fmt.Errorf("increasing auction retry: %s", err)
		}
	}

	return nil
}

// StorageDealFinalizedDeal report a deal that reached final status in the Filecoin network.
func (b *Broker) StorageDealFinalizedDeal(ctx context.Context, fad broker.FinalizedAuctionDeal) error {
	log.Debug("received a finalized deal...")

	// 1. Save the finalized deal in the storage-deal (successful or not)
	if err := b.store.SaveFinalizedDeal(ctx, fad); err != nil {
		return fmt.Errorf("adding finalized info to the store: %s", err)
	}

	sd, err := b.store.GetStorageDeal(ctx, fad.StorageDealID)
	if err != nil {
		return fmt.Errorf("get storage deal: %s", err)
	}
	deals, err := b.store.GetMinerDeals(ctx, sd.ID)
	if err != nil {
		return fmt.Errorf("adding miner deals: %s", err)
	}

	// 1.a If the finalized deal errored, we should create a new auction with replication factor 1,
	//     and we're done.
	if fad.ErrorCause != "" {
		var excludedMiners []string
		for _, deal := range deals {
			excludedMiners = append(excludedMiners, deal.MinerAddr)
		}
		log.Infof("creating new auction for failed deal with miner %s", fad.Miner)

		_, err = b.auctioneer.ReadyToAuction(
			ctx,
			sd.ID,
			sd.PayloadCid,
			int(sd.PieceSize),
			sd.DealDuration,
			1,
			b.conf.verifiedDeals,
			excludedMiners,
			sd.FilEpochDeadline,
			sd.Sources,
		)
		if err != nil {
			return fmt.Errorf("creating new auction for errored deal: %s", err)
		}
		if err := b.store.CountAuctionRetry(ctx, sd.ID); err != nil {
			return fmt.Errorf("increasing auction retry: %s", err)
		}
		return nil
	}

	// 2. We got and saved the successful deal. Now we check if this was the last one that we were
	//    waiting to be confirmed to meet the rep factor.
	var numConfirmedDeals int
	for _, deal := range deals {
		if deal.ErrorCause == "" && deal.DealID > 0 {
			numConfirmedDeals++
		}
	}

	// Are we done?
	if numConfirmedDeals == sd.RepFactor {
		if err := b.store.StorageDealSuccess(ctx, sd.ID); err != nil {
			return fmt.Errorf("moving to storage deal success: %s", err)
		}
	}

	return nil
}

// GetStorageDeal gets an existing storage deal. If the storage deal doesn't exists, it returns
// ErrNotFound.
func (b *Broker) GetStorageDeal(ctx context.Context, id broker.StorageDealID) (broker.StorageDeal, error) {
	sd, err := b.store.GetStorageDeal(ctx, id)
	if err == ErrNotFound {
		return broker.StorageDeal{}, ErrNotFound
	}
	if err != nil {
		return broker.StorageDeal{}, fmt.Errorf("get storage deal from store: %s", err)
	}

	return *sd, nil
}

// errorStorageDealAndRebatch does:
// - Move the StorageDeal to error.
// - Signal Packerd that all the underlying broker requests of the storage deal are ready to
//   be batched again.
// - Move the underlying broker requests of the storage deal to Batching.
// - Create an async job to unpin the Cid of the batch (since won't be relevant anymore).
func (b *Broker) errorStorageDealAndRebatch(ctx context.Context, id broker.StorageDealID, errCause string) error {
	brs, err := b.store.StorageDealError(ctx, id, errCause, true)
	if err != nil {
		return fmt.Errorf("moving storage deal to error status: %s", err)
	}

	log.Debugf("erroring storage deal %s, rebatching %d broker-requests: %s", id, len(brs), errCause)
	for i := range brs {
		br, err := b.store.GetBrokerRequest(ctx, brs[i])
		if err != nil {
			return fmt.Errorf("get broker request: %s", err)
		}
		if err := b.packer.ReadyToPack(ctx, br.ID, br.DataCid); err != nil {
			return fmt.Errorf("notifying packer of ready broker request: %s", err)
		}
	}

	return nil
}

// Close closes the broker.
func (b *Broker) Close() error {
	log.Info("closing broker...")
	b.onceClose.Do(func() {
		b.daemonCancelCtx()
		<-b.daemonClosed
	})
	return nil
}

func timeToFilEpoch(t time.Time) (uint64, error) {
	deadline := (t.Unix() - filecoinGenesisUnixEpoch) / 30
	if deadline <= 0 {
		return 0, fmt.Errorf("the provided deadline %s is before genesis", t)
	}

	return uint64(deadline), nil
}
