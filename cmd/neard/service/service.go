package service

import (
	"context"
	"net"
	"time"

	logging "github.com/ipfs/go-log/v2"
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
		server: grpc.NewServer(),
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

// LockInfo returns the LockInfo for the specified account id.
func (s *Service) LockInfo(ctx context.Context, req *chainapi.LockInfoRequest) (*chainapi.LockInfoResponse, error) {
	state := s.sc.GetState()
	if state.BlockHeight < int(req.BlockHeight) {
		return nil, status.Errorf(
			codes.FailedPrecondition,
			"specified block height %v is greater than current state block height %v",
			req.BlockHeight,
			state.BlockHeight,
		)
	}
	li, ok := state.LockedFunds[req.AccountId]
	if !ok {
		return nil, status.Errorf(codes.NotFound, "no lock info for account id %v", req.AccountId)
	}
	return &chainapi.LockInfoResponse{
		LockInfo: toPbLockInfo(li),
	}, nil
}

// HasFunds returns whether or not the specified account id has locked funds.
func (s *Service) HasFunds(ctx context.Context, req *chainapi.HasFundsRequest) (*chainapi.HasFundsResponse, error) {
	res, err := s.lc.HasLocked(ctx, req.BrokerId, req.AccountId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "calling has funds: %v", err)
	}
	return &chainapi.HasFundsResponse{
		HasFunds: res,
	}, nil
}

// State returns the entire Lock Box state.
func (s *Service) State(ctx context.Context, req *chainapi.StateRequest) (*chainapi.StateResponse, error) {
	state := s.sc.GetState()
	lockedFunds := make(map[string]*chainapi.LockInfo)
	for accountID, li := range state.LockedFunds {
		lockedFunds[accountID] = toPbLockInfo(li)
	}
	return &chainapi.StateResponse{
		LockedFunds: lockedFunds,
		BlockHeight: uint64(state.BlockHeight),
		BlockHash:   state.BlockHash,
	}, nil
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

func toPbLockInfo(li lockboxclient.LockInfo) *chainapi.LockInfo {
	return &chainapi.LockInfo{
		AccountId: li.AccountID,
		BrokerId:  li.BrokerID,
		Deposit: &chainapi.DepositInfo{
			Amount:     li.Deposit.Amount.String(),
			Sender:     li.Deposit.Sender,
			Expiration: li.Deposit.Expiration,
		},
	}
}
