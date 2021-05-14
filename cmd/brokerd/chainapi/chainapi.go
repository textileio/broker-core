package chainapi

import (
	"fmt"

	"github.com/textileio/broker-core/chainapi"
	"github.com/textileio/broker-core/cmd/neard/client"
	"google.golang.org/grpc"
)

// New returns a new chainapi.
func New(addr string) (chainapi.ChainAPI, error) {
	if addr == "" {
		return nil, fmt.Errorf("gRPC address is empty")
	}
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("creating client: %v", err)
	}
	return client.New(conn), nil
}
