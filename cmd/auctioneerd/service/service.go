package service

import (
	"context"
	"errors"
	"net"

	"github.com/gogo/status"
	golog "github.com/ipfs/go-log/v2"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/auctioneerd/auctioneer"
	"github.com/textileio/broker-core/cmd/auctioneerd/cast"
	"github.com/textileio/broker-core/dshelper/txndswrap"
	"github.com/textileio/broker-core/finalizer"
	pb "github.com/textileio/broker-core/gen/broker/auctioneer/v1"
	"github.com/textileio/broker-core/marketpeer"
	"github.com/textileio/broker-core/rpc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
)

var log = golog.Logger("auctioneer/service")

// Config defines params for Service configuration.
type Config struct {
	Listener net.Listener
	Peer     marketpeer.Config
	Auction  auctioneer.AuctionConfig
}

// Service is a gRPC service wrapper around an Auctioneer.
type Service struct {
	pb.UnimplementedAPIServiceServer

	server *grpc.Server
	lib    *auctioneer.Auctioneer

	finalizer *finalizer.Finalizer
}

var _ pb.APIServiceServer = (*Service)(nil)

// New returns a new Service.
func New(conf Config, store txndswrap.TxnDatastore, broker broker.Broker, fc auctioneer.FilClient) (*Service, error) {
	fin := finalizer.NewFinalizer()

	// Create auctioneer peer
	p, err := marketpeer.New(conf.Peer)
	if err != nil {
		return nil, fin.Cleanupf("creating peer: %v", err)
	}
	fin.Add(p)

	// Create auctioneer
	lib, err := auctioneer.New(p, store, broker, fc, conf.Auction)
	if err != nil {
		return nil, fin.Cleanupf("creating auctioneer: %v", err)
	}
	fin.Add(lib)

	s := &Service{
		server:    grpc.NewServer(),
		lib:       lib,
		finalizer: fin,
	}

	go func() {
		pb.RegisterAPIServiceServer(s.server, s)
		if err := s.server.Serve(conf.Listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			log.Errorf("server error: %v", err)
		}
	}()

	log.Info("service started")
	return s, nil
}

// Close the service.
func (s *Service) Close() error {
	rpc.StopServer(s.server)
	log.Info("service was shutdown")
	return s.finalizer.Cleanup(nil)
}

// Start the deal auction feed.
// If bootstrap is true, the peer will dial the configured bootstrap addresses
// before creating the deal auction feed.
func (s *Service) Start(bootstrap bool) error {
	return s.lib.Start(bootstrap)
}

// ReadyToAuction creates a new auction.
func (s *Service) ReadyToAuction(_ context.Context, req *pb.ReadyToAuctionRequest) (*pb.ReadyToAuctionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	if req.StorageDealId == "" {
		return nil, status.Error(codes.InvalidArgument, "storage deal id is empty")
	}
	if req.DealSize == 0 {
		return nil, status.Error(codes.InvalidArgument, "deal size must be greater than zero")
	}
	if req.DealDuration == 0 {
		return nil, status.Error(codes.InvalidArgument, "deal duration must be greater than zero")
	}
	if req.DealReplication == 0 {
		return nil, status.Error(codes.InvalidArgument, "deal replication must be greater than zero")
	}

	id, err := s.lib.CreateAuction(
		broker.StorageDealID(req.StorageDealId),
		req.DealSize,
		req.DealDuration,
		req.DealReplication,
		req.DealVerified,
	)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "creating auction: %s", err)
	}
	return &pb.ReadyToAuctionResponse{
		Id: string(id),
	}, nil
}

// GetAuction gets an auction by id.
func (s *Service) GetAuction(_ context.Context, req *pb.GetAuctionRequest) (*pb.GetAuctionResponse, error) {
	a, err := s.lib.GetAuction(broker.AuctionID(req.Id))
	if err != nil {
		return nil, err
	}
	return &pb.GetAuctionResponse{
		Auction: cast.AuctionToPb(*a),
	}, nil
}
