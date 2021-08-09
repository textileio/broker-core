package piecer

import (
	"context"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	commcid "github.com/filecoin-project/go-fil-commcid"
	commP "github.com/filecoin-project/go-fil-commp-hashhash"
	"github.com/ipfs/go-cid"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	"github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/ipld/go-car"
	"github.com/multiformats/go-multiaddr"
	"github.com/textileio/broker-core/broker"
	store "github.com/textileio/broker-core/cmd/piecerd/store"
	"github.com/textileio/broker-core/ipfsutil"
	mbroker "github.com/textileio/broker-core/msgbroker"
	logger "github.com/textileio/go-log/v2"
	"go.opentelemetry.io/otel/metric"
)

const maxPaddingSize = 32 << 30

var log = logger.Logger("piecer")

// Piecer provides a data-preparation pipeline for Batchs.
type Piecer struct {
	mb       mbroker.MsgBroker
	ipfsApis []ipfsutil.IpfsAPI

	store           *store.Store
	daemonFrequency time.Duration
	retryDelay      time.Duration
	padToSize       uint64

	newRequest      chan struct{}
	daemonCtx       context.Context
	daemonCancelCtx context.CancelFunc
	daemonClosed    chan struct{}

	statLastSize              int64
	metricLastSize            metric.Int64ValueObserver
	statLastDurationSeconds   int64
	metricLastDurationSeconds metric.Int64ValueObserver
	metricNewPrepare          metric.Int64Counter
	statLastPrepared          time.Time
	metricLastPrepared        metric.Int64ValueObserver
}

// New returns a new Piecer.
func New(
	postgresURI string,
	ipfsEndpoints []multiaddr.Multiaddr,
	mb mbroker.MsgBroker,
	daemonFrequency time.Duration,
	retryDelay time.Duration,
	padToSize uint64) (*Piecer, error) {
	ipfsApis := make([]ipfsutil.IpfsAPI, len(ipfsEndpoints))
	if padToSize&(padToSize-1) != 0 {
		return nil, fmt.Errorf("pad to size %d must be a positive power of two", padToSize)
	}

	if padToSize > maxPaddingSize {
		return nil, fmt.Errorf("pad to size can't be greater than 32GiB")
	}

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

	s, err := store.New(postgresURI)
	if err != nil {
		return nil, fmt.Errorf("initializing store: %s", err)
	}
	ctx, cls := context.WithCancel(context.Background())
	p := &Piecer{
		store:    s,
		ipfsApis: ipfsApis,
		mb:       mb,

		daemonFrequency: daemonFrequency,
		retryDelay:      retryDelay,
		padToSize:       padToSize,
		newRequest:      make(chan struct{}, 1),

		daemonCtx:       ctx,
		daemonCancelCtx: cls,
		daemonClosed:    make(chan struct{}),
	}
	p.initMetrics()
	go p.daemon()

	return p, nil
}

// ReadyToPrepare signals the Piecer that a new batch is ready to be prepared.
func (p *Piecer) ReadyToPrepare(ctx context.Context, sdID broker.BatchID, dataCid cid.Cid) error {
	if sdID == "" {
		return fmt.Errorf("batch id is empty")
	}
	if !dataCid.Defined() {
		return fmt.Errorf("data-cid is undefined")
	}

	if err := p.store.CreateUnpreparedBatch(ctx, sdID, dataCid); err != nil {
		return fmt.Errorf("creating unprepared-batch %s %s: %w", sdID, dataCid, err)
	}
	log.Debugf("saved unprepared-batch with batch %s and data-cid %s", sdID, dataCid)

	select {
	case p.newRequest <- struct{}{}:
	default:
	}

	return nil
}

// Close closes the piecer.
func (p *Piecer) Close() error {
	log.Info("closing piecer...")
	p.daemonCancelCtx()
	<-p.daemonClosed
	if err := p.store.Close(); err != nil {
		return fmt.Errorf("closing store: %s", err)
	}
	return nil
}

