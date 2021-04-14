package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/gogo/status"
	"github.com/ipfs/go-cid"
	"github.com/ipfs/go-datastore"
	logging "github.com/ipfs/go-log/v2"
	"github.com/textileio/broker-core/broker"
	brokeri "github.com/textileio/broker-core/cmd/brokerd/broker"
	packeri "github.com/textileio/broker-core/cmd/brokerd/packer"
	pb "github.com/textileio/broker-core/cmd/brokerd/pb/broker"
	pieceri "github.com/textileio/broker-core/cmd/brokerd/piecer"
	mongods "github.com/textileio/go-ds-mongo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var (
	log = logging.Logger("broker-service")
)

// Config provides configuration to the broker service.
type Config struct {
	GrpcListenAddress string

	MongoDBName string
	MongoURI    string
}

type Service struct {
	pb.UnimplementedAPIServiceServer

	config Config
	server *grpc.Server
	broker broker.Broker
}

var _ pb.APIServiceServer = (*Service)(nil)

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

	broker, err := brokeri.New(ds, packer, piecer)
	if err != nil {
		return nil, fmt.Errorf("creating broker implementation: %s", err)
	}

	s := &Service{
		config: config,
		server: grpc.NewServer(),
		broker: broker,
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

func (s *Service) CreateBR(ctx context.Context, r *pb.CreateBRRequest) (*pb.CreateBRResponse, error) {
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

	res := &pb.CreateBRResponse{
		Request: castBrokerRequestToProto(br),
	}

	return res, nil
}

func (s *Service) GetBR(ctx context.Context, r *pb.GetBRRequest) (*pb.GetBRResponse, error) {
	if r == nil {
		return nil, status.Error(codes.Internal, "empty request")
	}

	br, err := s.broker.Get(ctx, broker.BrokerRequestID(r.Id))
	if err != nil {
		return nil, status.Error(codes.Internal, fmt.Sprintf("get broker request: %s", err))
	}

	res := &pb.GetBRResponse{
		BrokerRequest: castBrokerRequestToProto(br),
	}

	return res, nil
}

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

	if errors != nil {
		return fmt.Errorf(strings.Join(errors, "\n"))
	}

	return nil
}

func castBrokerRequestToProto(br broker.BrokerRequest) *pb.BR {
	return &pb.BR{
		Id: string(br.ID),
		Meta: &pb.BRMetadata{
			Region: br.Metadata.Region,
		},
		StorageDealId: string(br.StorageDealID),
		CreatedAt:     timestamppb.New(br.CreatedAt),
		UpdatedAt:     timestamppb.New(br.UpdatedAt),
	}
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
