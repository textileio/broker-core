package dealermock

import (
	"context"
	"math/rand"
	"time"

	"github.com/textileio/broker-core/broker"
	dealeri "github.com/textileio/broker-core/dealer"
	mbroker "github.com/textileio/broker-core/msgbroker"
	logger "github.com/textileio/go-log/v2"
)

var log = logger.Logger("dealermock")

// Dealer provides a mocked implementation of Dealer. It reports successful
// deals to the broker after 1 sec.
type Dealer struct {
	mb mbroker.MsgBroker
}

// New returns a new Dealer.
func New(mb mbroker.MsgBroker) *Dealer {
	return &Dealer{
		mb: mb,
	}
}

// ReadyToCreateDeals registers deals to execute.
func (d *Dealer) ReadyToCreateDeals(ctx context.Context, sdb dealeri.AuctionDeals) error {
	log.Debugf("received ready to create deals %s", sdb.StorageDealID)
	go d.reportToBroker(sdb)
	return nil
}

func (d *Dealer) reportToBroker(sdb dealeri.AuctionDeals) {
	time.Sleep(time.Second)

	fd := broker.FinalizedDeal{
		StorageDealID:  sdb.StorageDealID,
		DealID:         rand.Int63(),
		DealExpiration: uint64(rand.Int63()),
		Miner:          "f0001",
	}
	if err := mbroker.PublishMsgFinalizedDeal(context.Background(), d.mb, fd); err != nil {
		log.Errorf("publishing finalized-deal msg to msgbroker: %s", err)
	}
}
