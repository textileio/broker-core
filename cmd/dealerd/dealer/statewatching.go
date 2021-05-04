package dealer

import (
	"context"
	"fmt"
	"time"

	"github.com/textileio/broker-core/cmd/dealerd/dealer/store"
	"github.com/textileio/broker-core/logging"
	"github.com/textileio/broker-core/ratelim"
)

func (d *Dealer) daemonDealMonitoring() {
	defer d.daemonWg.Done()

	for {
		select {
		case <-d.daemonCtx.Done():
			log.Infof("deal watching daemon closed")
			return
		case <-time.After(d.config.dealMonitoringFreq):
			if err := d.daemonDealMonitoringTick(); err != nil {
				log.Errorf("deal watcher tick: %s", err)
			}
		}
	}
}

func (d *Dealer) daemonDealMonitoringTick() error {
	rl, err := ratelim.New(d.config.dealMonitoringRateLim)
	if err != nil {
		return fmt.Errorf("create ratelim: %s", err)
	}

Loop1:
	for {
		select {
		case <-d.daemonCtx.Done():
			break Loop1
		default:
		}
		ps, err := d.store.GetAllAuctionDeals(store.WaitingConfirmation)
		if err != nil {
			return fmt.Errorf("get waiting-confirmation deals: %s", err)
		}
		if len(ps) == 0 {
			break
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*15)
		defer cancel()
		chainHeight, err := d.filclient.GetChainHeight(ctx)
		if err != nil {
			return fmt.Errorf("get chain height: %s", err)
		}
	Loop2:
		for _, aud := range ps {
			select {
			case <-d.daemonCtx.Done():
				break Loop2
			default:
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
	}

	return nil
}

func (d *Dealer) executeWaitingConfirmation(aud store.AuctionDeal, currentChainHeight uint64) error {
	if aud.DealID == 0 {
		dealID, stillHaveTime, err := d.tryResolvingDealID(aud, currentChainHeight)
		if err != nil {
			return fmt.Errorf("trying to resolve deal id: %s", err)
		}
		if dealID == 0 {
			if stillHaveTime {
				// No problem, we'll try later on a new iteration.
				return nil
			}

			// The miner lost the race, it's game-over.
			aud.Status = store.Error
			aud.ErrorCause = store.FailureUnfulfilledStartEpoch
			if err := d.store.SaveAuctionDeal(aud); err != nil {
				return fmt.Errorf("changing status to WaitingConfirmation: %s", err)
			}
			return nil
		}

		// We know the deal-id now. Persist it, and keep moving.
		aud.DealID = dealID
		if err := d.store.SaveAuctionDeal(aud); err != nil {
			return fmt.Errorf("updating auction deal in store: %s", err)
		}
	}

	// We can ask the chain now for final confirmation.
	// Now we can stop asking/trusting the miner for confirmation, and start asking
	// the chain.
	isActiveOnchain, expiration, err := d.filclient.CheckChainDeal(d.daemonCtx, aud.DealID)
	if err != nil {
		return fmt.Errorf("checking if deal is active on-chain: %s", err)
	}

	log.Debugf("deal %d active on-chain?: %v", aud.DealID, isActiveOnchain)
	if !isActiveOnchain {
		// Still not active on-chain.

		// If the miner still has time, let's check later again.
		if aud.StartEpoch > currentChainHeight {
			return nil
		}

		// The miner lost the race, it's game-over.
		aud.Status = store.Error
		aud.ErrorCause = store.FailureUnfulfilledStartEpoch
		if err := d.store.SaveAuctionDeal(aud); err != nil {
			return fmt.Errorf("changing status to WaitingConfirmation: %s", err)
		}

		return nil
	}

	aud.DealExpiration = expiration
	aud.Status = store.Success
	if err := d.store.SaveAuctionDeal(aud); err != nil {
		return fmt.Errorf("changing status to WaitingConfirmation: %s", err)
	}

	return nil
}

// tryResolvingDealID tries to resolve the deal-id from an AuctionDeal.
// It asks the miner for the message Cid that published the deal. If a DealID is returned,
// we can be sure is the correct one for AuctionDeal, since this method checks that the miner
// isn't playing tricks reporting a DealID from other data.
func (d *Dealer) tryResolvingDealID(aud store.AuctionDeal, currentChainEpoch uint64) (int64, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()
	pds, err := d.filclient.CheckDealStatusWithMiner(ctx, aud.Miner, aud.ProposalCid)
	if err != nil {
		return 0, false, fmt.Errorf("checking deal status with miner: %s", err)
	}
	log.Debugf("check-deal-status: %s", logging.MustJSONIndent(pds))

	if pds.PublishCid != nil {
		log.Debugf("miner published the deal in message %s, trying to resolve on-chain...", pds.PublishCid)
		ctx, cancel = context.WithTimeout(context.Background(), time.Second*20)
		defer cancel()
		dealID, err := d.filclient.ResolveDealIDFromMessage(ctx, aud.ProposalCid, *pds.PublishCid)
		if err != nil {
			return 0, false, fmt.Errorf("trying to resolve deal-id from message %s: %s", pds.PublishCid, err)
		}
		// Could we resolve by looking to the chain?, if yes, save it.
		// If no, no problem... we'll try again later when it might get confirmed.
		if dealID > 0 {
			return dealID, true, nil
		}
	}

	// Try our best but still can't know about the deal-id, return if there's still time to find out.
	return 0, aud.StartEpoch > currentChainEpoch, nil
}
