package service

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/ipfs/go-cid"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/textileio/bidbot/lib/auction"
	"github.com/textileio/bidbot/lib/datauri"
	"github.com/textileio/broker-core/broker"
	brokeri "github.com/textileio/broker-core/cmd/brokerd/broker"
	"github.com/textileio/broker-core/cmd/brokerd/cast"
	"github.com/textileio/broker-core/cmd/brokerd/store"
	"github.com/textileio/broker-core/common"
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

	PostgresURI string

	IPFSAPIMultiaddr string

	DealDuration         uint64
	DealReplication      uint32
	DefaultWalletAddress string
	VerifiedDeals        bool

	AuctionMaxRetries            int
	DefaultBatchDeadlineDuration time.Duration
	DefaultProposalStartOffset   time.Duration
}

// Service provides an implementation of the broker API.
type Service struct {
	pb.UnimplementedAPIServiceServer

	config Config
	server *grpc.Server

	broker *brokeri.Broker
}

var _ pb.APIServiceServer = (*Service)(nil)
var _ msgbroker.NewBatchCreatedListener = (*Service)(nil)
var _ msgbroker.NewBatchPreparedListener = (*Service)(nil)
var _ msgbroker.FinalizedDealListener = (*Service)(nil)
var _ msgbroker.AuctionClosedListener = (*Service)(nil)

// New returns a new Service.
func New(mb msgbroker.MsgBroker, config Config) (*Service, error) {
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("config is invalid: %s", err)
	}

	listener, err := net.Listen("tcp", config.ListenAddr)
	if err != nil {
		return nil, fmt.Errorf("getting net listener: %v", err)
	}
	defaultWalletAddress, err := address.NewFromString(config.DefaultWalletAddress)
	if err != nil {
		return nil, fmt.Errorf("invalid default wallet address: %v", err)
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
		ipfsClient,
		mb,
		brokeri.WithDealDuration(config.DealDuration),
		brokeri.WithDealReplication(config.DealReplication),
		brokeri.WithDefaultWalletAddress(defaultWalletAddress),
		brokeri.WithVerifiedDeals(config.VerifiedDeals),
		brokeri.WithAuctionMaxRetries(config.AuctionMaxRetries),
		brokeri.WithDefaultBatchDeadline(config.DefaultBatchDeadlineDuration),
		brokeri.WithProposalStartOffset(config.DefaultProposalStartOffset),
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

// CreatePreparedStorageRequest creates a new prepared StorageRequest.
func (s *Service) CreatePreparedStorageRequest(
	ctx context.Context,
	r *pb.CreatePreparedStorageRequestRequest) (*pb.CreatePreparedStorageRequestResponse, error) {
	log.Debugf("received prepared storage request")
	if r == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}

	payloadCid, err := cid.Decode(r.Cid)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid cid: %s", err)
	}

	pc, err := parsePreparedCAR(ctx, r)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid prepared car info: %s", err)
	}

	if r.Metadata.Origin == "" {
		return nil, status.Error(codes.InvalidArgument, "origin is empty")
	}
	providers := make([]address.Address, len(r.Metadata.Providers))
	for i, provStr := range r.Metadata.Providers {
		provAddr, err := address.NewFromString(provStr)
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "provider %s is invalid: %s", provStr, err)
		}
		if provAddr.Protocol() != address.ID {
			return nil, status.Errorf(codes.InvalidArgument, "%s should be an identity address", provStr)
		}
		providers[i] = provAddr
	}
	meta := broker.BatchMetadata{
		Origin:    r.Metadata.Origin,
		Tags:      make(map[string]string, len(r.Metadata.Tags)),
		Providers: providers,
	}
	for k, v := range meta.Tags {
		meta.Tags[k] = v
	}

	rw, err := parseRemoteWallet(r)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "remote wallet configuration is invalid: %s", err)
	}

	br, err := s.broker.CreatePrepared(ctx, payloadCid, pc, meta, rw)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "creating storage request: %s", err)
	}

	pbr, err := cast.StorageRequestToProto(br)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "converting result to proto: %s", err)
	}
	res := &pb.CreatePreparedStorageRequestResponse{
		Request: pbr,
	}

	return res, nil
}

