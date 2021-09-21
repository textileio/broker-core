package metrics

import (
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
)

// Meter provides the meter to be used to report metrics.
var Meter = metric.Must(global.Meter("authd"))
