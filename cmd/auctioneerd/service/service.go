package service

import (
	"context"
	"fmt"

	"github.com/ipfs/go-cid"
	golog "github.com/textileio/go-log/v2"

	format "github.com/ipfs/go-ipld-format"
	"github.com/textileio/bidbot/lib/auction"
	"github.com/textileio/bidbot/lib/dshelper/txndswrap"
	"github.com/textileio/bidbot/lib/filclient"
	core "github.com/textileio/broker-core/auctioneer"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/auctioneerd/auctioneer"
	mbroker "github.com/textileio/broker-core/msgbroker"
	"github.com/textileio/go-libp2p-pubsub-rpc/finalizer"
	rpcpeer "github.com/textileio/go-libp2p-pubsub-rpc/peer"
)

var log = golog.Logger("auctioneer/service")

// Config defines params for Service configuration.
type Config struct {
	Peer    rpcpeer.Config
	Auction auctioneer.AuctionConfig
}

// Service is a gRPC service wrapper around an Auctioneer.
type Service struct {
	mb   mbroker.MsgBroker
	peer *rpcpeer.Peer
	lib  *auctioneer.Auctioneer

	finalizer *finalizer.Finalizer
}

var _ mbroker.ReadyToAuctionListener = (*Service)(nil)
var _ mbroker.DealProposalAcceptedListener = (*Service)(nil)

// New returns a new Service.
func New(conf Config, store txndswrap.TxnDatastore, mb mbroker.MsgBroker, fc filclient.FilClient) (*Service, error) {
	fin := finalizer.NewFinalizer()

	// Create auctioneer peer
	p, err := rpcpeer.New(conf.Peer)
	if err != nil {
		return nil, fin.Cleanupf("creating peer: %v", err)
	}
	fin.Add(p)

	// Create auctioneer
	lib, err := auctioneer.New(p, store, mb, fc, conf.Auction)
	if err != nil {
		return nil, fin.Cleanupf("creating auctioneer: %v", err)
	}
	fin.Add(lib)

	s := &Service{
		mb:        mb,
		peer:      p,
		lib:       lib,
		finalizer: fin,
	}

	if err := mbroker.RegisterHandlers(mb, s); err != nil {
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

// DAGService returns the underlying peer's format.DAGService.
func (s *Service) DAGService() format.DAGService {
	return s.peer.DAGService()
}

// PeerInfo returns the peer's public information.
func (s *Service) PeerInfo() (*rpcpeer.Info, error) {
	return s.peer.Info()
}

// GetAuction gets the state of an auction by id. Mostly for test purpose.
func (s *Service) GetAuction(id auction.AuctionID) (*core.Auction, error) {
	return s.lib.GetAuction(id)
}

// OnReadyToAuction handles messagse from ready-to-auction topic.
func (s *Service) OnReadyToAuction(
	ctx context.Context,
	id auction.AuctionID,
	sdID broker.BatchID,
	payloadCid cid.Cid,
	dealSize, dealDuration uint64,
	dealReplication uint32,
	dealVerified bool,
	excludedMiners []string,
	filEpochDeadline uint64,
	sources auction.Sources,
) error {
	err := s.lib.CreateAuction(core.Auction{
		ID:               id,
		BatchID:          sdID,
		PayloadCid:       payloadCid,
		DealSize:         dealSize,
		DealDuration:     dealDuration,
		DealReplication:  dealReplication,
		DealVerified:     dealVerified,
		ExcludedMiners:   excludedMiners,
		FilEpochDeadline: filEpochDeadline,
		Sources:          sources,
	})
	if err != nil {
		return fmt.Errorf("processing ready-to-auction msg: %s", err)
	}

	return nil
}

// OnDealProposalAccepted receives an accepted deal proposal from a miner.
func (s *Service) OnDealProposalAccepted(
	ctx context.Context,
	auctionID auction.AuctionID,
	bidID auction.BidID,
	proposalCid cid.Cid,
) error {
	if err := s.lib.DeliverProposal(auctionID, bidID, proposalCid); err != nil {
		return fmt.Errorf("procesing deal-proposal-accepted msgg : %v", err)
	}
	return nil
}
