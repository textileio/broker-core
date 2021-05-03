package dealer

import (
	"context"
	"fmt"
	"sync"

	logger "github.com/ipfs/go-log/v2"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/dealerd/dealer/store"
	dealeri "github.com/textileio/broker-core/dealer"
	"github.com/textileio/broker-core/dshelper/txndswrap"
)

var log = logger.Logger("dealer")

type Dealer struct {
	config    config
	store     *store.Store
	broker    broker.Broker
	filclient Filclient

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
	broker broker.Broker,
	fc Filclient,
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
		broker:    broker,
		filclient: fc,

		daemonCtx:       ctx,
		daemonCancelCtx: cls,
		daemonClosed:    make(chan struct{}),
	}

	go d.daemon()

	return d, nil
}

func (d *Dealer) ReadyToCreateDeals(ctx context.Context, ad dealeri.AuctionDeals) error {
	auctionData := store.AuctionData{
		StorageDealID: ad.StorageDealID,
		PayloadCid:    ad.PayloadCid,
		PieceCid:      ad.PieceCid,
		PieceSize:     ad.PieceSize,
		Duration:      ad.Duration,
	}
	auctionDeals := make([]store.AuctionDeal, len(ad.Targets))
	for i, t := range ad.Targets {
		auctionDeal := store.AuctionDeal{
			Miner:               t.Miner,
			PricePerGiBPerEpoch: t.PricePerGiBPerEpoch,
			StartEpoch:          t.StartEpoch,
			Verified:            t.Verified,
			FastRetrieval:       t.FastRetrieval,
		}
		auctionDeals[i] = auctionDeal
	}
	if err := d.store.Create(auctionData, auctionDeals); err != nil {
		return fmt.Errorf("creating auction deals: %s", err)
	}

	return nil
}

// Close closes the dealer.
func (d *Dealer) Close() error {
	d.onceClose.Do(func() {
		d.daemonCancelCtx()
		<-d.daemonClosed
	})
	return nil
}

func (d *Dealer) daemon() {
	defer close(d.daemonClosed)

	d.daemonWg.Add(3)

	// daemonDealMaker make status changes from Pending -> (Watching | Error).
	// i.e: takes Pending deals, executes them, and leave them ready
	// to be confirmed on-chain.
	go d.daemonDealMaker()
	// daemonDealMonitoring makes status changes from Pending -> (Successs | Error)
	// i.e: monitors the fired deal until is confirmed on-chain.
	go d.daemonDealMonitoring()
	// daemonDealWatcher takes statuses (Success | Error) and reports the results
	// back to the broker, deleting them after geting ACK from it.
	go d.daemonDealReporter()

	<-d.daemonCtx.Done()
	log.Infof("closing dealer daemons")
	d.daemonWg.Wait() // Wait for all sub-daemons to finish.
	log.Infof("closing dealer daemons")
}
