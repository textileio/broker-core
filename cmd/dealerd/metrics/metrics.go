package metrics

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
)

var (
	// AttrOK is a metric tag to indicate a successful operation.
	AttrOK = attribute.Key("status").String("ok")
	// AttrError is a metric tag to indicate a failed operation.
	AttrError = attribute.Key("status").String("error")
	// Meter is the global meter used for metrics.
	Meter = metric.Must(global.Meter("dealerd"))
)

// MetricIncrCounter increments the specified Int64Counter by 1. Depending if err
// is nil or not, it will use AttrOK or AttrError respectively. This method is a helper
// for deferring in methods.
func MetricIncrCounter(ctx context.Context, err error, m metric.Int64Counter) {
	attr := AttrOK
	if err != nil {
		attr = AttrError
	}
	m.Add(ctx, 1, attr)
}
