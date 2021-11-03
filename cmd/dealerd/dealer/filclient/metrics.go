package filclient

import (
	"github.com/textileio/broker-core/cmd/dealerd/metrics"
	"go.opentelemetry.io/otel/attribute"
)

var (
	attrWalletSignature = attribute.Key("wallet")
	attrWalletType      = attribute.Key("wallet_type")
	attrLocalWallet     = attrWalletType.String("local")
	attrRemoteWallet    = attrWalletType.String("remote")
)

func (fc *FilClient) initMetrics() {
	fc.metricExecAuctionDeal = metrics.Meter.NewInt64Counter("dealer.filclient.execauctiondeal")
	fc.metricGetChainHeight = metrics.Meter.NewInt64Counter("dealer.filclient.getchainheight")
	fc.metricResolveDealIDFromMessage = metrics.Meter.NewInt64Counter("dealer.filclient.resolvedealidfrommessage")
	fc.metricCheckDealStatusWithStorageProvider =
		metrics.Meter.NewInt64Counter("dealer.filclient.checkdealstatuswithstorageprovider")
	fc.metricCheckChainDeal = metrics.Meter.NewInt64Counter("dealer.filclient.checkchaindeal")

	fc.metricFilAPIRequests = metrics.Meter.NewInt64Counter("dealer.filclient.filapi.requests")
	fc.metricFilAPIDurationMillis = metrics.Meter.NewInt64Histogram("dealer.filclient.filapi.duration.millis")
}
