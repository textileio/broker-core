package service

import (
	"context"
	"fmt"

	"github.com/ipfs/go-cid"
	"github.com/textileio/bidbot/lib/auction"
	"github.com/textileio/bidbot/lib/filclient"
	core "github.com/textileio/broker-core/auctioneer"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/auctioneerd/auctioneer"
	mbroker "github.com/textileio/broker-core/msgbroker"
	"github.com/textileio/go-libp2p-pubsub-rpc/finalizer"
	rpcpeer "github.com/textileio/go-libp2p-pubsub-rpc/peer"
	golog "github.com/textileio/go-log/v2"
)

var log = golog.Logger("auctioneer/service")

// Config defines params for Service configuration.
type Config struct {
	Peer               rpcpeer.Config
	Auction            auctioneer.AuctionConfig
	PostgresURI        string
	RecordBidbotEvents bool
}

// Service is a gRPC service wrapper around an Auctioneer.
type Service struct {
	mb        mbroker.MsgBroker
	lib       *auctioneer.Auctioneer
	finalizer *finalizer.Finalizer
}

var _ mbroker.ReadyToAuctionListener = (*Service)(nil)
var _ mbroker.DealProposalAcceptedListener = (*Service)(nil)
var _ mbroker.FinalizedDealListener = (*Service)(nil)

// New returns a new Service.
func New(conf Config, mb mbroker.MsgBroker, fc filclient.FilClient) (*Service, error) {
	fin := finalizer.NewFinalizer()

	// Create auctioneer
	lib, err := auctioneer.New(conf.Peer, conf.PostgresURI, mb, fc, conf.Auction, conf.RecordBidbotEvents)
	if err != nil {
		return nil, fin.Cleanupf("creating auctioneer: %v", err)
	}
	fin.Add(lib)

	s := &Service{
		mb:        mb,
		lib:       lib,
		finalizer: fin,
	}

	// OnDealProposalAccepted needs to notify bidbot about the proposal which requires NotifyTimeout. 2x to be safe.
	if err := mbroker.RegisterHandlers(mb, s, mbroker.WithACKDeadline(auctioneer.NotifyTimeout*2)); err != nil {
		return nil, fmt.Errorf("registering msgbroker handlers: %s", err)
	}

	log.Info("service started")
	return s, nil
}

// Close the service.
func (s *Service) Close() error {
	defer log.Info("service was shutdown")

	return s.finalizer.Cleanup(nil)
}

// Start the deal auction feed.
// If bootstrap is true, the peer will dial the configured bootstrap addresses
// before creating the deal auction feed.
func (s *Service) Start(bootstrap bool) error {
	return s.lib.Start(bootstrap)
}

// PeerInfo returns the peer's public information.
func (s *Service) PeerInfo() (*rpcpeer.Info, error) {
	return s.lib.PeerInfo()
}

// GetAuction gets the state of an auction by id. Mostly for test purpose.
func (s *Service) GetAuction(ctx context.Context, id auction.ID) (*core.Auction, error) {
	return s.lib.GetAuction(ctx, id)
}

// OnReadyToAuction handles messagse from ready-to-auction topic.
func (s *Service) OnReadyToAuction(
	ctx context.Context,
	id auction.ID,
	sdID broker.BatchID,
	payloadCid cid.Cid,
	dealSize uint64,
	dealDuration uint64,
	dealReplication uint32,
	dealVerified bool,
	excludedStorageProviders []string,
	filEpochDeadline uint64,
	sources auction.Sources,
	clientAddress string,
	providers []string,
) error {
	_, err := s.lib.GetAuction(ctx, id)
	if err == nil {
		log.Warnf("auction %s already exists, skip processing", id)
		return nil
	} else if err != auctioneer.ErrAuctionNotFound {
		return fmt.Errorf("get auction: %v", err)
	}
	err = s.lib.CreateAuction(ctx, core.Auction{
		ID:                       id,
		BatchID:                  sdID,
		PayloadCid:               payloadCid,
		DealSize:                 dealSize,
		DealDuration:             dealDuration,
		DealReplication:          dealReplication,
		DealVerified:             dealVerified,
		ExcludedStorageProviders: excludedStorageProviders,
		Providers:                providers,
		FilEpochDeadline:         filEpochDeadline,
		Sources:                  sources,
		ClientAddress:            clientAddress,
	})
	if err != nil {
		return fmt.Errorf("processing ready-to-auction msg: %s", err)
	}
	return nil
}

// OnDealProposalAccepted receives an accepted deal proposal from a storage-provider.
func (s *Service) OnDealProposalAccepted(
	ctx context.Context,
	auctionID auction.ID,
	bidID auction.BidID,
	proposalCid cid.Cid,
) error {
	if err := s.lib.DeliverProposal(ctx, auctionID, bidID, proposalCid); err != nil {
		return fmt.Errorf("procesing deal-proposal-accepted msg: %v", err)
	}
	return nil
}

// OnFinalizedDeal receives a finalized deal from a storage-provider.
func (s *Service) OnFinalizedDeal(ctx context.Context, _ mbroker.OperationID, fad broker.FinalizedDeal) error {
	if err := s.lib.MarkFinalizedDeal(ctx, fad); err != nil {
		return fmt.Errorf("procesing dfinalized-deal msg: %v", err)
	}
	return nil
}
