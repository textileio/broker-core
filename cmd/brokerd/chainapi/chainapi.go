package chainapi

import (
	"fmt"

	"github.com/textileio/broker-core/cmd/neard/client"
	"github.com/textileio/broker-core/reporter"
	"google.golang.org/grpc"
)

// New returns a new chainapi.
func New(addr string) (reporter.Reporter, error) {
	if addr == "" {
		return nil, fmt.Errorf("gRPC address is empty")
	}
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("creating client: %v", err)
	}
	return client.New(conn), nil
}
