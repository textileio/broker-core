package dealer

import (
	"context"
	"fmt"
	"time"

	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/dealerd/store"
	mbroker "github.com/textileio/broker-core/msgbroker"
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
		ctx, cancel := context.WithTimeout(d.daemonCtx, time.Second*15)
		defer cancel()
		aud, ok, err := d.store.GetNextPending(ctx, store.StatusReportFinalized)
		if err != nil {
			return fmt.Errorf("get successful deals: %s", err)
		}
		if !ok {
			break
		}
		if err := d.reportFinalizedAuctionDeal(ctx, aud); err != nil {
			log.Errorf("reporting finalized auction deal: %s", err)
			aud.ReadyAt = time.Now().Add(d.config.dealReportingRetryDelay)
			if err := d.store.SaveAndMoveAuctionDeal(ctx, aud, store.StatusReportFinalized); err != nil {
				return fmt.Errorf("saving reached deadline: %s", err)
			}
			return nil
		}
		cancel()
	}

	return nil
}

func (d *Dealer) reportFinalizedAuctionDeal(ctx context.Context, aud store.AuctionDeal) error {
	ad, err := d.store.GetAuctionData(ctx, aud.AuctionDataID)
	if err != nil {
		return fmt.Errorf("get auction data: %s", err)
	}
	fad := broker.FinalizedDeal{
		BatchID:           ad.BatchID,
		ErrorCause:        aud.ErrorCause,
		DealID:            aud.DealID,
		DealExpiration:    aud.DealExpiration,
		StorageProviderID: aud.StorageProviderID,
		AuctionID:         aud.AuctionID,
		BidID:             aud.BidID,
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()

	log.Debugf("reporting finalized deal (errorcause=%s)", aud.ErrorCause)
	if err := mbroker.PublishMsgFinalizedDeal(ctx, d.mb, fad); err != nil {
		return fmt.Errorf("publishing finalized-deal msg to msgbroker: %s", err)
	}

	// We mark the deal as Finalized so it no longer gets picked up by daemonDealReporterTick().
	if err := d.store.SaveAndMoveAuctionDeal(ctx, aud, store.StatusFinalized); err != nil {
		return fmt.Errorf("finalizing deal: %s", err)
	}

	return nil
}
