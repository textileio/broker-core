package filclient

import (
	"github.com/textileio/broker-core/cmd/dealerd/metrics"
	"go.opentelemetry.io/otel/metric"
)

func (fc *FilClient) initMetrics() error {
	fc.metricExecAuctionDeal = metric.Must(metrics.Meter).NewInt64Counter("dealer.filclient.execauctiondeal")
	fc.metricGetChainHeight = metric.Must(metrics.Meter).NewInt64Counter("dealer.filclient.getchainheight")
	fc.metricResolveDealIDFromMessage = metric.Must(metrics.Meter).NewInt64Counter("dealer.filclient.resolvedealidfrommessage")
	fc.metricCheckDealStatusWithMiner = metric.Must(metrics.Meter).NewInt64Counter("dealer.filclient.checkdealstatuswithmienr")
	fc.metricCheckChainDeal = metric.Must(metrics.Meter).NewInt64Counter("dealer.filclient.checkchaindeal")

	return nil
}
