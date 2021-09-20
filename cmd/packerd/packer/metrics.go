package packer

import (
	"context"

	"github.com/textileio/broker-core/cmd/packerd/metrics"
	"go.opentelemetry.io/otel/metric"
)

func (p *Packer) initMetrics() {
	p.metricNewBatch = metrics.Meter.NewInt64Counter(metrics.Prefix + ".batches_total")
	p.metricBatchErrors = metrics.Meter.NewInt64Counter(metrics.Prefix + ".batch_errors_total")
	p.metricBatchSizeTotal = metrics.Meter.NewInt64Counter(metrics.Prefix + ".batch_sizes_total")
	p.metricLastBatchCreated = metrics.Meter.NewInt64ValueObserver(
		metrics.Prefix+".last_batch_created_epoch",
		p.lastCreatedCb,
	)
	p.metricLastBatchCount = metrics.Meter.NewInt64ValueObserver(
		metrics.Prefix+".last_batch_count",
		p.lastCountCb,
	)
	p.metricLastBatchSize = metrics.Meter.NewInt64ValueObserver(
		metrics.Prefix+".last_batch_size",
		p.lastSizeCb,
	)
	p.metricLastBatchDuration = metrics.Meter.NewInt64ValueObserver(
		metrics.Prefix+".last_batch_duration",
		p.lastDurationCb,
	)
}

func (p *Packer) lastCreatedCb(ctx context.Context, r metric.Int64ObserverResult) {
	r.Observe(p.statLastBatch.Unix())
}

func (p *Packer) lastCountCb(ctx context.Context, r metric.Int64ObserverResult) {
	r.Observe(p.statLastBatchCount)
}

func (p *Packer) lastSizeCb(ctx context.Context, r metric.Int64ObserverResult) {
	r.Observe(p.statLastBatchSize)
}

func (p *Packer) lastDurationCb(ctx context.Context, r metric.Int64ObserverResult) {
	r.Observe(p.statLastBatchDuration)
}
