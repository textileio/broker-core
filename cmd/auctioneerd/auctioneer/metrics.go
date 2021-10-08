package auctioneer

import (
	"context"
	"time"

	"github.com/textileio/broker-core/cmd/auctioneerd/metrics"
	"go.opentelemetry.io/otel/metric"
)

func (a *Auctioneer) initMetrics() {
	a.metricNewAuction = metrics.Meter.NewInt64Counter(metrics.Prefix + ".auctions_total")
	a.metricNewFinalizedAuction = metrics.Meter.NewInt64Counter(metrics.Prefix + ".finalized_auctions_total")
	a.metricNewBid = metrics.Meter.NewInt64Counter(metrics.Prefix + ".bids_total")
	a.metricWinningBid = metrics.Meter.NewInt64Counter(metrics.Prefix + ".winning_bids_total")
	a.metricLastCreatedAuction = metrics.Meter.NewInt64ValueObserver(
		metrics.Prefix+".last_created_auction_epoch",
		a.lastCreatedAuctionCb)

	a.metricPubsubPeers = metrics.Meter.NewInt64ValueObserver(metrics.Prefix+".libp2p_pubsub_peers", a.lastPubsubPeersCb)
}

func (a *Auctioneer) lastCreatedAuctionCb(_ context.Context, r metric.Int64ObserverResult) {
	v := a.statLastCreatedAuction.Load()
	if v != nil {
		r.Observe(v.(time.Time).Unix())
	} else {
		r.Observe(0)
	}
}

func (a *Auctioneer) lastPubsubPeersCb(_ context.Context, r metric.Int64ObserverResult) {
	r.Observe(int64(len(a.commChannel.ListPeers())))
}
