package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"

	"github.com/ipfs/go-cid"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	"github.com/multiformats/go-multiaddr"
	"github.com/textileio/bidbot/lib/auction"
	"github.com/textileio/bidbot/lib/common"
	"github.com/textileio/broker-core/broker"
	auctioneeri "github.com/textileio/broker-core/cmd/brokerd/auctioneer"
	brokeri "github.com/textileio/broker-core/cmd/brokerd/broker"
	"github.com/textileio/broker-core/cmd/brokerd/cast"
	chainapii "github.com/textileio/broker-core/cmd/brokerd/chainapi"
	"github.com/textileio/broker-core/msgbroker"
	logger "github.com/textileio/go-log/v2"

	pb "github.com/textileio/broker-core/gen/broker/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	log = logger.Logger("broker/service")
)

// Config provides configuration to the broker service.
type Config struct {
	ListenAddr string

	PiecerAddr     string
	AuctioneerAddr string
	ReporterAddr   string

	PostgresURI string

	IPFSAPIMultiaddr string

	DealDuration    uint64
	DealReplication uint32
	VerifiedDeals   bool

	CARExportURL string

	AuctionMaxRetries int
}

// Service provides an implementation of the broker API.
type Service struct {
	pb.UnimplementedAPIServiceServer

	config Config
	server *grpc.Server

	broker *brokeri.Broker
}

var _ pb.APIServiceServer = (*Service)(nil)

// New returns a new Service.
func New(mb msgbroker.MsgBroker, config Config) (*Service, error) {
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("config is invalid: %s", err)
	}

	listener, err := net.Listen("tcp", config.ListenAddr)
	if err != nil {
		return nil, fmt.Errorf("getting net listener: %v", err)
	}

	auctioneer, err := auctioneeri.New(config.AuctioneerAddr)
	if err != nil {
		return nil, fmt.Errorf("creating auctioneer implementation: %s", err)
	}

	reporter, err := chainapii.New(config.ReporterAddr)
	if err != nil {
		return nil, fmt.Errorf("creating reporter implementation: %s", err)
	}

	ma, err := multiaddr.NewMultiaddr(config.IPFSAPIMultiaddr)
	if err != nil {
		return nil, fmt.Errorf("parsing ipfs client multiaddr: %s", err)
	}
	ipfsClient, err := httpapi.NewApi(ma)
	if err != nil {
		return nil, fmt.Errorf("creating ipfs client: %s", err)
	}

	broker, err := brokeri.New(
		config.PostgresURI,
		auctioneer,
		reporter,
		ipfsClient,
		mb,
		brokeri.WithDealDuration(config.DealDuration),
		brokeri.WithDealReplication(config.DealReplication),
		brokeri.WithVerifiedDeals(config.VerifiedDeals),
		brokeri.WithCARExportURL(config.CARExportURL),
		brokeri.WithAuctionMaxRetries(config.AuctionMaxRetries),
	)
	if err != nil {
		return nil, fmt.Errorf("creating broker implementation: %s", err)
	}

	s := &Service{
		config: config,
		server: grpc.NewServer(grpc.UnaryInterceptor(common.GrpcLoggerInterceptor(log))),
		broker: broker,
	}

	if err := msgbroker.RegisterHandlers(mb, s); err != nil {
		return nil, fmt.Errorf("registering msgbroker handlers: %s", err)
	}

	go func() {
		pb.RegisterAPIServiceServer(s.server, s)
		if err := s.server.Serve(listener); err != nil && !errors.Is(err, grpc.ErrServerStopped) {
			log.Errorf("server error: %v", err)
		}
	}()

	log.Infof("service listening at %s", config.ListenAddr)

	return s, nil
}

