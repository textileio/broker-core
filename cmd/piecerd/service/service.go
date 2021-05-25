package service

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/ipfs/go-cid"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	golog "github.com/ipfs/go-log/v2"
	"github.com/multiformats/go-multiaddr"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/brokerd/client"
	"github.com/textileio/broker-core/cmd/piecerd/piecer"
	"github.com/textileio/broker-core/dshelper"
	"github.com/textileio/broker-core/finalizer"
	pb "github.com/textileio/broker-core/gen/broker/piecer/v1"
	"github.com/textileio/broker-core/rpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var log = golog.Logger("piecer/service")

// Config defines params for Service configuration.
type Config struct {
	ListenAddr string

	MongoDBName string
	MongoURI    string

	IpfsAPIMultiaddr string
	BrokerAPIAddr    string
}

// Service is a gRPC service wrapper around a piecer.
type Service struct {
	pb.UnimplementedAPIServiceServer

	server    *grpc.Server
	piecer    *piecer.Piecer
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
	lib, err := piecer.New(ds, ipfsClient, brokerClient)
	if err != nil {
		return nil, fin.Cleanupf("creating piecer: %v", err)
	}
	fin.Add(lib)

	s := &Service{
		server:    grpc.NewServer(),
		piecer:    lib,
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

// ReadyToPrepare indicates that a batch is ready to be prepared.
func (s *Service) ReadyToPrepare(ctx context.Context, r *pb.ReadyToPrepareRequest) (*pb.ReadyToPrepareResponse, error) {
	if r == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	if r.StorageDealId == "" {
		return nil, status.Error(codes.InvalidArgument, "storage deal id is empty")
	}
	dataCid, err := cid.Decode(r.DataCid)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "decoding data cid: %s", err)
	}
	if !dataCid.Defined() {
		return nil, status.Error(codes.InvalidArgument, "data cid is undefined")
	}

	if err := s.piecer.ReadyToPrepare(ctx, broker.StorageDealID(r.StorageDealId), dataCid); err != nil {
		return nil, status.Errorf(codes.Internal, "queuing data-cid to be prepared: %s", err)
	}

	return &pb.ReadyToPrepareResponse{}, nil
}

// Close the service.
func (s *Service) Close() error {
	rpc.StopServer(s.server)
	log.Info("service was shutdown")
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
