package service

import (
	"context"
	"net"
	"time"

	ec "github.com/ethereum/go-ethereum/common"
	"github.com/textileio/bidbot/lib/common"
	"github.com/textileio/broker-core/cmd/chainapis/ethshared/contractclient"
	"github.com/textileio/broker-core/gen/broker/chainapi/v1"
	logging "github.com/textileio/go-log/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

var (
	log *logging.ZapEventLogger
)

// Service implements the chainservice for NEAR.
type Service struct {
	chainapi.UnimplementedChainApiServiceServer
	contractClients map[string]*contractclient.BridgeProvider
	server          *grpc.Server
}

// NewService creates a new Service.
func NewService(
	listener net.Listener,
	daemonName string,
	contractClients map[string]*contractclient.BridgeProvider,
) (*Service, error) {
	log = logging.Logger(daemonName)
	s := &Service{
		contractClients: contractClients,
		server:          grpc.NewServer(grpc.UnaryInterceptor(common.GrpcLoggerInterceptor(log))),
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
	contractClient, ok := s.contractClients[req.ChainId]
	if !ok {
		return nil, status.Errorf(codes.InvalidArgument, "unsupported chain id: %s", req.ChainId)
	}
	res, err := contractClient.HasDeposit(nil, ec.HexToAddress(req.Depositee))
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
	// TODO: Actuyally implement it, even though it will always return true in our usage.
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
