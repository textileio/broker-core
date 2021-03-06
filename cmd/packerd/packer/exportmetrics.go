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
		mOpenBatchesCidCount metric.Int64GaugeObserver
		mOpenBatchesBytes    metric.Int64GaugeObserver
		mOpenBatchesCount    metric.Int64GaugeObserver
		mDoneBatchesCount    metric.Int64GaugeObserver
		mDoneBatchesBytes    metric.Int64GaugeObserver

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
	mOpenBatchesCidCount = batchObs.NewInt64GaugeObserver(metrics.Prefix + ".open_batches_cid_count")
	mOpenBatchesBytes = batchObs.NewInt64GaugeObserver(metrics.Prefix + ".open_batches_bytes")
	mOpenBatchesCount = batchObs.NewInt64GaugeObserver(metrics.Prefix + ".open_batches_count")
	mDoneBatchesCount = batchObs.NewInt64GaugeObserver(metrics.Prefix + ".done_batches_count")
	mDoneBatchesBytes = batchObs.NewInt64GaugeObserver(metrics.Prefix + ".done_batches_bytes")

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

		// We override all done origins with non-open batches to produce zero
		// values for prometheus.
		for _, doneOrigin := range doneStats {
			var foundOpen bool
			for _, openOrigin := range openStats {
				if openOrigin.Origin == doneOrigin.Origin {
					foundOpen = true
					break
				}
			}
			if !foundOpen {
				lastOpenBatchesStats = append(lastOpenBatchesStats,
					store.OpenBatchStats{
						Origin:   doneOrigin.Origin,
						CidCount: 0,
						Bytes:    0,
						Count:    0,
					})
			}
		}
		lock.Unlock()

		<-time.After(p.exportMetricsFreq)
	}
}
