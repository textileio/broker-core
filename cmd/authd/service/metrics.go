package service

import (
	"github.com/textileio/broker-core/cmd/authd/metrics"
)

var prefix = "authd"

func (s *Service) initMetrics() {
	s.metricGrpcRequests = metrics.Meter.NewInt64Counter(prefix + ".grpc_requests_total")
	s.metricGrpcRequestDuration = metrics.Meter.NewFloat64ValueRecorder(prefix + ".grpc_request_duration")
}
