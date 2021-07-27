package filclient

import (
	"github.com/textileio/broker-core/cmd/dealerd/metrics"
)

func (fc *FilClient) initMetrics() {
	fc.metricExecAuctionDeal = metrics.Meter.NewInt64Counter("dealer.filclient.execauctiondeal")
	fc.metricGetChainHeight = metrics.Meter.NewInt64Counter("dealer.filclient.getchainheight")
	fc.metricResolveDealIDFromMessage = metrics.Meter.NewInt64Counter("dealer.filclient.resolvedealidfrommessage")
	fc.metricCheckDealStatusWithStatusProvider = metrics.Meter.NewInt64Counter("dealer.filclient.checkdealstatuswithminer")
	fc.metricCheckChainDeal = metrics.Meter.NewInt64Counter("dealer.filclient.checkchaindeal")
}
