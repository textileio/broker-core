package packer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	cbornode "github.com/ipfs/go-ipld-cbor"
	logger "github.com/ipfs/go-log/v2"
	"github.com/multiformats/go-multihash"
	"github.com/textileio/broker-core/broker"
	packeri "github.com/textileio/broker-core/packer"
)

var (
	cutFrequency = time.Second * 20

	log = logger.Logger("packer")
)

// Packer provides batching strategies to bundle multiple
// BrokerRequest into a StorageDeal.
type Packer struct {
	lock  sync.Mutex
	queue []br

	ipfs   *httpapi.HttpApi
	broker broker.Broker

	onceClose       sync.Once
	daemonCtx       context.Context
	daemonCancelCtx context.CancelFunc
	daemonClosed    chan struct{}
}

type br struct {
	id      broker.BrokerRequestID
	dataCid cid.Cid
}

var _ packeri.Packer = (*Packer)(nil)

// New returns a new Packer.
func New(ds datastore.TxnDatastore, ipfsClient *httpapi.HttpApi, broker broker.Broker) (*Packer, error) {
	ctx, cls := context.WithCancel(context.Background())
	p := &Packer{
		ipfs:            ipfsClient,
		broker:          broker,
		daemonCtx:       ctx,
		daemonCancelCtx: cls,
		daemonClosed:    make(chan struct{}),
	}

	go p.daemon()

	return p, nil
}

// ReadyToPack signals the packer that there's a new BrokerRequest that can be
// considered. Packer will notify the broker async when the final StorageDeal
// that contains this BrokerRequest gets created.
func (p *Packer) ReadyToPack(ctx context.Context, id broker.BrokerRequestID, dataCid cid.Cid) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.queue = append(p.queue, br{id: id, dataCid: dataCid})

	return nil
}

// Close closes the packer.
func (p *Packer) Close() error {
	p.onceClose.Do(func() {
		p.daemonCancelCtx()
		<-p.daemonClosed
	})
	return nil
}

func (p *Packer) daemon() {
	defer close(p.daemonClosed)
	for {
		select {
		case <-p.daemonCtx.Done():
			log.Infof("packer closed")
			return
		case <-time.After(cutFrequency):
			if err := p.pack(p.daemonCtx); err != nil {
				log.Errorf("packing: %s", err)
			}
		}
	}
}

func (p *Packer) pack(ctx context.Context) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	if len(p.queue) == 0 {
		return nil
	}

	log.Infof("packing %d broker requests...", len(p.queue))

	// Do some simply cbor array batching.
	// Totally reasonable things to do.. just some downsides.
	//
	// We don't do dag-size checking to fit in sector-sizes here
	// but we need to do that to always cut DAG sizes smaller than a sector.
	//
	// A better batching and dag-size checking will happen soon anyway.
	cidArray := make([]cid.Cid, len(p.queue))
	brids := make([]broker.BrokerRequestID, len(p.queue))
	for i, br := range p.queue {
		cidArray[i] = br.dataCid
		brids[i] = br.id
	}
	n, err := cbornode.WrapObject(cidArray, multihash.SHA2_256, -1)
	if err != nil {
		return fmt.Errorf("marshaling cbor array: %s", err)
	}

	if err := p.ipfs.Dag().Add(ctx, n); err != nil {
		return fmt.Errorf("adding root node of batch to ipfs: %s", err)
	}

	sdID, err := p.broker.CreateStorageDeal(ctx, n.Cid(), brids)
	if err != nil {
		return fmt.Errorf("creating storage deal: %s", err)
	}

	log.Infof("storage deal created: {id: %s, cid: %s}", sdID, n.Cid())
	p.queue = p.queue[:0]

	return nil
}
