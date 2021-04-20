package packer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	cbornode "github.com/ipfs/go-ipld-cbor"
	logger "github.com/ipfs/go-log/v2"
	"github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multihash"
	"github.com/textileio/broker-core/broker"
	packeri "github.com/textileio/broker-core/packer"
)

// TODO: wire gRPC client whenever ready.
// For the moment, we do a quick impl that:
// - Has an internal queue of things to pack
// - Every take all queued things and creates a batch
// - Signals the broker about the batch
//
// This is a embedded mock of what is going to be an external
// service dependency. The loopback to notifying the
// broker about packed data would come from an API call.
// Here we simply wired some interface since it's all living in the
// same binary. So there're some interfaces that will go away.
// (i.e. things here will be way simpler in the final impl).

var (
	cutFrequency = time.Second * 10

	log = logger.Logger("packer")
)

// Packer provides batching strategies to bundle multiple
// BrokerRequest into a StorageDeal.
type Packer struct {
	lock  sync.Mutex
	queue []broker.BrokerRequest

	broker broker.Broker
	ipfs   *httpapi.HttpApi

	onceClose       sync.Once
	daemonCtx       context.Context
	daemonCancelCtx context.CancelFunc
	daemonClosed    chan struct{}
}

var _ packeri.Packer = (*Packer)(nil)

// New returns a new Packer.
func New(ipfsAPIMultiaddr string) (*Packer, error) {
	ma, err := multiaddr.NewMultiaddr(ipfsAPIMultiaddr)
	if err != nil {
		return nil, fmt.Errorf("parsing ipfs client multiaddr: %s", err)
	}
	client, err := httpapi.NewApi(ma)
	if err != nil {
		return nil, fmt.Errorf("creating ipfs client: %s", err)
	}

	ctx, cls := context.WithCancel(context.Background())
	p := &Packer{
		ipfs:            client,
		daemonCtx:       ctx,
		daemonCancelCtx: cls,
		daemonClosed:    make(chan struct{}),
	}

	go p.daemon()

	return p, nil
}

// SetBroker is a *temporary* method that will go away. Now just
// being useful for the embbeded packer in brokerd.
func (p *Packer) SetBroker(b broker.Broker) {
	p.broker = b
}

// ReadyToPack signals the packer that there's a new BrokerRequest that can be
// considered. Packer will notify the broker async when the final StorageDeal
// that contains this BrokerRequest gets created.
func (p *Packer) ReadyToPack(ctx context.Context, br broker.BrokerRequest) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.queue = append(p.queue, br)

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
		cidArray[i] = br.DataCid
		brids[i] = br.ID
	}
	n, err := cbornode.WrapObject(cidArray, multihash.SHA2_256, -1)
	if err != nil {
		return fmt.Errorf("marshaling cbor array: %s", err)
	}

	if err := p.ipfs.Dag().Add(ctx, n); err != nil {
		return fmt.Errorf("adding root node of batch to ipfs: %s", err)
	}

	srg := broker.BrokerRequestGroup{
		Cid:                    n.Cid(),
		GroupedStorageRequests: brids,
	}

	sd, err := p.broker.CreateStorageDeal(ctx, srg)
	if err != nil {
		return fmt.Errorf("creating storage deal: %s", err)
	}
	log.Infof("storage deal created: {id: %s, cid: %s}", sd.ID, sd.Cid)
	p.queue = p.queue[:0]

	return nil
}
