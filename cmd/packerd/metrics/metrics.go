package metrics

import (
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
)

// Prefix specifies the prefix to be used in exported metrics.
const Prefix = "packerd"

// Meter provides the meter to be used to report metrics.
var Meter = metric.Must(global.Meter(Prefix))
