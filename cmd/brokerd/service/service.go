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
	"github.com/textileio/broker-core/broker"
	auctioneercast "github.com/textileio/broker-core/cmd/auctioneerd/cast"
	auctioneeri "github.com/textileio/broker-core/cmd/brokerd/auctioneer"
	brokeri "github.com/textileio/broker-core/cmd/brokerd/broker"
	"github.com/textileio/broker-core/cmd/brokerd/cast"
	chainapii "github.com/textileio/broker-core/cmd/brokerd/chainapi"
	dealeri "github.com/textileio/broker-core/cmd/brokerd/dealer"
	packeri "github.com/textileio/broker-core/cmd/brokerd/packer"
	pieceri "github.com/textileio/broker-core/cmd/brokerd/piecer"
	"github.com/textileio/broker-core/cmd/common"
	logging "github.com/textileio/go-log/v2"

	"github.com/textileio/broker-core/dshelper"
	pb "github.com/textileio/broker-core/gen/broker/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	log = logging.Logger("broker/service")
)

// Config provides configuration to the broker service.
type Config struct {
	ListenAddr string

	PiecerAddr     string
	PackerAddr     string
	AuctioneerAddr string
	DealerAddr     string
	ReporterAddr   string

	MongoDBName string
	MongoURI    string

	IPFSAPIMultiaddr string

	DealDuration    uint64
	DealReplication uint32
	VerifiedDeals   bool

	CARExportURL string
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
func New(config Config) (*Service, error) {
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("config is invalid: %s", err)
	}

	listener, err := net.Listen("tcp", config.ListenAddr)
	if err != nil {
		return nil, fmt.Errorf("getting net listener: %v", err)
	}

	ds, err := dshelper.NewMongoTxnDatastore(config.MongoURI, config.MongoDBName)
	if err != nil {
		return nil, fmt.Errorf("creating datastore: %s", err)
	}

	packer, err := packeri.New(config.PackerAddr)
	if err != nil {
		return nil, fmt.Errorf("creating packer implementation: %s", err)
	}

	piecer, err := pieceri.New(config.PiecerAddr)
	if err != nil {
		return nil, fmt.Errorf("creating piecer implementation: %s", err)
	}

	auctioneer, err := auctioneeri.New(config.AuctioneerAddr)
	if err != nil {
		return nil, fmt.Errorf("creating auctioneer implementation: %s", err)
	}

	dealer, err := dealeri.New(config.DealerAddr)
	if err != nil {
		return nil, fmt.Errorf("creating dealer implementation: %s", err)
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
		ds,
		packer,
		piecer,
		auctioneer,
		dealer,
		reporter,
		ipfsClient,
		brokeri.WithDealDuration(config.DealDuration),
		brokeri.WithDealReplication(config.DealReplication),
		brokeri.WithVerifiedDeals(config.VerifiedDeals),
		brokeri.WithCARExportURL(config.CARExportURL),
	)
	if err != nil {
		return nil, fmt.Errorf("creating broker implementation: %s", err)
	}

	s := &Service{
		config: config,
		server: grpc.NewServer(grpc.UnaryInterceptor(common.GrpcLoggerInterceptor(log))),
		broker: broker,
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

	// Validate rep factor.
	if r.PreparedCAR.RepFactor <= 0 {
		return nil, status.Error(codes.InvalidArgument, "rep-factor should be positive")
	}

	pc.Deadline = r.PreparedCAR.Deadline.AsTime()

	if r.PreparedCAR.CarUrl != nil {
		url, err := url.Parse(r.PreparedCAR.CarUrl.Url)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "parsing CAR URL: %s", err)
		}
		if url.Scheme != "http" && url.Scheme != "https" {
			return nil, status.Error(codes.InvalidArgument, "CAR URL scheme should be http(s)")
		}

		pc.Sources.CARURL = &broker.CARURL{
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
		pc.Sources.CARIPFS = &broker.CARIPFS{
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

// GetBrokerRequest gets an existing broker request.
func (s *Service) GetBrokerRequest(
	ctx context.Context,
	r *pb.GetBrokerRequestRequest) (*pb.GetBrokerRequestResponse, error) {
	if r == nil {
		return nil, status.Error(codes.Internal, "empty request")
	}

	br, err := s.broker.Get(ctx, broker.BrokerRequestID(r.Id))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get broker request: %s", err)
	}

	pbr, err := cast.BrokerRequestToProto(br)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "converting result to proto: %s", err)
	}
	res := &pb.GetBrokerRequestResponse{
		BrokerRequest: pbr,
	}

	return res, nil
}

