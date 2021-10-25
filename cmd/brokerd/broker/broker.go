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
	"github.com/textileio/broker-core/cmd/brokerd/store"
	"github.com/textileio/broker-core/common"
	"github.com/textileio/broker-core/dealer"
	"github.com/textileio/broker-core/metrics"
	"github.com/textileio/broker-core/msgbroker"
	"github.com/textileio/broker-core/storeutil"
)

const (
	filecoinGenesisUnixEpoch   = 1598306400
	msgAuctionDeadlineExceeded = "auction deadline exceeded"
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
	ipfsClient *httpapi.HttpApi
	mb         msgbroker.MsgBroker

	conf config

	onceClose       sync.Once
	daemonCtx       context.Context
	daemonCancelCtx context.CancelFunc
	daemonClosed    chan struct{}

	lock    sync.Mutex
	entropy *ulid.MonotonicEntropy

	metricStartedAuctions      metric.Int64Counter
	metricStartedBytes         metric.Int64Counter
	metricFinishedAuctions     metric.Int64Counter
	metricFinishedBytes        metric.Int64Counter
	metricRebatches            metric.Int64Counter
	metricRebatchedBytes       metric.Int64Counter
	metricBatchFinalityMinutes metric.Int64ValueRecorder
	metricReauctions           metric.Int64Counter
	metricReauctionedBytes     metric.Int64Counter

	metricUnpinTotal        metric.Int64Counter
	statTotalRecursivePins  int64
	metricRecursivePinCount metric.Int64ValueObserver
}

