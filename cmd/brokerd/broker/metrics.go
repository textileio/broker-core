package broker

import (
	"context"

	"github.com/textileio/broker-core/cmd/brokerd/metrics"
	"go.opentelemetry.io/otel/metric"
)

var prefix = "brokerd"

func (b *Broker) initMetrics() {
	b.metricUnpinTotal = metrics.Meter.NewInt64Counter(prefix + ".unpin_total")
	b.metricRecursivePinCount = metrics.Meter.NewInt64ValueObserver(prefix+".pins_total", b.lastTotalPinCountCb)
}

func (b *Broker) lastTotalPinCountCb(ctx context.Context, r metric.Int64ObserverResult) {
	r.Observe(b.statTotalRecursivePins)
}
