package packer

import (
	"context"
	"sync"
	"time"

	"github.com/textileio/broker-core/cmd/packerd/metrics"
	"github.com/textileio/broker-core/cmd/packerd/store"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func (p *Packer) daemonExportMetrics() {
	var (
		mOpenBatchesCidCount metric.Int64ValueObserver
		mOpenBatchesBytes    metric.Int64ValueObserver
		mOpenBatchesCount    metric.Int64ValueObserver
		mDoneBatchesCount    metric.Int64ValueObserver
		mDoneBatchesBytes    metric.Int64ValueObserver

		lock      sync.Mutex
		lastStats *store.Stats
	)

	batchObs := metrics.Meter.NewBatchObserver(func(ctx context.Context, result metric.BatchObserverResult) {
		lock.Lock()
		defer lock.Unlock()
		if lastStats == nil {
			return
		}
		result.Observe(
			[]attribute.KeyValue{},
			mOpenBatchesCidCount.Observation(lastStats.OpenBatchesCidCount),
			mOpenBatchesBytes.Observation(lastStats.OpenBatchesBytes),
			mOpenBatchesCount.Observation(lastStats.OpenBatchesCount),
			mDoneBatchesCount.Observation(lastStats.DoneBatchesCount),
			mDoneBatchesBytes.Observation(lastStats.DoneBatchesBytes),
		)
	})
	mOpenBatchesCidCount = batchObs.NewInt64ValueObserver(metrics.Prefix + ".open_batches_cid_count")
	mOpenBatchesBytes = batchObs.NewInt64ValueObserver(metrics.Prefix + ".open_batches_bytes")
	mOpenBatchesCount = batchObs.NewInt64ValueObserver(metrics.Prefix + ".open_batches_count")
	mDoneBatchesCount = batchObs.NewInt64ValueObserver(metrics.Prefix + ".done_batches_count")
	mDoneBatchesBytes = batchObs.NewInt64ValueObserver(metrics.Prefix + ".done_batches_bytes")

	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		start := time.Now()
		stats, err := p.store.GetStats(ctx)
		if err != nil {
			cancel()
			log.Errorf("get metrics stats: %s", err)
			<-time.After(p.exportMetricsFreq)
			continue
		}
		cancel()
		log.Debugf("metrics stats took %dms", time.Since(start).Milliseconds())

		lock.Lock()
		lastStats = &stats
		lock.Unlock()

		<-time.After(p.exportMetricsFreq)
	}
}
