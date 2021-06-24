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
		metricStatusCounter metric.Int64ValueObserver
		countMap            map[storagemarket.StorageDealStatus]int64
		err                 error
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
	metricStatusCounter = batchObs.NewInt64ValueObserver(metrics.Prefix + ".deal_status_count")

	for {
		start := time.Now()
		countMap, err = d.store.GetAuctionDealStatusCounts()
		if err != nil {
			log.Errorf("metrics count statuses: %s", err)
			continue
		}
		log.Debugf("get auction deal status counts took %dms", time.Since(start).Milliseconds())
		<-time.After(d.config.exportStatusesCountFrequency)
	}
}
