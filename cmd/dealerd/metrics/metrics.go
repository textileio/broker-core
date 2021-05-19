package metrics

import (
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
)

const Prefix = "dealerd"

// Meter is the global meter used for metrics.
var Meter = metric.Must(global.Meter(Prefix))
