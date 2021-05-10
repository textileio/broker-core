package dealermock

import (
	"context"
	"math/rand"
	"time"

	logger "github.com/ipfs/go-log/v2"
	"github.com/textileio/broker-core/broker"
	dealeri "github.com/textileio/broker-core/dealer"
)

var log = logger.Logger("dealermock")

// Dealer provides a mocked implementation of Dealer. It reports successful
// deals to the broker after 1 sec.
type Dealer struct {
	broker broker.Broker
}

// New returns a new Dealer.
func New(broker broker.Broker) *Dealer {
	return &Dealer{
		broker: broker,
	}
}

// ReadyToCreateDeals registers deals to execute.
func (d *Dealer) ReadyToCreateDeals(ctx context.Context, sdb dealeri.AuctionDeals) error {
	go d.reportToBroker(sdb)
	return nil
}

func (d *Dealer) reportToBroker(sdb dealeri.AuctionDeals) {
	time.Sleep(time.Second)

	res := []broker.FinalizedAuctionDeal{
		{
			StorageDealID:  sdb.StorageDealID,
			DealID:         rand.Int63(),
			DealExpiration: uint64(rand.Int63()),
			Miner:          "f0001",
		},
	}
	if err := d.broker.StorageDealFinalizedDeals(context.Background(), res); err != nil {
		log.Errorf("reporting finalized deals to broker: %s", err)
	}
}
