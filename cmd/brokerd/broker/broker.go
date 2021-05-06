package broker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	logger "github.com/ipfs/go-log/v2"
	"github.com/textileio/broker-core/auctioneer"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/brokerd/store"
	"github.com/textileio/broker-core/dealer"
	"github.com/textileio/broker-core/dshelper/txndswrap"
	"github.com/textileio/broker-core/packer"
	"github.com/textileio/broker-core/piecer"
	"github.com/textileio/broker-core/reporter"
)

var (
	// ErrNotFound is returned when the broker request doesn't exist.
	ErrNotFound = fmt.Errorf("broker request not found")
	// ErrInvalidCid is returned when the Cid is undefined.
	ErrInvalidCid = fmt.Errorf("the cid can't be undefined")
	// ErrEmptyGroup is returned when an empty storage deal group
	// is received.
	ErrEmptyGroup = fmt.Errorf("the storage deal group is empty")

	errAllDealsFailed = "all winning bids deals failed"

	log = logger.Logger("broker")
)

// Broker creates and tracks request to store Cids in
// the Filecoin network.
type Broker struct {
	store        *store.Store
	packer       packer.Packer
	piecer       piecer.Piecer
	auctioneer   auctioneer.Auctioneer
	dealer       dealer.Dealer
	reporter     reporter.Reporter
	dealDuration uint64
}

// New creates a Broker backed by the provided `ds`.
func New(
	ds txndswrap.TxnDatastore,
	packer packer.Packer,
	piecer piecer.Piecer,
	auctioneer auctioneer.Auctioneer,
	dealer dealer.Dealer,
	reporter reporter.Reporter,
	dealEpochs uint64,
) (*Broker, error) {
	if dealEpochs < broker.MinDealEpochs {
		return nil, fmt.Errorf("deal epochs is less than minimum allowed: %d", broker.MinDealEpochs)
	}
	if dealEpochs > broker.MaxDealEpochs {
		return nil, fmt.Errorf("deal epochs is greater than maximum allowed: %d", broker.MaxDealEpochs)
	}

	store, err := store.New(txndswrap.Wrap(ds, "/broker-store"))
	if err != nil {
		return nil, fmt.Errorf("initializing broker request store: %s", err)
	}

	b := &Broker{
		store:        store,
		packer:       packer,
		piecer:       piecer,
		dealer:       dealer,
		auctioneer:   auctioneer,
		reporter:     reporter,
		dealDuration: dealEpochs,
	}
	return b, nil
}

var _ broker.Broker = (*Broker)(nil)

