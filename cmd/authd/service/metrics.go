package service

import (
	"github.com/textileio/broker-core/cmd/authd/metrics"
)

var prefix = "authd"

func (s *Service) initMetrics() {
	s.metricGrpcRequests = metrics.Meter.NewInt64Counter(prefix + ".grpc_requests_total")
	s.metricGrpcRequestDurationMillis = metrics.Meter.NewInt64ValueRecorder(prefix + ".grpc_request_duration_millis")
}
