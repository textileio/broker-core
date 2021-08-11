package service

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/textileio/broker-core/auctioneer"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/apid/store"
	pb "github.com/textileio/broker-core/gen/broker/v1"
	"github.com/textileio/broker-core/msgbroker"
	logging "github.com/textileio/go-log/v2"
)

var (
	log = logging.Logger("service")
)

// Service collects data from other daemons and runs the API server.
type Service struct {
	store *store.Store
}

var _ msgbroker.AuctionEventsListener = (*Service)(nil)

// New creates the service.
func New(mb msgbroker.MsgBroker, httpAddr string, postgresURI string) (*Service, error) {
	s, err := store.New(postgresURI)
	if err != nil {
		return nil, err
	}
	service := &Service{store: s}
	if err := msgbroker.RegisterHandlers(mb, service); err != nil {
		return nil, fmt.Errorf("registering msgbroker handlers: %s", err)
	}

	httpServer := &http.Server{
		Addr:              httpAddr,
		ReadHeaderTimeout: time.Second * 5,
		Handler:           http.NewServeMux(),
	}
	log.Info("running HTTP API...")
	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Errorf("stopping http server: %s", err)
		}
	}()

	return service, nil
}

// OnAuctionStarted .
func (s *Service) OnAuctionStarted(ctx context.Context, t time.Time, a *pb.AuctionSummary) {
	if err := s.store.CreateOrUpdateAuction(ctx, a); err != nil {
		log.Error(err)
	}
}

// OnAuctionBidReceived .
func (s *Service) OnAuctionBidReceived(ctx context.Context, t time.Time, a *pb.AuctionSummary, b *auctioneer.Bid) {
	err := s.store.CreateOrUpdateAuction(ctx, a)
	if err == nil {
		err = s.store.CreateOrUpdateBid(ctx, a.Id, b)
	}
	if err != nil {
		log.Error(err)
	}
}

// OnAuctionWinnerSelected .
func (s *Service) OnAuctionWinnerSelected(ctx context.Context, t time.Time, a *pb.AuctionSummary, b *auctioneer.Bid) {
	err := s.store.CreateOrUpdateAuction(ctx, a)
	if err == nil {
		err = s.store.CreateOrUpdateBid(ctx, a.Id, b)
	}
	if err == nil {
		err = s.store.WonBid(ctx, a.Id, b, t)
	}
	if err != nil {
		log.Error(err)
	}
}

// OnAuctionWinnerAcked .
func (s *Service) OnAuctionWinnerAcked(ctx context.Context, t time.Time, a *pb.AuctionSummary, b *auctioneer.Bid) {
	err := s.store.CreateOrUpdateAuction(ctx, a)
	if err == nil {
		err = s.store.CreateOrUpdateBid(ctx, a.Id, b)
	}
	if err == nil {
		err = s.store.AckedBid(ctx, a.Id, b, t)
	}
	if err != nil {
		log.Error(err)
	}
}

// OnAuctionProposalCidDelivered .
func (s *Service) OnAuctionProposalCidDelivered(ctx context.Context, ts time.Time,
	auctionID, bidID, proposalCid, errorCause string) {
	if err := s.store.ProposalDelivered(ctx, ts, auctionID, bidID, proposalCid, errorCause); err != nil {
		log.Error(err)
	}
}

// OnAuctionClosed .
func (s *Service) OnAuctionClosed(ctx context.Context, opID msgbroker.OperationID, ca broker.ClosedAuction) error {
	if err := s.store.AuctionClosed(ctx, ca, time.Now()); err != nil {
		log.Error(err)
	}
	return nil
}
