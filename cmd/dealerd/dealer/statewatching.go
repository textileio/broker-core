package dealer

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/textileio/broker-core/cmd/dealerd/store"
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

	ctx, cancel := context.WithTimeout(d.daemonCtx, time.Second*15)
	defer cancel()
	chainHeight, err := d.filclient.GetChainHeight(ctx)
	if err != nil {
		return fmt.Errorf("get chain height: %s", err)
	}

	for {
		if d.daemonCtx.Err() != nil {
			break
		}

		ctx, cancel := context.WithTimeout(d.daemonCtx, time.Second*15)
		defer cancel()
		aud, ok, err := d.store.GetNextPending(ctx, store.StatusConfirmation)
		if err != nil {
			cancel()
			return fmt.Errorf("get waiting-confirmation deals: %s", err)
		}
		if !ok {
			cancel()
			break
		}

		rl.Exec(func() error {
			ctx, cancel := context.WithTimeout(d.daemonCtx, time.Second*30)
			defer cancel()
			if err := d.executeWaitingConfirmation(ctx, aud, chainHeight); err != nil {
				log.Errorf("executing waiting-confirmation: %s", err)
			}
			// We're not interested in ratelim error inspection.
			return nil
		})
	}
	rl.Wait()

	return nil
}

func (d *Dealer) executeWaitingConfirmation(
	ctx context.Context,
	aud store.AuctionDeal,
	currentChainHeight uint64) error {
	rw, err := d.store.GetRemoteWallet(ctx, aud.AuctionDataID)
	if err != nil {
		return fmt.Errorf("get remote wallet info: %s", err)
	}
	dealID, status := d.tryResolvingDealID(aud, rw)
	if status != storagemarket.StorageDealUnknown {
		mds, ok, err := d.store.GetStatusForStatusID(ctx, status)
		if !ok {
			return fmt.Errorf("no status for status id: %d", status)
		}
		if err != nil {
			return fmt.Errorf("getting status for status id: %s", err)
		}
		aud.MarketDealStatus = mds
	}
	stillHaveTime := aud.StartEpoch >= currentChainHeight

	if aud.DealID == 0 {
		log.Debugf("%s deal without deal-id, trying resolving with storage-provider %s",
			aud.ProposalCid, aud.StorageProviderID)
		if dealID == 0 {
			if stillHaveTime {
				// No problem, we'll try later on a new iteration.
				log.Debugf("still can't resolve the %s deal-id with %s, but have time...", aud.ProposalCid, aud.StorageProviderID)
				aud.ReadyAt = time.Now().Add(d.config.dealWatchingResolveDealIDRetryDelay)
				if err := d.store.SaveAndMoveAuctionDeal(ctx, aud, store.StatusConfirmation); err != nil {
					return fmt.Errorf("saving retry resolve deal id: %s", err)
				}
				return nil
			}

			// The storage-provider lost the race, it's game-over.
			log.Warnf("still can't resolve %s deal-id with %s and time is over; failing", aud.ProposalCid, aud.StorageProviderID)
			aud.ErrorCause = failureUnfulfilledStartEpoch
			aud.ReadyAt = time.Unix(0, 0)
			if err := d.store.SaveAndMoveAuctionDeal(ctx, aud, store.StatusReportFinalized); err != nil {
				return fmt.Errorf("saving reached deadline: %s", err)
			}
			return nil
		}

		log.Infof("%s deal-id %d with storage-provider %s resolved!", aud.ProposalCid, dealID, aud.StorageProviderID)
		// We know the deal-id now. Persist it, and keep moving.
		aud.DealID = dealID
		if err := d.store.SaveAndMoveAuctionDeal(ctx, aud, store.AuctionDealStatus(aud.Status)); err != nil {
			return fmt.Errorf("updating deal-id: %s", err)
		}
	}

	log.Debugf("%s checking on-chain confirmation for deal-id %d with storage provider %s",
		aud.ProposalCid, aud.DealID, aud.StorageProviderID)
	// We can ask the chain now for final confirmation.
	// Now we can stop asking/trusting the storage-provider for confirmation, and start asking
	// the chain.
	isActiveOnchain, expiration, slashed, err := d.filclient.CheckChainDeal(ctx, aud.DealID)
	if err != nil {
		log.Errorf("checking if deal %d is active on-chain: %s", aud.DealID, err)

		aud.ReadyAt = time.Now().Add(d.config.dealWatchingCheckChainRetryDelay)
		if err := d.store.SaveAndMoveAuctionDeal(ctx, aud, store.StatusConfirmation); err != nil {
			return fmt.Errorf("saving auction deal: %s", err)
		}
		return nil
	}

	if slashed {
		aud.ReadyAt = time.Unix(0, 0)
		aud.ErrorCause = fmt.Sprintf("the deal %d was slashed", aud.DealID)
		if err := d.store.SaveAndMoveAuctionDeal(ctx, aud, store.StatusReportFinalized); err != nil {
			return fmt.Errorf("saving auction deal: %s", err)
		}
	}

	if !isActiveOnchain {
		// Still not active on-chain.

		// If the storage-provider still has time, let's check later again.
		if aud.StartEpoch > currentChainHeight {
			log.Debugf("%s/%d/%s not active, we have time...", aud.ProposalCid, aud.DealID, aud.StorageProviderID)
			aud.ReadyAt = time.Now().Add(d.config.dealWatchingCheckChainRetryDelay)
			if err := d.store.SaveAndMoveAuctionDeal(ctx, aud, store.StatusConfirmation); err != nil {
				return fmt.Errorf("saving auction deal: %s", err)
			}
			return nil
		}

		// The storage-provider lost the race, it's game-over.
		log.Warnf("%s/%d/%s not active, reached deadline, gameover", aud.ProposalCid, aud.DealID, aud.StorageProviderID)
		aud.ErrorCause = failureUnfulfilledStartEpoch
		aud.ReadyAt = time.Unix(0, 0)
		if err := d.store.SaveAndMoveAuctionDeal(ctx, aud, store.StatusReportFinalized); err != nil {
			return fmt.Errorf("changing status to WaitingConfirmation: %s", err)
		}

		return nil
	}

	log.Infof("%s deal-id %d with storage-provider %s confirmed on-chain!",
		aud.ProposalCid, aud.DealID, aud.StorageProviderID)
	aud.DealExpiration = expiration
	aud.ReadyAt = time.Unix(0, 0)
	if err := d.store.SaveAndMoveAuctionDeal(ctx, aud, store.StatusReportFinalized); err != nil {
		return fmt.Errorf("saving auction deal: %s", err)
	}

	return nil
}

