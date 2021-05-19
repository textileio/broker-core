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
	"github.com/multiformats/go-multiaddr"
	"github.com/textileio/broker-core/broker"
	"go.opentelemetry.io/otel/metric"
)

var log = logger.Logger("piecer")

// Piecer provides a data-preparation pipeline for StorageDeals.
type Piecer struct {
	broker broker.Broker
	ipfs   *httpapi.HttpApi

	checkQueue chan struct{}
	lock       sync.Mutex
	queue      []pendingStorageDeal

	onceClose       sync.Once
	daemonCtx       context.Context
	daemonCancelCtx context.CancelFunc
	daemonClosed    chan struct{}

	statLastPrepared   time.Time
	metricNewPrepare   metric.Int64Counter
	metricLastPrepared metric.Int64ValueObserver
}

// New returns a nice Piecer.
func New(ipfsAPIMultiaddr string) (*Piecer, error) {
	ma, err := multiaddr.NewMultiaddr(ipfsAPIMultiaddr)
	if err != nil {
		return nil, fmt.Errorf("parsing ipfs client multiaddr: %s", err)
	}
	client, err := httpapi.NewApi(ma)
	if err != nil {
		return nil, fmt.Errorf("creating ipfs client: %s", err)
	}

	ctx, cls := context.WithCancel(context.Background())
	p := &Piecer{
		ipfs:            client,
		checkQueue:      make(chan struct{}, 1),
		daemonCtx:       ctx,
		daemonCancelCtx: cls,
		daemonClosed:    make(chan struct{}),
	}

	p.initMetrics()

	go p.daemon()

	return p, nil
}

// SetBroker is a *temporary* method that will go away. Now just
// being useful for the embbeded piecer in brokerd.
func (p *Piecer) SetBroker(b broker.Broker) {
	p.broker = b
}

// ReadyToPrepare signals the Piecer that a new StorageDeal is ready to be prepared.
// Piecer will call the broker async with the end result.
func (p *Piecer) ReadyToPrepare(ctx context.Context, id broker.StorageDealID, dataCid cid.Cid) error {
	p.lock.Lock()
	defer p.lock.Unlock()
	p.queue = append(p.queue, pendingStorageDeal{id: id, dataCid: dataCid})

	select {
	case p.checkQueue <- struct{}{}:
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
	for {
		select {
		case <-p.daemonCtx.Done():
			log.Infof("piecer closed")
			return
		case <-p.checkQueue:
			for {
				p.lock.Lock()
				if len(p.queue) == 0 {
					p.lock.Unlock()
					break
				}
				tip := p.queue[0]
				p.lock.Unlock()
				if err := p.prepare(p.daemonCtx, tip); err != nil {
					log.Errorf("preparing: %s", err)
				}
				p.lock.Lock()
				p.queue = p.queue[1:]
				p.lock.Unlock()
			}
		}
	}
}

func (p *Piecer) prepare(ctx context.Context, psd pendingStorageDeal) error {
	start := time.Now()
	log.Debugf("preparing storage deal %s with cid %s", psd.id, psd.dataCid)

	prCAR, pwCAR := io.Pipe()
	var errCarGen error
	go func() {
		defer func() {
			if err := pwCAR.Close(); err != nil {
				errCarGen = err
			}
		}()
		if err := car.WriteCar(ctx, p.ipfs.Dag(), []cid.Cid{psd.dataCid}, pwCAR); err != nil {
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
	log.Debugf("preparation of storage deal %s took %.2f seconds", psd.id, time.Since(start).Seconds())

	if err := p.broker.StorageDealPrepared(ctx, psd.id, dpr); err != nil {
		return fmt.Errorf("signaling broker that storage deal is prepared: %s", err)
	}

	p.metricNewPrepare.Add(ctx, 1)
	p.statLastPrepared = time.Now()

	return nil
}

type pendingStorageDeal struct {
	id      broker.StorageDealID
	dataCid cid.Cid
}
