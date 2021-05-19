package packer

import (
	"context"

	"github.com/textileio/broker-core/cmd/packerd/metrics"
	"go.opentelemetry.io/otel/metric"
)

func (fc *Packer) initMetrics() {
	fc.metricNewBatch = metrics.Meter.NewInt64Counter(metrics.Prefix + ".batches_total")
	fc.metricLastCreated = metrics.Meter.NewInt64ValueObserver(metrics.Prefix+".last_created_epoch", fc.lastCreatedCb)
}

func (fc *Packer) lastCreatedCb(ctx context.Context, r metric.Int64ObserverResult) {
	r.Observe(fc.statLastBatch.Unix())
}
