package packer

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	ipld "github.com/ipfs/go-ipld-format"
	logger "github.com/ipfs/go-log/v2"
	"github.com/ipfs/go-merkledag"
	"github.com/ipfs/go-unixfs"
	"github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/multiformats/go-multibase"
	"github.com/textileio/broker-core/broker"
	packeri "github.com/textileio/broker-core/packer"
)

var (
	log = logger.Logger("packer")
)

// Packer provides batching strategies to bundle multiple
// BrokerRequest into a StorageDeal.
type Packer struct {
	batchFrequency time.Duration
	batchLimit     int64

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
func New(
	ds datastore.TxnDatastore,
	ipfsClient *httpapi.HttpApi,
	broker broker.Broker,
	opts ...Option) (*Packer, error) {
	cfg := defaultConfig
	for _, op := range opts {
		if err := op(&cfg); err != nil {
			return nil, fmt.Errorf("applying option: %s", err)
		}
	}

	ctx, cls := context.WithCancel(context.Background())
	p := &Packer{
		batchFrequency: cfg.frequency,
		batchLimit:     calcBatchLimit(cfg.sectorSize),

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
	if dataCid.Version() != 1 {
		return fmt.Errorf("only cidv1 is supported")
	}
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
		case <-time.After(p.batchFrequency):
			if err := p.pack(p.daemonCtx); err != nil {
				log.Errorf("packing: %s", err)
			}
		}
	}
}

func (p *Packer) pack(ctx context.Context) error {
	p.lock.Lock()
	copyQueue := p.queue
	p.lock.Unlock()
	if len(copyQueue) == 0 {
		return nil
	}

	log.Infof("packing %d broker requests...", len(p.queue))

	batchCid, brids, err := p.batchQueue(ctx, copyQueue)
	if err != nil {
		return fmt.Errorf("batching cids: %s", err)
	}

	p.lock.Lock()
	for _, brid := range brids {
		for i := range p.queue {
			if p.queue[i].id == brid {
				// Remove while keeping order. This keeps fairness.
				p.queue = append(p.queue[:i], p.queue[i+1:]...)
				break
			}
		}
	}
	p.lock.Unlock()

	sdID, err := p.broker.CreateStorageDeal(ctx, batchCid, brids)
	if err != nil {
		return fmt.Errorf("creating storage deal: %s", err)
	}

	log.Infof("storage deal created: {id: %s, cid: %s}", sdID, batchCid)

	return nil
}

func (p *Packer) batchQueue(ctx context.Context, brs []br) (cid.Cid, []broker.BrokerRequestID, error) {
	batchRoot := unixfs.EmptyDirNode()
	batchRoot.SetCidBuilder(merkledag.V1CidPrefix())
	var currentSize int64

	var addedStorageRequestIDs []broker.BrokerRequestID
	for _, br := range brs {
		n, err := p.ipfs.Dag().Get(ctx, br.dataCid)
		if err != nil {
			return cid.Undef, nil, fmt.Errorf("getting node by cid: %s", err)
		}
		ns, err := n.Stat()
		if err != nil {
			return cid.Undef, nil, fmt.Errorf("getting node stat: %s", err)
		}

		// If adding this data to the DAG exceeds our limit size for a batch,
		// we skip it and try with the next. Skipped ones still keep original order,
		// so they naturally have priority in the next batch to get in.
		if currentSize+int64(ns.CumulativeSize) > p.batchLimit {
			log.Infof("skipping cid %s which doesn't fit in sector-limit bounds: %s", br.dataCid)
			continue
		}

		// 1- We get a base32 Cid.
		base32Cid, err := br.dataCid.StringOfBase(multibase.Base32)
		if err != nil {
			return cid.Undef, nil, fmt.Errorf("transforming to base32 cid: %s", err)
		}
		base32CidLen := len(base32Cid)

		// 2- Build path through 3 layers.
		layer0LinkName := "ba.." + base32Cid[base32CidLen-2:base32CidLen]
		layer1Node, err := p.getOrCreateLayerNode(ctx, batchRoot, layer0LinkName)
		if err != nil {
			return cid.Undef, nil, fmt.Errorf("get/create layer0 node: %s", err)
		}

		layer1LinkName := "ba.." + base32Cid[base32CidLen-4:base32CidLen]
		layer2Node, err := p.getOrCreateLayerNode(ctx, layer1Node, layer1LinkName)
		if err != nil {
			return cid.Undef, nil, fmt.Errorf("get/create layer1 node: %s", err)
		}

		_, err = layer2Node.GetNodeLink(base32Cid)
		if err != nil && err != merkledag.ErrLinkNotFound {
			return cid.Undef, nil, fmt.Errorf("getting target node: %s", err)
		}
		if err == merkledag.ErrLinkNotFound {
			if err := layer2Node.AddNodeLink(base32Cid, n); err != nil {
				return cid.Undef, nil, fmt.Errorf("adding target node link: %s", err)
			}

			if err := updateNodeLink(layer1Node, layer1LinkName, layer2Node); err != nil {
				return cid.Undef, nil, fmt.Errorf("updating layer 2 node: %s", err)
			}

			if err := updateNodeLink(batchRoot, layer0LinkName, layer1Node); err != nil {
				return cid.Undef, nil, fmt.Errorf("updating layer 1 node: %s", err)
			}

			err = p.ipfs.Dag().AddMany(ctx, []ipld.Node{batchRoot, layer1Node, layer2Node})
			if err != nil {
				return cid.Undef, nil, fmt.Errorf("adding updated nodes: %s", err)
			}
		} else {
			// The "else" case is interesting. In this situation, we already batched the same dataCid
			// in the dag. Maybe some other BrokerRequest had the same Cid and we got 2x1.
			// That's to say, we don't do anything in the batch DAG.
			// It isn't a problem, both StorageRequests are fulfilled.
			// Let's log a warning just to be aware that this is a lucky situation and not
			// something firing often.
			log.Warnf("lucky! cid %s is already in the batch", base32Cid)
		}

		addedStorageRequestIDs = append(addedStorageRequestIDs, br.id)

		currentSize, err = p.dagSize(ctx, batchRoot.Cid())
		if err != nil {
			return cid.Undef, nil, fmt.Errorf("calculating dag size: %s", err)
		}
	}

	if err := p.ipfs.Pin().Add(ctx, path.IpfsPath(batchRoot.Cid()), options.Pin.Recursive(true)); err != nil {
		return cid.Undef, nil, fmt.Errorf("pinning batch root: %s", err)
	}

	return batchRoot.Cid(), addedStorageRequestIDs, nil
}

