package dealer

import (
	"context"
	"fmt"
	"time"

	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/dealerd/dealer/store"
)

func (d *Dealer) daemonDealReporter() {
	defer d.daemonWg.Done()

	for {
		select {
		case <-d.daemonCtx.Done():
			log.Infof("deal reporter daemon closed")
			return
		case <-time.After(d.config.dealReportingFreq):
			if err := d.daemonDealReporterTick(); err != nil {
				log.Errorf("deal reporter tick: %s", err)
			}
		}
	}
}

func (d *Dealer) daemonDealReporterTick() error {
	for {
		adSuccess, err := d.store.GetAllAuctionDeals(store.Success)
		if err != nil {
			return fmt.Errorf("get successful deals: %s", err)
		}
		adError, err := d.store.GetAllAuctionDeals(store.Error)
		if err != nil {
			return fmt.Errorf("get errored deals: %s", err)
		}
		ads := append(adSuccess, adError...)
		if len(ads) == 0 {
			break
		}

		res := make([]broker.FinalizedAuctionDeal, len(ads))
		for i, aud := range ads {
			ad, err := d.store.GetAuctionData(aud.AuctionDataID)
			if err != nil {
				return fmt.Errorf("get auction data: %s", err)
			}
			res[i] = broker.FinalizedAuctionDeal{
				StorageDealID:  ad.StorageDealID,
				ErrorCause:     aud.ErrorCause,
				DealID:         aud.DealID,
				DealExpiration: aud.DealExpiration,
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		// We report finalized Auction Deals to the broker.
		if err := d.broker.StorageDealFinalizedDeals(ctx, res); err != nil {
			return fmt.Errorf("reporting auction deal results to the broker: %s", err)
		}

		// We are safe to remove those from our store. This will indirectly remove also the linked
		// AuctionData, if no pending/in-progress AuctionDeals exist for them.
		if err := d.store.RemoveAuctionDeals(ads); err != nil {
			return fmt.Errorf("removing auction deals: %s", err)
		}
	}

	return nil
}
