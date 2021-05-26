package piecer

import (
	"fmt"

	"github.com/textileio/broker-core/cmd/piecerd/client"
	"github.com/textileio/broker-core/piecer"
	"google.golang.org/grpc"
)

// New returns a new auctioneer.
func New(addr string) (piecer.Piecer, error) {
	if addr == "" {
		return nil, fmt.Errorf("gRPC address is empty")
	}
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("creating client: %v", err)
	}
	return client.NewClient(conn), nil
}
