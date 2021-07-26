package broker

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	"github.com/oklog/ulid/v2"
	logger "github.com/textileio/go-log/v2"
	"go.opentelemetry.io/otel/metric"

	"github.com/textileio/bidbot/lib/auction"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/chainapi"
	"github.com/textileio/broker-core/cmd/brokerd/store"
	"github.com/textileio/broker-core/dealer"
	mbroker "github.com/textileio/broker-core/msgbroker"
)

const (
	filecoinGenesisUnixEpoch = 1598306400
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
	chainAPI   chainapi.ChainAPI
	ipfsClient *httpapi.HttpApi
	mb         mbroker.MsgBroker

	conf config

	onceClose       sync.Once
	daemonCtx       context.Context
	daemonCancelCtx context.CancelFunc
	daemonClosed    chan struct{}

	lock    sync.Mutex
	entropy *ulid.MonotonicEntropy

	metricUnpinTotal        metric.Int64Counter
	statTotalRecursivePins  int64
	metricRecursivePinCount metric.Int64ValueObserver
}

// New creates a Broker backed by the provided `ds`.
func New(
	postgresURI string,
	chainAPI chainapi.ChainAPI,
	ipfsClient *httpapi.HttpApi,
	mb mbroker.MsgBroker,
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
		chainAPI:   chainAPI,
		ipfsClient: ipfsClient,
		mb:         mb,

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

// Create creates a new StorageRequest with the provided Cid and
// Metadata configuration.
func (b *Broker) Create(ctx context.Context, dataCid cid.Cid) (broker.StorageRequest, error) {
	if !dataCid.Defined() {
		return broker.StorageRequest{}, ErrInvalidCid
	}

	now := time.Now()
	brID, err := b.newID()
	if err != nil {
		return broker.StorageRequest{}, fmt.Errorf("generating id: %s", err)
	}
	br := broker.StorageRequest{
		ID:        broker.StorageRequestID(brID),
		DataCid:   dataCid,
		Status:    broker.RequestBatching,
		CreatedAt: now,
		UpdatedAt: now,
	}
	log.Debugf("creating broker-request %s with dataCid %s", brID, dataCid)
	if err := b.store.CreateStorageRequest(ctx, br); err != nil {
		return broker.StorageRequest{}, fmt.Errorf("create broker request in store: %s", err)
	}

	log.Debugf("publishing broker-request %s with dataCid %s in ready-to-batch topic", brID, dataCid)
	dataCids := []mbroker.ReadyToBatchData{{StorageRequestID: br.ID, DataCid: br.DataCid}}
	if err := mbroker.PublishMsgReadyToBatch(ctx, b.mb, dataCids); err != nil {
		return broker.StorageRequest{}, fmt.Errorf("publishing to msg broker: %s", err)
	}

	return br, nil
}

// CreatePrepared creates a broker request for prepared data.
func (b *Broker) CreatePrepared(
	ctx context.Context,
	payloadCid cid.Cid,
	pc broker.PreparedCAR) (br broker.StorageRequest, err error) {
	log.Debugf("creating prepared car broker request")
	if !payloadCid.Defined() {
		return broker.StorageRequest{}, ErrInvalidCid
	}

	now := time.Now()
	brID, err := b.newID()
	if err != nil {
		return broker.StorageRequest{}, fmt.Errorf("generating id: %s", err)
	}
	br = broker.StorageRequest{
		ID:        broker.StorageRequestID(brID),
		DataCid:   payloadCid,
		Status:    broker.RequestAuctioning,
		CreatedAt: now,
		UpdatedAt: now,
	}

	if pc.RepFactor == 0 {
		pc.RepFactor = int(b.conf.dealReplication)
	}

	filEpochDeadline, err := timeToFilEpoch(pc.Deadline)
	if err != nil {
		return broker.StorageRequest{}, fmt.Errorf("calculating FIL epoch deadline: %s", err)
	}
	sdID, err := b.newID()
	if err != nil {
		return broker.StorageRequest{}, fmt.Errorf("generating storage deal id: %s", err)
	}
	sd := broker.Batch{
		// TODO(jsign): this might change depending if gRPC is still used for this API.
		ID:                 broker.BatchID(sdID),
		RepFactor:          pc.RepFactor,
		DealDuration:       int(b.conf.dealDuration),
		Status:             broker.BatchStatusAuctioning,
		Sources:            pc.Sources,
		DisallowRebatching: true,
		FilEpochDeadline:   filEpochDeadline,

		// We fill what packer+piecer usually do.
		PayloadCid: payloadCid,
		PieceCid:   pc.PieceCid,
		PieceSize:  pc.PieceSize,
	}

	ctx, err = b.store.CtxWithTx(ctx)
	if err != nil {
		return broker.StorageRequest{}, err
	}
	defer func() {
		err = b.store.FinishTxForCtx(ctx, err)
	}()

	log.Debugf("creating prepared broker request")
	if err = b.store.CreateStorageRequest(ctx, br); err != nil {
		return broker.StorageRequest{}, fmt.Errorf("saving broker request in store: %s", err)
	}

	log.Debugf("creating prepared broker-request payload-cid %s, storage-deal %s", payloadCid, sdID)
	if err := b.store.CreateBatch(ctx, &sd, []broker.StorageRequestID{br.ID}); err != nil {
		return broker.StorageRequest{}, fmt.Errorf("creating storage deal: %w", err)
	}

	auctionID, err := b.newID()
	if err != nil {
		return broker.StorageRequest{}, fmt.Errorf("generating auction id: %s", err)
	}
	if err = mbroker.PublishMsgReadyToAuction(
		ctx,
		b.mb,
		auction.AuctionID(auctionID),
		sd.ID,
		sd.PayloadCid,
		int(sd.PieceSize),
		sd.DealDuration,
		sd.RepFactor,
		b.conf.verifiedDeals,
		nil,
		sd.FilEpochDeadline,
		sd.Sources,
	); err != nil {
		return broker.StorageRequest{}, fmt.Errorf("publish ready to auction %s: %s", sd.ID, err)
	}
	log.Debugf("created prepared auction %s for storage-deal %s", auctionID, sdID)

	return br, nil
}

// GetStorageRequestInfo gets a StorageRequest by id. If doesn't exist, it returns ErrNotFound.
func (b *Broker) GetStorageRequestInfo(
	ctx context.Context,
	brID broker.StorageRequestID) (broker.StorageRequestInfo, error) {
	br, err := b.store.GetStorageRequest(ctx, brID)
	if err == store.ErrNotFound {
		return broker.StorageRequestInfo{}, ErrNotFound
	}
	if err != nil {
		return broker.StorageRequestInfo{}, fmt.Errorf("get broker request from store: %s", err)
	}

	bri := broker.StorageRequestInfo{
		StorageRequest: br,
	}

	if br.BatchID != "" {
		deals, err := b.store.GetDeals(ctx, br.BatchID)
		if err != nil {
			return broker.StorageRequestInfo{}, fmt.Errorf("get storage-deal: %s", err)
		}
		for _, deal := range deals {
			if deal.DealID == 0 {
				continue
			}
			di := broker.StorageRequestDeal{
				MinerID:    deal.MinerID,
				DealID:     deal.DealID,
				Expiration: deal.DealExpiration,
			}
			bri.Deals = append(bri.Deals, di)
		}
	}
	log.Debugf("get broker-request info with id %s, storage-deal %s", brID, br.BatchID)

	return bri, nil
}

// CreateNewBatch creates a Batch that contains multiple StorageRequest.
func (b *Broker) CreateNewBatch(
	ctx context.Context,
	batchID broker.BatchID,
	batchCid cid.Cid,
	brIDs []broker.StorageRequestID) (broker.BatchID, error) {
	if !batchCid.Defined() {
		return "", ErrInvalidCid
	}
	if len(brIDs) == 0 {
		return "", ErrEmptyGroup
	}
	for i := range brIDs {
		if len(brIDs[i]) == 0 {
			return "", fmt.Errorf("storage requests id can't be empty")
		}
	}

	cidURL, err := url.Parse(batchCid.String())
	if err != nil {
		return "", fmt.Errorf("creating cid url fragment: %s", err)
	}
	sd := broker.Batch{
		ID:                 batchID,
		PayloadCid:         batchCid,
		RepFactor:          int(b.conf.dealReplication),
		DealDuration:       int(b.conf.dealDuration),
		Status:             broker.BatchStatusPreparing,
		DisallowRebatching: false,
		FilEpochDeadline:   0,
		Sources: auction.Sources{
			CARURL: &auction.CARURL{
				URL: *b.conf.carExportURL.ResolveReference(cidURL),
			},
		},
	}

	if err := b.store.CreateBatch(ctx, &sd, brIDs); err != nil {
		return "", fmt.Errorf("creating storage deal: %w", err)
	}
	log.Debugf("new storage-deal %s created with batchCid %s with %d broker-requests", sd.ID, batchCid, len(brIDs))

	return sd.ID, nil
}

// NewBatchPrepared contains information of a prepared batch.
func (b *Broker) NewBatchPrepared(
	ctx context.Context,
	sdID broker.BatchID,
	dpr broker.DataPreparationResult,
) (err error) {
	if sdID == "" {
		return fmt.Errorf("the storage deal id is empty")
	}
	if err := dpr.Validate(); err != nil {
		return fmt.Errorf("the data preparation result is invalid: %s", err)
	}

	ctx, err = b.store.CtxWithTx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		err = b.store.FinishTxForCtx(ctx, err)
	}()

	sd, err := b.store.GetBatch(ctx, sdID)
	if err != nil {
		return fmt.Errorf("get stoarge deal: %s", err)
	}

	auctionID, err := b.newID()
	if err != nil {
		return fmt.Errorf("generating auction id: %s", err)
	}

	if err := b.store.BatchToAuctioning(ctx, sdID, dpr.PieceCid, dpr.PieceSize); err != nil {
		return fmt.Errorf("saving piecer output in storage deal: %s", err)
	}

	if err := mbroker.PublishMsgReadyToAuction(
		ctx,
		b.mb,
		auction.AuctionID(auctionID),
		sdID,
		sd.PayloadCid,
		int(dpr.PieceSize),
		sd.DealDuration,
		sd.RepFactor,
		b.conf.verifiedDeals,
		nil,
		sd.FilEpochDeadline,
		sd.Sources,
	); err != nil {
		return fmt.Errorf("publishing ready to create auction msg: %s", err)
	}

	log.Debugf("storage-deal %s was prepared piece-cid %s piece-size %d in auction %s...",
		sdID, dpr.PieceCid, dpr.PieceSize, auctionID)

	return nil
}

