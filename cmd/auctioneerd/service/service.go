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
	"github.com/textileio/broker-core/cmd/brokerd/client"
	"github.com/textileio/broker-core/dshelper"
	"github.com/textileio/broker-core/finalizer"
	pb "github.com/textileio/broker-core/gen/broker/auctioneer/v1"
	"github.com/textileio/broker-core/marketpeer"
	"github.com/textileio/broker-core/rpc"
	"google.golang.org/grpc"
)

var log = golog.Logger("auctioneer/service")

// Config defines params for Service configuration.
type Config struct {
	RepoPath   string
	ListenAddr string
	BrokerAddr string
	Peer       marketpeer.Config
	Auction    auctioneer.AuctionConfig
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
func New(conf Config) (*Service, error) {
	if err := validateConfig(conf); err != nil {
		return nil, fmt.Errorf("invalid config: %s", err)
	}

	fin := finalizer.NewFinalizer()

	// Create broker client
	broker, err := client.New(conf.BrokerAddr)
	if err != nil {
		return nil, fmt.Errorf("creating broker client: %s", err)
	}
	fin.Add(broker)

	// Create auctioneer peer
	p, err := marketpeer.New(conf.Peer)
	if err != nil {
		return nil, fin.Cleanupf("creating peer: %v", err)
	}
	fin.Add(p)

	// Create auctioneer
	store, err := dshelper.NewBadgerTxnDatastore(filepath.Join(conf.RepoPath, "auctionq"))
	if err != nil {
		return nil, fin.Cleanupf("creating repo: %v", err)
	}
	fin.Add(store)
	lib, err := auctioneer.New(p, store, broker, auctioneer.AuctionConfig{
		Duration: conf.Auction.Duration,
	})
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

func validateConfig(conf Config) error {
	if conf.ListenAddr == "" {
		return fmt.Errorf("listen address is empty")
	}
	// TODO: Re-enable when mocks are in place.
	// if conf.BrokerAddr == "" {
	// 	return fmt.Errorf("broker address is empty")
	// }
	return nil
}

// Close the service.
func (s *Service) Close() error {
	rpc.StopServer(s.server)
	log.Info("service was shutdown")
	return s.finalizer.Cleanup(nil)
}

// Bootstrap the market peer against well-known network peers.
func (s *Service) Bootstrap() {
	s.lib.Bootstrap()
}

// EnableMDNS enables an MDNS discovery service.
// This is useful on a local network (testing).
func (s *Service) EnableMDNS(intervalSecs int) error {
	return s.lib.EnableMDNS(intervalSecs)
}

// CreateAuction creates a new auction.
func (s *Service) CreateAuction(_ context.Context, req *pb.CreateAuctionRequest) (*pb.CreateAuctionResponse, error) {
	id, err := s.lib.CreateAuction(req.DealId, req.DealSize, req.DealDuration)
	if err != nil {
		return nil, err
	}
	return &pb.CreateAuctionResponse{
		Id: id,
	}, nil
}

// GetAuction gets an auction by id.
func (s *Service) GetAuction(_ context.Context, req *pb.GetAuctionRequest) (*pb.GetAuctionResponse, error) {
	a, err := s.lib.GetAuction(req.Id)
	if err != nil {
		return nil, err
	}
	return &pb.GetAuctionResponse{
		Auction: cast.AuctionToPb(a),
	}, nil
}
