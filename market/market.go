package market

import (
	"context"
	"sync"

	"github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log/v2"
	iface "github.com/ipfs/interface-go-ipfs-core"
	"github.com/libp2p/go-libp2p-core/protocol"
	host "github.com/libp2p/go-libp2p-host"
	"github.com/oklog/ulid/v2"
	"github.com/textileio/broker-core/market/message"
	"github.com/textileio/broker-core/pubsub"
)

var log = logging.Logger("threads/registry")

const ProtocolDeals protocol.ID = "/textile/deals/0.0.1"

type AuctionID ulid.ULID

type Bid struct {
}

type Auctioneer interface {
	// Self() peer.ID

	StartAuction(msg message.MarketMessage) error
	StopAuction(id string)
}

type impl struct {
	host  host.Host
	ps    iface.PubSubAPI
	topic *pubsub.Topic
	// cache *cache.TokenCache
	reqs map[AuctionID]chan Bid

	ctx    context.Context
	cancel context.CancelFunc

	semaphores *rpc.SemaphorePool
	lk         sync.Mutex
}

func NewAuctioneer(host host.Host, store datastore.Datastore) {

}

func (i *impl) StartAuction(msg message.MarketMessage) error {

}
