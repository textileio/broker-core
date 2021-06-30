package packer

import (
	"context"
	"time"

	"github.com/textileio/broker-core/cmd/packerd/metrics"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func (p *Packer) daemonExportMetrics() {
	var (
		metricPendingCounter metric.Int64ValueObserver
		pendingCount         int64
		metricPendingBytes   metric.Int64ValueObserver
		pendingBytes         int64
		metricOpenBatchCount metric.Int64ValueObserver
		openBatches          int64
	)

	batchObs := metrics.Meter.NewBatchObserver(func(ctx context.Context, result metric.BatchObserverResult) {
		result.Observe(
			[]attribute.KeyValue{},
			metricPendingCounter.Observation(pendingCount),
			metricPendingBytes.Observation(pendingBytes),
			metricOpenBatchCount.Observation(openBatches),
		)
	})
	metricPendingCounter = batchObs.NewInt64ValueObserver(metrics.Prefix + ".pending_count")
	metricPendingBytes = batchObs.NewInt64ValueObserver(metrics.Prefix + ".pending_bytes")
	metricOpenBatchCount = batchObs.NewInt64ValueObserver(metrics.Prefix + ".open_batch_count")

	for {
		start := time.Now()
		newPendingCount, newPendingBytes, newOpenBatches, err := p.store.GetStats()
		if err != nil {
			log.Errorf("getting stats: %s", err)
			continue
		}
		log.Debugf("get stats took %dms", time.Since(start).Milliseconds())

		pendingCount = newPendingCount
		pendingBytes = newPendingBytes
		openBatches = newOpenBatches

		<-time.After(p.exportMetricsFreq)
	}
}
