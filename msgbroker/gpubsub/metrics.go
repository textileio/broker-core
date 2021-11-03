package gpubsub

import (
	"context"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

type metricsCollector interface {
	onPublish(context.Context, string, error)
	onHandle(context.Context, string, time.Duration, error)
}

type noopMetricsCollector struct{}

func (noopMetricsCollector) onPublish(context.Context, string, error) {}
func (noopMetricsCollector) onHandle(context.Context, string, time.Duration, error) {
}

type otelMetricsCollector struct {
	metricPublishedMessages           metric.Int64Counter
	metricPublishMessageErrors        metric.Int64Counter
	metricHandledMessages             metric.Int64Counter
	metricHandleMessageErrors         metric.Int64Counter
	metricHandleMessageDurationMillis metric.Int64Histogram
}

func (c *otelMetricsCollector) onPublish(ctx context.Context, topicName string, err error) {
	label := attribute.String("topic", topicName)
	c.metricPublishedMessages.Add(ctx, 1, label)
	if err != nil {
		c.metricPublishMessageErrors.Add(ctx, 1, label)
	}
}

func (c *otelMetricsCollector) onHandle(ctx context.Context, topicName string, timeTaken time.Duration, err error) {
	label := attribute.String("topic", topicName)
	c.metricHandledMessages.Add(ctx, 1, label)
	c.metricHandleMessageDurationMillis.Record(ctx, timeTaken.Milliseconds(), label)
	if err != nil {
		c.metricHandleMessageErrors.Add(ctx, 1, label)
	}
}

func (p *PubsubMsgBroker) initMetrics(meter metric.MeterMust) {
	p.metrics = &otelMetricsCollector{
		metricPublishedMessages:           meter.NewInt64Counter("gpubsub_published_messages_total"),
		metricPublishMessageErrors:        meter.NewInt64Counter("gpubsub_publish_message_errors_total"),
		metricHandledMessages:             meter.NewInt64Counter("gpubsub_handled_messages_total"),
		metricHandleMessageErrors:         meter.NewInt64Counter("gpubsub_handle_message_errors_total"),
		metricHandleMessageDurationMillis: meter.NewInt64Histogram("gpubsub_handle_message_duration_millis"),
	}
}
