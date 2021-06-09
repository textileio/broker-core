package service

import (
	"context"
	"net"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"github.com/textileio/broker-core/cmd/common"
	"github.com/textileio/broker-core/cmd/neard/lockboxclient"
	"github.com/textileio/broker-core/cmd/neard/statecache"
	"github.com/textileio/broker-core/gen/broker/chainapi/v1"
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
	sc     *statecache.StateCache
	lc     *lockboxclient.Client
	server *grpc.Server
}

// NewService creates a new Service.
func NewService(listener net.Listener, stateCache *statecache.StateCache, lc *lockboxclient.Client) (*Service, error) {
	s := &Service{
		sc:     stateCache,
		lc:     lc,
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
	res, err := s.lc.HasDeposit(ctx, req.BrokerId, req.AccountId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "calling has deposit: %v", err)
	}
	return &chainapi.HasDepositResponse{
		HasDeposit: res,
	}, nil
}

// UpdatePayload reports storage info back to the smart contract.
func (s *Service) UpdatePayload(
	ctx context.Context,
	req *chainapi.UpdatePayloadRequest,
) (*chainapi.UpdatePayloadResponse, error) {
	var dealInfos []lockboxclient.DealInfo
	for _, info := range req.Options.Deals {
		dealInfos = append(dealInfos, lockboxclient.DealInfo{
			DealID:     info.DealId,
			MinerID:    info.MinerId,
			Expiration: info.Expiration,
		})
	}
	opts := lockboxclient.PayloadOptions{
		PieceCid: req.Options.PieceCid,
		Deals:    dealInfos,
		DataCids: req.Options.DataCids,
	}
	if err := s.lc.UpdatePayload(ctx, req.PayloadCid, opts); err != nil {
		return nil, status.Errorf(codes.Internal, "calling update payload: %v", err)
	}
	return &chainapi.UpdatePayloadResponse{}, nil
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
