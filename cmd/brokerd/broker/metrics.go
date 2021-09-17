package broker

import (
	"context"

	"github.com/textileio/broker-core/cmd/brokerd/metrics"
	"go.opentelemetry.io/otel/metric"
)

var prefix = "brokerd"

func (b *Broker) initMetrics() {
	b.metricStartedBatches = metrics.Meter.NewInt64Counter(prefix + ".started_batches_total")
	b.metricStartedBytes = metrics.Meter.NewInt64Counter(prefix + ".started_bytes_total")
	b.metricFinishedBatches = metrics.Meter.NewInt64Counter(prefix + ".finished_batches_total")
	b.metricFinishedBytes = metrics.Meter.NewInt64Counter(prefix + ".finished_bytes_total")
	b.metricErroredBatches = metrics.Meter.NewInt64Counter(prefix + ".errored_batches_total")
	b.metricErroredBytes = metrics.Meter.NewInt64Counter(prefix + ".errored_bytes_total")
	b.metricRebatches = metrics.Meter.NewInt64Counter(prefix + ".rebatches_total")
	b.metricRebatchedBytes = metrics.Meter.NewInt64Counter(prefix + ".rebatched_bytes_total")
	b.metricBatchDuration = metrics.Meter.NewFloat64ValueRecorder(prefix + ".batch_duration")

	b.metricUnpinTotal = metrics.Meter.NewInt64Counter(prefix + ".unpin_total")
	b.metricRecursivePinCount = metrics.Meter.NewInt64ValueObserver(prefix+".pins_total", b.lastTotalPinCountCb)
}

func (b *Broker) lastTotalPinCountCb(ctx context.Context, r metric.Int64ObserverResult) {
	r.Observe(b.statTotalRecursivePins)
}
