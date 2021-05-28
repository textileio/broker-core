package dealer

import (
	"context"
	"fmt"
	"time"

	"github.com/textileio/broker-core/cmd/dealerd/dealer/store"
	"github.com/textileio/broker-core/ratelim"
)

var (
	failureDealMakingMaxRetries = "reached maximum amount of deal-making retries"
)

func (d *Dealer) daemonDealMaker() {
	defer d.daemonWg.Done()

	for {
		select {
		case <-d.daemonCtx.Done():
			log.Infof("deal maker daemon closed")
			return
		case <-time.After(d.config.dealMakingFreq):
			if err := d.daemonDealMakerTick(); err != nil {
				log.Errorf("deal maker tick: %s", err)
			}
		}
	}
}

func (d *Dealer) daemonDealMakerTick() error {
	rl, err := ratelim.New(d.config.dealMakingRateLim)
	if err != nil {
		return fmt.Errorf("create ratelim: %s", err)
	}

	for {
		if d.daemonCtx.Err() != nil {
			break
		}

		aud, ok, err := d.store.GetNext(store.PendingDealMaking)
		if err != nil {
			return fmt.Errorf("get pending auction deals: %s", err)
		}
		if !ok {
			break
		}
		rl.Exec(func() error {
			if err := d.executePendingDealMaking(d.daemonCtx, aud); err != nil {
				log.Errorf("executing pending deal making: %s", err)
			}
			// We're not interested in ratelim error inspection.
			return nil
		})
	}
	rl.Wait()

	return nil
}

func (d *Dealer) executePendingDealMaking(ctx context.Context, aud store.AuctionDeal) error {
	ad, err := d.store.GetAuctionData(aud.AuctionDataID)
	if err != nil {
		return fmt.Errorf("get auction data %s: %s", aud.AuctionDataID, err)
	}
	proposalCid, retry, err := d.filclient.ExecuteAuctionDeal(d.daemonCtx, ad, aud)
	if err != nil {
		return fmt.Errorf("executing auction deal: %s", err)
	}

	// If retry, then we move from ExecutingDealMaking back to PendingDealMaking
	// with some delay as to retry again. If we tried dealMakingMaxRetries,
	// we give up and error the auction deal.
	if retry {
		aud.Retries++
		if aud.Retries > d.config.dealMakingMaxRetries {
			aud.ErrorCause = failureDealMakingMaxRetries
			aud.ReadyAt = time.Unix(0, 0)
			if err := d.store.SaveAndMoveAuctionDeal(aud, store.PendingDealMaking); err != nil {
				return fmt.Errorf("saving auction deal: %s", err)
			}
			return nil
		}

		aud.ReadyAt = time.Now().Add(d.config.dealMakingRetryDelay)
		if err := d.store.SaveAndMoveAuctionDeal(aud, store.PendingDealMaking); err != nil {
			return fmt.Errorf("saving auction deal: %s", err)
		}
		return nil
	}

	aud.Retries = 0
	aud.ProposalCid = proposalCid
	aud.ReadyAt = time.Unix(0, 0)

	if err := d.store.SaveAndMoveAuctionDeal(aud, store.PendingConfirmation); err != nil {
		return fmt.Errorf("changing status to WaitingConfirmation: %s", err)
	}

	if err := d.broker.StorageDealProposalAccepted(ctx, ad.StorageDealID, aud.Miner, proposalCid); err != nil {
		return fmt.Errorf("signaling broker of accepted proposal %s: %s", proposalCid, err)
	}

	return nil
}
