package broker

import (
	"context"
	"fmt"
	"time"

	"github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/textileio/broker-core/cmd/brokerd/store"
)

func (b *Broker) daemonUnpinner() {
	defer close(b.daemonClosed)

	for {
		select {
		case <-b.daemonCtx.Done():
			log.Info("broker unpinner daemon closed")
			return
		case <-time.After(b.conf.unpinnerFrequency):
		}
		for {
			uj, ok, err := b.store.UnpinJobGetNext()
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
				if err := b.store.UnpinJobMoveToPending(uj.ID, b.conf.unpinnerRetryDelay); err != nil {
					log.Errorf("moving again to pending: %s", err)
				}
				break
			}

			if err := b.store.DeleteExecuting(uj.ID); err != nil {
				log.Errorf("removing finalized unpin job: %s", err)
				if err := b.store.UnpinJobMoveToPending(uj.ID, b.conf.unpinnerRetryDelay); err != nil {
					log.Errorf("unpin job moving again to pending: %s", err)
				}
				break
			}
		}
	}
}

func (b *Broker) unpinCid(ctx context.Context, uj store.UnpinJob) error {
	log.Debugf("unpinning %s", uj.Cid)
	if err := b.ipfsClient.Pin().Rm(ctx, path.IpfsPath(uj.Cid), options.Pin.RmRecursive(true)); err != nil {
		return fmt.Errorf("unpinning %s: %s", uj.Cid, err)
	}
	return nil
}