// CreatePreparedBrokerRequest creates a new prepared BrokerRequest.
func (s *Service) CreatePreparedBrokerRequest(
	ctx context.Context,
	r *pb.CreatePreparedBrokerRequestRequest) (*pb.CreatePreparedBrokerRequestResponse, error) {
	log.Debugf("received prepared broker request")
	if r == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	if r.PreparedCAR == nil {
		return nil, status.Error(codes.InvalidArgument, "empty prepared car information")
	}

	c, err := cid.Decode(r.Cid)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid cid: %s", err)
	}

	var pc broker.PreparedCAR
	// Validate PieceCid.
	pc.PieceCid, err = cid.Decode(r.PreparedCAR.PieceCid)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "piece-cid is invalid: %s", err)
	}
	if pc.PieceCid.Prefix().Codec != broker.CodecFilCommitmentUnsealed {
		return nil, status.Error(codes.InvalidArgument, "piece-cid must be have fil-commitment-unsealed codec")
	}

	// Validate PieceSize.
	if r.PreparedCAR.PieceSize <= 0 {
		return nil, status.Error(codes.InvalidArgument, "piece-size must be greater than zero")
	}
	if r.PreparedCAR.PieceSize&(r.PreparedCAR.PieceSize-1) != 0 {
		return nil, status.Error(codes.InvalidArgument, "piece-size must be a power of two")
	}
	if r.PreparedCAR.PieceSize > broker.MaxPieceSize {
		return nil, status.Errorf(codes.InvalidArgument, "piece-size can't be greater than %d", broker.MaxPieceSize)
	}
	pc.PieceSize = r.PreparedCAR.PieceSize

	// Validate rep factor.
	if r.PreparedCAR.RepFactor < 0 {
		return nil, status.Error(codes.InvalidArgument, "rep-factor can't be negative")
	}
	pc.RepFactor = int(r.PreparedCAR.RepFactor)

	pc.Deadline = r.PreparedCAR.Deadline.AsTime()

	if r.PreparedCAR.CarUrl != nil {
		url, err := url.Parse(r.PreparedCAR.CarUrl.Url)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "parsing CAR URL: %s", err)
		}
		if url.Scheme != "http" && url.Scheme != "https" {
			return nil, status.Error(codes.InvalidArgument, "CAR URL scheme should be http(s)")
		}

		pc.Sources.CARURL = &auction.CARURL{
			URL: *url,
		}
	}

	if r.PreparedCAR.CarIpfs != nil {
		carCid, err := cid.Decode(r.PreparedCAR.CarIpfs.Cid)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "car cid isn't valid: %s", err)
		}
		maddrs := make([]multiaddr.Multiaddr, len(r.PreparedCAR.CarIpfs.Multiaddrs))
		for i, smaddr := range r.PreparedCAR.CarIpfs.Multiaddrs {
			maddr, err := multiaddr.NewMultiaddr(smaddr)
			if err != nil {
				return nil, status.Errorf(codes.InvalidArgument, "invalid multiaddr %s: %s", smaddr, err)
			}
			maddrs[i] = maddr
		}
		pc.Sources.CARIPFS = &auction.CARIPFS{
			Cid:        carCid,
			Multiaddrs: maddrs,
		}
	}

	if pc.Sources.CARURL == nil && pc.Sources.CARIPFS == nil {
		return nil, status.Error(codes.InvalidArgument, "at least one download source must be specified")
	}

	br, err := s.broker.CreatePrepared(ctx, c, pc)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "creating storage request: %s", err)
	}

	pbr, err := cast.BrokerRequestToProto(br)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "converting result to proto: %s", err)
	}
	res := &pb.CreatePreparedBrokerRequestResponse{
		Request: pbr,
	}

	return res, nil
}

// CreateBrokerRequest creates a new BrokerRequest.
func (s *Service) CreateBrokerRequest(
	ctx context.Context,
	r *pb.CreateBrokerRequestRequest) (*pb.CreateBrokerRequestResponse, error) {
	log.Debugf("received broker request")
	if r == nil {
		return nil, status.Error(codes.Internal, "empty request")
	}

	c, err := cid.Decode(r.Cid)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid cid: %s", err)
	}

	br, err := s.broker.Create(ctx, c)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "creating storage request: %s", err)
	}

	pbr, err := cast.BrokerRequestToProto(br)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "converting result to proto: %s", err)
	}
	res := &pb.CreateBrokerRequestResponse{
		Request: pbr,
	}

	return res, nil
}

