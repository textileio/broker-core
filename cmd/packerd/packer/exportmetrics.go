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
		metricDoneBatchCount metric.Int64ValueObserver
		doneBatchCount       int64
		metricDoneBatchBytes metric.Int64ValueObserver
		doneBatchBytes       int64
	)

	batchObs := metrics.Meter.NewBatchObserver(func(ctx context.Context, result metric.BatchObserverResult) {
		result.Observe(
			[]attribute.KeyValue{},
			metricPendingCounter.Observation(pendingCount),
			metricPendingBytes.Observation(pendingBytes),
			metricOpenBatchCount.Observation(openBatches),
			metricDoneBatchCount.Observation(doneBatchCount),
			metricDoneBatchBytes.Observation(doneBatchBytes),
		)
	})
	metricPendingCounter = batchObs.NewInt64ValueObserver(metrics.Prefix + ".pending_count")
	metricPendingBytes = batchObs.NewInt64ValueObserver(metrics.Prefix + ".pending_bytes")
	metricOpenBatchCount = batchObs.NewInt64ValueObserver(metrics.Prefix + ".open_batch_count")
	metricDoneBatchCount = batchObs.NewInt64ValueObserver(metrics.Prefix + ".done_batch_count")
	metricDoneBatchBytes = batchObs.NewInt64ValueObserver(metrics.Prefix + ".done_batch_bytes")

	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		start := time.Now()
		stats, err := p.store.GetStats(ctx)
		if err != nil {
			cancel()
			log.Errorf("get metrics stats: %s", err)
			continue
		}
		log.Debugf("metrics stats took %dms", time.Since(start).Milliseconds())

		pendingCount = stats.PendingStorageRequestsCount
		pendingBytes = stats.OpenBatchBytes
		openBatches = stats.OpenBatchCount
		doneBatchCount = stats.DoneBatchCount
		doneBatchBytes = stats.DoneBatchBytes

		<-time.After(p.exportMetricsFreq)
	}
}
