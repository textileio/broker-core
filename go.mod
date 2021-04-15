module github.com/textileio/broker-core

go 1.16

require (
	github.com/google/uuid v1.2.0
	github.com/ipfs/go-cid v0.0.7
	github.com/ipfs/go-ipfs-files v0.0.8
	github.com/ipfs/go-ipfs-http-client v0.1.0
	github.com/ipfs/go-log/v2 v2.1.3
	github.com/ipfs/interface-go-ipfs-core v0.4.0
	github.com/multiformats/go-multiaddr v0.3.1
	github.com/phayes/freeport v0.0.0-20180830031419-95f893ade6f2
	github.com/spf13/cobra v1.1.3
	github.com/spf13/viper v1.7.1
	github.com/stretchr/testify v1.7.0
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.19.0
	go.opentelemetry.io/contrib/instrumentation/runtime v0.19.0
	go.opentelemetry.io/otel/exporters/metric/prometheus v0.19.0
	go.opentelemetry.io/otel/metric v0.19.0
	go.uber.org/zap v1.16.0
	golang.org/x/net v0.0.0-20210414194228-064579744ee0 // indirect
	golang.org/x/sys v0.0.0-20210415045647-66c3f260301c // indirect
	google.golang.org/genproto v0.0.0-20210415145412-64678f1ae2d5 // indirect
	google.golang.org/grpc v1.37.0
	google.golang.org/protobuf v1.26.0
)
