package metrics

import (
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
)

var Meter = metric.Must(global.Meter("brokerd"))