// CreateStorageRequest creates a new StorageRequest.
func (s *Service) CreateStorageRequest(
	ctx context.Context,
	r *pb.CreateStorageRequestRequest) (*pb.CreateStorageRequestResponse, error) {
	log.Debugf("received storage request")
	if r == nil {
		return nil, status.Error(codes.Internal, "empty request")
	}
	c, err := cid.Decode(r.Cid)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid cid: %s", err)
	}
	if r.Origin == "" {
		return nil, status.Error(codes.InvalidArgument, "origin is empty")
	}

	br, err := s.broker.Create(ctx, c, r.Origin)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "creating storage request: %s", err)
	}

	pbr, err := cast.StorageRequestToProto(br)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "converting result to proto: %s", err)
	}
	res := &pb.CreateStorageRequestResponse{
		Request: pbr,
	}

	return res, nil
}

// GetStorageRequestInfo gets information about a storage request by id.
func (s *Service) GetStorageRequestInfo(
	ctx context.Context,
	r *pb.GetStorageRequestInfoRequest) (*pb.GetStorageRequestInfoResponse, error) {
	if r == nil {
		return nil, status.Error(codes.Internal, "empty request")
	}

	br, err := s.broker.GetStorageRequestInfo(ctx, broker.StorageRequestID(r.Id))
	if err != nil {
		return nil, status.Errorf(codes.Internal, "get storage request: %s", err)
	}

	res, err := cast.StorageRequestInfoToProto(br)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "converting result to proto: %s", err)
	}

	return res, nil
}

// OnNewBatchCreated handles new messages in new-batch-created topic.
func (s *Service) OnNewBatchCreated(
	ctx context.Context,
	id broker.BatchID,
	batchCid cid.Cid,
	batchSize int64,
	brids []broker.StorageRequestID,
	origin string,
	manifest []byte,
	carURL *url.URL) error {
	if _, err := s.broker.CreateNewBatch(ctx, id, batchCid, batchSize, brids, origin, manifest, carURL); err != nil {
		if errors.Is(err, store.ErrBatchExists) {
			log.Warnf("batch ID %s already created, acking", id)
			return nil
		}
		return fmt.Errorf("creating batch: %s", err)
	}
	log.Debugf("new batch created: %s", id)

	return nil
}

// OnAuctionClosed handles new messages in auction-closed topic.
func (s *Service) OnAuctionClosed(ctx context.Context, opID msgbroker.OperationID, au broker.ClosedAuction) error {
	if err := s.broker.BatchAuctioned(ctx, opID, au); err != nil {
		if errors.Is(err, brokeri.ErrOperationIDExists) {
			log.Warnf("operation %s already exists, acking", opID)
			return nil
		}
		return fmt.Errorf("processing closed auction: %s", err)
	}
	return nil
}

// OnNewBatchPrepared handles new messages in new-batch-prepared topic.
func (s *Service) OnNewBatchPrepared(
	ctx context.Context,
	id broker.BatchID,
	pr broker.DataPreparationResult) error {
	if err := s.broker.NewBatchPrepared(ctx, id, pr); err != nil {
		// idempotency is taken care of by NewBatchPrepared
		return fmt.Errorf("processing new prepared batch: %s", err)
	}

	return nil
}

