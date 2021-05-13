package auctioneer

import (
	"fmt"

	"github.com/textileio/broker-core/auctioneer"
	"github.com/textileio/broker-core/cmd/auctioneerd/client"
	"google.golang.org/grpc"
)

// New returns a new auctioneer.
func New(addr string) (auctioneer.Auctioneer, error) {
	if addr == "" {
		return nil, fmt.Errorf("gRPC address is empty")
	}
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("creating client: %v", err)
	}
	return client.New(conn), nil
}
