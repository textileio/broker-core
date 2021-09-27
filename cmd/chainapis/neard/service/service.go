package service

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/textileio/broker-core/cmd/chainapis/neard/providerclient"
	"github.com/textileio/broker-core/common"
	"github.com/textileio/broker-core/gen/broker/chainapi/v1"
	logging "github.com/textileio/go-log/v2"
	"github.com/textileio/near-api-go/keys"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

var (
	log = logging.Logger("service")
)

// Service implements the chainservice for NEAR.
type Service struct {
	chainapi.UnimplementedChainApiServiceServer
	pcs    map[string]*providerclient.Client
	server *grpc.Server
}

// NewService creates a new Service.
func NewService(listener net.Listener, pcs map[string]*providerclient.Client) (*Service, error) {
	s := &Service{
		pcs:    pcs,
		server: grpc.NewServer(grpc.UnaryInterceptor(common.GrpcLoggerInterceptor(log))),
	}
	go func() {
		log.Infof("Starting service in here %v...", listener.Addr().String())
		chainapi.RegisterChainApiServiceServer(s.server, s)
		reflection.Register(s.server)
		if err := s.server.Serve(listener); err != nil {
			log.Errorf("serve error: %v", err)
		}
	}()
	return s, nil
}

// HasDeposit returns whether or not the specified account id has deposited funds.
func (s *Service) HasDeposit(
	ctx context.Context,
	req *chainapi.HasDepositRequest,
) (*chainapi.HasDepositResponse, error) {
	pc, ok := s.pcs[req.ChainId]
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "unsupported chain id: %s", req.ChainId)
	}
	res, err := pc.HasDeposit(ctx, req.Depositee)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "calling has deposit: %v", err)
	}
	return &chainapi.HasDepositResponse{
		HasDeposit: res,
	}, nil
}

// OwnsPublicKey returns whether or not the specified account owns the provided public key.
func (s *Service) OwnsPublicKey(
	ctx context.Context,
	req *chainapi.OwnsPublicKeyRequest,
) (*chainapi.OwnsPublicKeyResponse, error) {
	pc, ok := s.pcs[req.ChainId]
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "unsupported chain id: %s", req.ChainId)
	}

	key, err := keys.NewPublicKeyFromString(req.PublicKey)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "parsing public key: %v", err)
	}

	_, err = pc.NearClient.Account(req.AccountId).ViewAccessKey(ctx, key)
	if err != nil && strings.Contains(err.Error(), "does not exist while viewing") {
		return &chainapi.OwnsPublicKeyResponse{OwnsPublicKey: false}, nil
	} else if err != nil {
		return nil, status.Errorf(codes.Internal, "viewing access key: %v", err)
	}

	return &chainapi.OwnsPublicKeyResponse{OwnsPublicKey: true}, nil
}

// Close stops the server and cleans up all internally created resources.
func (s *Service) Close() error {
	stopped := make(chan struct{})
	go func() {
		s.server.GracefulStop()
		close(stopped)
	}()
	t := time.NewTimer(10 * time.Second)
	select {
	case <-t.C:
		s.server.Stop()
	case <-stopped:
		t.Stop()
	}
	log.Info("gRPC server stopped")

	return nil
}
