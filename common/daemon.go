package common

import (
	"context"
	"fmt"
	"net/http"
	"time"

	logger "github.com/textileio/go-log/v2"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric/global"
	"go.opentelemetry.io/otel/sdk/metric/aggregator/histogram"
	controller "go.opentelemetry.io/otel/sdk/metric/controller/basic"
	"go.opentelemetry.io/otel/sdk/metric/export/aggregation"
	processor "go.opentelemetry.io/otel/sdk/metric/processor/basic"
	selector "go.opentelemetry.io/otel/sdk/metric/selector/simple"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// SetupInstrumentation starts a metrics endpoint.
func SetupInstrumentation(prometheusAddr string) error {
	config := prometheus.Config{
		// 0.25 is for near deposit, others for duration in either millis, seconds, or minutes
		DefaultHistogramBoundaries: []float64{0.25, 1, 10, 100, 1000, 10000},
	}
	c := controller.New(
		processor.NewFactory(
			selector.NewWithHistogramDistribution(
				histogram.WithExplicitBoundaries(config.DefaultHistogramBoundaries),
			),
			aggregation.CumulativeTemporalitySelector(),
			processor.WithMemory(true),
		),
	)
	exporter, err := prometheus.New(config, c)
	if err != nil {
		return fmt.Errorf("failed to initialize prometheus exporter %v", err)
	}
	global.SetMeterProvider(exporter.MeterProvider())
	http.HandleFunc("/metrics", exporter.ServeHTTP)
	go func() {
		_ = http.ListenAndServe(prometheusAddr, nil)
	}()

	if err := runtime.Start(runtime.WithMinimumReadMemStatsInterval(time.Second)); err != nil {
		return fmt.Errorf("starting Go runtime metrics: %s", err)
	}

	return nil
}

// GrpcLoggerInterceptor logs any error produced by processing requests, and catches/recovers
// from panics.
func GrpcLoggerInterceptor(log *logger.ZapEventLogger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context, req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler) (res interface{}, err error) {
		// Recover from any panic caused by this request processing.
		defer func() {
			if r := recover(); r != nil {
				log.Errorf("panic: %s", r)
				err = status.Errorf(codes.Internal, "panic: %s", r)
			}
		}()

		res, err = handler(ctx, req)
		grpcErrCode := status.Code(err)
		if grpcErrCode != codes.OK {
			log.Error(err)
		}
		return
	}
}
