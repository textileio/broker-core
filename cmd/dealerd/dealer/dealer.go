package dealer

import (
	"context"
	"fmt"
	"sync"

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

	onceClose       sync.Once
	daemonCtx       context.Context
	daemonCancelCtx context.CancelFunc
	daemonClosed    chan struct{}
}

var _ dealeri.Dealer = (*Dealer)(nil)

// New returns a new Packer.
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

	ctx, cls := context.WithCancel(context.Background())
	d := &Dealer{
		store:  store,
		broker: broker,

		daemonCtx:       ctx,
		daemonCancelCtx: cls,
		daemonClosed:    make(chan struct{}),
	}

	go d.daemon()

	return d, nil
}

// Close closes the dealer.
func (p *Dealer) Close() error {
	p.onceClose.Do(func() {
		p.daemonCancelCtx()
		<-p.daemonClosed
	})
	return nil
}

func (p *Dealer) daemon() {
	defer close(p.daemonClosed)
	for {
		select {
		case <-p.daemonCtx.Done():
			log.Infof("dealer closed")
			return
		}
	}
}

func (p *Dealer) makeDeals(ctx context.Context) {
}
