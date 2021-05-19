package piecer

import (
	"context"

	"github.com/textileio/broker-core/cmd/brokerd/metrics"
	"go.opentelemetry.io/otel/metric"
)

var prefix = "piecerd"

func (p *Piecer) initMetrics() {
	p.metricNewPrepare = metrics.Meter.NewInt64Counter(prefix + ".prepared_total")
	p.metricLastPrepared = metrics.Meter.NewInt64ValueObserver(prefix+".last_prepared_epoch", p.lastPreparedCb)
}

func (p *Piecer) lastPreparedCb(ctx context.Context, r metric.Int64ObserverResult) {
	r.Observe(p.statLastPrepared.Unix())
}
