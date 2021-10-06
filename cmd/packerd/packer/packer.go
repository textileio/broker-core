package packer

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	aggregator "github.com/filecoin-project/go-dagaggregator-unixfs"
	"github.com/ipfs/go-cid"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	"github.com/ipfs/interface-go-ipfs-core/options"
	ipfspath "github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/ipld/go-car"
	"github.com/multiformats/go-multiaddr"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/packerd/store"
	"github.com/textileio/broker-core/ipfsutil"
	"github.com/textileio/broker-core/metrics"
	mbroker "github.com/textileio/broker-core/msgbroker"
	"github.com/textileio/broker-core/storeutil"
	logger "github.com/textileio/go-log/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	log = logger.Logger("packer")
)

// Packer provides batching strategies to bundle multiple
// StorageRequest into a Batch.
type Packer struct {
	daemonFreq        time.Duration
	exportMetricsFreq time.Duration
	retryDelay        time.Duration

	ipfsApis []ipfsutil.IpfsAPI
	store    *store.Store
	pinner   *httpapi.HttpApi
	mb       mbroker.MsgBroker

	carUploader  CARUploader
	carExportURL *url.URL

	daemonCtx       context.Context
	daemonCancelCtx context.CancelFunc
	daemonClosed    chan struct{}

	metricNewBatch                metric.Int64Counter
	metricBatchSizeTotal          metric.Int64Counter
	statLastBatch                 time.Time
	metricLastBatchCreated        metric.Int64ValueObserver
	statLastBatchCount            int64
	metricLastBatchCount          metric.Int64ValueObserver
	statLastBatchSize             int64
	metricLastBatchSize           metric.Int64ValueObserver
	statLastBatchDuration         int64
	metricLastBatchDurationMillis metric.Int64ValueObserver
}

// CARUploader provides blob storage for CAR files.
type CARUploader interface {
	Store(context.Context, string, io.Reader) (string, error)
}

// New returns a new Packer.
func New(
	postgresURI string,
	pinnerClient *httpapi.HttpApi,
	ipfsEndpoints []multiaddr.Multiaddr,
	mb mbroker.MsgBroker,
	opts ...Option) (*Packer, error) {
	cfg := defaultConfig
	for _, op := range opts {
		if err := op(&cfg); err != nil {
			return nil, fmt.Errorf("applying option: %s", err)
		}
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %s", err)
	}

	batchMaxSize := calcBatchLimit(cfg.sectorSize)
	store, err := store.New(postgresURI, batchMaxSize, cfg.batchMinSize)
	if err != nil {
		return nil, fmt.Errorf("init store: %s", err)
	}

	ipfsApis := make([]ipfsutil.IpfsAPI, len(ipfsEndpoints))
	for i, endpoint := range ipfsEndpoints {
		api, err := httpapi.NewApi(endpoint)
		if err != nil {
			return nil, fmt.Errorf("creating ipfs api: %s", err)
		}
		coreapi, err := api.WithOptions(options.Api.Offline(true))
		if err != nil {
			return nil, fmt.Errorf("creating offline core api: %s", err)
		}
		ipfsApis[i] = ipfsutil.IpfsAPI{Address: endpoint, API: coreapi}
	}

	ctx, cls := context.WithCancel(context.Background())
	p := &Packer{
		daemonFreq:        cfg.daemonFreq,
		exportMetricsFreq: cfg.exportMetricsFreq,
		retryDelay:        cfg.retryDelay,
		carUploader:       cfg.carUploader,
		carExportURL:      cfg.carExportURL,
		ipfsApis:          ipfsApis,

		store:  store,
		pinner: pinnerClient,
		mb:     mb,

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
func (p *Packer) ReadyToBatch(
	ctx context.Context,
	opID mbroker.OperationID,
	rtbds []mbroker.ReadyToBatchData) (err error) {
	start := time.Now()
	ctx, err = p.store.CtxWithTx(ctx)
	if err != nil {
		return fmt.Errorf("creating ctx with tx: %s", err)
	}
	defer func() {
		err = storeutil.FinishTxForCtx(ctx, err)
	}()

	for _, rtbd := range rtbds {
		srID := rtbd.StorageRequestID
		dataCid := rtbd.DataCid
		log.Debugf("received ready to pack storage-request %s, data-cid %s", srID, dataCid)

		node, err := p.pinner.Dag().Get(ctx, dataCid)
		if err != nil {
			return fmt.Errorf("get node for data-cid %s: %s", dataCid, err)
		}
		stat, err := node.Stat()
		if err != nil {
			return fmt.Errorf("get stat for data-cid %s: %s", dataCid, err)
		}
		size := int64(stat.CumulativeSize)

		if err := p.store.AddStorageRequestToOpenBatch(
			ctx,
			string(opID),
			srID,
			dataCid,
			size,
			rtbd.Origin,
		); err != nil {
			return fmt.Errorf("add storage-request to open batch: %w", err)
		}
	}
	log.Debugf("packing %d storage-requests took %dms", len(rtbds), time.Since(start).Milliseconds())

	return nil
}

// Close closes the packer.
func (p *Packer) Close() error {
	log.Info("closing daemon...")
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
			log.Info("daemon closed")
			return
		case <-time.After(p.daemonFreq):
			for {
				count, err := p.pack(p.daemonCtx)
				if err != nil {
					log.Errorf("packing: %s", err)
					p.metricNewBatch.Add(p.daemonCtx, 1, metrics.AttrOK)
					break
				}
				if count == 0 {
					break
				}
				p.metricNewBatch.Add(p.daemonCtx, 1, metrics.AttrError)
			}
		}
	}
}