// BatchAuctioned is called by the Auctioneer with the result of the Batch auction.
func (b *Broker) BatchAuctioned(ctx context.Context, au broker.ClosedAuction) (err error) {
	log.Debugf("auction %s closed storage-deal %s  with %d winning bids, errorCause %s",
		au.ID, au.BatchID, len(au.WinningBids), au.ErrorCause)

	if au.Status != broker.AuctionStatusFinalized {
		return errors.New("auction status should be final")
	}

	ctx, err = b.store.CtxWithTx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		err = b.store.FinishTxForCtx(ctx, err)
	}()

	sd, err := b.store.GetBatch(ctx, au.BatchID)
	if err != nil {
		return fmt.Errorf("storage-deal not found: %s", err)
	}

	// If the auction failed, we didn't have at least 1 bid.
	if au.ErrorCause != "" {
		switch sd.DisallowRebatching {
		case false:
			// The batch can be rebatched. We switch the storage deal to error status,
			// and also signal the store to liberate the underlying broker requests to Pending.
			// This way they can be signaled to be re-batched.
			log.Debugf("auction %s closed with error %s, rebatching...", au.ID, au.ErrorCause)
			if err := b.errorBatchAndRebatch(ctx, au.BatchID, au.ErrorCause); err != nil {
				return fmt.Errorf("erroring storage deal and rebatching: %s", err)
			}
		case true:
			_, err := b.store.BatchError(ctx, sd.ID, au.ErrorCause, false)
			if err != nil {
				return fmt.Errorf("moving storage deal to error status: %s", err)
			}
		}
		return nil
	}

	if len(au.WinningBids) != sd.RepFactor {
		return fmt.Errorf("winning bids expected %d, got %d", sd.RepFactor, len(au.WinningBids))
	}

	if sd.Status != broker.BatchStatusAuctioning {
		return fmt.Errorf("storage-deal isn't in expected status: %s", sd.Status)
	}

	ads := dealer.AuctionDeals{
		BatchID:    sd.ID,
		PayloadCid: sd.PayloadCid,
		PieceCid:   sd.PieceCid,
		PieceSize:  sd.PieceSize,
		Duration:   au.DealDuration,
		Proposals:  make([]dealer.Proposal, len(au.WinningBids)),
	}

	var i int
	for id, bid := range au.WinningBids {
		ads.Proposals[i] = dealer.Proposal{
			MinerID:             bid.MinerID,
			PricePerGiBPerEpoch: bid.Price,
			StartEpoch:          bid.StartEpoch,
			Verified:            au.DealVerified,
			FastRetrieval:       bid.FastRetrieval,
			AuctionID:           au.ID,
			BidID:               id,
		}
		i++
	}

	if err := b.store.AddDeals(ctx, au); err != nil {
		return fmt.Errorf("adding miner deals: %s", err)
	}

	log.Debugf("publishing ready to create deals for auction %s, storage-deal", au.ID, au.BatchID)
	if err := mbroker.PublishMsgReadyToCreateDeals(ctx, b.mb, ads); err != nil {
		return fmt.Errorf("sending ready to create deals msg to msgbroker: %s", err)
	}

	return nil
}