// CreateStorageDeal deal creates a storage deal.
func (s *Service) CreateStorageDeal(
	ctx context.Context,
	r *pb.CreateStorageDealRequest) (*pb.CreateStorageDealResponse, error) {
	if r == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	batchCid, err := cid.Decode(r.BatchCid)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid cid: %s", r.BatchCid)
	}

	brids := make([]broker.BrokerRequestID, len(r.BrokerRequestIds))
	for i, id := range r.BrokerRequestIds {
		if id == "" {
			return nil, status.Error(codes.InvalidArgument, "broker request id can't be empty")
		}
		brids[i] = broker.BrokerRequestID(id)
	}

	sd, err := s.broker.CreateStorageDeal(ctx, batchCid, brids)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "creating storage deal: %s", err)
	}

	return &pb.CreateStorageDealResponse{Id: string(sd)}, nil
}

// StorageDealAuctioned indicated that an auction has completed.
func (s *Service) StorageDealAuctioned(
	ctx context.Context,
	r *pb.StorageDealAuctionedRequest) (*pb.StorageDealAuctionedResponse, error) {
	if r == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	auction, err := auctioneercast.AuctionFromPb(r.Auction)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid auction: %s", err)
	}

	if err := s.broker.StorageDealAuctioned(ctx, auction); err != nil {
		log.Errorf("storage deal auctioned: %s", err)
		return nil, status.Errorf(codes.Internal, "storage deal auctioned: %s", err)
	}

	return &pb.StorageDealAuctionedResponse{}, nil
}

// StorageDealPrepared indicates that a data batch is prepared for Filecoin onboarding.
func (s *Service) StorageDealPrepared(
	ctx context.Context,
	r *pb.StorageDealPreparedRequest) (*pb.StorageDealPreparedResponse, error) {
	if r == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	pieceCid, err := cid.Decode(r.PieceCid)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "decoding piece cid: %s", err)
	}
	id := broker.StorageDealID(r.StorageDealId)
	pr := broker.DataPreparationResult{
		PieceCid:  pieceCid,
		PieceSize: r.PieceSize,
	}
	if err := s.broker.StorageDealPrepared(ctx, id, pr); err != nil {
		return nil, status.Errorf(codes.Internal, "calling storage deal prepared: %s", err)
	}

	return &pb.StorageDealPreparedResponse{}, nil
}

// StorageDealFinalizedDeal reports the result of finalized deals.
func (s *Service) StorageDealFinalizedDeal(
	ctx context.Context,
	r *pb.StorageDealFinalizedDealRequest) (*pb.StorageDealFinalizedDealResponse, error) {
	if r == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	if r.StorageDealId == "" {
		return nil, status.Error(codes.InvalidArgument, "storage deal id is empty")
	}
	if r.DealId <= 0 {
		return nil, status.Errorf(
			codes.InvalidArgument,
			"deal id is %d and should be positive",
			r.DealId)
	}
	fad := broker.FinalizedAuctionDeal{
		StorageDealID:  broker.StorageDealID(r.StorageDealId),
		DealID:         r.DealId,
		DealExpiration: r.DealExpiration,
		Miner:          r.MinerId,
		ErrorCause:     r.ErrorCause,
	}

	if err := s.broker.StorageDealFinalizedDeal(ctx, fad); err != nil {
		return nil, status.Errorf(codes.Internal, "processing finalized deals: %s", err)
	}

	return &pb.StorageDealFinalizedDealResponse{}, nil
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
	if conf.PackerAddr == "" {
		return errors.New("packer api addr is empty")
	}
	if conf.AuctioneerAddr == "" {
		return errors.New("auctioneer api addr is empty")
	}
	if conf.DealerAddr == "" {
		return errors.New("dealer api addr is empty")
	}
	if conf.ReporterAddr == "" {
		return errors.New("reporter api addr is empty")
	}
	if conf.MongoDBName == "" {
		return errors.New("mongo db name is empty")
	}
	if conf.MongoURI == "" {
		return errors.New("mongo uri is empty")
	}
	if conf.IPFSAPIMultiaddr == "" {
		return errors.New("ipfs api multiaddress is empty")
	}
	if conf.DealDuration < broker.MinDealDuration {
		return fmt.Errorf("deal duration is less than minimum allowed: %d", broker.MinDealDuration)
	}
	if conf.DealDuration > broker.MaxDealDuration {
		return fmt.Errorf("deal duration is greater than maximum allowed: %d", broker.MaxDealDuration)
	}
	if conf.DealReplication < broker.MinDealReplication {
		return fmt.Errorf("deal replication is less than minimum allowed: %d", broker.MinDealDuration)
	}
	if conf.DealReplication > broker.MaxDealReplication {
		return fmt.Errorf("deal replication is greater than maximum allowed: %d", broker.MaxDealDuration)
	}
	return nil
}
