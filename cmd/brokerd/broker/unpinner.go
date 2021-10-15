package broker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ipfs/go-cid"
	cmds "github.com/ipfs/go-ipfs-cmds"
	"github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/textileio/broker-core/cmd/brokerd/store"
	"github.com/textileio/broker-core/metrics"
)

func (b *Broker) daemonUnpinner() {
	defer close(b.daemonClosed)

	go b.exportIPFSMetrics()
	for {
		select {
		case <-b.daemonCtx.Done():
			log.Info("broker unpinner daemon closed")
			return
		case <-time.After(b.conf.unpinnerFrequency):
		}
		for {
			uj, ok, err := b.store.UnpinJobGetNext(b.daemonCtx)
			if err != nil {
				log.Errorf("get next unpin job: %s", err)
				break
			}
			if !ok {
				log.Debug("no remaning unpin jobs")
				break
			}

			if err := b.unpinCid(b.daemonCtx, uj); err != nil {
				log.Errorf("unpinning %s: %s", uj.Cid, err)
				if err := b.store.UnpinJobMoveToPending(b.daemonCtx, uj.ID, b.conf.unpinnerRetryDelay); err != nil {
					log.Errorf("moving again to pending: %s", err)
				}
				continue
			}

			if err := b.store.DeleteExecuting(b.daemonCtx, uj.ID); err != nil {
				log.Errorf("removing finalized unpin job: %s", err)
				if err := b.store.UnpinJobMoveToPending(b.daemonCtx, uj.ID, b.conf.unpinnerRetryDelay); err != nil {
					log.Errorf("unpin job moving again to pending: %s", err)
				}
			}
		}
	}
}

func (b *Broker) unpinCid(ctx context.Context, uj store.UnpinJob) (err error) {
	defer func() {
		metrics.MetricIncrCounter(ctx, err, b.metricUnpinTotal)
	}()
	cid, err := cid.Parse(uj.Cid)
	if err != nil {
		return err
	}
	log.Debugf("unpinning %s", uj.Cid)
	if err := b.ipfsClient.Pin().Rm(ctx, path.IpfsPath(cid), options.Pin.RmRecursive(true)); err != nil {
		// (jsign): This is a workaround for a bug we discovered in the ipfs-cluster IPFS proxy.
		//          https://github.com/ipfs/ipfs-cluster/issues/1366. For now we'll assume this error
		//          is always that the data was already unpinned most probably due to some retry.
		//          We should change this whenever the bug is resolved.
		var ctmErr *cmds.Error
		if ok := errors.As(err, &ctmErr); ok && ctmErr.Code == 0 {
			log.Warnf("%s unpinning was skipped due to ipfs-cluster known bug: %s", uj.Cid, err)
			return nil
		}
		return fmt.Errorf("unpinning %s: %s", uj.Cid, err)
	}
	log.Debugf("%s was unpinned", uj.Cid)
	return nil
}

func (b *Broker) exportIPFSMetrics() {
	for {
		<-time.After(b.conf.exportPinCountFrequency)
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*120)
		ch, err := b.ipfsClient.Pin().Ls(ctx, options.Pin.Ls.Recursive())
		if err != nil {
			log.Errorf("getting total pin count: %s", err)
			cancel()
			continue
		}
		cancel()
		var total int64
		for range ch {
			total++
		}
		b.statTotalRecursivePins = total
	}
}
