package texbroker

import (
	"context"
	"fmt"

	"github.com/ipfs/go-cid"
	logger "github.com/ipfs/go-log/v2"
	"github.com/textileio/broker-core/broker"
	pb "github.com/textileio/broker-core/gen/broker/v1"
	v1 "github.com/textileio/broker-core/proto/broker/v1"
	"github.com/textileio/broker-core/util"
	"google.golang.org/grpc"
)

var (
	log = logger.Logger("texbroker")
)

// TexBroker provides an interface with the Broker subsystem.
type TexBroker struct {
	c    pb.APIServiceClient
	conn *grpc.ClientConn
}

var _ broker.BrokerRequestor = (*TexBroker)(nil)

// New returns a new TexBroker.
func New(brokerAPIAddr string) (*TexBroker, error) {
	conn, err := grpc.Dial(brokerAPIAddr, util.GetClientRPCOpts(brokerAPIAddr)...)
	if err != nil {
		return nil, err
	}
	b := &TexBroker{
		c:    pb.NewAPIServiceClient(conn),
		conn: conn,
	}

	return b, nil
}

// Create creates a new BrokerRequest.
func (tb *TexBroker) Create(ctx context.Context, c cid.Cid, meta broker.Metadata) (broker.BrokerRequest, error) {
	log.Debugf("creating broker request for cid %s", c)

	req := &pb.CreateBrokerRequestRequest{
		Cid: c.String(),
		Meta: &pb.BrokerRequestMetadata{
			Region: meta.Region,
		},
	}
	res, err := tb.c.CreateBrokerRequest(ctx, req)
	if err != nil {
		return broker.BrokerRequest{}, fmt.Errorf("creating broker request: %s", err)
	}

	br, err := v1.FromProtoBrokerRequest(res.Request)
	if err != nil {
		return broker.BrokerRequest{}, fmt.Errorf("decoding proto response: %s", err)
	}

	return br, nil
}

// Get gets a broker request from its ID.
func (tb *TexBroker) Get(ctx context.Context, id broker.BrokerRequestID) (broker.BrokerRequest, error) {
	return broker.BrokerRequest{
		ID:     id,
		Status: broker.RequestUnknown,
	}, nil
}

func (tb *TexBroker) Close() error {
	if err := tb.conn.Close(); err != nil {
		return fmt.Errorf("closing gRPC client: %s", err)
	}
	return nil
}