// GetBrokerRequestInfo gets information about a broker request by id.
func (s *Service) GetBrokerRequestInfo(
	ctx context.Context,
	r *pb.GetBrokerRequestInfoRequest) (*pb.GetBrokerRequestInfoResponse, error) {
	if r == nil {
		return nil, status.Error(codes.Internal, "empty request")
	}

	br, err := s.broker.GetBrokerRequestInfo(ctx, broker.BrokerRequestID(r.Id))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get broker request: %s", err)
	}

	res, err := cast.BrokerRequestInfoToProto(br)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "converting result to proto: %s", err)
	}

	return res, nil
}

// OnNewBatchCreated handles new messages in new-batch-created topic.
func (s *Service) OnNewBatchCreated(
	ctx context.Context,
	id broker.StorageDealID,
	batchCid cid.Cid, brids []broker.BrokerRequestID) error {
	if _, err := s.broker.CreateNewBatch(ctx, id, batchCid, brids); err != nil {
		return fmt.Errorf("creating storage deal: %s", err)
	}
	log.Debugf("new batch created: %s", id)

	return nil
}

// StorageDealAuctioned indicated that an auction has completed.
func (s *Service) StorageDealAuctioned(
	ctx context.Context,
	r *pb.StorageDealAuctionedRequest) (*pb.StorageDealAuctionedResponse, error) {
	if r == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	auction, err := cast.ClosedAuctionFromPb(r)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid auction: %s", err)
	}

	if err := s.broker.StorageDealAuctioned(ctx, auction); err != nil {
		log.Errorf("storage deal auctioned: %s", err)
		return nil, status.Errorf(codes.Internal, "storage deal auctioned: %s", err)
	}

	return &pb.StorageDealAuctionedResponse{}, nil
}

// OnNewBatchPrepared handles new messages in new-batch-prepared topic.
func (s *Service) OnNewBatchPrepared(
	ctx context.Context,
	id broker.StorageDealID,
	pr broker.DataPreparationResult) error {
	if err := s.broker.NewBatchPrepared(ctx, id, pr); err != nil {
		return fmt.Errorf("processing new prepared batch: %s", err)
	}

	return nil
}

// OnFinalizedDeal handles new messages in the finalized-deal topic.
func (s *Service) OnFinalizedDeal(ctx context.Context, fd broker.FinalizedDeal) error {
	if err := s.broker.StorageDealFinalizedDeal(ctx, fd); err != nil {
		return fmt.Errorf("processing finalized deal: %s", err)
	}

	return nil
}

// Close gracefully closes the service.
func (s *Service) Close() error {
	defer log.Info("service closed")

	var errors []string
	defer log.Info("service was shutdown with %d errors", len(errors))

	log.Info("closing gRPC server")
	s.server.GracefulStop()

	return nil
}

func validateConfig(conf Config) error {
	if conf.ListenAddr == "" {
		return errors.New("service listen addr is empty")
	}
	if conf.PiecerAddr == "" {
		return errors.New("piecer api addr is empty")
	}
	if conf.AuctioneerAddr == "" {
		return errors.New("auctioneer api addr is empty")
	}
	if conf.ReporterAddr == "" {
		return errors.New("reporter api addr is empty")
	}
	if conf.PostgresURI == "" {
		return errors.New("postgres uri is empty")
	}
	if conf.IPFSAPIMultiaddr == "" {
		return errors.New("ipfs api multiaddress is empty")
	}
	if conf.DealDuration < auction.MinDealDuration {
		return fmt.Errorf("deal duration is less than minimum allowed: %d", auction.MinDealDuration)
	}
	if conf.DealDuration > auction.MaxDealDuration {
		return fmt.Errorf("deal duration is greater than maximum allowed: %d", auction.MaxDealDuration)
	}
	if conf.DealReplication < broker.MinDealReplication {
		return fmt.Errorf("deal replication is less than minimum allowed: %d", broker.MinDealReplication)
	}
	if conf.DealReplication > broker.MaxDealReplication {
		return fmt.Errorf("deal replication is greater than maximum allowed: %d", broker.MaxDealReplication)
	}
	return nil
}
