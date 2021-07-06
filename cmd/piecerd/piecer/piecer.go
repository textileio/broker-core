package piecer

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	commcid "github.com/filecoin-project/go-fil-commcid"
	commP "github.com/filecoin-project/go-fil-commp-hashhash"
	"github.com/ipfs/go-cid"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	format "github.com/ipfs/go-ipld-format"
	iface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/ipfs/interface-go-ipfs-core/options"
	ipfspath "github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/ipld/go-car"
	"github.com/multiformats/go-multiaddr"
	"github.com/textileio/bidbot/lib/dshelper/txndswrap"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/piecerd/piecer/store"
	pbBroker "github.com/textileio/broker-core/gen/broker/v1"
	"github.com/textileio/broker-core/msgbroker"
	pieceri "github.com/textileio/broker-core/piecer"
	logger "github.com/textileio/go-log/v2"
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/protobuf/proto"
)

var log = logger.Logger("piecer")

// Piecer provides a data-preparation pipeline for StorageDeals.
type Piecer struct {
	mb       msgbroker.MsgBroker
	ipfsApis []ipfsAPI

	store           *store.Store
	daemonFrequency time.Duration
	retryDelay      time.Duration
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

type ipfsAPI struct {
	address multiaddr.Multiaddr
	api     iface.CoreAPI
}

var _ pieceri.Piecer = (*Piecer)(nil)

// New returns a new Piecer.
func New(
	ds txndswrap.TxnDatastore,
	ipfsEndpoints []multiaddr.Multiaddr,
	mb msgbroker.MsgBroker,
	daemonFrequency time.Duration,
	retryDelay time.Duration) (*Piecer, error) {
	ipfsApis := make([]ipfsAPI, len(ipfsEndpoints))
	for i, endpoint := range ipfsEndpoints {
		api, err := httpapi.NewApi(endpoint)
		if err != nil {
			return nil, fmt.Errorf("creating ipfs api: %s", err)
		}
		coreapi, err := api.WithOptions(options.Api.Offline(true))
		if err != nil {
			return nil, fmt.Errorf("creating offline core api: %s", err)
		}
		ipfsApis[i] = ipfsAPI{address: endpoint, api: coreapi}
	}

	ctx, cls := context.WithCancel(context.Background())
	p := &Piecer{
		store:    store.New(txndswrap.Wrap(ds, "/store")),
		ipfsApis: ipfsApis,
		mb:       mb,

		daemonFrequency: daemonFrequency,
		retryDelay:      retryDelay,
		newRequest:      make(chan struct{}, 1),

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
	log.Info("closing piecer...")
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
			log.Info("piecer closed")
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
				break
			}

			if err := p.prepare(p.daemonCtx, usd); err != nil {
				log.Errorf("preparing: %s", err)
				if err := p.store.MoveToPending(usd.ID, p.retryDelay); err != nil {
					log.Errorf("moving again to pending: %s", err)
				}
				break
			}

			if err := p.store.Delete(usd.ID); err != nil {
				log.Errorf("deleting: %s", err)
				if err := p.store.MoveToPending(usd.ID, p.retryDelay); err != nil {
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

	nodeGetter, err := p.getNodeGetterForCid(usd.DataCid)
	if err != nil {
		return fmt.Errorf("get node getter for cid %s: %s", usd.DataCid, err)
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
	duration := time.Since(start).Seconds()
	log.Debugf("preparation of storage deal %s took %.2f seconds", usd.StorageDealID, duration)

	sdp := &pbBroker.StorageDealPreparedRequest{
		MsgId:         msgID,
		StorageDealId: string(usd.StorageDealID),
		PieceCid:      dpr.PieceCid.Bytes(),
		PieceSize:     dpr.PieceSize,
	}
	sdpBytes, err := proto.Marshal(sdp)
	if err != nil {
		return fmt.Errorf("signaling broker that storage deal is prepared: %s", err)
	}
	if err := p.mb.PublishMsg(ctx, "new-prepared-batch", sdpBytes); err != nil {
		return fmt.Errorf("publishing new-prepared-batch message: %s", err)
	}

	p.metricNewPrepare.Add(ctx, 1)
	p.statLastPrepared = time.Now()
	p.statLastSize = int64(dpr.PieceSize)
	p.statLastDurationSeconds = int64(duration)

	return nil
}

func (p *Piecer) getNodeGetterForCid(c cid.Cid) (format.NodeGetter, error) {
	var ng format.NodeGetter

	rand.Shuffle(len(p.ipfsApis), func(i, j int) {
		p.ipfsApis[i], p.ipfsApis[j] = p.ipfsApis[j], p.ipfsApis[i]
	})

	log.Debug("core-api lookup for cid")
	for _, coreapi := range p.ipfsApis {
		ctx, cls := context.WithTimeout(context.Background(), time.Second*5)
		defer cls()
		_, ok, err := coreapi.api.Pin().IsPinned(ctx, ipfspath.IpfsPath(c))
		if err != nil {
			log.Errorf("checking if %s is pinned in %s: %s", c, coreapi.address, err)
			continue
		}
		if !ok {
			continue
		}
		log.Debugf("found core-api for cid: %s", coreapi.address)
		ng = coreapi.api.Dag()
		break
	}

	if ng == nil {
		return nil, fmt.Errorf("node getter for cid not found")
	}

	return ng, nil
}
