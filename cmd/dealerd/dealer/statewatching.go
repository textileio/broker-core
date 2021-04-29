package dealer

import (
	"context"
	"fmt"
	"time"

	"github.com/textileio/broker-core/cmd/common"
	"github.com/textileio/broker-core/cmd/dealerd/dealer/store"
	"github.com/textileio/broker-core/ratelim"
)

var (
	dealWatcherFreq    = time.Second * 15
	dealWatcherRateLim = 20
)

func (d *Dealer) daemonDealWatcher() {
	defer d.daemonWg.Done()

	for {
		select {
		case <-d.daemonCtx.Done():
			log.Infof("deal watching daemon closed")
			return
		case <-time.After(dealWatcherFreq):
			if err := d.daemonDealWatcherTick(); err != nil {
				log.Errorf("deal watcher tick: %s", err)
			}
		}
	}
}

func (d *Dealer) daemonDealWatcherTick() error {
	rl, err := ratelim.New(dealWatcherRateLim)
	if err != nil {
		return fmt.Errorf("create ratelim: %s", err)
	}
	defer rl.Wait()

	for {
		ps, err := d.store.GetAllAuctionDeals(store.WaitingConfirmation)
		if err != nil {
			return fmt.Errorf("get waiting-confirmation deals: %s", err)
		}
		if len(ps) == 0 {
			break
		}
		chainHeight, err := d.filclient.GetChainHeight()
		if err != nil {
			return fmt.Errorf("get chain height: %s", err)
		}
		for _, aud := range ps {
			select {
			case <-d.daemonCtx.Done():
				break
			default:
			}

			if err != nil {
				log.Errorf("get next waiting-confirmation deal: %s", err)
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
	}

	return nil
}

func (d *Dealer) executeWaitingConfirmation(aud store.AuctionDeal, currentChainHeight int64) error {
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
		if err := d.store.SaveAuctionDeal(aud); err != nil {
			return fmt.Errorf("updating auction deal in store: %s", err)
		}
	}

	// We can ask the chain now for final confirmation.
	// Now we can stop asking/trusting the miner for confirmation, and start asking
	// the chain.
	isActiveOnchain, err := d.filclient.CheckChainDeal(d.daemonCtx, aud.DealID)
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

	aud.Status = store.Success
	if err := d.store.SaveAuctionDeal(aud); err != nil {
		return fmt.Errorf("changing status to WaitingConfirmation: %s", err)
	}

	return nil
}

func (d *Dealer) tryResolvingDealID(aud store.AuctionDeal, currentChainEpoch int64) (int64, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()
	pds, err := d.filclient.CheckDealStatusWithMiner(ctx, aud.Miner, aud.ProposalCid)
	if err != nil {
		return 0, false, fmt.Errorf("checking deal status with miner: %s", err)
	}
	log.Debugf("check-deal-status: %s", common.MustJsonIndent(pds))

	// Do best effort to resolve DealID if possible.
	if pds.DealID > 0 {
		log.Debugf("deal-id %d already resolved by miner %s", pds.DealID, aud.Miner)
		return int64(pds.DealID), true, nil
	}

	if pds.PublishCid != nil {
		log.Debugf("miner published the deal in message %s, trying to resolve on-chain...", pds.PublishCid)
		ctx, cancel = context.WithTimeout(context.Background(), time.Second*20)
		defer cancel()
		dealID, err := d.filclient.ResolveDealIDFromMessage(ctx, aud, pds.PublishCid)
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