// New creates a new Broker.
func New(
	postgresURI string,
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

	ctx, cls := context.WithCancel(context.Background())
	b := &Broker{
		store:      s,
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
	meta broker.BatchMetadata,
	rw *broker.RemoteWallet) (sr broker.StorageRequest, err error) {
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
	sr = broker.StorageRequest{
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
	if pc.ProposalStartOffset < 0 {
		return broker.StorageRequest{}, fmt.Errorf("proposals start offset is negative")
	}
	proposalStartOffset := b.conf.defaultProposalStartOffset
	if pc.ProposalStartOffset != 0 {
		proposalStartOffset = pc.ProposalStartOffset
	}
	batchDeadline := time.Now().Add(b.conf.defaultBatchDeadline)
	if !pc.Deadline.IsZero() {
		if pc.Deadline.Before(batchDeadline) {
			log.Warnf("storage-request tighter than default auction duration %s", sr.ID)
		}
		batchDeadline = pc.Deadline
	}
	batchDeadlineEpoch, err := timeToFilEpoch(batchDeadline)
	if err != nil {
		return broker.StorageRequest{}, fmt.Errorf("calculating batch epoch deadline: %s", err)
	}

	batchID, err := b.newID()
	if err != nil {
		return broker.StorageRequest{}, fmt.Errorf("generating batch id: %s", err)
	}
	ba := &broker.Batch{
		ID:                  broker.BatchID(batchID),
		RepFactor:           pc.RepFactor,
		DealDuration:        int(b.conf.dealDuration),
		Status:              broker.BatchStatusAuctioning,
		Sources:             pc.Sources,
		DisallowRebatching:  true,
		FilEpochDeadline:    batchDeadlineEpoch,
		ProposalStartOffset: proposalStartOffset,
		Origin:              meta.Origin,
		Tags:                meta.Tags,
		Providers:           meta.Providers,

		PayloadCid:  payloadCid,
		PayloadSize: nil, // On prepared CARs this information isn't provided.
		PieceCid:    pc.PieceCid,
		PieceSize:   pc.PieceSize,
	}

	ctx, err = b.store.CtxWithTx(ctx)
	if err != nil {
		return broker.StorageRequest{}, err
	}
	defer func() {
		err = storeutil.FinishTxForCtx(ctx, err)
	}()

	log.Debugf("creating prepared storage-request")
	if err = b.store.CreateStorageRequest(ctx, sr); err != nil {
		return broker.StorageRequest{}, fmt.Errorf("saving storage request in store: %s", err)
	}

	log.Debugf("creating prepared batch payload-cid %s, batch %s", payloadCid, batchID)
	if err := b.store.CreateBatch(ctx, ba, []broker.StorageRequestID{sr.ID}, nil, rw); err != nil {
		return broker.StorageRequest{}, fmt.Errorf("creating batch: %w", err)
	}
	sr.BatchID = ba.ID

	proposalStartOffsetEpoch, err := timeToFilEpoch(time.Now().Add(proposalStartOffset))
	if err != nil {
		return broker.StorageRequest{}, fmt.Errorf("calculating auction epoch deadline: %s", err)
	}

	auctionID, err := b.startAuction(ctx, ba, rw, ba.RepFactor, ba.PieceSize, proposalStartOffsetEpoch)
	if err != nil {
		return broker.StorageRequest{}, err
	}
	log.Debugf("created prepared auction %s for batch %s", auctionID, batchID)
	return sr, nil
}

// GetStorageRequestInfo gets a StorageRequest by id. If doesn't exist, it returns ErrNotFound.
func (b *Broker) GetStorageRequestInfo(
	ctx context.Context,
	srID broker.StorageRequestID) (broker.StorageRequestInfo, error) {
	sr, err := b.store.GetStorageRequest(ctx, srID)
	if err == store.ErrNotFound {
		return broker.StorageRequestInfo{}, ErrNotFound
	}
	if err != nil {
		return broker.StorageRequestInfo{}, fmt.Errorf("get storage-request from store: %s", err)
	}

	sri := broker.StorageRequestInfo{
		StorageRequest: sr,
	}

	if sr.BatchID != "" {
		deals, err := b.store.GetDeals(ctx, sr.BatchID)
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
			sri.Deals = append(sri.Deals, di)
		}
	}
	log.Debugf("get storage-request info with id %s, batch %s", srID, sr.BatchID)

	return sri, nil
}

// CreateNewBatch creates a Batch that contains multiple StorageRequest.
func (b *Broker) CreateNewBatch(
	ctx context.Context,
	batchID broker.BatchID,
	batchCid cid.Cid,
	batchSize int64,
	brIDs []broker.StorageRequestID,
	origin string,
	manifest []byte,
	carURL *url.URL) (broker.BatchID, error) {
	if !batchCid.Defined() {
		return "", ErrInvalidCid
	}
	if len(brIDs) == 0 {
		return "", ErrEmptyGroup
	}
	if batchSize == 0 {
		return "", errors.New("batch size can't be empty")
	}
	for i := range brIDs {
		if len(brIDs[i]) == 0 {
			return "", fmt.Errorf("storage requests id can't be empty")
		}
	}

	ba := broker.Batch{
		ID:                  batchID,
		PayloadCid:          batchCid,
		PayloadSize:         &batchSize,
		RepFactor:           int(b.conf.dealReplication),
		DealDuration:        int(b.conf.dealDuration),
		Status:              broker.BatchStatusPreparing,
		Origin:              origin,
		DisallowRebatching:  false,
		FilEpochDeadline:    0, // For internal batches, we currently don't have hard deadlines.
		ProposalStartOffset: b.conf.defaultProposalStartOffset,
		Sources: auction.Sources{
			CARURL: &auction.CARURL{
				URL: *carURL,
			},
		},
	}

	if err := b.store.CreateBatch(ctx, &ba, brIDs, manifest, nil); err != nil {
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

	ctx, err = b.store.CtxWithTx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		err = storeutil.FinishTxForCtx(ctx, err)
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

	proposalStartEpoch, err := timeToFilEpoch(time.Now().Add(ba.ProposalStartOffset))
	if err != nil {
		return fmt.Errorf("calculating proposal start epoch: %s", err)
	}
	auctionID, err := b.startAuction(ctx, ba, nil, ba.RepFactor, dpr.PieceSize, proposalStartEpoch)
	if err != nil {
		return err
	}
	log.Debugf("batch %s was prepared piece-cid %s piece-size %d in auction %s...",
		batchID, dpr.PieceCid, dpr.PieceSize, auctionID)
	return nil
}

func (b *Broker) startAuction(ctx context.Context,
	ba *broker.Batch,
	rw *broker.RemoteWallet,
	repFactor int,
	pieceSize uint64,
	filEpochDeadline uint64) (auction.ID, error) {
	auctionID, err := b.newID()
	if err != nil {
		return "", fmt.Errorf("generating auction id: %s", err)
	}
	var clientAddress string
	if rw != nil {
		clientAddress = common.StringifyAddr(rw.WalletAddr)
	} else {
		clientAddress = common.StringifyAddr(b.conf.defaultWalletAddr)
	}

	excludedSPs, err := b.store.GetExcludedStorageProviders(ctx, ba.PieceCid, ba.Origin)
	if err != nil {
		return "", fmt.Errorf("get excluded storage providers for %s:%s: %s", ba.PieceCid, ba.Origin, err)
	}
	// If this was a direct2provider auctioned batch, we still need at least 1 non-excluded
	// miner to have the possibility of closing successfully. If we need to re-auction and don't
	// have room for any miner to win, then re-auctioning won't make sense and it will fail.
	if len(ba.Providers) > 0 {
		var excludedTargetedProviders []string
		for _, targetedSP := range ba.Providers {
			for _, excludedSP := range excludedSPs {
				if common.StringifyAddr(targetedSP) == excludedSP {
					excludedTargetedProviders = append(excludedTargetedProviders, excludedSP)
					break
				}
			}
		}
		if len(excludedTargetedProviders) == len(ba.Providers) {
			errCause := "no available miners can be selected for re-auctioning"
			log.Warn(errCause)
			_, err := b.batchError(ctx, ba, errCause, false)
			return "", fmt.Errorf("erroring batch for %q: %s", errCause, err)
		}
	}
	if err := msgbroker.PublishMsgReadyToAuction(
		ctx,
		b.mb,
		auction.ID(auctionID),
		ba.ID,
		ba.PayloadCid,
		pieceSize,
		ba.DealDuration,
		repFactor,
		b.conf.verifiedDeals,
		excludedSPs,
		filEpochDeadline,
		ba.Sources,
		clientAddress,
		ba.Providers,
	); err != nil {
		return "", fmt.Errorf("publishing ready to create auction msg: %s", err)
	}
	b.metricStartedAuctions.Add(ctx, 1)
	b.metricStartedBytes.Add(ctx, int64(pieceSize))

	return auction.ID(auctionID), nil
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
		err = storeutil.FinishTxForCtx(ctx, err)
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

	if ba.Status == broker.BatchStatusError {
		log.Warnf("batch %s has already errored, ignored closed auction %s", ba.ID, au.ID)
		return nil
	}
	if ba.Status != broker.BatchStatusAuctioning && ba.Status != broker.BatchStatusDealMaking {
		return fmt.Errorf("batch isn't in expected status: %s", ba.Status)
	}

	// If the auction failed, we didn't have at least 1 bid.
	if au.ErrorCause != "" {
		switch ba.DisallowRebatching {
		case false:
			// The batch can be rebatched. We switch the batch to error status,
			// and also signal the store to liberate the underlying storage requests to Pending.
			// This way they can be signaled to be re-batched.
			if err := b.errorBatchAndRebatch(ctx, ba, au.ErrorCause); err != nil {
				return fmt.Errorf("erroring batch and rebatching: %s", err)
			}
		case true:
			_, err := b.batchError(ctx, ba, au.ErrorCause, false)
			return err
		}
		return nil
	}

	rw, err := b.store.GetRemoteWalletConfig(ctx, au.BatchID)
	if err != nil {
		return fmt.Errorf("get remote wallet config: %s", err)
	}

	adID, err := b.newID()
	if err != nil {
		return fmt.Errorf("generating auction deal id: %s", err)
	}

	acceptedWinners, err := b.store.AddDeals(ctx, au)
	if err != nil {
		return fmt.Errorf("adding auction accepted winners: %s", err)
	}

	// If not all auction winners were accepted, then we should fire an auction to fill the gap.
	if len(acceptedWinners) != len(au.WinningBids) {
		propStartEpoch, err := timeToFilEpoch(time.Now().Add(ba.ProposalStartOffset))
		if err != nil {
			return fmt.Errorf("calculating auction start offset: %s", err)
		}
		_, err = b.startAuction(ctx, ba, rw, len(au.WinningBids)-len(acceptedWinners), ba.PieceSize, propStartEpoch)
		if err != nil {
			return fmt.Errorf("creating new auction for discarded miners: %s", err)
		}
	}

	// If we had not accepted winners, we don't have to fire new deals, so return early.
	if len(acceptedWinners) == 0 {
		log.Infof("auction %s didn't fire deals since all winners weren't accepted", au.ID)
		return nil
	}

	ads := dealer.AuctionDeals{
		ID:           adID,
		BatchID:      ba.ID,
		PayloadCid:   ba.PayloadCid,
		PieceCid:     ba.PieceCid,
		PieceSize:    ba.PieceSize,
		Duration:     au.DealDuration,
		Proposals:    make([]dealer.Proposal, len(acceptedWinners)),
		RemoteWallet: rw,
	}

	for i := range acceptedWinners {
		for bidID, bid := range au.WinningBids {
			if bid.StorageProviderID == acceptedWinners[i] {
				ads.Proposals[i] = dealer.Proposal{
					StorageProviderID:   bid.StorageProviderID,
					PricePerGiBPerEpoch: bid.Price,
					StartEpoch:          bid.StartEpoch,
					Verified:            au.DealVerified,
					FastRetrieval:       bid.FastRetrieval,
					AuctionID:           au.ID,
					BidID:               bidID,
				}
				break
			}
		}
	}

	// Fire the deals only with accepted winners.
	log.Debugf("publishing ready to create deals for auction %s, batch %s", au.ID, au.BatchID)
	if err := msgbroker.PublishMsgReadyToCreateDeals(ctx, b.mb, ads); err != nil {
		return fmt.Errorf("sending ready to create deals msg to msgbroker: %s", err)
	}

	return nil
}

// BatchFinalizedDeal report a deal that reached final status in the Filecoin network.
func (b *Broker) BatchFinalizedDeal(ctx context.Context,
	opID msgbroker.OperationID,
	fad broker.FinalizedDeal) (err error) {
	log.Debugf("finalized deal from auction (%s, %s), batch %s, storage-provider %s",
		fad.AuctionID, fad.BidID, fad.BatchID, fad.StorageProviderID)

	ctx, err = b.store.CtxWithTx(ctx)
	if err != nil {
		return err
	}
	defer func() {
		err = storeutil.FinishTxForCtx(ctx, err)
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
		log.Infof("creating new auction for failed deal with storage-provider %s", fad.StorageProviderID)

		isMinerMisconfigured := strings.Contains(fad.ErrorCause, "deal rejected")

		proposalStartEpoch, err := timeToFilEpoch(time.Now().Add(ba.ProposalStartOffset))
		if err != nil {
			return fmt.Errorf("calculating auction start offset: %s", err)
		}

		// If the Batch has a specific deadline, check if the re-auction can be run
		// considering the broker auction duration. If doesn't fit before the known deadline
		// we can't reauction and we fail the batch.
		//
		// If the batch doesn't have a specified deadline (==0), we can re-auction without deadline
		// constraint.
		if !isMinerMisconfigured && ba.FilEpochDeadline > 0 && ba.FilEpochDeadline < proposalStartEpoch {
			errCause := fmt.Sprintf(
				"re-auction deadline %d is greater than batch deadline  %d: "+msgAuctionDeadlineExceeded,
				proposalStartEpoch, ba.FilEpochDeadline)
			log.Warn(errCause)
			_, err := b.batchError(ctx, ba, errCause, false)
			return err
		}

		rw, err := b.store.GetRemoteWalletConfig(ctx, ba.ID)
		if err != nil {
			return fmt.Errorf("get remote wallet config: %s", err)
		}
		_, err = b.startAuction(ctx, ba, rw, 1, ba.PieceSize, proposalStartEpoch)
		if err != nil {
			return fmt.Errorf("creating new auction for errored deal: %s", err)
		}
		b.metricReauctions.Add(ctx, 1)
		b.metricReauctionedBytes.Add(ctx, int64(ba.PieceSize))
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

	log.Debugf("auction %s, batch %s finalized deal-id %d: %d/%d",
		fad.AuctionID, ba.ID, fad.DealID, numConfirmedDeals, ba.RepFactor)
	// Are we done?
	if numConfirmedDeals == ba.RepFactor {
		return b.batchSuccess(ctx, ba)
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
func (b *Broker) errorBatchAndRebatch(ctx context.Context, ba *broker.Batch, errCause string) error {
	brs, err := b.batchError(ctx, ba, errCause, true)
	if err != nil {
		return err
	}

	log.Debugf("erroring batch %s, rebatching %d broker-requests: %s", ba.ID, len(brs), errCause)
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

func (b *Broker) batchSuccess(ctx context.Context, ba *broker.Batch) error {
	if err := b.store.BatchSuccess(ctx, ba.ID); err != nil {
		return fmt.Errorf("moving to batch success: %s", err)
	}
	b.metricFinishedAuctions.Add(ctx, 1, metrics.AttrOK)
	b.metricFinishedBytes.Add(ctx, int64(ba.PieceSize), metrics.AttrOK)
	b.metricBatchFinalityMinutes.Record(ctx, int64(time.Since(ba.CreatedAt).Minutes()))
	log.Debugf("batch %s success", ba.ID)
	return nil
}

func (b *Broker) batchError(ctx context.Context, ba *broker.Batch, errorCause string,
	rebatch bool) (brIDs []broker.StorageRequestID, err error) {
	brs, err := b.store.BatchError(ctx, ba.ID, errorCause, rebatch)
	if err != nil {
		return nil, fmt.Errorf("moving batch to error status: %s", err)
	}
	b.metricFinishedAuctions.Add(ctx, 1, metrics.AttrError)
	b.metricFinishedBytes.Add(ctx, int64(ba.PieceSize), metrics.AttrError)
	if rebatch {
		b.metricRebatches.Add(ctx, 1)
		b.metricRebatchedBytes.Add(ctx, int64(ba.PieceSize))
	}
	return brs, nil
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
