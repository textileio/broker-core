package dealer

import (
	"fmt"

	"github.com/textileio/broker-core/cmd/dealerd/client"
	"github.com/textileio/broker-core/dealer"
	"google.golang.org/grpc"
)

// New returns a new dealer.
func New(addr string) (dealer.Dealer, error) {
	if addr == "" {
		return nil, fmt.Errorf("gRPC address is empty")
	}
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("creating client: %v", err)
	}
	return client.NewClient(conn), nil
}
