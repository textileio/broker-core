package broker

import (
	"github.com/textileio/broker-core/cmd/brokerd/metrics"
)

var prefix = "brokerd"

func (b *Broker) initMetrics() {
	b.metricUnpinTotal = metrics.Meter.NewInt64Counter(prefix + ".unpin_total")
}
