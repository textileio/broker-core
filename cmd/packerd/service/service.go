package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"path/filepath"

	"github.com/ipfs/go-cid"
	golog "github.com/ipfs/go-log/v2"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/packerd/packer"
	"github.com/textileio/broker-core/finalizer"
	pb "github.com/textileio/broker-core/gen/broker/packer/v1"
	kt "github.com/textileio/broker-core/keytransform"
	"github.com/textileio/broker-core/rpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var log = golog.Logger("packer/service")

// Config defines params for Service configuration.
type Config struct {
	RepoPath         string
	ListenAddr       string
	IpfsAPIMultiaddr string
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
	fin := finalizer.NewFinalizer()

	// TODO: wire mongo?
	store, err := kt.NewBadgerStore(filepath.Join(conf.RepoPath, "packerstore"))
	if err != nil {
		return nil, fin.Cleanupf("creating repo: %v", err)
	}
	fin.Add(store)

	lib, err := packer.New(store, conf.IpfsAPIMultiaddr)
	if err != nil {
		return nil, fin.Cleanupf("creating packer: %v", err)
	}
	fin.Add(lib)

	s := &Service{
		server:    grpc.NewServer(),
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

// CreateAuction creates a new auction.
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
	rpc.StopServer(s.server)
	log.Info("service was shutdown")
	return s.finalizer.Cleanup(nil)
}
