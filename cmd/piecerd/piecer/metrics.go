package piecer

import (
	"context"

	"github.com/textileio/broker-core/cmd/brokerd/metrics"
	"go.opentelemetry.io/otel/metric"
)

var prefix = "piecerd"

func (p *Piecer) initMetrics() {
	p.metricNewPrepare = metrics.Meter.NewInt64Counter(prefix + ".prepared_total")
	p.metricLastPrepared = metrics.Meter.NewInt64GaugeObserver(prefix+".last_prepared_epoch", p.lastPreparedCb)
	p.metricLastSize = metrics.Meter.NewInt64GaugeObserver(prefix+".last_size", p.lastSizeCb)
	p.metricLastDurationSeconds = metrics.Meter.NewInt64GaugeObserver(
		prefix+".last_duration_seconds",
		p.lastDurationSecondsCb)
}

func (p *Piecer) lastPreparedCb(ctx context.Context, r metric.Int64ObserverResult) {
	r.Observe(p.statLastPrepared.Unix())
}

func (p *Piecer) lastSizeCb(ctx context.Context, r metric.Int64ObserverResult) {
	r.Observe(p.statLastSize)
}

func (p *Piecer) lastDurationSecondsCb(ctx context.Context, r metric.Int64ObserverResult) {
	r.Observe(p.statLastDurationSeconds)
}