func (p *Packer) pack(ctx context.Context) (int, error) {
	ctx, cls := context.WithTimeout(ctx, time.Hour)
	defer cls()
	batchID, batchSize, srs, origin, ok, err := p.store.GetNextReadyBatch(ctx)
	if err != nil {
		return 0, fmt.Errorf("get next ready batch: %s", err)
	}
	if !ok {
		return 0, nil
	}

	log.Debugf("preparing ready batch-id %s with %d storage-request", batchID, len(srs))
	start := time.Now()
	batchCid, manifest, carURL, err := p.createDAGForBatch(ctx, srs)
	if err != nil {
		return 0, fmt.Errorf("creating dag for batch: %s", err)
	}

	if err := p.store.MoveBatchToStatus(ctx, batchID, p.retryDelay, store.StatusDone); err != nil {
		return 0, fmt.Errorf("moving batch %s to done: %s", batchID, err)
	}

	srIDs := make([]broker.StorageRequestID, len(srs))
	for i, sr := range srs {
		srIDs[i] = sr.StorageRequestID
	}
	if err := mbroker.PublishMsgNewBatchCreated(
		ctx,
		p.mb,
		batchID,
		batchCid,
		srIDs,
		origin,
		manifest,
		carURL); err != nil {
		return 0, fmt.Errorf("publishing msg to broker: %s", err)
	}

	label := attribute.String("origin", origin)
	p.metricBatchSizeTotal.Add(ctx, batchSize, label)
	p.statLastBatch = time.Now()
	p.statLastBatchCount = int64(len(srIDs))
	p.statLastBatchSize = batchSize
	p.statLastBatchDuration = time.Since(start).Milliseconds()
	log.Infof(
		"created batch-id: %s, batch-cid: %s, num-storage-requests: %d, batch-size: %d",
		batchID, batchCid, len(srIDs), batchSize)

	return len(srIDs), nil
}

