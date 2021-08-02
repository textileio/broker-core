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
	"github.com/textileio/broker-core/msgbroker"
)

const (
	filecoinGenesisUnixEpoch = 1598306400
)

var (
	// ErrNotFound is returned when the storage request doesn't exist.
	ErrNotFound = errors.New("storage request not found")
	// ErrInvalidCid is returned when the Cid is undefined.
	ErrInvalidCid = errors.New("the cid can't be undefined")
	// ErrEmptyGroup is returned when an empty batch group
	// is received.
	ErrEmptyGroup = errors.New("the batch group is empty")
	// ErrOperationIDExists indicates that the opeation already exists,
	// which basically means the function being called with the same data again.
	ErrOperationIDExists = errors.New("operation-id already exists")

	log = logger.Logger("broker")
)

// Broker creates and tracks request to store Cids in
// the Filecoin network.
type Broker struct {
	store      *store.Store
	chainAPI   chainapi.ChainAPI
	ipfsClient *httpapi.HttpApi
	mb         msgbroker.MsgBroker

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
	mb msgbroker.MsgBroker,
	opts ...Option,
) (*Broker, error) {
	s, err := store.New(postgresURI)
	if err != nil {
		return nil, fmt.Errorf("initializing storage request store: %s", err)
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
func (b *Broker) Create(ctx context.Context, dataCid cid.Cid, origin string) (broker.StorageRequest, error) {
	if !dataCid.Defined() {
		return broker.StorageRequest{}, ErrInvalidCid
	}

	now := time.Now()
	srID, err := b.newID()
	if err != nil {
		return broker.StorageRequest{}, fmt.Errorf("generating id: %s", err)
	}
	br := broker.StorageRequest{
		ID:        broker.StorageRequestID(srID),
		DataCid:   dataCid,
		Status:    broker.RequestBatching,
		Origin:    origin,
		CreatedAt: now,
		UpdatedAt: now,
	}
	log.Debugf("creating storage-request %s with data-cid %s from origin %s", srID, dataCid, origin)
	if err := b.store.CreateStorageRequest(ctx, br); err != nil {
		return broker.StorageRequest{}, fmt.Errorf("create storage request in store: %s", err)
	}

	log.Debugf("publishing storage-request %s with data-cid %s in ready-to-batch topic", srID, dataCid)
	dataCids := []msgbroker.ReadyToBatchData{{StorageRequestID: br.ID, DataCid: br.DataCid, Origin: origin}}
	if err := msgbroker.PublishMsgReadyToBatch(ctx, b.mb, dataCids); err != nil {
		return broker.StorageRequest{}, fmt.Errorf("publishing to msg broker: %s", err)
	}

	return br, nil
}

// CreatePrepared creates a storage request for prepared data.
func (b *Broker) CreatePrepared(
	ctx context.Context,
	payloadCid cid.Cid,
	pc broker.PreparedCAR,
	meta broker.BatchMetadata) (br broker.StorageRequest, err error) {
	log.Debugf("creating prepared car storage request")
	if !payloadCid.Defined() {
		return broker.StorageRequest{}, ErrInvalidCid
	}
	if meta.Origin == "" {
		return broker.StorageRequest{}, fmt.Errorf("origin is empty")
	}

	now := time.Now()
	srID, err := b.newID()
	if err != nil {
		return broker.StorageRequest{}, fmt.Errorf("generating id: %s", err)
	}
	br = broker.StorageRequest{
		ID:        broker.StorageRequestID(srID),
		DataCid:   payloadCid,
		Status:    broker.RequestAuctioning,
		Origin:    meta.Origin,
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
	batchID, err := b.newID()
	if err != nil {
		return broker.StorageRequest{}, fmt.Errorf("generating batch id: %s", err)
	}
	ba := broker.Batch{
		ID:                 broker.BatchID(batchID),
		RepFactor:          pc.RepFactor,
		DealDuration:       int(b.conf.dealDuration),
		Status:             broker.BatchStatusAuctioning,
		Sources:            pc.Sources,
		DisallowRebatching: true,
		FilEpochDeadline:   filEpochDeadline,
		Origin:             meta.Origin,
		Tags:               meta.Tags,

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

	log.Debugf("creating prepared storage-request")
	if err = b.store.CreateStorageRequest(ctx, br); err != nil {
		return broker.StorageRequest{}, fmt.Errorf("saving storage request in store: %s", err)
	}

	log.Debugf("creating prepared batch payload-cid %s, batch %s", payloadCid, batchID)
	if err := b.store.CreateBatch(ctx, &ba, []broker.StorageRequestID{br.ID}); err != nil {
		return broker.StorageRequest{}, fmt.Errorf("creating batch: %w", err)
	}

	auctionID, err := b.newID()
	if err != nil {
		return broker.StorageRequest{}, fmt.Errorf("generating auction id: %s", err)
	}
	if err = msgbroker.PublishMsgReadyToAuction(
		ctx,
		b.mb,
		auction.ID(auctionID),
		ba.ID,
		ba.PayloadCid,
		int(ba.PieceSize),
		ba.DealDuration,
		ba.RepFactor,
		b.conf.verifiedDeals,
		nil,
		ba.FilEpochDeadline,
		ba.Sources,
	); err != nil {
		return broker.StorageRequest{}, fmt.Errorf("publish ready to auction %s: %s", ba.ID, err)
	}
	log.Debugf("created prepared auction %s for batch %s", auctionID, batchID)

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
		return broker.StorageRequestInfo{}, fmt.Errorf("get storage-request from store: %s", err)
	}

	bri := broker.StorageRequestInfo{
		StorageRequest: br,
	}

	if br.BatchID != "" {
		deals, err := b.store.GetDeals(ctx, br.BatchID)
		if err != nil {
			return broker.StorageRequestInfo{}, fmt.Errorf("get deals: %s", err)
		}
		for _, deal := range deals {
			if deal.DealID == 0 {
				continue
			}
			di := broker.StorageRequestDeal{
				StorageProviderID: deal.StorageProviderID,
				DealID:            deal.DealID,
				Expiration:        deal.DealExpiration,
			}
			bri.Deals = append(bri.Deals, di)
		}
	}
	log.Debugf("get storage-request info with id %s, batch %s", brID, br.BatchID)

	return bri, nil
}

// CreateNewBatch creates a Batch that contains multiple StorageRequest.
func (b *Broker) CreateNewBatch(
	ctx context.Context,
	batchID broker.BatchID,
	batchCid cid.Cid,
	brIDs []broker.StorageRequestID,
	origin string) (broker.BatchID, error) {
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
	ba := broker.Batch{
		ID:                 batchID,
		PayloadCid:         batchCid,
		RepFactor:          int(b.conf.dealReplication),
		DealDuration:       int(b.conf.dealDuration),
		Status:             broker.BatchStatusPreparing,
		Origin:             origin,
		DisallowRebatching: false,
		FilEpochDeadline:   0,
		Sources: auction.Sources{
			CARURL: &auction.CARURL{
				URL: *b.conf.carExportURL.ResolveReference(cidURL),
			},
		},
	}

	if err := b.store.CreateBatch(ctx, &ba, brIDs); err != nil {
		return "", fmt.Errorf("creating batch: %w", err)
	}
	log.Debugf("new batch %s created with batchCid %s with %d broker-requests", ba.ID, batchCid, len(brIDs))

	return ba.ID, nil
}

// NewBatchPrepared contains information of a prepared batch.
func (b *Broker) NewBatchPrepared(
	ctx context.Context,
	batchID broker.BatchID,
	dpr broker.DataPreparationResult,
) (err error) {
	if batchID == "" {
		return fmt.Errorf("the batch id is empty")
	}
	if err := dpr.Validate(); err != nil {
		return fmt.Errorf("the data preparation result is invalid: %s", err)
	}

	auctionID, err := b.newID()
	if err != nil {
		return fmt.Errorf("generating auction id: %s", err)
	}

	ctx, err = b.store.CtxWithTx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		err = b.store.FinishTxForCtx(ctx, err)
	}()

	ba, err := b.store.GetBatch(ctx, batchID)
	if err != nil {
		return fmt.Errorf("get batch: %s", err)
	}

	// Take care of correct state transitions.
	switch ba.Status {
	case broker.BatchStatusPreparing:
		if ba.PieceCid.Defined() || ba.PieceSize > 0 {
			return fmt.Errorf("piece cid and size should be empty: %s %d", ba.PieceCid, ba.PieceSize)
		}
	case broker.BatchStatusAuctioning:
		if ba.PieceCid != dpr.PieceCid {
			return fmt.Errorf("piececid different from registered: %s %s", ba.PieceCid, dpr.PieceCid)
		}
		if ba.PieceSize != dpr.PieceSize {
			return fmt.Errorf("piece size different from registered: %d %d", ba.PieceSize, dpr.PieceSize)
		}
		// auction is in process, do nothing
		return nil
	default:
		return fmt.Errorf("wrong storage request status transition, tried moving to %s", ba.Status)
	}

	if err := b.store.BatchToAuctioning(ctx, batchID, dpr.PieceCid, dpr.PieceSize); err != nil {
		return fmt.Errorf("saving piecer output in batch: %s", err)
	}

	if err := msgbroker.PublishMsgReadyToAuction(
		ctx,
		b.mb,
		auction.ID(auctionID),
		batchID,
		ba.PayloadCid,
		int(dpr.PieceSize),
		ba.DealDuration,
		ba.RepFactor,
		b.conf.verifiedDeals,
		nil,
		ba.FilEpochDeadline,
		ba.Sources,
	); err != nil {
		return fmt.Errorf("publishing ready to create auction msg: %s", err)
	}

	log.Debugf("batch %s was prepared piece-cid %s piece-size %d in auction %s...",
		batchID, dpr.PieceCid, dpr.PieceSize, auctionID)

	return nil
}

// BatchAuctioned is called by the Auctioneer with the result of the Batch auction.
func (b *Broker) BatchAuctioned(ctx context.Context, opID msgbroker.OperationID, au broker.ClosedAuction) (err error) {
	log.Debugf("auction %s closed batch %s  with %d winning bids, errorCause %s",
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

	if exists, err := b.store.OperationExists(ctx, string(opID)); err != nil {
		return fmt.Errorf("checking if operation exists: %s", err)
	} else if exists {
		return ErrOperationIDExists
	}

	ba, err := b.store.GetBatch(ctx, au.BatchID)
	if err != nil {
		return fmt.Errorf("get batch: %s", err)
	}

	if ba.Status != broker.BatchStatusAuctioning {
		return fmt.Errorf("batch isn't in expected status: %s", ba.Status)
	}

	// If the auction failed, we didn't have at least 1 bid.
	if au.ErrorCause != "" {
		switch ba.DisallowRebatching {
		case false:
			// The batch can be rebatched. We switch the batch to error status,
			// and also signal the store to liberate the underlying storage requests to Pending.
			// This way they can be signaled to be re-batched.
			log.Debugf("auction %s closed with error %s, rebatching...", au.ID, au.ErrorCause)
			if err := b.errorBatchAndRebatch(ctx, au.BatchID, au.ErrorCause); err != nil {
				return fmt.Errorf("erroring batch and rebatching: %s", err)
			}
		case true:
			_, err := b.store.BatchError(ctx, ba.ID, au.ErrorCause, false)
			if err != nil {
				return fmt.Errorf("moving batch to error status: %s", err)
			}
		}
		return nil
	}

	if len(au.WinningBids) != ba.RepFactor {
		return fmt.Errorf("winning bids expected %d, got %d", ba.RepFactor, len(au.WinningBids))
	}

	adID, err := b.newID()
	if err != nil {
		return fmt.Errorf("generating auction deal id: %s", err)
	}

	ads := dealer.AuctionDeals{
		ID:         adID,
		BatchID:    ba.ID,
		PayloadCid: ba.PayloadCid,
		PieceCid:   ba.PieceCid,
		PieceSize:  ba.PieceSize,
		Duration:   au.DealDuration,
		Proposals:  make([]dealer.Proposal, len(au.WinningBids)),
	}

	var i int
	for id, bid := range au.WinningBids {
		ads.Proposals[i] = dealer.Proposal{
			StorageProviderID:   bid.StorageProviderID,
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
		return fmt.Errorf("adding storage-provider deals: %s", err)
	}

	log.Debugf("publishing ready to create deals for auction %s, batch", au.ID, au.BatchID)
	if err := msgbroker.PublishMsgReadyToCreateDeals(ctx, b.mb, ads); err != nil {
		return fmt.Errorf("sending ready to create deals msg to msgbroker: %s", err)
	}

	return nil
}

// BatchFinalizedDeal report a deal that reached final status in the Filecoin network.
func (b *Broker) BatchFinalizedDeal(ctx context.Context,
	opID msgbroker.OperationID,
	fad broker.FinalizedDeal) (err error) {
	log.Debugf("finalized deal from auction (%s, %s), batch %s, deal-id %s, storage-provider %s",
		fad.AuctionID, fad.BidID, fad.BatchID, fad.DealID, fad.StorageProviderID)

	ctx, err = b.store.CtxWithTx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		err = b.store.FinishTxForCtx(ctx, err)
	}()

	if exists, err := b.store.OperationExists(ctx, string(opID)); err != nil {
		return fmt.Errorf("checking if operation exists: %s", err)
	} else if exists {
		return ErrOperationIDExists
	}

	// 1. Save the finalized deal in the batch (successful or not)
	if err := b.store.SaveDeals(ctx, fad); err != nil {
		return fmt.Errorf("adding finalized info to the store: %s", err)
	}

	ba, err := b.store.GetBatch(ctx, fad.BatchID)
	if err != nil {
		return fmt.Errorf("get batch: %s", err)
	}
	deals, err := b.store.GetDeals(ctx, ba.ID)
	if err != nil {
		return fmt.Errorf("adding storage-provider deals: %s", err)
	}

	// 1.a If the finalized deal errored, we should create a new auction with replication factor 1,
	//     and we're done.
	if fad.ErrorCause != "" {
		var excludedStorageProviders []string
		for _, deal := range deals {
			excludedStorageProviders = append(excludedStorageProviders, deal.StorageProviderID)
		}
		log.Infof("creating new auction for failed deal with storage-provider %s", fad.StorageProviderID)

		auctionID, err := b.newID()
		if err != nil {
			return fmt.Errorf("generating auction id: %s", err)
		}
		if err := msgbroker.PublishMsgReadyToAuction(
			ctx,
			b.mb,
			auction.ID(auctionID),
			ba.ID,
			ba.PayloadCid,
			int(ba.PieceSize),
			ba.DealDuration,
			1,
			b.conf.verifiedDeals,
			excludedStorageProviders,
			ba.FilEpochDeadline,
			ba.Sources,
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

	log.Debugf("auction %s, batch %s finalized deal: %d/%d", fad.AuctionID, ba.ID, numConfirmedDeals, ba.RepFactor)
	// Are we done?
	if numConfirmedDeals == ba.RepFactor {
		if err := b.store.BatchSuccess(ctx, ba.ID); err != nil {
			return fmt.Errorf("moving to batch success: %s", err)
		}
		log.Debugf("batch %s success", ba.ID)
	}

	return nil
}

// GetBatch gets an existing batch. If the batch doesn't exists, it returns
// ErrNotFound.
func (b *Broker) GetBatch(ctx context.Context, id broker.BatchID) (broker.Batch, error) {
	sd, err := b.store.GetBatch(ctx, id)
	if err == ErrNotFound {
		return broker.Batch{}, ErrNotFound
	}
	if err != nil {
		return broker.Batch{}, fmt.Errorf("get batch from store: %s", err)
	}

	return *sd, nil
}

// errorBatchAndRebatch does:
// - Move the Batch to error.
// - Signal Packerd that all the underlying storage requests of the batch are ready to
//   be batched again.
// - Move the underlying storage requests of the batch to Batching.
// - Create an async job to unpin the Cid of the batch (since won't be relevant anymore).
func (b *Broker) errorBatchAndRebatch(ctx context.Context, id broker.BatchID, errCause string) error {
	brs, err := b.store.BatchError(ctx, id, errCause, true)
	if err != nil {
		return fmt.Errorf("moving batch to error status: %s", err)
	}

	log.Debugf("erroring batch %s, rebatching %d broker-requests: %s", id, len(brs), errCause)
	dataCids := make([]msgbroker.ReadyToBatchData, len(brs))
	for i := range brs {
		br, err := b.store.GetStorageRequest(ctx, brs[i])
		if err != nil {
			return fmt.Errorf("get storage request: %s", err)
		}
		dataCids[i] = msgbroker.ReadyToBatchData{StorageRequestID: br.ID, DataCid: br.DataCid, Origin: br.Origin}
	}
	if err := msgbroker.PublishMsgReadyToBatch(ctx, b.mb, dataCids); err != nil {
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
