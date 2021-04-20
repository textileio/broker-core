package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/ipfs/go-cid"
	logging "github.com/ipfs/go-log/v2"
	"github.com/textileio/broker-core/broker"
	auctioneeri "github.com/textileio/broker-core/cmd/brokerd/auctioneer"
	brokeri "github.com/textileio/broker-core/cmd/brokerd/broker"
	"github.com/textileio/broker-core/cmd/brokerd/cast"
	packeri "github.com/textileio/broker-core/cmd/brokerd/packer"
	pieceri "github.com/textileio/broker-core/cmd/brokerd/piecer"
	"github.com/textileio/broker-core/cmd/common"
	pb "github.com/textileio/broker-core/gen/broker/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	log = logging.Logger("broker/service")
)

// Config provides configuration to the broker service.
type Config struct {
	GrpcListenAddress string

	AuctioneerAddr string
	PackerAddr     string

	MongoDBName string
	MongoURI    string

	IpfsMultiaddr string
}

// Service provides an implementation of the broker API.
type Service struct {
	pb.UnimplementedAPIServiceServer

	config Config
	server *grpc.Server

	broker *brokeri.Broker
	packer *packeri.Packer
}

var _ pb.APIServiceServer = (*Service)(nil)

// New returns a new Service.
func New(config Config) (*Service, error) {
	listener, err := net.Listen("tcp", config.GrpcListenAddress)
	if err != nil {
		return nil, fmt.Errorf("getting net listener: %v", err)
	}

	ds, err := common.CreateMongoTxnDatastore(config.MongoURI, config.MongoDBName)
	if err != nil {
		return nil, fmt.Errorf("creating datastore: %s", err)
	}

	packer, err := packeri.New(config.PackerAddr)
	if err != nil {
		return nil, fmt.Errorf("creating packer implementation: %s", err)
	}

	piecer, err := pieceri.New(config.IpfsMultiaddr)
	if err != nil {
		return nil, fmt.Errorf("creating piecer implementation: %s", err)
	}

	auctioneer, err := auctioneeri.New(config.AuctioneerAddr)
	if err != nil {
		return nil, fmt.Errorf("creating auctioneer implementation: %s", err)
	}

	broker, err := brokeri.New(ds, packer, piecer, auctioneer)
	if err != nil {
		return nil, fmt.Errorf("creating broker implementation: %s", err)
	}

	// TODO: this line will go away soon when piecer lives in its own daemon
	piecer.SetBroker(broker)

	s := &Service{
		config: config,
		server: grpc.NewServer(),
		broker: broker,
		packer: packer,
	}
	go func() {
		pb.RegisterAPIServiceServer(s.server, s)
		if err := s.server.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			log.Errorf("server error: %v", err)
		}
	}()

	log.Infof("service listening at %s", config.GrpcListenAddress)

	return s, nil
}

// CreateBrokerRequest creates a new BrokerRequest.
func (s *Service) CreateBrokerRequest(
	ctx context.Context,
	r *pb.CreateBrokerRequestRequest) (*pb.CreateBrokerRequestResponse, error) {
	if r == nil {
		return nil, status.Error(codes.Internal, "empty request")
	}

	c, err := cid.Decode(r.Cid)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid cid: %s", err)
	}

	meta := broker.Metadata{}
	if r.Meta != nil {
		meta.Region = r.Meta.Region
	}
	if err := meta.Validate(); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid metadata: %s", err)
	}

	br, err := s.broker.Create(ctx, c, meta)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "creating storage request: %s", err)
	}

	pbr, err := cast.BrokerRequestToProto(br)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "converting result to proto: %s", err)
	}
	res := &pb.CreateBrokerRequestResponse{
		Request: pbr,
	}

	return res, nil
}

// GetBrokerRequest gets an existing broker request.
func (s *Service) GetBrokerRequest(
	ctx context.Context,
	r *pb.GetBrokerRequestRequest) (*pb.GetBrokerRequestResponse, error) {
	if r == nil {
		return nil, status.Error(codes.Internal, "empty request")
	}

	br, err := s.broker.Get(ctx, broker.BrokerRequestID(r.Id))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get broker request: %s", err)
	}

	pbr, err := cast.BrokerRequestToProto(br)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "converting result to proto: %s", err)
	}
	res := &pb.GetBrokerRequestResponse{
		BrokerRequest: pbr,
	}

	return res, nil
}

// CreateStorageDeal deal creates a storage deal.
func (s *Service) CreateStorageDeal(
	ctx context.Context,
	r *pb.CreateStorageDealRequest) (*pb.CreateStorageDealResponse, error) {
	if r == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	batchCid, err := cid.Decode(r.BatchCid)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid cid: %s", r.BatchCid)
	}

	brids := make([]broker.BrokerRequestID, len(r.BrokerRequestIds))
	for i, id := range r.BrokerRequestIds {
		if id == "" {
			return nil, status.Error(codes.InvalidArgument, "broker request id can't be empty")
		}
		brids[i] = broker.BrokerRequestID(id)
	}

	sd, err := s.broker.CreateStorageDeal(ctx, batchCid, brids)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "creating storage deal: %s", err)
	}

	return &pb.CreateStorageDealResponse{Id: string(sd.ID)}, nil
}

// Close gracefully closes the service.
func (s *Service) Close() error {
	var errors []string
	defer log.Info("service was shutdown with %d errors", len(errors))

	stopped := make(chan struct{})
	go func() {
		s.server.GracefulStop()
		close(stopped)
	}()
	timer := time.NewTimer(10 * time.Second)
	select {
	case <-timer.C:
		s.server.Stop()
	case <-stopped:
		timer.Stop()
	}

	if err := s.packer.Close(); err != nil {
		errors = append(errors, err.Error())
	}

	if errors != nil {
		return fmt.Errorf(strings.Join(errors, "\n"))
	}

	return nil
}