// BatchFinalizedDeal report a deal that reached final status in the Filecoin network.
func (b *Broker) BatchFinalizedDeal(ctx context.Context, fad broker.FinalizedDeal) (err error) {
	log.Debugf("finalized deal from auction (%s, %s), storage-deal %s, deal-id %s, miner %s",
		fad.AuctionID, fad.BidID, fad.BatchID, fad.DealID, fad.Miner)

	ctx, err = b.store.CtxWithTx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		err = b.store.FinishTxForCtx(ctx, err)
	}()

	// 1. Save the finalized deal in the storage-deal (successful or not)
	if err := b.store.SaveDeals(ctx, fad); err != nil {
		return fmt.Errorf("adding finalized info to the store: %s", err)
	}

	sd, err := b.store.GetBatch(ctx, fad.BatchID)
	if err != nil {
		return fmt.Errorf("get storage deal: %s", err)
	}
	deals, err := b.store.GetDeals(ctx, sd.ID)
	if err != nil {
		return fmt.Errorf("adding miner deals: %s", err)
	}

	// 1.a If the finalized deal errored, we should create a new auction with replication factor 1,
	//     and we're done.
	if fad.ErrorCause != "" {
		var excludedMiners []string
		for _, deal := range deals {
			excludedMiners = append(excludedMiners, deal.MinerID)
		}
		log.Infof("creating new auction for failed deal with miner %s", fad.Miner)

		auctionID, err := b.newID()
		if err != nil {
			return fmt.Errorf("generating auction id: %s", err)
		}
		if err := mbroker.PublishMsgReadyToAuction(
			ctx,
			b.mb,
			auction.AuctionID(auctionID),
			sd.ID,
			sd.PayloadCid,
			int(sd.PieceSize),
			sd.DealDuration,
			1,
			b.conf.verifiedDeals,
			excludedMiners,
			sd.FilEpochDeadline,
			sd.Sources,
		); err != nil {
			return fmt.Errorf("creating new auction for errored deal: %s", err)
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

	log.Debugf("auction %s, storage-deal %s finalized deal: %d/%d", fad.AuctionID, sd.ID, numConfirmedDeals, sd.RepFactor)
	// Are we done?
	if numConfirmedDeals == sd.RepFactor {
		if err := b.store.BatchSuccess(ctx, sd.ID); err != nil {
			return fmt.Errorf("moving to storage deal success: %s", err)
		}
		log.Debugf("storage deal %s success", sd.ID)
	}

	return nil
}

// GetBatch gets an existing storage deal. If the storage deal doesn't exists, it returns
// ErrNotFound.
func (b *Broker) GetBatch(ctx context.Context, id broker.BatchID) (broker.Batch, error) {
	sd, err := b.store.GetBatch(ctx, id)
	if err == ErrNotFound {
		return broker.Batch{}, ErrNotFound
	}
	if err != nil {
		return broker.Batch{}, fmt.Errorf("get storage deal from store: %s", err)
	}

	return *sd, nil
}

// errorBatchAndRebatch does:
// - Move the Batch to error.
// - Signal Packerd that all the underlying broker requests of the storage deal are ready to
//   be batched again.
// - Move the underlying broker requests of the storage deal to Batching.
// - Create an async job to unpin the Cid of the batch (since won't be relevant anymore).
func (b *Broker) errorBatchAndRebatch(ctx context.Context, id broker.BatchID, errCause string) error {
	brs, err := b.store.BatchError(ctx, id, errCause, true)
	if err != nil {
		return fmt.Errorf("moving storage deal to error status: %s", err)
	}

	log.Debugf("erroring storage deal %s, rebatching %d broker-requests: %s", id, len(brs), errCause)
	dataCids := make([]mbroker.ReadyToBatchData, len(brs))
	for i := range brs {
		br, err := b.store.GetStorageRequest(ctx, brs[i])
		if err != nil {
			return fmt.Errorf("get broker request: %s", err)
		}
		dataCids[i] = mbroker.ReadyToBatchData{StorageRequestID: br.ID, DataCid: br.DataCid}
	}
	if err := mbroker.PublishMsgReadyToBatch(ctx, b.mb, dataCids); err != nil {
		return fmt.Errorf("publishing to msg broker: %s", err)
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

func (b *Broker) newID() (string, error) {
	b.lock.Lock()
	// Not deferring unlock since can be recursive.

	if b.entropy == nil {
		b.entropy = ulid.Monotonic(rand.Reader, 0)
	}
	id, err := ulid.New(ulid.Timestamp(time.Now().UTC()), b.entropy)
	if errors.Is(err, ulid.ErrMonotonicOverflow) {
		b.entropy = nil
		b.lock.Unlock()
		return b.newID()
	} else if err != nil {
		b.lock.Unlock()
		return "", fmt.Errorf("generating id: %v", err)
	}
	b.lock.Unlock()
	return strings.ToLower(id.String()), nil
}

func timeToFilEpoch(t time.Time) (uint64, error) {
	deadline := (t.Unix() - filecoinGenesisUnixEpoch) / 30
	if deadline <= 0 {
		return 0, fmt.Errorf("the provided deadline %s is before genesis", t)
	}

	return uint64(deadline), nil
}
