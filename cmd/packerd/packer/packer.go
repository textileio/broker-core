package packer

import (
	"context"
	"fmt"
	"time"

	aggregator "github.com/filecoin-project/go-dagaggregator-unixfs"
	"github.com/ipfs/go-cid"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	"github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/packerd/store"
	mbroker "github.com/textileio/broker-core/msgbroker"
	"github.com/textileio/broker-core/storeutil"
	logger "github.com/textileio/go-log/v2"
	"go.opentelemetry.io/otel/metric"
)

var (
	log = logger.Logger("packer")
)

// Packer provides batching strategies to bundle multiple
// BrokerRequest into a StorageDeal.
type Packer struct {
	daemonFreq        time.Duration
	exportMetricsFreq time.Duration

	store *store.Store
	ipfs  *httpapi.HttpApi
	mb    mbroker.MsgBroker

	daemonCtx       context.Context
	daemonCancelCtx context.CancelFunc
	daemonClosed    chan struct{}

	metricNewBatch          metric.Int64Counter
	statLastBatch           time.Time
	metricLastBatchCreated  metric.Int64ValueObserver
	statLastBatchCount      int64
	metricLastBatchCount    metric.Int64ValueObserver
	statLastBatchSize       int64
	metricLastBatchSize     metric.Int64ValueObserver
	statLastBatchDuration   int64
	metricLastBatchDuration metric.Int64ValueObserver
}

// New returns a new Packer.
func New(
	postgresURI string,
	ipfsClient *httpapi.HttpApi,
	mb mbroker.MsgBroker,
	opts ...Option) (*Packer, error) {
	cfg := defaultConfig
	for _, op := range opts {
		if err := op(&cfg); err != nil {
			return nil, fmt.Errorf("applying option: %s", err)
		}
	}

	if cfg.batchMinSize <= 0 {
		return nil, fmt.Errorf("batch min size should be positive")
	}

	batchMaxSize := calcBatchLimit(cfg.sectorSize)
	store, err := store.New(postgresURI, batchMaxSize, cfg.batchMinSize)
	if err != nil {
		return nil, fmt.Errorf("initializing store: %s", err)
	}

	ctx, cls := context.WithCancel(context.Background())
	p := &Packer{
		daemonFreq:        cfg.daemonFreq,
		exportMetricsFreq: cfg.exportMetricsFreq,

		store: store,
		ipfs:  ipfsClient,
		mb:    mb,

		daemonCtx:       ctx,
		daemonCancelCtx: cls,
		daemonClosed:    make(chan struct{}),
	}
	p.initMetrics()

	go p.daemon()
	go p.daemonExportMetrics()

	return p, nil
}

// ReadyToBatch includes a StorageRequest in an open batch.
func (p *Packer) ReadyToBatch(ctx context.Context, opID string, srID broker.BrokerRequestID, dataCid cid.Cid) error {
	log.Debugf("received ready to pack storage-request %s, data-cid %s", srID, dataCid)

	node, err := p.ipfs.Dag().Get(ctx, dataCid)
	if err != nil {
		return fmt.Errorf("get node for data-cid: %s", dataCid, err)
	}
	stat, err := node.Stat()
	if err != nil {
		return fmt.Errorf("get stat for data-cid %s: %s", dataCid, err)
	}
	size := int64(stat.CumulativeSize)

	ctx, err = p.store.CtxWithTx(ctx)
	if err != nil {
		return fmt.Errorf("creating ctx with tx: %s", err)
	}
	defer func() {
		err = storeutil.FinishTxForCtx(ctx, err)
	}()
	if err := p.store.AddStorageRequestToOpenBatch(
		ctx,
		opID,
		srID,
		dataCid,
		size,
	); err != nil {
		return fmt.Errorf("add storage-request to open batch: %s", err)
	}

	return nil
}

// Close closes the packer.
func (p *Packer) Close() error {
	log.Info("closing packer...")
	p.daemonCancelCtx()
	<-p.daemonClosed
	if err := p.store.Close(); err != nil {
		return fmt.Errorf("closing store: %s", err)
	}
	return nil
}