// tryResolvingDealID tries to resolve the deal-id from an AuctionDeal.
// It asks the storage-provider for the message Cid that published the deal. If a DealID is returned,
// we can be sure is the correct one for AuctionDeal, since this method checks that the storage-provider
// isn't playing tricks reporting a DealID from other data.
func (d *Dealer) tryResolvingDealID(
	aud store.AuctionDeal,
	rw *store.RemoteWallet) (int64, storagemarket.StorageDealStatus) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*20)
	defer cancel()

	publishCid, status, err :=
		d.filclient.CheckDealStatusWithStorageProvider(ctx, aud.StorageProviderID, aud.ProposalCid, rw)
	if err != nil {
		log.Infof("checking deal status with storage-provider: %s", err)
		return 0, 0
	}
	log.Debugf("%s check-deal-status: %s", aud.ID, storagemarket.DealStates[status])

	if publishCid != nil {
		log.Debugf("%s storage-provider published the deal in message %s, trying to resolve on-chain...",
			aud.ID, publishCid)
		ctx, cancel = context.WithTimeout(context.Background(), time.Second*20)
		defer cancel()

		ad, err := d.store.GetAuctionData(ctx, aud.AuctionDataID)
		if err != nil {
			log.Infof("get auction data: %s", err)
			return 0, 0
		}
		dealID, err := d.filclient.ResolveDealIDFromMessage(
			ctx, *publishCid, aud.StorageProviderID, ad.PieceCid, aud.StartEpoch)
		if err != nil {
			log.Errorf("trying to resolve deal-id from message %s: %s", publishCid, err)
			return 0, status
		}
		// Could we resolve by looking to the chain?, if yes, save it.
		// If no, no problem... we'll try again later when it might get confirmed.
		if dealID > 0 {
			return dealID, status
		}
	}

	return 0, status
}
