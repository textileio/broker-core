package dealer

import (
	"fmt"
	"time"

	"github.com/textileio/broker-core/cmd/dealerd/dealer/store"
	"github.com/textileio/broker-core/ratelim"
)

var (
	dealMakerFreq    = time.Second * 5
	dealMakerRateLim = 20
)

func (d *Dealer) daemonDealMaker() {
	defer d.daemonWg.Done()

	for {
		select {
		case <-d.daemonCtx.Done():
			log.Infof("deal maker daemon closed")
			return
		case <-time.After(dealMakerFreq):
			if err := d.daemonDealMakerTick(); err != nil {
				log.Errorf("deal maker tick: %s", err)
			}
		}
	}
}

func (d *Dealer) daemonDealMakerTick() error {
	rl, err := ratelim.New(dealMakerRateLim)
	if err != nil {
		return fmt.Errorf("create ratelim: %s", err)
	}
	defer rl.Wait()

	for {
		ps, err := d.store.GetAllAuctionDeals(store.Pending)
		if err != nil {
			return fmt.Errorf("get pending auction deals: %s", err)
		}
		if len(ps) == 0 {
			break
		}
		for _, aud := range ps {
			select {
			case <-d.daemonCtx.Done():
				break
			default:
			}

			if err != nil {
				log.Errorf("get next pending deal: %s", err)
				break
			}

			rl.Exec(func() error {
				if err := d.executePending(aud); err != nil {
					log.Errorf("executing Pending: %s", err)
				}
				// We're not interested in ratelim error inspection.
				return nil
			})
		}
	}

	return nil
}

func (d *Dealer) executePending(aud store.AuctionDeal) error {
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

	return nil
}