func (p *Piecer) daemon() {
	defer close(p.daemonClosed)

	p.newRequest <- struct{}{}
	for {
		select {
		case <-p.daemonCtx.Done():
			log.Info("piecer closed")
			return
		case <-p.newRequest:
		case <-time.After(p.daemonFrequency):
		}
		for {
			usd, ok, err := p.store.GetNextPending(p.daemonCtx)
			if err != nil {
				log.Errorf("get next unprepared batch: %s", err)
				break
			}
			if !ok {
				break
			}

			if err := p.prepare(p.daemonCtx, usd); err != nil {
				log.Errorf("preparing batch %s, data-cid %s: %s", usd.BatchID, usd.DataCid, err)
				if err := p.store.MoveToStatus(p.daemonCtx, usd.BatchID, p.retryDelay, store.StatusPending); err != nil {
					log.Errorf("moving again to pending: %s", err)
				}
				break
			}

			if err := p.store.MoveToStatus(p.daemonCtx, usd.BatchID, 0, store.StatusDone); err != nil {
				log.Errorf("deleting batch %s, data-cid %s: %s", usd.BatchID, usd.DataCid, err)
				if err := p.store.MoveToStatus(p.daemonCtx, usd.BatchID, p.retryDelay, store.StatusPending); err != nil {
					log.Errorf("moving again to pending: %s", err)
				}
				break
			}
		}
	}
}

func (p *Piecer) prepare(ctx context.Context, usd store.UnpreparedBatch) error {
	start := time.Now()
	log.Debugf("preparing batch %s with data-cid %s", usd.BatchID, usd.DataCid)

	nodeGetter, found := ipfsutil.GetNodeGetterForCid(p.ipfsApis, usd.DataCid)
	if !found {
		return fmt.Errorf("node getter for data cid %s not found", usd.DataCid)
	}

	prCAR, pwCAR := io.Pipe()
	var errCarGen error
	go func() {
		defer func() {
			if err := pwCAR.Close(); err != nil {
				errCarGen = err
			}
		}()
		if err := car.WriteCar(ctx, nodeGetter, []cid.Cid{usd.DataCid}, pwCAR); err != nil {
			errCarGen = err
			return
		}
	}()

	var (
		errCommP error
		wg       sync.WaitGroup
		dpr      broker.DataPreparationResult
	)
	wg.Add(1)
	go func() {
		defer wg.Done()

		cp := &commP.Calc{}
		_, err := io.Copy(cp, prCAR)
		if err != nil {
			errCommP = fmt.Errorf("copying data to aggregator: %s", err)
			return
		}

		rawCommP, ps, err := cp.Digest()
		if err != nil {
			errCommP = fmt.Errorf("calculating final digest: %s", err)
			return
		}

		if ps < p.padToSize {
			log.Debugf("padding commP from %d to %d", ps, p.padToSize)
			rawCommP, err = commP.PadCommP(rawCommP, ps, p.padToSize)
			if err != nil {
				errCommP = fmt.Errorf("padding commp: %s", err)
				return
			}
			ps = p.padToSize
		}

		pcid, err := commcid.DataCommitmentV1ToCID(rawCommP)
		if err != nil {
			errCommP = fmt.Errorf("converting commP to cid: %s", err)
			return
		}

		dpr = broker.DataPreparationResult{
			PieceSize: ps,
			PieceCid:  pcid,
		}
	}()
	wg.Wait()
	if errCarGen != nil || errCommP != nil {
		return fmt.Errorf("write car err: %s, commP err: %s", errCarGen, errCommP)
	}

	duration := time.Since(start).Seconds()
	log.Debugf("prepared of batch %s, data-cid %s, piece-size %s, piece-cid %s took %.2f seconds",
		usd.BatchID, usd.DataCid, humanize.IBytes(dpr.PieceSize), dpr.PieceCid, duration)

	if err := mbroker.PublishMsgNewBatchPrepared(ctx, p.mb, usd.BatchID, dpr.PieceCid, dpr.PieceSize); err != nil {
		return fmt.Errorf("publish message to message broker: %s", err)
	}

	p.metricNewPrepare.Add(ctx, 1)
	p.statLastPrepared = time.Now()
	p.statLastSize = int64(dpr.PieceSize)
	p.statLastDurationSeconds = int64(duration)

	return nil
}
