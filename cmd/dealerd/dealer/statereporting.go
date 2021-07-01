package dealer

import (
	"context"
	"fmt"
	"time"

	"github.com/textileio/bidbot/lib/broker"
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
		aud, ok, err := d.store.GetNext(store.PendingReportFinalized)
		if err != nil {
			return fmt.Errorf("get successful deals: %s", err)
		}
		if !ok {
			break
		}
		if err := d.reportFinalizedAuctionDeal(aud); err != nil {
			log.Errorf("reporting finalized auction deal: %s", err)
			aud.ReadyAt = time.Now().Add(d.config.dealReportingRetryDelay)
			if err := d.store.SaveAndMoveAuctionDeal(aud, store.PendingReportFinalized); err != nil {
				return fmt.Errorf("saving reached deadline: %s", err)
			}
			return nil
		}
	}

	return nil
}

func (d *Dealer) reportFinalizedAuctionDeal(aud store.AuctionDeal) error {
	ad, err := d.store.GetAuctionData(aud.AuctionDataID)
	if err != nil {
		return fmt.Errorf("get auction data: %s", err)
	}
	fad := broker.FinalizedAuctionDeal{
		StorageDealID:  ad.StorageDealID,
		ErrorCause:     aud.ErrorCause,
		DealID:         aud.DealID,
		DealExpiration: aud.DealExpiration,
		Miner:          aud.Miner,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	log.Debugf("reporting finalized auction deal (errorcause=%s)", aud.ErrorCause)
	// We report finalized Auction Deals to the broker.
	if err := d.broker.StorageDealFinalizedDeal(ctx, fad); err != nil {
		return fmt.Errorf("reporting auction deal results to the broker: %s", err)
	}

	// We are safe to remove it from our store. This will indirectly remove also the linked
	// AuctionData, if no pending/in-progress AuctionDeals exist for them.
	if err := d.store.RemoveAuctionDeal(aud); err != nil {
		return fmt.Errorf("removing auction deals: %s", err)
	}

	return nil
}