func (p *Packer) createDAGForBatch(ctx context.Context, srs []store.StorageRequest) (cid.Cid, []byte, string, error) {
	lst := make([]aggregator.AggregateDagEntry, len(srs))
	for i, sr := range srs {
		dataCid, err := cid.Decode(sr.DataCid)
		if err != nil {
			return cid.Undef, nil, "", fmt.Errorf("decoding cid %s: %s", dataCid, err)
		}
		lst[i] = aggregator.AggregateDagEntry{
			RootCid:                   dataCid,
			UniqueBlockCumulativeSize: uint64(sr.Size),
		}
	}
	start := time.Now()
	root, entries, err := aggregator.Aggregate(ctx, p.pinner.Dag(), lst)
	if err != nil {
		return cid.Undef, nil, "", fmt.Errorf("aggregating cids: %s", err)
	}
	log.Debugf("aggregation took %dms", time.Since(start).Milliseconds())

	if err := p.pinBatchDAG(ctx, root); err != nil {
		return cid.Undef, nil, "", fmt.Errorf("pinning batch dag: %s", err)
	}

	manifestJSON := bytes.NewBuffer(nil)
	if err := aggregator.EncodeManifestJSON(entries, manifestJSON); err != nil {
		return cid.Undef, nil, "", fmt.Errorf("encoding manifest json: %s", err)
	}
	log.Debugf("aggregation+pinning took %dms", time.Since(start).Milliseconds())

	carURL, err := p.getCARURL(ctx, root)
	if err != nil {
		return cid.Undef, nil, "", fmt.Errorf("get car url: %s", err)
	}

	return root, manifestJSON.Bytes(), carURL, nil
}

func (p *Packer) pinBatchDAG(ctx context.Context, root cid.Cid) error {
	log.Debugf("pinning %s", root)
	if err := p.pinner.Pin().Add(ctx, ipfspath.IpfsPath(root), options.Pin.Recursive(true)); err != nil {
		return fmt.Errorf("pinning batch root: %s", err)
	}
	// When using ipfs-cluster, the pinning API pins the DAG async.
	// We want to confirm the DAG is pinned in some underlying go-ipfs node before moving on as
	// to avoid noise while getting the corresponding go-ipfs node when trying to use the data.
	for {
		log.Debugf("confirming dag %s is pinned", root)
		ok, err := ipfsutil.IsPinned(ctx, p.ipfsApis, root)
		if err != nil || !ok {
			log.Debugf("dag %s isn't confirmed to be pinned (err: %s, ok: %v)", root, err, ok)
			time.Sleep(time.Second * 10)
			continue
		}
		break
	}
	log.Debugf("dag %s pinning confirmed", root)

	return nil
}

func (p *Packer) getCARURL(ctx context.Context, root cid.Cid) (string, error) {
	if p.carUploader != nil {
		start := time.Now()
		log.Debugf("uploading generating and uploading CAR file to external bucket")
		ng, found := ipfsutil.GetNodeGetterForCid(p.ipfsApis, root)
		if !found {
			return "", fmt.Errorf("get node getter for cid %s not found", root)
		}
		pr, pw := io.Pipe()
		defer func() {
			_ = pr.Close()
		}()
		go func() {
			defer func() {
				_ = pw.Close()
			}()

			if err := car.WriteCar(ctx, ng, []cid.Cid{root}, pw); err != nil {
				err = fmt.Errorf("generating car %s: %v", root, err)
				log.Errorf("writing car file to car uploader: %s", err)
				_ = pw.CloseWithError(err)
			}
		}()
		carURL, err := p.carUploader.Store(ctx, root.String(), pr)
		if err != nil {
			return "", fmt.Errorf("uploading car file: %s", err)
		}
		log.Debugf("car generated in external url %s in %.2f seconds", carURL, time.Since(start).Seconds())

		return carURL, nil
	}

	cidURL, err := url.Parse(root.String())
	if err != nil {
		return "", fmt.Errorf("creating cid url fragment: %s", err)
	}
	carURL := p.carExportURL.ResolveReference(cidURL).String()
	log.Debugf("car generated in dynamic endpoint url %s", carURL)

	return carURL, nil
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
