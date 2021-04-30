package service

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/ipfs/go-cid"
	golog "github.com/ipfs/go-log/v2"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/brokerd/client"
	"github.com/textileio/broker-core/cmd/dealerd/dealer"
	dealeri "github.com/textileio/broker-core/dealer"
	"github.com/textileio/broker-core/dshelper"
	"github.com/textileio/broker-core/finalizer"
	pb "github.com/textileio/broker-core/gen/broker/dealer/v1"
	"github.com/textileio/broker-core/rpc"
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

	BrokerAPIAddr string
}

// Service is a gRPC service wrapper around an packer.
type Service struct {
	pb.UnimplementedAPIServiceServer

	server    *grpc.Server
	dealer    *dealer.Dealer
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

	broker, err := client.New(conf.BrokerAPIAddr)
	if err != nil {
		return nil, fmt.Errorf("creating broker client: %s", err)
	}
	opts := []dealer.Option{}
	lib, err := dealer.New(ds, broker, opts...)
	if err != nil {
		return nil, fin.Cleanupf("creating dealer: %v", err)
	}
	fin.Add(lib)

	s := &Service{
		server:    grpc.NewServer(),
		dealer:    lib,
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

func (s *Service) ReadyToCreateDeals(
	ctx context.Context,
	r *pb.ReadyToCreateDealsRequest) (*pb.ReadyToCreateDealsResponse, error) {
	if r == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if r.StorageDealId == "" {
		return nil, status.Error(codes.InvalidArgument, "storage deal id is empty")
	}

	payloadCid, err := cid.Decode(r.PayloadCid)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "parsing payload cid %s: %s", r.PayloadCid, err)
	}
	pieceCid, err := cid.Decode(r.PieceCid)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "parsing piece cid %s: %s", r.PieceCid, err)
	}

	ad := dealeri.AuctionDeals{
		StorageDealID: broker.StorageDealID(r.StorageDealId),
		PayloadCid:    payloadCid,
		PieceCid:      pieceCid,
		PieceSize:     r.PieceSize,
		Duration:      r.Duration,
		Targets:       make([]dealeri.AuctionDealsTarget, len(r.Targets)),
	}
	for i, t := range r.Targets {
		if t.Miner == "" {
			return nil, status.Errorf(codes.InvalidArgument, "miner addr is empty")
		}
		if t.PricePerGibPerEpoch < 0 {
			return nil, status.Errorf(codes.InvalidArgument, "price per gib per epoch is negative")
		}
		if t.StartEpoch == 0 {
			return nil, status.Errorf(codes.InvalidArgument, "start epoch should be positive")
		}
		ad.Targets[i] = dealeri.AuctionDealsTarget{
			Miner:               t.Miner,
			PricePerGiBPerEpoch: t.PricePerGibPerEpoch,
			StartEpoch:          t.StartEpoch,
			Verified:            t.Verified,
			FastRetrieval:       t.FastRetrieval,
		}
	}

	if err := s.dealer.ReadyToCreateDeals(ctx, ad); err != nil {
		return nil, status.Errorf(codes.Internal, "processing ready to create deals: %s", err)
	}

	return &pb.ReadyToCreateDealsResponse{}, nil
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
