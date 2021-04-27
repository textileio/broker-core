package dealer

import (
	"context"
	"fmt"
	"sync"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/filecoin-project/lotus/chain/wallet"
	"github.com/ipfs/go-datastore"
	logger "github.com/ipfs/go-log/v2"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/dealerd/dealer/store"
	dealeri "github.com/textileio/broker-core/dealer"
	"github.com/textileio/broker-core/dshelper/txndswrap"
)

var (
	log = logger.Logger("dealer")
)

type Dealer struct {
	store  *store.Store
	broker broker.Broker

	walletAddr address.Address
	wallet     *wallet.LocalWallet
	gateway    api.GatewayAPI

	onceClose       sync.Once
	daemonCtx       context.Context
	daemonCancelCtx context.CancelFunc
	daemonClosed    chan struct{}
}

var _ dealeri.Dealer = (*Dealer)(nil)

// New returns a new Dealer.
func New(
	ds datastore.TxnDatastore,
	broker broker.Broker,
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

	memks := wallet.NewMemKeyStore()
	w, err := wallet.NewWallet(memks)
	if err != nil {
		return nil, fmt.Errorf("creating local wallet: %s", err)
	}

	ki := &types.KeyInfo{
		Type:       cfg.keyType,
		PrivateKey: cfg.keyPrivate,
	}
	waddr, err := w.WalletImport(context.Background(), ki)
	if err != nil {
		return nil, fmt.Errorf("importing wallet addr: %s", err)
	}

	ctx, cls := context.WithCancel(context.Background())
	d := &Dealer{
		store:  store,
		broker: broker,

		walletAddr: waddr,
		wallet:     w,

		daemonCtx:       ctx,
		daemonCancelCtx: cls,
		daemonClosed:    make(chan struct{}),
	}

	go d.daemon()

	return d, nil
}

func (d *Dealer) ReadyToCreateDeals(ctx context.Context, ad dealeri.AuctionDeals) error {
	auctionData := store.AuctionData{
		AuctionID:  ad.AuctionID,
		PayloadCid: ad.PayloadCid,
		PieceCid:   ad.PieceCid,
		PieceSize:  ad.PieceSize,
		Duration:   ad.Duration,
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
		return fmt.Errorf("enqueueing auction deals: %s", err)
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
	for {
		select {
		case <-d.daemonCtx.Done():
			log.Infof("dealer closed")
			return
		}
	}
}

func (d *Dealer) makeDeals(ctx context.Context) {
}
