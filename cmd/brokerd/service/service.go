package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log/v2"
	"github.com/textileio/broker-core/broker"
	auctioneeri "github.com/textileio/broker-core/cmd/brokerd/auctioneer"
	brokeri "github.com/textileio/broker-core/cmd/brokerd/broker"
	packeri "github.com/textileio/broker-core/cmd/brokerd/packer"
	pieceri "github.com/textileio/broker-core/cmd/brokerd/piecer"
	pb "github.com/textileio/broker-core/gen/broker/v1"
	mongods "github.com/textileio/go-ds-mongo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	log = logging.Logger("broker/service")
)

// Config provides configuration to the broker service.
type Config struct {
	GrpcListenAddress string

	AuctioneerAddr string

	MongoDBName string
	MongoURI    string
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

	ds, err := createDatastore(config)
	if err != nil {
		return nil, fmt.Errorf("creating datastore: %s", err)
	}

	packer, err := packeri.New()
	if err != nil {
		return nil, fmt.Errorf("creating packer implementation: %s", err)
	}

	piecer, err := pieceri.New()
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

	// TODO: this line will go away soon, now it's just needed for emmbeding a packer
	// in broked binary.
	packer.SetBroker(broker)

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
func (s *Service) CreateBrokerRequest(ctx context.Context, r *pb.CreateBrokerRequestRequest) (*pb.CreateBrokerRequestResponse, error) {
	if r == nil {
		return nil, status.Error(codes.Internal, "empty request")
	}

	c, err := cid.Decode(r.Cid)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid cid: %s", err))
	}

	meta := broker.Metadata{}
	if r.Meta != nil {
		meta.Region = r.Meta.Region
	}
	if err := meta.Validate(); err != nil {
		return nil, status.Error(codes.InvalidArgument, fmt.Sprintf("invalid metadata: %s", err))
	}

	br, err := s.broker.Create(ctx, c, meta)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("creating storage request: %s", err))
	}

	pbr, err := castBrokerRequestToProto(br)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("converting result to proto: %s", err))
	}
	res := &pb.CreateBrokerRequestResponse{
		Request: pbr,
	}

	return res, nil
}

// GetBrokerRequest gets an existing broker request.
func (s *Service) GetBrokerRequest(ctx context.Context, r *pb.GetBrokerRequestRequest) (*pb.GetBrokerRequestResponse, error) {
	if r == nil {
		return nil, status.Error(codes.Internal, "empty request")
	}

	br, err := s.broker.Get(ctx, broker.BrokerRequestID(r.Id))
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("get broker request: %s", err))
	}

	pbr, err := castBrokerRequestToProto(br)
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("converting result to proto: %s", err))
	}
	res := &pb.GetBrokerRequestResponse{
		BrokerRequest: pbr,
	}

	return res, nil
}

// Close gracefully closes the service.
func (s *Service) Close() error {
	var errors []string
	defer log.Info("service was shutdown with %d errors", len(errors))

	// TODO: close gRPC server when exists
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

func castBrokerRequestToProto(br broker.BrokerRequest) (*pb.BrokerRequest, error) {
	var pbStatus pb.BrokerRequestStatus
	switch br.Status {
	case broker.RequestUnknown:
		pbStatus = pb.BrokerRequestStatus_UNSPECIFIED
	case broker.RequestBatching:
		pbStatus = pb.BrokerRequestStatus_BATCHING
	case broker.RequestPreparing:
		pbStatus = pb.BrokerRequestStatus_PREPARING
	case broker.RequestAuctioning:
		pbStatus = pb.BrokerRequestStatus_AUCTIONING
	case broker.RequestDealMaking:
		pbStatus = pb.BrokerRequestStatus_DEALMAKING
	case broker.BrokerRequestSuccess:
		pbStatus = pb.BrokerRequestStatus_SUCCESS
	default:
		return nil, fmt.Errorf("unknown status: %d", br.Status)

	}

	return &pb.BrokerRequest{
		Id:      string(br.ID),
		DataCid: br.DataCid.String(),
		Status:  pbStatus,
		Meta: &pb.BrokerRequestMetadata{
			Region: br.Metadata.Region,
		},
		StorageDealId: string(br.StorageDealID),
		CreatedAt:     timestamppb.New(br.CreatedAt),
		UpdatedAt:     timestamppb.New(br.UpdatedAt),
	}, nil
}

func createDatastore(conf Config) (datastore.TxnDatastore, error) {
	log.Info("Opening Mongo database...")

	mongoCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	if conf.MongoURI == "" {
		return nil, fmt.Errorf("mongo uri is empty")
	}
	if conf.MongoDBName == "" {
		return nil, fmt.Errorf("mongo database name is empty")
	}
	ds, err := mongods.New(mongoCtx, conf.MongoURI, conf.MongoDBName)
	if err != nil {
		return nil, fmt.Errorf("opening mongo datastore: %s", err)
	}

	return ds, nil
}
