package metrics

import (
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
)

const Prefix = "auctioneerd"

var Meter = metric.Must(global.Meter(Prefix))
