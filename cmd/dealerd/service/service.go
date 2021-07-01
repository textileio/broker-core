package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"

	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/lotus/api/v0api"
	"github.com/ipfs/go-cid"
	"github.com/textileio/bidbot/lib/auction"
	"github.com/textileio/bidbot/lib/common"
	"github.com/textileio/bidbot/lib/dshelper"
	"github.com/textileio/bidbot/lib/finalizer"
	"github.com/textileio/broker-core/cmd/brokerd/client"
	"github.com/textileio/broker-core/cmd/dealerd/dealer"
	"github.com/textileio/broker-core/cmd/dealerd/dealer/filclient"
	"github.com/textileio/broker-core/cmd/dealerd/dealermock"
	dealeri "github.com/textileio/broker-core/dealer"
	pb "github.com/textileio/broker-core/gen/broker/dealer/v1"
	"github.com/textileio/broker-core/rpc"
	golog "github.com/textileio/go-log/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var log = golog.Logger("dealer/service")

// Config defines params for Service configuration.
type Config struct {
	ListenAddr string

	MongoDBName string
	MongoURI    string

	LotusGatewayURL         string
	LotusExportedWalletAddr string

	AllowUnverifiedDeals             bool
	MaxVerifiedPricePerGiBPerEpoch   int64
	MaxUnverifiedPricePerGiBPerEpoch int64

	BrokerAPIAddr string

	Mock bool
}

// Service is a gRPC service wrapper around an packer.
type Service struct {
	pb.UnimplementedAPIServiceServer

	server    *grpc.Server
	dealer    dealeri.Dealer
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
		return nil, fin.Cleanupf("creating broker client: %s", err)
	}

	var lib dealeri.Dealer
	if conf.Mock {
		log.Warnf("running in mocked mode")
		lib = dealermock.New(broker)
	} else {
		var lotusAPI v0api.FullNodeStruct
		closer, err := jsonrpc.NewMergeClient(context.Background(), conf.LotusGatewayURL, "Filecoin",
			[]interface{}{
				&lotusAPI.CommonStruct.Internal,
				&lotusAPI.Internal,
			},
			http.Header{},
		)
		if err != nil {
			return nil, fin.Cleanupf("creating lotus gateway client: %s", err)
		}
		fin.Add(&nopCloser{closer})

		filclient, err := filclient.New(
			&lotusAPI,
			filclient.WithExportedKey(conf.LotusExportedWalletAddr),
			filclient.WithAllowUnverifiedDeals(conf.AllowUnverifiedDeals),
			filclient.WithMaxPriceLimits(conf.MaxVerifiedPricePerGiBPerEpoch, conf.MaxUnverifiedPricePerGiBPerEpoch),
		)
		if err != nil {
			return nil, fin.Cleanupf("creating filecoin client: %s", err)
		}
		libi, err := dealer.New(ds, broker, filclient)
		if err != nil {
			return nil, fin.Cleanupf("creating dealer: %v", err)
		}
		fin.Add(libi)
		lib = libi
	}

	s := &Service{
		server:    grpc.NewServer(grpc.UnaryInterceptor(common.GrpcLoggerInterceptor(log))),
		dealer:    lib,
		finalizer: fin,
	}

	listener, err := net.Listen("tcp", conf.ListenAddr)
	if err != nil {
		return nil, fin.Cleanupf("getting net listener: %v", err)
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

// ReadyToCreateDeals registers deals that are ready to be executed.
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
		StorageDealID: auction.StorageDealID(r.StorageDealId),
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
	defer log.Info("service was shutdown")

	log.Info("closing gRPC server...")
	rpc.StopServer(s.server)

	return s.finalizer.Cleanup(nil)
}

func validateConfig(conf Config) error {
	if conf.BrokerAPIAddr == "" {
		return errors.New("broker api addr is empty")
	}
	if conf.ListenAddr == "" {
		return errors.New("service listen addr is empty")
	}
	if conf.MongoDBName == "" {
		return errors.New("mongo db name is empty")
	}
	if conf.MongoURI == "" {
		return errors.New("mongo uri is empty")
	}

	return nil
}

type nopCloser struct {
	f func()
}

func (np *nopCloser) Close() error {
	np.f()
	return nil
}
