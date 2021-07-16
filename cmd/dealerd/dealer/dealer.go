package dealer

import (
	"context"
	"fmt"
	"sync"

	"github.com/textileio/bidbot/lib/dshelper/txndswrap"
	"github.com/textileio/bidbot/lib/logging"
	"github.com/textileio/broker-core/cmd/dealerd/dealer/store"
	dealeri "github.com/textileio/broker-core/dealer"
	mbroker "github.com/textileio/broker-core/msgbroker"
	logger "github.com/textileio/go-log/v2"
)

var log = logger.Logger("dealer")

// Dealer creates, monitors and reports deals in the Filecoin network.
type Dealer struct {
	config    config
	store     *store.Store
	filclient FilClient
	mb        mbroker.MsgBroker

	onceClose       sync.Once
	daemonCtx       context.Context
	daemonCancelCtx context.CancelFunc
	daemonClosed    chan struct{}
	daemonWg        sync.WaitGroup
}

var _ dealeri.Dealer = (*Dealer)(nil)

// New returns a new Dealer.
func New(
	ds txndswrap.TxnDatastore,
	mb mbroker.MsgBroker,
	fc FilClient,
	opts ...Option) (*Dealer, error) {
	cfg := defaultConfig
	for _, op := range opts {
		if err := op(&cfg); err != nil {
			return nil, fmt.Errorf("applying option: %s", err)
		}
	}

	store, err := store.New(txndswrap.Wrap(ds, "/queue"))
	if err != nil {
		return nil, fmt.Errorf("initializing store: %s", err)
	}

	ctx, cls := context.WithCancel(context.Background())
	d := &Dealer{
		config:    cfg,
		store:     store,
		filclient: fc,
		mb:        mb,

		daemonCtx:       ctx,
		daemonCancelCtx: cls,
		daemonClosed:    make(chan struct{}),
	}

	go d.daemons()

	return d, nil
}

// ReadyToCreateDeals signal the dealer that new deals are ready to be executed.
func (d *Dealer) ReadyToCreateDeals(ctx context.Context, ad dealeri.AuctionDeals) error {
	auctionData := &store.AuctionData{
		StorageDealID: ad.StorageDealID,
		PayloadCid:    ad.PayloadCid,
		PieceCid:      ad.PieceCid,
		PieceSize:     ad.PieceSize,
		Duration:      ad.Duration,
	}
	log.Debugf("ready to create deals auction data: %s", logging.MustJSONIndent(auctionData))
	auctionDeals := make([]*store.AuctionDeal, len(ad.Proposals))
	for i, t := range ad.Proposals {
		auctionDeal := &store.AuctionDeal{
			Miner:               t.Miner,
			PricePerGiBPerEpoch: t.PricePerGiBPerEpoch,
			StartEpoch:          t.StartEpoch,
			Verified:            t.Verified,
			FastRetrieval:       t.FastRetrieval,
		}
		auctionDeals[i] = auctionDeal
		log.Debugf("%s auction deal: %s", auctionData.StorageDealID, logging.MustJSONIndent(auctionDeal))
	}
	if err := d.store.Create(auctionData, auctionDeals); err != nil {
		return fmt.Errorf("creating auction deals: %s", err)
	}

	return nil
}

// Close closes the dealer.
func (d *Dealer) Close() error {
	log.Info("closing dealer...")
	d.onceClose.Do(func() {
		d.daemonCancelCtx()
		<-d.daemonClosed
	})
	return nil
}

func (d *Dealer) daemons() {
	defer close(d.daemonClosed)

	// We don't count the metrics daemon since we don't need to wait for it.
	d.daemonWg.Add(3)

	// daemonDealMaker makes status transitions:
	// PendingDealMaking <--> ExecutingDealMaking --> PendingConfirmation
	go d.daemonDealMaker()
	// daemonDealMonitorer makes status transitions:
	// PendingConfirmation <--> ExecutingConfirmation --> PendingReportFinalized
	go d.daemonDealMonitorer()
	// daemonDealWatcher takes records in PendingReportFinalized and reports back the
	// result to the broker. If the broker ACKs correctly, then it deletes them.
	go d.daemonDealReporter()

	go d.daemonExportMetrics()

	<-d.daemonCtx.Done()
	log.Info("closing dealer daemons")
	d.daemonWg.Wait() // Wait for all sub-daemons to finish.
	log.Info("closing dealer daemons")
}
