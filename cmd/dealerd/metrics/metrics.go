package metrics

import (
	"context"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
)

var (
	AttrOK    = attribute.Key("status").String("ok")
	AttrError = attribute.Key("status").String("error")
	Meter     = global.Meter("dealerd")
)

func MetricIncrCounter(ctx context.Context, err error, m metric.Int64Counter) {
	attr := AttrOK
	if err != nil {
		attr = AttrError
	}
	m.Add(ctx, 1, attr)
}
