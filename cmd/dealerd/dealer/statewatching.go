package dealer

import (
	"context"
	"fmt"
	"time"

	"github.com/textileio/broker-core/cmd/dealerd/dealer/store"
	"github.com/textileio/broker-core/logging"
	"github.com/textileio/broker-core/ratelim"
)

var failureUnfulfilledStartEpoch = "the deal won't be active on-chain"

func (d *Dealer) daemonDealMonitorer() {
	defer d.daemonWg.Done()

	for {
		select {
		case <-d.daemonCtx.Done():
			log.Infof("deal watching daemon closed")
			return
		case <-time.After(d.config.dealWatchingFreq):
			if err := d.daemonDealMonitoringTick(); err != nil {
				log.Errorf("deal watcher tick: %s", err)
			}
		}
	}
}

func (d *Dealer) daemonDealMonitoringTick() error {
	rl, err := ratelim.New(d.config.dealWatchingRateLim)
	if err != nil {
		return fmt.Errorf("create ratelim: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
	defer cancel()
	chainHeight, err := d.filclient.GetChainHeight(ctx)
	if err != nil {
		return fmt.Errorf("get chain height: %s", err)
	}

	for {
		if d.daemonCtx.Err() != nil {
			break
		}

		aud, ok, err := d.store.GetNext(store.PendingConfirmation)
		if err != nil {
			return fmt.Errorf("get waiting-confirmation deals: %s", err)
		}
		if !ok {
			break
		}

		rl.Exec(func() error {
			if err := d.executeWaitingConfirmation(aud, chainHeight); err != nil {
				log.Errorf("executing waiting-confirmation: %s", err)
			}
			// We're not interested in ratelim error inspection.
			return nil
		})
	}
	rl.Wait()

	return nil
}

func (d *Dealer) executeWaitingConfirmation(aud store.AuctionDeal, currentChainHeight uint64) error {
	if aud.DealID == 0 {
		log.Debugf("%s deal without deal-id, trying resolving with miner %s", aud.ProposalCid, aud.Miner)
		dealID, stillHaveTime := d.tryResolvingDealID(aud, currentChainHeight)
		if dealID == 0 {
			if stillHaveTime {
				// No problem, we'll try later on a new iteration.
				log.Debugf("still can't resolve the %s deal-id with %s, but have time...", aud.ProposalCid, aud.Miner)
				aud.ReadyAt = time.Now().Add(d.config.dealWatchingResolveDealIDRetryDelay)
				if err := d.store.SaveAndMoveAuctionDeal(aud, store.PendingConfirmation); err != nil {
					return fmt.Errorf("saving retry resolve deal id: %s", err)
				}
				return nil
			}

			// The miner lost the race, it's game-over.
			log.Warnf("still can't resolve %s deal-id with %s and time is over; failing", aud.ProposalCid, aud.Miner)
			aud.ErrorCause = failureUnfulfilledStartEpoch
			aud.ReadyAt = time.Unix(0, 0)
			if err := d.store.SaveAndMoveAuctionDeal(aud, store.PendingReportFinalized); err != nil {
				return fmt.Errorf("saving reached deadline: %s", err)
			}
			return nil
		}

		log.Infof("%s deal-id %d with miner %s resolved!", aud.ProposalCid, dealID, aud.Miner)
		// We know the deal-id now. Persist it, and keep moving.
		aud.DealID = dealID
		if err := d.store.SaveAndMoveAuctionDeal(aud, aud.Status); err != nil {
			return fmt.Errorf("updating deal-id: %s", err)
		}
	}

	log.Debugf("%s checking on-chain confirmation for deal-id %d with miner %s", aud.ProposalCid, aud.DealID, aud.Miner)
	// We can ask the chain now for final confirmation.
	// Now we can stop asking/trusting the miner for confirmation, and start asking
	// the chain.
	isActiveOnchain, expiration, slashed, err := d.filclient.CheckChainDeal(d.daemonCtx, aud.DealID)
	if err != nil {
		log.Errorf("checking if deal %d is active on-chain: %s", aud.DealID, err)

		aud.ReadyAt = time.Now().Add(d.config.dealWatchingCheckChainRetryDelay)
		if err := d.store.SaveAndMoveAuctionDeal(aud, store.PendingConfirmation); err != nil {
			return fmt.Errorf("saving auction deal: %s", err)
		}
		return nil
	}

	if slashed {
		aud.ReadyAt = time.Unix(0, 0)
		aud.ErrorCause = fmt.Sprintf("the deal %d was slashed", aud.DealID)
		if err := d.store.SaveAndMoveAuctionDeal(aud, store.PendingReportFinalized); err != nil {
			return fmt.Errorf("saving auction deal: %s", err)
		}
	}

	if !isActiveOnchain {
		// Still not active on-chain.

		// If the miner still has time, let's check later again.
		if aud.StartEpoch > currentChainHeight {
			log.Debugf("%s deal-id %d with miner %s not active on chain, but we have time...", aud.ProposalCid, aud.DealID, aud.Miner)
			aud.ReadyAt = time.Now().Add(d.config.dealWatchingCheckChainRetryDelay)
			if err := d.store.SaveAndMoveAuctionDeal(aud, store.PendingConfirmation); err != nil {
				return fmt.Errorf("saving auction deal: %s", err)
			}
			return nil
		}

		// The miner lost the race, it's game-over.
		log.Warnf("%s deal-id %d with miner %s not active on chain and reached deadline; it's over", aud.ProposalCid, aud.DealID, aud.Miner)
		aud.ErrorCause = failureUnfulfilledStartEpoch
		aud.ReadyAt = time.Unix(0, 0)
		if err := d.store.SaveAndMoveAuctionDeal(aud, store.PendingReportFinalized); err != nil {
			return fmt.Errorf("changing status to WaitingConfirmation: %s", err)
		}

		return nil
	}

	log.Infof("%s deal-id %d with miner %s confirmed on-chain!", aud.ProposalCid, aud.DealID, aud.Miner)
	aud.DealExpiration = expiration
	aud.ReadyAt = time.Unix(0, 0)
	if err := d.store.SaveAndMoveAuctionDeal(aud, store.PendingReportFinalized); err != nil {
		return fmt.Errorf("saving auction deal: %s", err)
	}

	return nil
}

// tryResolvingDealID tries to resolve the deal-id from an AuctionDeal.
// It asks the miner for the message Cid that published the deal. If a DealID is returned,
// we can be sure is the correct one for AuctionDeal, since this method checks that the miner
// isn't playing tricks reporting a DealID from other data.
func (d *Dealer) tryResolvingDealID(aud store.AuctionDeal, currentChainEpoch uint64) (int64, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()
	pds, err := d.filclient.CheckDealStatusWithMiner(ctx, aud.Miner, aud.ProposalCid)
	if err != nil {
		log.Errorf("checking deal status with miner: %s", err)
		return 0, true
	}
	log.Debugf("%s check-deal-status: %s", aud.ID, logging.MustJSONIndent(pds))

	if pds.PublishCid != nil {
		log.Debugf("%s miner published the deal in message %s, trying to resolve on-chain...", aud.ID, pds.PublishCid)
		ctx, cancel = context.WithTimeout(context.Background(), time.Second*20)
		defer cancel()
		dealID, err := d.filclient.ResolveDealIDFromMessage(ctx, aud.ProposalCid, *pds.PublishCid)
		if err != nil {
			log.Errorf("trying to resolve deal-id from message %s: %s", pds.PublishCid, err)
			return 0, true
		}
		// Could we resolve by looking to the chain?, if yes, save it.
		// If no, no problem... we'll try again later when it might get confirmed.
		if dealID > 0 {
			return dealID, true
		}
	}

	// Try our best but still can't know about the deal-id, return if there's still time to find out.
	return 0, aud.StartEpoch > currentChainEpoch
}
