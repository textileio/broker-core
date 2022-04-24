package filclient

import (
	"context"
	"time"

	"github.com/textileio/broker-core/cmd/dealerd/metrics"
	gmetrics "github.com/textileio/broker-core/metrics"
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

func (fc *FilClient) collectAPIMetrics(ctx context.Context, methodName string, err error, duration time.Duration) {
	labels := []attribute.KeyValue{
		attribute.String("method", methodName),
	}
	if err != nil {
		labels = append(labels, gmetrics.AttrError)
	} else {
		labels = append(labels, gmetrics.AttrOK)
	}

	fc.metricFilAPIRequests.Add(ctx, 1, labels...)
	fc.metricFilAPIDurationMillis.Record(ctx, duration.Milliseconds(), labels...)
}
