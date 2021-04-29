package dealer

import (
	"fmt"
	"time"

	"github.com/textileio/broker-core/cmd/dealerd/dealer/store"
	"github.com/textileio/broker-core/ratelim"
)

func (d *Dealer) daemonDealReporter() {
	defer d.daemonWg.Done()

	for {
		select {
		case <-d.daemonCtx.Done():
			log.Infof("deal reporter daemon closed")
			return
		case <-time.After(dealWatcherFreq):
			if err := d.daemonDealReporterTick(); err != nil {
				log.Errorf("deal reporter tick: %s", err)
			}
		}
	}
}

func (d *Dealer) daemonDealReporterTick() error {
	rl, err := ratelim.New(dealWatcherRateLim)
	if err != nil {
		return fmt.Errorf("create ratelim: %s", err)
	}
	defer rl.Wait()

	for {
		adSuccess, err := d.store.GetAllAuctionDeals(store.Success)
		if err != nil {
			return fmt.Errorf("get success deals: %s", err)
		}
		adError, err := d.store.GetAllAuctionDeals(store.Error)
		if err != nil {
			return fmt.Errorf("get success deals: %s", err)
		}
		ads := append(adSuccess, adError...)
		if len(ads) == 0 {
			break
		}

		// We report finalized Auction Deals to the broker.
		if err := d.broker.ReportAuctionDealResults(ctx, results); err != nil {
			return fmt.Errorf("reporting auction deal results to the broker: %s", err)
		}

		// We are safe to remove those from our store. This will indirectly remove also the linked
		// AuctionData, if no pending/in-progress AuctionDeals exist for them.
		if err := d.store.RemoveAuctionDeals(auctionDealIDs); err != nil {
			return fmt.Errorf("removing auction deals: %s", err)
		}
	}

	return nil
}
