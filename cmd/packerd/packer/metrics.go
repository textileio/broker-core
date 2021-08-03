package packer

import (
	"context"

	"github.com/textileio/broker-core/cmd/packerd/metrics"
	"go.opentelemetry.io/otel/metric"
)

func (fc *Packer) initMetrics() {
	fc.metricNewBatch = metrics.Meter.NewInt64Counter(metrics.Prefix + ".batches_total")
	fc.metricBatchSizeTotal = metrics.Meter.NewInt64Counter(metrics.Prefix + ".batch_sizes_total")
	fc.metricLastBatchCreated = metrics.Meter.NewInt64ValueObserver(
		metrics.Prefix+".last_batch_created_epoch",
		fc.lastCreatedCb,
	)
	fc.metricLastBatchCount = metrics.Meter.NewInt64ValueObserver(
		metrics.Prefix+".last_batch_count",
		fc.lastCountCb,
	)
	fc.metricLastBatchSize = metrics.Meter.NewInt64ValueObserver(
		metrics.Prefix+".last_batch_size",
		fc.lastSizeCb,
	)
	fc.metricLastBatchDuration = metrics.Meter.NewInt64ValueObserver(
		metrics.Prefix+".last_batch_duration",
		fc.lastDurationCb,
	)
}

func (fc *Packer) lastCreatedCb(ctx context.Context, r metric.Int64ObserverResult) {
	r.Observe(fc.statLastBatch.Unix())
}

func (fc *Packer) lastCountCb(ctx context.Context, r metric.Int64ObserverResult) {
	r.Observe(fc.statLastBatchCount)
}

func (fc *Packer) lastSizeCb(ctx context.Context, r metric.Int64ObserverResult) {
	r.Observe(fc.statLastBatchSize)
}

func (fc *Packer) lastDurationCb(ctx context.Context, r metric.Int64ObserverResult) {
	r.Observe(fc.statLastBatchDuration)
}
