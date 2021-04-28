package dealer

import (
	"fmt"
	"time"

	"github.com/textileio/broker-core/cmd/dealerd/dealer/store"
	"github.com/textileio/broker-core/ratelim"
)

var (
	dealMakerFreq    = time.Second * 5
	dealMakerRateLim = 20
)

func (d *Dealer) daemonDealMaker() {
	defer d.daemonWg.Done()

	for {
		select {
		case <-d.daemonCtx.Done():
			log.Infof("deal maker daemon closed")
			return
		case <-time.After(dealMakerFreq):
			if err := d.daemonDealMakerTick(); err != nil {
				log.Errorf("deal maker tick: %s", err)
			}
		}
	}
}

func (d *Dealer) daemonDealMakerTick() error {
	rl, err := ratelim.New(dealMakerRateLim)
	if err != nil {
		return fmt.Errorf("create ratelim: %s", err)
	}
	defer rl.Wait()

	for {
		select {
		case <-d.daemonCtx.Done():
			break
		default:
		}

		p, err := d.store.GetNextPending()
		if err == store.ErrNotFound {
			break
		}
		if err != nil {
			log.Errorf("get next pending deal: %s", err)
			break
		}

		rl.Exec(func() error {
			if err := d.executePending(p); err != nil {
				log.Errorf("executing Pending: %s", err)
			}
			// We're not interested in ratelim error inspection.
			return nil
		})
	}

	return nil
}

func (d *Dealer) executePending(aud store.AuctionDeal) error {
	ad, err := d.store.GetAuctionData(aud.AuctionDataID)
	if err != nil {
		return fmt.Errorf("get auction data %s: %s", aud.AuctionDataID, err)
	}

	if err := d.store.StatusChange(aud.ID, store.WaitingConfirmation); err != nil {
		return fmt.Errorf("changing status to WaitingConfirmation: %s", err)
	}

	return nil
}

func (d *Dealer) daemonDealWatcher() {
	defer d.daemonWg.Done()
}

func (d *Dealer) daemonDealReporter() {
	defer d.daemonWg.Done()
}
