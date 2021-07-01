package service

import (
	"context"
	"errors"
	"net"

	"github.com/gogo/status"
	"github.com/ipfs/go-cid"
	format "github.com/ipfs/go-ipld-format"
	"github.com/textileio/bidbot/lib/auction"
	"github.com/textileio/bidbot/lib/common"
	"github.com/textileio/bidbot/lib/dshelper/txndswrap"
	"github.com/textileio/bidbot/lib/filclient"
	"github.com/textileio/bidbot/lib/finalizer"
	"github.com/textileio/bidbot/lib/marketpeer"
	core "github.com/textileio/broker-core/auctioneer"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/auctioneerd/auctioneer"
	"github.com/textileio/broker-core/cmd/auctioneerd/cast"
	pb "github.com/textileio/broker-core/gen/broker/auctioneer/v1"
	"github.com/textileio/broker-core/rpc"
	golog "github.com/textileio/go-log/v2"
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
	peer   *marketpeer.Peer
	lib    *auctioneer.Auctioneer

	finalizer *finalizer.Finalizer
}

var _ pb.APIServiceServer = (*Service)(nil)

// New returns a new Service.
func New(conf Config, store txndswrap.TxnDatastore, broker broker.Broker, fc filclient.FilClient) (*Service, error) {
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
		server:    grpc.NewServer(grpc.UnaryInterceptor(common.GrpcLoggerInterceptor(log))),
		peer:      p,
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
	defer log.Info("service was shutdown")

	log.Info("closing gRPC server...")
	rpc.StopServer(s.server)

	return s.finalizer.Cleanup(nil)
}

// Start the deal auction feed.
// If bootstrap is true, the peer will dial the configured bootstrap addresses
// before creating the deal auction feed.
func (s *Service) Start(bootstrap bool) error {
	return s.lib.Start(bootstrap)
}

// DAGService returns the underlying peer's format.DAGService.
func (s *Service) DAGService() format.DAGService {
	return s.peer.DAGService()
}

// PeerInfo returns the peer's public information.
func (s *Service) PeerInfo() (*marketpeer.PeerInfo, error) {
	return s.peer.Info()
}

// GetAuction gets the state of an auction by id. Mostly for test purpose.
func (s *Service) GetAuction(id auction.AuctionID) (*core.Auction, error) {
	return s.lib.GetAuction(id)
}

// ReadyToAuction creates a new auction.
func (s *Service) ReadyToAuction(_ context.Context, req *pb.ReadyToAuctionRequest) (*pb.ReadyToAuctionResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	if req.StorageDealId == "" {
		return nil, status.Error(codes.InvalidArgument, "storage deal id is empty")
	}
	if req.PayloadCid == "" {
		return nil, status.Error(codes.InvalidArgument, "payload cid is empty")
	}
	payloadCid, err := cid.Parse(req.PayloadCid)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "payload cid unparseable")
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
	sources, err := cast.SourcesFromPb(req.Sources)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "decoding sources: %v", err)
	}

	id, err := s.lib.CreateAuction(core.Auction{
		StorageDealID:    broker.StorageDealID(req.StorageDealId),
		PayloadCid:       payloadCid,
		DealSize:         req.DealSize,
		DealDuration:     req.DealDuration,
		DealReplication:  req.DealReplication,
		DealVerified:     req.DealVerified,
		ExcludedMiners:   req.ExcludedMiners,
		FilEpochDeadline: req.FilEpochDeadline,
		Sources:          sources,
	})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "%v", err)
	}
	return &pb.ReadyToAuctionResponse{
		Id: string(id),
	}, nil
}

// ProposalAccepted receives an accepted deal proposal from a miner.
func (s *Service) ProposalAccepted(
	_ context.Context,
	req *pb.ProposalAcceptedRequest,
) (*pb.ProposalAcceptedResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	if req.AuctionId == "" {
		return nil, status.Error(codes.InvalidArgument, "auction id is required")
	}
	if req.BidId == "" {
		return nil, status.Error(codes.InvalidArgument, "bid id is required")
	}
	proposalCid, err := cid.Decode(req.ProposalCid)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid proposal cid")
	}
	if err := s.lib.DeliverProposal(auction.AuctionID(req.AuctionId), auction.BidID(req.BidId), proposalCid); err != nil {
		return nil, status.Errorf(codes.Internal, "delivering proposal: %v", err)
	}
	return &pb.ProposalAcceptedResponse{}, nil
}
