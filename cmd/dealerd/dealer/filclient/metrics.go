package filclient

import (
	"github.com/textileio/broker-core/cmd/dealerd/metrics"
)

func (fc *FilClient) initMetrics() {
	fc.metricExecAuctionDeal = metrics.Meter.NewInt64Counter("dealer.filclient.execauctiondeal")
	fc.metricGetChainHeight = metrics.Meter.NewInt64Counter("dealer.filclient.getchainheight")
	fc.metricResolveDealIDFromMessage = metrics.Meter.NewInt64Counter("dealer.filclient.resolvedealidfrommessage")
	fc.metricCheckDealStatusWithStorageProvider =
		metrics.Meter.NewInt64Counter("dealer.filclient.checkdealstatuswithstorageprovider")
	fc.metricCheckChainDeal = metrics.Meter.NewInt64Counter("dealer.filclient.checkchaindeal")
}
