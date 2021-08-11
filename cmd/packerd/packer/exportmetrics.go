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

		lock                 sync.Mutex
		lastOpenBatchesStats []store.OpenBatchStats
		lastDoneBatchesStats []store.DoneBatchStats
	)

	batchObs := metrics.Meter.NewBatchObserver(func(ctx context.Context, result metric.BatchObserverResult) {
		lock.Lock()
		defer lock.Unlock()
		if lastOpenBatchesStats == nil || lastDoneBatchesStats == nil {
			return
		}
		for _, stat := range lastOpenBatchesStats {
			result.Observe(
				[]attribute.KeyValue{attribute.Key("origin").String(stat.Origin)},
				mOpenBatchesCidCount.Observation(stat.CidCount),
				mOpenBatchesBytes.Observation(stat.Bytes),
				mOpenBatchesCount.Observation(stat.Count),
			)
		}
		for _, stat := range lastDoneBatchesStats {
			result.Observe(
				[]attribute.KeyValue{attribute.Key("origin").String(stat.Origin)},
				mDoneBatchesCount.Observation(stat.Count),
				mDoneBatchesBytes.Observation(stat.Bytes),
			)
		}
	})
	mOpenBatchesCidCount = batchObs.NewInt64ValueObserver(metrics.Prefix + ".open_batches_cid_count")
	mOpenBatchesBytes = batchObs.NewInt64ValueObserver(metrics.Prefix + ".open_batches_bytes")
	mOpenBatchesCount = batchObs.NewInt64ValueObserver(metrics.Prefix + ".open_batches_count")
	mDoneBatchesCount = batchObs.NewInt64ValueObserver(metrics.Prefix + ".done_batches_count")
	mDoneBatchesBytes = batchObs.NewInt64ValueObserver(metrics.Prefix + ".done_batches_bytes")

	for {
		ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
		start := time.Now()
		openStats, doneStats, err := p.store.GetStats(ctx)
		if err != nil {
			cancel()
			log.Errorf("get metrics stats: %s", err)
			<-time.After(p.exportMetricsFreq)
			continue
		}
		cancel()
		log.Debugf("metrics stats took %dms", time.Since(start).Milliseconds())

		lock.Lock()
		lastOpenBatchesStats = openStats
		lastDoneBatchesStats = doneStats
		lock.Unlock()

		<-time.After(p.exportMetricsFreq)
	}
}
