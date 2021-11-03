package broker

import (
	"context"

	"github.com/textileio/broker-core/cmd/brokerd/metrics"
	"go.opentelemetry.io/otel/metric"
)

var prefix = "brokerd"

func (b *Broker) initMetrics() {
	b.metricStartedAuctions = metrics.Meter.NewInt64Counter(prefix + ".started_auctions_total")
	b.metricStartedBytes = metrics.Meter.NewInt64Counter(prefix + ".started_bytes_total")
	b.metricFinishedAuctions = metrics.Meter.NewInt64Counter(prefix + ".finished_auctions_total")
	b.metricFinishedBytes = metrics.Meter.NewInt64Counter(prefix + ".finished_bytes_total")
	b.metricRebatches = metrics.Meter.NewInt64Counter(prefix + ".rebatches_total")
	b.metricRebatchedBytes = metrics.Meter.NewInt64Counter(prefix + ".rebatched_bytes_total")
	b.metricBatchFinalityMinutes = metrics.Meter.NewInt64Histogram(prefix + ".batch_finality_minutes")
	b.metricReauctions = metrics.Meter.NewInt64Counter(prefix + ".reauctions_total")
	b.metricReauctionedBytes = metrics.Meter.NewInt64Counter(prefix + ".reauctioned_bytes_total")

	b.metricUnpinTotal = metrics.Meter.NewInt64Counter(prefix + ".unpin_total")
	b.metricRecursivePinCount = metrics.Meter.NewInt64GaugeObserver(prefix+".pins_total", b.lastTotalPinCountCb)
}

func (b *Broker) lastTotalPinCountCb(ctx context.Context, r metric.Int64ObserverResult) {
	r.Observe(b.statTotalRecursivePins)
}