// Create creates a new BrokerRequest with the provided Cid and
// Metadata configuration.
func (b *Broker) Create(ctx context.Context, c cid.Cid, meta broker.Metadata) (broker.BrokerRequest, error) {
	if !c.Defined() {
		return broker.BrokerRequest{}, ErrInvalidCid
	}

	if err := meta.Validate(); err != nil {
		return broker.BrokerRequest{}, fmt.Errorf("invalid metadata: %s", err)
	}

	now := time.Now()
	br := broker.BrokerRequest{
		ID:        broker.BrokerRequestID(uuid.New().String()),
		DataCid:   c,
		Status:    broker.RequestBatching,
		Metadata:  meta,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := b.store.SaveBrokerRequest(ctx, br); err != nil {
		return broker.BrokerRequest{}, fmt.Errorf("saving broker request in store: %s", err)
	}

	// We notify the Packer that this BrokerRequest is ready to be considered.
	// We'll receive a call to `(*Broker).CreateStorageDeal(...)` which will contain
	// this BrokerRequest, and continue with the bidding process..
	if err := b.packer.ReadyToPack(ctx, br.ID, br.DataCid); err != nil {
		// TODO: there's room for improvement here. We can mark this broker-request
		// to be retried in signaling the packer, to avoid be orphaned.
		// Under normal circumstances this shouldn't happen.
		// We can simply save BrokerRequest with a "ReadyToBatch", and have some daemon
		// making sure of notifying and then switching to "Batching"; shoudn't be a big deal.
		return broker.BrokerRequest{}, fmt.Errorf("notifying packer of ready broker request: %s", err)
	}

	return br, nil
}

// Get gets a BrokerRequest by id. If doesn't exist, it returns ErrNotFound.
func (b *Broker) Get(ctx context.Context, ID broker.BrokerRequestID) (broker.BrokerRequest, error) {
	br, err := b.store.GetBrokerRequest(ctx, ID)
	if err == store.ErrNotFound {
		return broker.BrokerRequest{}, ErrNotFound
	}
	if err != nil {
		return broker.BrokerRequest{}, fmt.Errorf("get broker request from store: %s", err)
	}

	return br, nil
}

// CreateStorageDeal creates a StorageDeal that contains multiple BrokerRequest. This API is most probably
// caused by Packer. When Packer batches enough pending BrokerRequests in a BrokerRequestGroup, it signals
// the Broker to create a StorageDeal. This StorageDeal should be prepared (piece-size/commP) before publishing
// it in the feed.
func (b *Broker) CreateStorageDeal(
	ctx context.Context,
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

	now := time.Now()
	sd := broker.StorageDeal{
		PayloadCid:       batchCid,
		Status:           broker.StorageDealPreparing,
		BrokerRequestIDs: brids,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	// Transactionally we:
	// - Move involved BrokerRequest statuses to `Preparing`.
	// - Link each BrokerRequest with the StorageDeal.
	// - Save the `StorageDeal` in the store.
	if err := b.store.CreateStorageDeal(ctx, &sd); err != nil {
		return "", fmt.Errorf("creating storage deal: %w", err)
	}

	log.Debugf("creating storage deal %s created, signaling piecer...", sd.ID)
	// Signal Piecer that there's work to do. It will eventually call us
	// through PreparedStorageDeal(...).
	if err := b.piecer.ReadyToPrepare(ctx, sd.ID, sd.PayloadCid); err != nil {
		// TODO: same possible improvement as described in `ReadyToPack`
		// applies here.
		return "", fmt.Errorf("signaling piecer: %s", err)
	}

	return sd.ID, nil
}

// StorageDealPrepared is called by Prepared to notify that the data preparation stage is done,
// and to continue with the storage deal process.
func (b *Broker) StorageDealPrepared(
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

	log.Debugf("storage deal %s was prepared, signaling auctioneer...", id)
	// Signal the Auctioneer to create an auction. It will eventually call StorageDealAuctioned(..) to tell
	// us about who won things.
	auctionID, err := b.auctioneer.ReadyToAuction(ctx, id, dpr.PieceSize, b.dealDuration)
	if err != nil {
		return fmt.Errorf("signaling auctioneer to create auction: %s", err)
	}

	if err := b.store.StorageDealToAuctioning(ctx, id, auctionID, dpr.PieceCid, dpr.PieceSize); err != nil {
		return fmt.Errorf("saving piecer output in storage deal: %s", err)
	}

	log.Debugf("created auction %s", auctionID)
	return nil
}

// StorageDealAuctioned is called by the Auctioneer with the result of the StorageDeal auction.
func (b *Broker) StorageDealAuctioned(ctx context.Context, auction broker.Auction) error {
	log.Debugf("storage deal %s was auctioned, signaling dealer...", auction.StorageDealID)

	if auction.Status != broker.AuctionStatusEnded && auction.Status != broker.AuctionStatusError {
		return errors.New("auction status should be final")
	}

	// If the auction returned status is error, we switch the storage deal to error status,
	// and also signal the store to liberate the underlying broker requests to Pending.
	// This way they can be signaled to be re-batched.
	if auction.Status == broker.AuctionStatusError {
		if err := b.errorStorageDealAndRebatch(ctx, auction.StorageDealID, auction.Error); err != nil {
			return fmt.Errorf("erroring storage deal and rebatching: %s", err)
		}
		return nil
	}

	if len(auction.WinningBids) == 0 {
		return fmt.Errorf("winning bids list is empty")
	}

	sd, err := b.store.GetStorageDeal(ctx, auction.StorageDealID)
	if err != nil {
		return fmt.Errorf("storage deal not found: %s", err)
	}

	ads := dealer.AuctionDeals{
		StorageDealID: sd.ID,
		PayloadCid:    sd.PayloadCid,
		PieceCid:      sd.PieceCid,
		PieceSize:     sd.PieceSize,
		Duration:      auction.DealDuration,
		Targets:       make([]dealer.AuctionDealsTarget, len(auction.WinningBids)),
	}

	for i, wbid := range auction.WinningBids {
		bid, ok := auction.Bids[wbid]
		if !ok {
			return fmt.Errorf("winning bid %s wasn't found in bid map", wbid)
		}
		ads.Targets[i] = dealer.AuctionDealsTarget{
			Miner:               bid.MinerID,
			PricePerGiBPerEpoch: bid.AskPrice,
			StartEpoch:          bid.StartEpoch,
			Verified:            true, // Hardcoded for now.
			FastRetrieval:       bid.FastRetrieval,
		}
	}

	if err := b.dealer.ReadyToCreateDeals(ctx, ads); err != nil {
		return fmt.Errorf("signaling dealer to execute winning bids: %s", err)
	}

	if err := b.store.StorageDealToDealMaking(ctx, auction); err != nil {
		return fmt.Errorf("moving storage deal to deal making: %s", err)
	}

	return nil
}

// StorageDealFinalizedDeals reports deals that reached final status in the Filecoin network.
func (b *Broker) StorageDealFinalizedDeals(ctx context.Context, fads []broker.FinalizedAuctionDeal) error {
	for i := range fads {
		if err := b.store.StorageDealFinalizedDeal(fads[i]); err != nil {
			return fmt.Errorf("adding finalized info to the store: %s", err)
		}

		sd, err := b.store.GetStorageDeal(ctx, fads[i].StorageDealID)
		if err != nil {
			return fmt.Errorf("get storage deal: %s", err)
		}

		// Only report the deal to the chain if it was successful.
		if fads[i].ErrorCause == "" {
			if err := b.reportFinalizedAuctionDeal(ctx, sd, fads[i]); err != nil {
				return fmt.Errorf("reporting finalized auction deal to the chain: %s", err)
			}
		}

		// Do we got the last finalized transaction?
		if len(sd.Deals) == len(sd.Auction.WinningBids) {
			// If we have at least one successful deal, then we succeed.
			finalStatus := broker.StorageDealError
			for i := range sd.Deals {
				if sd.Deals[i].ErrorCause == "" {
					finalStatus = broker.StorageDealSuccess
				}
			}
			sd.Status = finalStatus

			switch finalStatus {
			case broker.StorageDealSuccess:
				if err := b.store.StorageDealSuccess(ctx, sd.ID); err != nil {
					return fmt.Errorf("moving to storage deal success: %s", err)
				}
			case broker.StorageDealError:
				if err := b.errorStorageDealAndRebatch(ctx, sd.ID, errAllDealsFailed); err != nil {
					return fmt.Errorf("erroring storage deal and rebatching storage requests: %s", err)
				}
			}
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

	return sd, nil
}

func (b *Broker) reportFinalizedAuctionDeal(ctx context.Context, sd broker.StorageDeal, fad broker.FinalizedAuctionDeal) error {
	deals := make([]reporter.DealInfo, len(sd.Deals))
	for i := range sd.Deals {
		d := reporter.DealInfo{
			DealID:     sd.Deals[i].DealID,
			MinerID:    sd.Deals[i].Miner,
			Expiration: sd.Deals[i].DealExpiration,
		}
		deals[i] = d
	}
	dataCids := make([]cid.Cid, len(sd.BrokerRequestIDs))
	for i := range sd.BrokerRequestIDs {
		br, err := b.store.GetBrokerRequest(ctx, sd.BrokerRequestIDs[i])
		if err != nil {
			return fmt.Errorf("get broker request: %s", err)
		}
		dataCids[i] = br.DataCid
	}
	if err := b.reporter.ReportStorageInfo(ctx, sd.PayloadCid, sd.PieceCid, deals, dataCids); err != nil {
		return fmt.Errorf("reporting storage info: %s", err)
	}
	return nil
}

func (b *Broker) errorStorageDealAndRebatch(ctx context.Context, id broker.StorageDealID, errCause string) error {
	brs, err := b.store.StorageDealError(ctx, id, errCause)
	if err != nil {
		return fmt.Errorf("moving storage deal to error status: %s", err)
	}

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
