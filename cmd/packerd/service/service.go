package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/ipfs/go-cid"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	"github.com/multiformats/go-multiaddr"
	"github.com/textileio/bidbot/lib/common"
	"github.com/textileio/bidbot/lib/dshelper"
	"github.com/textileio/bidbot/lib/finalizer"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/brokerd/client"
	"github.com/textileio/broker-core/cmd/packerd/packer"
	pb "github.com/textileio/broker-core/gen/broker/packer/v1"
	"github.com/textileio/broker-core/rpc"
	golog "github.com/textileio/go-log/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var log = golog.Logger("packer/service")

// Config defines params for Service configuration.
type Config struct {
	ListenAddr string

	MongoDBName string
	MongoURI    string

	IpfsAPIMultiaddr string
	BrokerAPIAddr    string

	DaemonFrequency        time.Duration
	ExportMetricsFrequency time.Duration

	TargetSectorSize int64
	BatchMinSize     uint
}

// Service is a gRPC service wrapper around an packer.
type Service struct {
	pb.UnimplementedAPIServiceServer

	server    *grpc.Server
	packer    *packer.Packer
	finalizer *finalizer.Finalizer
}

var _ pb.APIServiceServer = (*Service)(nil)

// New returns a new Service.
func New(conf Config) (*Service, error) {
	if err := validateConfig(conf); err != nil {
		return nil, fmt.Errorf("config is invalid: %s", err)
	}

	fin := finalizer.NewFinalizer()

	ds, err := dshelper.NewMongoTxnDatastore(conf.MongoURI, conf.MongoDBName)
	if err != nil {
		return nil, fmt.Errorf("creating datastore: %s", err)
	}
	fin.Add(ds)

	ma, err := multiaddr.NewMultiaddr(conf.IpfsAPIMultiaddr)
	if err != nil {
		return nil, fmt.Errorf("parsing ipfs client multiaddr: %s", err)
	}
	ipfsClient, err := httpapi.NewApi(ma)
	if err != nil {
		return nil, fmt.Errorf("creating ipfs client: %s", err)
	}
	brokerClient, err := client.New(conf.BrokerAPIAddr)
	if err != nil {
		return nil, fmt.Errorf("creating broker client: %s", err)
	}
	opts := []packer.Option{
		packer.WithDaemonFrequency(conf.DaemonFrequency),
		packer.WithSectorSize(conf.TargetSectorSize),
		packer.WithBatchMinSize(conf.BatchMinSize),
	}
	lib, err := packer.New(ds, ipfsClient, brokerClient, opts...)
	if err != nil {
		return nil, fin.Cleanupf("creating packer: %v", err)
	}
	fin.Add(lib)

	s := &Service{
		server:    grpc.NewServer(grpc.UnaryInterceptor(common.GrpcLoggerInterceptor(log))),
		packer:    lib,
		finalizer: fin,
	}

	listener, err := net.Listen("tcp", conf.ListenAddr)
	if err != nil {
		return nil, fmt.Errorf("getting net listener: %v", err)
	}
	go func() {
		pb.RegisterAPIServiceServer(s.server, s)
		if err := s.server.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			log.Errorf("server error: %v", err)
		}
	}()

	log.Infof("service listening at %s", conf.ListenAddr)
	return s, nil
}

// ReadyToPack indicates that a broker request is ready to be packed.
func (s *Service) ReadyToPack(ctx context.Context, r *pb.ReadyToPackRequest) (*pb.ReadyToPackResponse, error) {
	if r == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	if r.BrokerRequestId == "" {
		return nil, status.Error(codes.InvalidArgument, "broker request id is empty")
	}
	dataCid, err := cid.Decode(r.DataCid)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "decoding data cid: %s", err)
	}

	if err := s.packer.ReadyToPack(ctx, broker.BrokerRequestID(r.BrokerRequestId), dataCid); err != nil {
		return nil, status.Errorf(codes.Internal, "queuing broker request: %s", err)
	}

	return &pb.ReadyToPackResponse{}, nil
}

// Close the service.
func (s *Service) Close() error {
	defer log.Info("service was shutdown")

	log.Info("closing gRPC server..")
	rpc.StopServer(s.server)

	return s.finalizer.Cleanup(nil)
}

func validateConfig(conf Config) error {
	if conf.BrokerAPIAddr == "" {
		return fmt.Errorf("broker api addr is empty")
	}
	if conf.IpfsAPIMultiaddr == "" {
		return fmt.Errorf("ipfs api multiaddr is empty")
	}
	if conf.ListenAddr == "" {
		return fmt.Errorf("service listen addr is empty")
	}
	if conf.MongoDBName == "" {
		return fmt.Errorf("mongo db name is empty")
	}
	if conf.MongoURI == "" {
		return fmt.Errorf("mongo uri is empty")
	}

	return nil
}
