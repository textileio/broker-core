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
	logger "github.com/ipfs/go-log/v2"
	"github.com/ipld/go-car"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/piecerd/piecer/store"
	"github.com/textileio/broker-core/dshelper/txndswrap"
	pieceri "github.com/textileio/broker-core/piecer"
	"go.opentelemetry.io/otel/metric"
)

var log = logger.Logger("piecer")

// Piecer provides a data-preparation pipeline for StorageDeals.
type Piecer struct {
	broker broker.Broker
	ipfs   *httpapi.HttpApi

	store           *store.Store
	daemonFrequency time.Duration
	newRequest      chan struct{}

	onceClose       sync.Once
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

var _ pieceri.Piecer = (*Piecer)(nil)

// New returns a new Piecer.
func New(ds txndswrap.TxnDatastore, ipfs *httpapi.HttpApi, b broker.Broker, daemonFrequency time.Duration) (*Piecer, error) {
	ctx, cls := context.WithCancel(context.Background())
	p := &Piecer{
		store:  store.New(txndswrap.Wrap(ds, "/store")),
		ipfs:   ipfs,
		broker: b,

		daemonFrequency: daemonFrequency,
		newRequest:      make(chan struct{}),

		daemonCtx:       ctx,
		daemonCancelCtx: cls,
		daemonClosed:    make(chan struct{}),
	}
	p.initMetrics()
	go p.daemon()

	return p, nil
}

// ReadyToPrepare signals the Piecer that a new StorageDeal is ready to be prepared.
// Piecer will call the broker async with the end result.
func (p *Piecer) ReadyToPrepare(ctx context.Context, id broker.StorageDealID, dataCid cid.Cid) error {
	if id == "" {
		return fmt.Errorf("storage deal id is empty")
	}
	if !dataCid.Defined() {
		return fmt.Errorf("data-cid is undefined")
	}

	if err := p.store.Create(id, dataCid); err != nil {
		return fmt.Errorf("saving %s %s: %s", id, dataCid, err)
	}

	select {
	case p.newRequest <- struct{}{}:
	default:
	}

	return nil
}

// Close closes the piecer.
func (p *Piecer) Close() error {
	p.onceClose.Do(func() {
		p.daemonCancelCtx()
		<-p.daemonClosed
	})
	return nil
}

func (p *Piecer) daemon() {
	defer close(p.daemonClosed)

	p.newRequest <- struct{}{}
	for {
		select {
		case <-p.daemonCtx.Done():
			log.Infof("piecer closed")
			return
		case <-p.newRequest:
		case <-time.After(p.daemonFrequency):
		}
		for {
			usd, ok, err := p.store.GetNext()
			if err != nil {
				log.Errorf("get next unprepared batch: %s", err)
				break
			}
			if !ok {
				log.Debug("no remaning unprepared batches")
			}

			if err := p.prepare(p.daemonCtx, usd); err != nil {
				log.Errorf("preparing: %s", err)
				if err := p.store.MoveToPending(usd.ID); err != nil {
					log.Errorf("moving again to pending: %s", err)
				}
				break
			}

			if err := p.store.Delete(usd.ID); err != nil {
				log.Errorf("deleting: %s", err)
				if err := p.store.MoveToPending(usd.ID); err != nil {
					log.Errorf("moving again to pending: %s", err)
				}
				break
			}
		}
	}
}

func (p *Piecer) prepare(ctx context.Context, usd store.UnpreparedStorageDeal) error {
	start := time.Now()
	log.Debugf("preparing storage deal %s with data-cid %s", usd.StorageDealID, usd.DataCid)

	prCAR, pwCAR := io.Pipe()
	var errCarGen error
	go func() {
		defer func() {
			if err := pwCAR.Close(); err != nil {
				errCarGen = err
			}
		}()
		if err := car.WriteCar(ctx, p.ipfs.Dag(), []cid.Cid{usd.DataCid}, pwCAR); err != nil {
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

	log.Debugf("piece-size: %s, piece-cid: %s", humanize.IBytes(dpr.PieceSize), dpr.PieceCid)
	log.Debugf("preparation of storage deal %s took %.2f seconds", usd.StorageDealID, time.Since(start).Seconds())

	if err := p.broker.StorageDealPrepared(ctx, usd.StorageDealID, dpr); err != nil {
		return fmt.Errorf("signaling broker that storage deal is prepared: %s", err)
	}

	p.metricNewPrepare.Add(ctx, 1)
	p.statLastPrepared = time.Now()

	return nil
}