func (p *Packer) daemon() {
	defer close(p.daemonClosed)
	for {
		select {
		case <-p.daemonCtx.Done():
			log.Info("packer closed")
			return
		case <-time.After(p.daemonFreq):
			for {
				count, err := p.pack(p.daemonCtx)
				if err != nil {
					log.Errorf("packing: %s", err)
					break
				}
				if count == 0 {
					break
				}
			}
		}
	}
}

// TODO(jsign): review logging.
func (p *Packer) pack(ctx context.Context) (int, error) {
	// TODO(jsign): rename bbrs
	batch, bbrs, ok, err := p.store.GetNextReadyBatch()
	if err != nil {
		return 0, fmt.Errorf("get next ready batch: %s", err)
	}
	if !ok {
		return 0, nil
	}
	log.Debugf("preparing ready batch-id %s with %d storage-request", batch.BatchID, len(bbrs))

	start := time.Now()
	batchCid, err := p.createDAGForBatch(ctx, bbrs)
	if err != nil {
		return 0, fmt.Errorf("creating dag for batch: %s", err)
	}

	if err := p.store.MoveBatchToStatus(ctx, batch.BatchID, store.StatusDone); err != nil {
		return 0, fmt.Errorf("moving batch %s to done: %s", batch.BatchID, err)
	}

	brids := make([]broker.BrokerRequestID, len(bbrs))
	for i, bbr := range bbrs {
		brids[i] = bbr.BrokerRequestID
	}
	if err := mbroker.PublishMsgNewBatchCreated(ctx, p.mb, batch.BatchID, batchCid, brids); err != nil {
		return 0, fmt.Errorf("publishing msg to broker: %s", err)
	}

	p.metricNewBatch.Add(ctx, 1)
	p.statLastBatch = time.Now()
	p.statLastBatchCount = int64(len(bbrs))
	p.statLastBatchSize = batch.Size
	p.statLastBatchDuration = time.Since(start).Milliseconds()
	log.Infof(
		"batch created: {batch-id: %s, batch-cid: %s, num-storage-requests: %d, batch-size: %d}",
		batch.BatchID, batchCid, len(bbrs), batch.Size)

	return len(bbrs), nil
}

func (p *Packer) createDAGForBatch(ctx context.Context, bbrs []store.StorageRequest) (cid.Cid, error) {
	lst := make([]aggregator.AggregateDagEntry, len(bbrs))
	for i, bbr := range bbrs {
		lst[i] = aggregator.AggregateDagEntry{
			RootCid:                   bbr.DataCid,
			UniqueBlockCumulativeSize: uint64(bbr.Size),
		}
	}
	start := time.Now()
	root, err := aggregator.Aggregate(ctx, p.ipfs.Dag(), lst)
	if err != nil {
		return cid.Undef, fmt.Errorf("aggregating cids: %s", err)
	}
	log.Debugf("aggregation took %dms", time.Since(start).Milliseconds())
	if err := p.ipfs.Pin().Add(ctx, path.IpfsPath(root), options.Pin.Recursive(true)); err != nil {
		return cid.Undef, fmt.Errorf("pinning batch root: %s", err)
	}
	log.Debugf("aggregation+pinning took %dms", time.Since(start).Milliseconds())

	return root, nil
}

// calcBatchLimit does a worst-case estimation of the maximum size the batch DAG can have to fit
// in the specified sector size.
// Rationale of the calculation:
// - Mul 127/128: Account fr32 padding in the sector.
// - Div 1.04: This is an estimation of CAR serialization overhead.
//             2(varint)+34(cid-size)+~1024(small-block) -> ~4% overhead.
//
// Note that this limit is a worst-case scenario for a perfectly calculated size for a DAG,
// that's to say, accounting for deduplication. TL;DR: safe boundaries.
func calcBatchLimit(sectorSize int64) int64 {
	return int64(float64(sectorSize) * float64(127) / float64(128) / 1.04)
}
