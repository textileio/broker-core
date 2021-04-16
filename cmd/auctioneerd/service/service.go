package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"path/filepath"

	golog "github.com/ipfs/go-log/v2"
	"github.com/textileio/broker-core/cmd/auctioneerd/auctioneer"
	"github.com/textileio/broker-core/cmd/auctioneerd/cast"
	"github.com/textileio/broker-core/finalizer"
	pb "github.com/textileio/broker-core/gen/broker/auctioneer/v1"
	kt "github.com/textileio/broker-core/keytransform"
	"github.com/textileio/broker-core/marketpeer"
	"github.com/textileio/broker-core/rpc"
	"google.golang.org/grpc"
)

var log = golog.Logger("auctioneer/service")

type Config struct {
	RepoPath   string
	ListenAddr string
	Peer       marketpeer.Config
}

type Service struct {
	pb.UnimplementedAPIServiceServer

	server    *grpc.Server
	lib       *auctioneer.Auctioneer
	finalizer *finalizer.Finalizer
}

var _ pb.APIServiceServer = (*Service)(nil)

func New(conf Config) (*Service, error) {
	fin := finalizer.NewFinalizer()

	// Create auctioneer peer
	p, err := marketpeer.New(conf.Peer)
	if err != nil {
		return nil, fin.Cleanupf("creating peer: %v", err)
	}
	fin.Add(p)

	// Create auctioneer
	store, err := kt.NewBadgerStore(filepath.Join(conf.RepoPath, "auctionq"))
	if err != nil {
		return nil, fin.Cleanupf("creating repo: %v", err)
	}
	fin.Add(store)
	lib, err := auctioneer.New(p, store)
	if err != nil {
		return nil, fin.Cleanupf("creating auctioneer: %v", err)
	}
	fin.Add(lib)

	s := &Service{
		server:    grpc.NewServer(),
		lib:       lib,
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

func (s *Service) Close() error {
	rpc.StopServer(s.server)
	log.Info("service was shutdown")
	return s.finalizer.Cleanup(nil)
}

func (s *Service) Bootstrap() {
	s.lib.Bootstrap()
}

func (s *Service) EnableMDNS(intervalSecs int) error {
	return s.lib.EnableMDNS(intervalSecs)
}

func (s *Service) CreateAuction(_ context.Context, _ *pb.CreateAuctionRequest) (*pb.CreateAuctionResponse, error) {
	id, err := s.lib.CreateAuction()
	if err != nil {
		return nil, err
	}
	return &pb.CreateAuctionResponse{
		Id: id,
	}, nil
}

func (s *Service) GetAuction(_ context.Context, req *pb.GetAuctionRequest) (*pb.GetAuctionResponse, error) {
	a, err := s.lib.GetAuction(req.Id)
	if err != nil {
		return nil, err
	}
	return &pb.GetAuctionResponse{
		Auction: cast.AuctionToPb(a),
	}, nil
}