// OnFinalizedDeal handles new messages in the finalized-deal topic.
func (s *Service) OnFinalizedDeal(ctx context.Context, opID msgbroker.OperationID, fd broker.FinalizedDeal) error {
	if err := s.broker.BatchFinalizedDeal(ctx, opID, fd); err != nil {
		if errors.Is(err, brokeri.ErrOperationIDExists) {
			log.Warnf("operation %s already exists, acking", opID)
			return nil
		}
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

func parsePreparedCAR(ctx context.Context, r *pb.CreatePreparedStorageRequestRequest) (broker.PreparedCAR, error) {
	if r.PreparedCar == nil {
		return broker.PreparedCAR{}, errors.New("empty prepared car information")
	}

	var pc broker.PreparedCAR
	var err error

	// Validate PieceCid.
	pc.PieceCid, err = cid.Decode(r.PreparedCar.PieceCid)
	if err != nil {
		return broker.PreparedCAR{}, fmt.Errorf("piece-cid is invalid: %s", err)
	}
	if pc.PieceCid.Prefix().Codec != broker.CodecFilCommitmentUnsealed {
		return broker.PreparedCAR{}, errors.New("piece-cid must be have fil-commitment-unsealed codec")
	}

	// Validate PieceSize.
	if r.PreparedCar.PieceSize <= 0 {
		return broker.PreparedCAR{}, errors.New("piece-size must be greater than zero")
	}
	if r.PreparedCar.PieceSize&(r.PreparedCar.PieceSize-1) != 0 {
		return broker.PreparedCAR{}, errors.New("piece-size must be a power of two")
	}
	if r.PreparedCar.PieceSize > broker.MaxPieceSize {
		return broker.PreparedCAR{}, fmt.Errorf("piece-size can't be greater than %d", broker.MaxPieceSize)
	}
	pc.PieceSize = r.PreparedCar.PieceSize

	// Validate rep factor.
	if r.PreparedCar.RepFactor < 0 {
		return broker.PreparedCAR{}, errors.New("rep-factor can't be negative")
	}
	pc.RepFactor = int(r.PreparedCar.RepFactor)

	pc.Deadline = r.PreparedCar.Deadline.AsTime()

	pc.ProposalStartOffset = 0
	if r.PreparedCar.ProposalStartOffset != nil {
		pc.ProposalStartOffset = r.PreparedCar.ProposalStartOffset.AsDuration()
	}

	if r.PreparedCar.CarUrl != nil {
		url, err := url.Parse(r.PreparedCar.CarUrl.Url)
		if err != nil {
			return broker.PreparedCAR{}, fmt.Errorf("parsing CAR URL: %s", err)
		}
		if url.Scheme != "http" && url.Scheme != "https" {
			return broker.PreparedCAR{}, errors.New("CAR URL scheme should be http(s)")
		}

		pc.Sources.CARURL = &auction.CARURL{
			URL: *url,
		}
	}

	if r.PreparedCar.CarIpfs != nil {
		carCid, err := cid.Decode(r.PreparedCar.CarIpfs.Cid)
		if err != nil {
			return broker.PreparedCAR{}, fmt.Errorf("car cid isn't valid: %s", err)
		}
		maddrs := make([]multiaddr.Multiaddr, len(r.PreparedCar.CarIpfs.Multiaddrs))
		for i, smaddr := range r.PreparedCar.CarIpfs.Multiaddrs {
			maddr, err := multiaddr.NewMultiaddr(smaddr)
			if err != nil {
				return broker.PreparedCAR{}, fmt.Errorf("invalid multiaddr %s: %s", smaddr, err)
			}
			maddrs[i] = maddr
		}
		pc.Sources.CARIPFS = &auction.CARIPFS{
			Cid:        carCid,
			Multiaddrs: maddrs,
		}
	}

	if pc.Sources.CARURL == nil && pc.Sources.CARIPFS == nil {
		return broker.PreparedCAR{}, errors.New("at least one download source must be specified")
	}

	u, err := datauri.NewFromSources(r.Cid, pc.Sources)
	if err != nil {
		return broker.PreparedCAR{}, fmt.Errorf("creating data URI from sources: %s", err)
	}
	if err := u.Validate(ctx); err != nil {
		return broker.PreparedCAR{}, fmt.Errorf("validating sources: %s", err)
	}

	return pc, nil
}

func parseRemoteWallet(r *pb.CreatePreparedStorageRequestRequest) (*broker.RemoteWallet, error) {
	if r.RemoteWallet == nil {
		return nil, nil
	}
	peerID, err := peer.Decode(r.RemoteWallet.PeerId)
	if err != nil {
		return nil, fmt.Errorf("invalid peer-id: %s", err)
	}
	if err := peerID.Validate(); err != nil {
		return nil, fmt.Errorf("empty peer-id: %s", err)
	}
	if r.RemoteWallet.AuthToken == "" {
		return nil, fmt.Errorf("empty authorization token: %s", err)
	}
	waddr, err := address.NewFromString(r.RemoteWallet.WalletAddr)
	if err != nil {
		return nil, fmt.Errorf("parsing wallet address: %s", err)
	}
	if waddr.Empty() {
		return nil, errors.New("wallet address is invalid (empty)")
	}
	multiaddrs := make([]multiaddr.Multiaddr, 0, len(r.RemoteWallet.Multiaddrs))
	for _, smaddr := range r.RemoteWallet.Multiaddrs {
		maddr, err := multiaddr.NewMultiaddr(smaddr)
		if err != nil {
			return nil, fmt.Errorf("multiaddress %s is invalid: %s", smaddr, err)
		}
		multiaddrs = append(multiaddrs, maddr)
	}
	rw := &broker.RemoteWallet{
		PeerID:     peerID,
		WalletAddr: waddr,
		AuthToken:  r.RemoteWallet.AuthToken,
		Multiaddrs: multiaddrs,
	}

	return rw, nil
}
