package metrics

import (
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
)

const Prefix = "packerd"

var Meter = metric.Must(global.Meter(Prefix))
