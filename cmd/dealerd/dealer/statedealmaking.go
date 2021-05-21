package dealer

import (
	"context"
	"fmt"
	"time"

	"github.com/textileio/broker-core/cmd/dealerd/dealer/store"
	"github.com/textileio/broker-core/ratelim"
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

Loop1:
	for {
		select {
		case <-d.daemonCtx.Done():
			break Loop1
		default:
		}

		ps, err := d.store.GetAllAuctionDeals(store.Pending)
		if err != nil {
			return fmt.Errorf("get pending auction deals: %s", err)
		}
		if len(ps) == 0 {
			break
		}
	Loop2:
		for _, aud := range ps {
			select {
			case <-d.daemonCtx.Done():
				break Loop2
			default:
			}
			rl.Exec(func() error {
				if err := d.executePending(d.daemonCtx, aud); err != nil {
					// TODO: most probably we want to include logic
					// to retry a bounded number of times (or similar)
					// when went stop trying to make the deal with the miner.
					log.Errorf("executing Pending: %s", err)
				}
				// We're not interested in ratelim error inspection.
				return nil
			})
		}
		rl.Wait()
	}

	return nil
}

func (d *Dealer) executePending(ctx context.Context, aud store.AuctionDeal) error {
	ad, err := d.store.GetAuctionData(aud.AuctionDataID)
	if err != nil {
		return fmt.Errorf("get auction data %s: %s", aud.AuctionDataID, err)
	}
	proposalCid, err := d.filclient.ExecuteAuctionDeal(d.daemonCtx, ad, aud)
	if err != nil {
		return fmt.Errorf("executing auction deal: %s", err)
	}

	aud.Status = store.WaitingConfirmation
	aud.ProposalCid = proposalCid
	if err := d.store.SaveAuctionDeal(aud); err != nil {
		return fmt.Errorf("changing status to WaitingConfirmation: %s", err)
	}

	if err := d.broker.StorageDealProposalAccepted(ctx, aud.Miner, proposalCid); err != nil {
		return fmt.Errorf("signaling broker of accepted proposal %s: %s", proposalCid, err)
	}

	return nil
}
