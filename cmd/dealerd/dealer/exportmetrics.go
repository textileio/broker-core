package dealer

import (
	"context"
	"time"

	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/textileio/broker-core/cmd/dealerd/metrics"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

func (d *Dealer) daemonExportMetrics() {
	var (
		metricStatusCounter metric.Int64GaugeObserver
		countMap            = map[storagemarket.StorageDealStatus]int64{}
	)
	attrStatus := attribute.Key("status")

	batchObs := metrics.Meter.NewBatchObserver(func(ctx context.Context, result metric.BatchObserverResult) {
		for status, count := range countMap {
			result.Observe(
				[]attribute.KeyValue{attrStatus.String(storagemarket.DealStates[status])},
				metricStatusCounter.Observation(count),
			)
		}
	})
	metricStatusCounter = batchObs.NewInt64GaugeObserver(metrics.Prefix + ".deal_status_count")

	for {
		start := time.Now()
		newCountMap, err := d.store.GetStatusCounts(context.Background())
		if err != nil {
			log.Errorf("metrics count statuses: %s", err)
			time.Sleep(d.config.exportStatusesCountFrequency)
			continue
		}
		for status := range countMap {
			countMap[status] = 0
		}
		for status, val := range newCountMap {
			countMap[status] = val
		}

		log.Debugf("get auction deal status counts took %dms", time.Since(start).Milliseconds())
		time.Sleep(d.config.exportStatusesCountFrequency)
	}
}