func (p *Packer) getOrCreateLayerNode(
	ctx context.Context,
	parentNode *merkledag.ProtoNode,
	layerLinkName string) (*merkledag.ProtoNode, error) {
	layer0Link, err := parentNode.GetNodeLink(layerLinkName)
	if err != nil && err != merkledag.ErrLinkNotFound {
		return nil, fmt.Errorf("getting node by name: %s", err)
	}
	var layerNode *merkledag.ProtoNode
	if err == merkledag.ErrLinkNotFound {
		layerNode = unixfs.EmptyDirNode()
		if err := parentNode.AddNodeLink(layerLinkName, layerNode); err != nil {
			return nil, fmt.Errorf("adding link: %s", err)
		}
	} else {
		l1Node, err := p.ipfs.ResolveNode(ctx, path.IpfsPath(layer0Link.Cid))
		if err != nil {
			return nil, fmt.Errorf("getting layer node: %s", err)
		}
		layerNode = l1Node.(*merkledag.ProtoNode)
	}

	return layerNode, nil
}

func updateNodeLink(node *merkledag.ProtoNode, linkName string, newNode *merkledag.ProtoNode) error {
	if err := node.RemoveNodeLink(linkName); err != nil {
		return fmt.Errorf("removing link 0 name: %s", err)
	}
	if err := node.AddNodeLink(linkName, newNode); err != nil {
		return fmt.Errorf("updating batch root node link: %s", err)
	}
	return nil
}

func (p *Packer) dagSize(ctx context.Context, c cid.Cid) (int64, error) {
	stat, err := p.ipfs.Object().Stat(ctx, path.IpfsPath(c))
	if err != nil {
		return 0, fmt.Errorf("getting node by cid: %s", err)
	}

	return int64(stat.CumulativeSize), nil
}

// calcBatchLimit does a worst-case estimation of the maximum size the batch DAG can have to fit
// in the specified sector size.
// Rationale of the calculation:
// - Mul 127/128: Account fr32 padding in the sector.
// - Div 1.04: This is an estimation of CAR serialization overhead.
//             2(varint)+34(cid-size)+~1024(small-block) -> ~4% overhead.
//
// Note that this limit is a worst-case scenario for a perfectly calculated size for a DAG,
// that's to say, accouting fro deduplication. TL;DR: safe cboundaries.
func calcBatchLimit(sectorSize int64) int64 {
	return int64(float64(sectorSize) * float64(127) / float64(128) / 1.04)
}

/*
// (jsign): Leaving this other dag-size implementation which is correct for DAGs that aren't UnixFS fully.
// But it's very slow. Unfortunately, there's no workaround for this slow method (`ipfs dag stat` does exactly
// this thing).

func (p *Packer) dagSizeTraversal(ctx context.Context, c cid.Cid) (uint64, error) {
	n, err := p.ipfs.Dag().Get(ctx, c)
	if err != nil {
		return 0, fmt.Errorf("getting node by cid: %s", err)
	}

	var size uint64
	err = traverse.Traverse(n, traverse.Options{
		DAG:            p.ipfs.Dag(),
		Order:          traverse.DFSPre,
		ErrFunc:        nil,
		SkipDuplicates: true,
		Func: func(current traverse.State) error {
			size += uint64(len(current.Node.RawData()))
			return nil
		},
	})
	if err != nil {
		return 0, fmt.Errorf("traversing the DAG: %s", err)
	}

	return size, nil
}
*/
