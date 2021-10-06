package msgbroker

import (
	"context"
	"errors"
	"fmt"
	"net/url"

	"github.com/filecoin-project/go-address"
	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/textileio/bidbot/lib/auction"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/common"
	"github.com/textileio/broker-core/dealer"
	pb "github.com/textileio/broker-core/gen/broker/v1"
	"google.golang.org/protobuf/proto"
)

// TopicHandler is function that processes a received message.
// If no error is returned, the message will be automatically acked.
// If an error is returned, the message will be automatically nacked.
type TopicHandler func(context.Context, []byte) error

// MsgBroker is a message-broker for async message communication.
type MsgBroker interface {
	// RegisterTopicHandler registers a handler to a topic, with a defined
	// subscription defined by the underlying implementation. Is highly recommended
	// to register handlers in a type-safe way using RegisterHandlers().
	RegisterTopicHandler(topic TopicName, handler TopicHandler, opts ...Option) error

	// PublishMsg publishes a message to the desired topic.
	PublishMsg(ctx context.Context, topicName TopicName, data []byte) error
}

// TopicName is a topic name.
type TopicName string

const (
	// NewBatchCreatedTopic is the topic name for new-batch-created messages.
	NewBatchCreatedTopic TopicName = "new-batch-created"
	// NewBatchPreparedTopic is the topic name for new-batch-prepared messages.
	NewBatchPreparedTopic = "new-batch-prepared"
	// ReadyToBatchTopic is the topic name for ready-to-batch messages.
	ReadyToBatchTopic = "ready-to-batch"
	// ReadyToCreateDealsTopic is the topic name for ready-to-create-deals messages.
	ReadyToCreateDealsTopic = "ready-to-create-deals"
	// FinalizedDealTopic is the topic name for finalized-deal messages.
	FinalizedDealTopic = "finalized-deal"
	// DealProposalAcceptedTopic is the topic name for deal-proposal-accepted messages.
	DealProposalAcceptedTopic = "deal-proposal-accepted"
	// ReadyToAuctionTopic is the topic name for ready-to-auction messages.
	ReadyToAuctionTopic = "ready-to-auction"

	// AuctionClosedTopic is the topic name for auction-closed messages.
	AuctionClosedTopic = "auction-closed"
	// AuctionStartedTopic is the topic name for auction-started messages.
	AuctionStartedTopic = "auction-started"
	// AuctionBidReceivedTopic is the topic name for auction-bid-received messages.
	AuctionBidReceivedTopic = "auction-bid-received"
	// AuctionWinnerSelectedTopic is the topic name for auction-winner-selected messages.
	AuctionWinnerSelectedTopic = "auction-winner-selected"
	// AuctionWinnerAckedTopic is the topic name for auction-winner-acked messages.
	AuctionWinnerAckedTopic = "auction-winner-acked"
	// AuctionProposalCidDeliveredTopic is the topic name for auction-proposal-cid-delivered messages.
	AuctionProposalCidDeliveredTopic = "auction-proposal-cid-delivered"
)

// OperationID is a unique identifier for messages.
type OperationID string

// NewBatchCreatedListener is a handler for new-batch-created topic.
type NewBatchCreatedListener interface {
	OnNewBatchCreated(
		context.Context,
		broker.BatchID,
		cid.Cid,
		[]broker.StorageRequestID,
		string,
		[]byte,
		*url.URL) error
}

// NewBatchPreparedListener is a handler for new-batch-prepared topic.
type NewBatchPreparedListener interface {
	OnNewBatchPrepared(context.Context, broker.BatchID, broker.DataPreparationResult) error
}

// ReadyToBatchListener is a handler for ready-to-batch topic.
type ReadyToBatchListener interface {
	OnReadyToBatch(context.Context, OperationID, []ReadyToBatchData) error
}

// ReadyToBatchData contains storage request data information to be batched.
type ReadyToBatchData struct {
	StorageRequestID broker.StorageRequestID
	DataCid          cid.Cid
	Origin           string
}

// ReadyToCreateDealsListener is a handler for ready-to-create-deals topic.
type ReadyToCreateDealsListener interface {
	OnReadyToCreateDeals(context.Context, dealer.AuctionDeals) error
}

// FinalizedDealListener is a handler for finalized-deal topic.
type FinalizedDealListener interface {
	OnFinalizedDeal(context.Context, OperationID, broker.FinalizedDeal) error
}

// DealProposalAcceptedListener is a handler for deal-proposal-accepted topic.
type DealProposalAcceptedListener interface {
	OnDealProposalAccepted(context.Context, auction.ID, auction.BidID, cid.Cid) error
}

// ReadyToAuctionListener is a handler for finalized-deal topic.
type ReadyToAuctionListener interface {
	OnReadyToAuction(
		ctx context.Context,
		id auction.ID,
		BatchID broker.BatchID,
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
	) error
}

// AuctionClosedListener is a handler for auction-closed topic.
type AuctionClosedListener interface {
	OnAuctionClosed(context.Context, OperationID, broker.ClosedAuction) error
}

// RegisterHandlers automatically calls mb.RegisterTopicHandler in the methods that
// s might satisfy on known XXXListener interfaces. This allows to automatically wire
// s to receive messages from topics of implemented handlers.
func RegisterHandlers(mb MsgBroker, s interface{}, opts ...Option) error {
	var countRegistered int
	if l, ok := s.(NewBatchCreatedListener); ok {
		countRegistered++
		err := mb.RegisterTopicHandler(NewBatchCreatedTopic, func(ctx context.Context, data []byte) error {
			r := &pb.NewBatchCreated{}
			if err := proto.Unmarshal(data, r); err != nil {
				return fmt.Errorf("unmarshal new batch created: %s", err)
			}
			if r.Id == "" {
				return errors.New("batch id is empty")
			}
			baID := broker.BatchID(r.Id)

			baCid, err := cid.Cast(r.BatchCid)
			if err != nil {
				return fmt.Errorf("decoding batch cid: %s", err)
			}
			if !baCid.Defined() {
				return errors.New("data cid is undefined")
			}
			if len(r.StorageRequestIds) == 0 {
				return errors.New("storage requests list is empty")
			}
			srIDs := make([]broker.StorageRequestID, len(r.StorageRequestIds))
			for i, id := range r.StorageRequestIds {
				if id == "" {
					return errors.New("storage request id can't be empty")
				}
				srIDs[i] = broker.StorageRequestID(id)
			}
			if r.Origin == "" {
				return errors.New("origin is empty")
			}
			if len(r.Manifest) == 0 {
				return errors.New("manifest is empty")
			}
			if r.CarUrl == "" {
				return errors.New("car url is empty")
			}
			carURL, err := url.ParseRequestURI(r.CarUrl)
			if err != nil {
				return fmt.Errorf("parsing car url %s: %s", r.CarUrl, err)
			}

			if err := l.OnNewBatchCreated(ctx, baID, baCid, srIDs, r.Origin, r.Manifest, carURL); err != nil {
				return fmt.Errorf("calling on-new-batch-created handler: %s", err)
			}
			return nil
		}, opts...)
		if err != nil {
			return fmt.Errorf("registering handler for new-batch-created topic: %s", err)
		}
	}

	if l, ok := s.(NewBatchPreparedListener); ok {
		countRegistered++
		err := mb.RegisterTopicHandler(NewBatchPreparedTopic, func(ctx context.Context, data []byte) error {
			r := &pb.NewBatchPrepared{}
			if err := proto.Unmarshal(data, r); err != nil {
				return fmt.Errorf("unmarshal new batch prepared: %s", err)
			}
			if r.Id == "" {
				return errors.New("batch id is empty")
			}
			pieceCid, err := cid.Cast(r.PieceCid)
			if err != nil {
				return fmt.Errorf("decoding piece cid: %s", err)
			}
			if r.PieceSize == 0 {
				return fmt.Errorf("piece size is zero")
			}
			pr := broker.DataPreparationResult{
				PieceCid:  pieceCid,
				PieceSize: r.PieceSize,
			}
			id := broker.BatchID(r.Id)
			if err := l.OnNewBatchPrepared(ctx, id, pr); err != nil {
				return fmt.Errorf("calling on-new-batch-prepared handler: %s", err)
			}
			return nil
		}, opts...)
		if err != nil {
			return fmt.Errorf("registering handler for new-batch-prepared topic: %s", err)
		}
	}

	if l, ok := s.(ReadyToBatchListener); ok {
		countRegistered++
		err := mb.RegisterTopicHandler(ReadyToBatchTopic, func(ctx context.Context, data []byte) error {
			r := &pb.ReadyToBatch{}
			if err := proto.Unmarshal(data, r); err != nil {
				return fmt.Errorf("unmarshal ready to batch: %s", err)
			}
			if r.OperationId == "" {
				return fmt.Errorf("operation-id is empty")
			}
			if len(r.DataCids) == 0 {
				return errors.New("data cids list can't be empty")
			}

			rtb := make([]ReadyToBatchData, len(r.DataCids))
			for i := range r.DataCids {
				if r.DataCids[i].StorageRequestId == "" {
					return errors.New("storage request id is empty")
				}
				brID := broker.StorageRequestID(r.DataCids[i].StorageRequestId)
				dataCid, err := cid.Cast(r.DataCids[i].DataCid)
				if err != nil {
					return fmt.Errorf("decoding data cid: %s", err)
				}
				if !dataCid.Defined() {
					return errors.New("data cid is undefined")
				}
				if r.DataCids[i].Origin == "" {
					return errors.New("origin is empty")
				}
				rtb[i] = ReadyToBatchData{
					StorageRequestID: brID,
					DataCid:          dataCid,
					Origin:           r.DataCids[i].Origin,
				}
			}

			if err := l.OnReadyToBatch(ctx, OperationID(r.OperationId), rtb); err != nil {
				return fmt.Errorf("calling ready-to-batch handler: %s", err)
			}
			return nil
		}, opts...)
		if err != nil {
			return fmt.Errorf("registering handler for ready-to-batch topic: %s", err)
		}
	}

	if l, ok := s.(ReadyToCreateDealsListener); ok {
		countRegistered++
		err := mb.RegisterTopicHandler(ReadyToCreateDealsTopic, func(ctx context.Context, data []byte) error {
			r := &pb.ReadyToCreateDeals{}
			if err := proto.Unmarshal(data, r); err != nil {
				return fmt.Errorf("unmarshal ready-to-create-deals: %s", err)
			}
			if r.Id == "" {
				return errors.New("id is empty")
			}

			if r.BatchId == "" {
				return errors.New("batch id is empty")
			}

			payloadCid, err := cid.Cast(r.PayloadCid)
			if err != nil {
				return fmt.Errorf("parsing payload cid %s: %s", r.PayloadCid, err)
			}
			if !payloadCid.Defined() {
				return errors.New("payloadcid is undefined")
			}
			pieceCid, err := cid.Cast(r.PieceCid)
			if err != nil {
				return fmt.Errorf("parsing piece cid %s: %s", r.PieceCid, err)
			}
			if !pieceCid.Defined() {
				return errors.New("piececid is undefined")
			}
			if r.Duration == 0 {
				return errors.New("duration is zero")
			}
			if len(r.Proposals) == 0 {
				return fmt.Errorf("batch %s: list of proposals is empty", r.BatchId)
			}
			ads := dealer.AuctionDeals{
				ID:         r.Id,
				BatchID:    broker.BatchID(r.BatchId),
				PayloadCid: payloadCid,
				PieceCid:   pieceCid,
				PieceSize:  r.PieceSize,
				Duration:   r.Duration,
				Proposals:  make([]dealer.Proposal, len(r.Proposals)),
			}
			for i, t := range r.Proposals {
				if t.StorageProviderId == "" {
					return errors.New("storage-provider-id is empty")
				}
				if t.PricePerGibPerEpoch < 0 {
					return errors.New("price per gib per epoch is negative")
				}
				if t.StartEpoch == 0 {
					return errors.New("start epoch should be positive")
				}
				if t.AuctionId == "" {
					return errors.New("auction-id is empty")
				}
				if t.BidId == "" {
					return errors.New("bid-id is empty")
				}
				ads.Proposals[i] = dealer.Proposal{
					StorageProviderID:   t.StorageProviderId,
					PricePerGiBPerEpoch: t.PricePerGibPerEpoch,
					StartEpoch:          t.StartEpoch,
					Verified:            t.Verified,
					FastRetrieval:       t.FastRetrieval,
					AuctionID:           auction.ID(t.AuctionId),
					BidID:               auction.BidID(t.BidId),
				}
			}
			if r.RemoteWallet != nil {
				peerID, err := peer.Decode(r.RemoteWallet.PeerId)
				if err != nil {
					return fmt.Errorf("decoding peer-id: %s", err)
				}
				if err := peerID.Validate(); err != nil {
					return fmt.Errorf("peer id is invalid: %s", err)
				}
				if r.RemoteWallet.AuthToken == "" {
					return errors.New("remote wallet auth token is empty")
				}
				waddr, err := address.NewFromString(r.RemoteWallet.WalletAddr)
				if err != nil {
					return fmt.Errorf("decoding wallet addr %s: %s", r.RemoteWallet.WalletAddr, err)
				}
				if waddr.Empty() {
					return errors.New("remote wallet wallet addr is empty")
				}
				maddrs := make([]multiaddr.Multiaddr, len(r.RemoteWallet.Multiaddrs))
				for i, strmaddr := range r.RemoteWallet.Multiaddrs {
					maddr, err := multiaddr.NewMultiaddr(strmaddr)
					if err != nil {
						return fmt.Errorf("parsing multiaddr %s: %s", strmaddr, err)
					}
					maddrs[i] = maddr
				}
				ads.RemoteWallet = &broker.RemoteWallet{
					PeerID:     peerID,
					AuthToken:  r.RemoteWallet.AuthToken,
					WalletAddr: waddr,
					Multiaddrs: maddrs,
				}
			}
			if err := l.OnReadyToCreateDeals(ctx, ads); err != nil {
				return fmt.Errorf("calling ready-to-create-deals handler: %s", err)
			}
			return nil
		}, opts...)
		if err != nil {
			return fmt.Errorf("registering handler for ready-to-create-deals topic: %s", err)
		}
	}

	if l, ok := s.(FinalizedDealListener); ok {
		countRegistered++
		err := mb.RegisterTopicHandler(FinalizedDealTopic, func(ctx context.Context, data []byte) error {
			r := &pb.FinalizedDeal{}
			if err := proto.Unmarshal(data, r); err != nil {
				return fmt.Errorf("unmarshal finalized deal msg: %s", err)
			}
			if r.OperationId == "" {
				return errors.New("operation id is empty")
			}
			if r.BatchId == "" {
				return errors.New("batch id is empty")
			}
			if r.ErrorCause == "" {
				if r.DealId <= 0 {
					return fmt.Errorf(
						"deal id is %d and should be positive",
						r.DealId)
				}
				if r.DealExpiration <= 0 {
					return fmt.Errorf(
						"deal expiration is %d and should be positive",
						r.DealExpiration)
				}
			}
			if r.StorageProviderId == "" {
				return errors.New("storage-provider-id is empty")
			}
			if r.AuctionId == "" {
				return errors.New("auction-id is empty")
			}
			if r.BidId == "" {
				return errors.New("bid-id is empty")
			}
			fd := broker.FinalizedDeal{
				BatchID:           broker.BatchID(r.BatchId),
				DealID:            r.DealId,
				DealExpiration:    r.DealExpiration,
				StorageProviderID: r.StorageProviderId,
				ErrorCause:        r.ErrorCause,
				AuctionID:         auction.ID(r.AuctionId),
				BidID:             auction.BidID(r.BidId),
			}

			if err := l.OnFinalizedDeal(ctx, OperationID(r.OperationId), fd); err != nil {
				return fmt.Errorf("calling finalized-deal handler: %s", err)
			}
			return nil
		}, opts...)
		if err != nil {
			return fmt.Errorf("registering handler for finalized-deal topic: %s", err)
		}
	}

	if l, ok := s.(DealProposalAcceptedListener); ok {
		countRegistered++
		err := mb.RegisterTopicHandler(DealProposalAcceptedTopic, func(ctx context.Context, data []byte) error {
			r := &pb.DealProposalAccepted{}
			if err := proto.Unmarshal(data, r); err != nil {
				return fmt.Errorf("unmarshal deal-proposal-accepted msg: %s", err)
			}
			if r.BatchId == "" {
				return errors.New("batch id is empty")
			}
			if r.StorageProviderId == "" {
				return errors.New("storage-provider-id is empty")
			}
			proposalCid, err := cid.Cast(r.ProposalCid)
			if err != nil || !proposalCid.Defined() {
				return errors.New("invalid proposal cid")
			}
			if !proposalCid.Defined() {
				return errors.New("proposal cid is undefined")
			}
			if r.AuctionId == "" {
				return errors.New("auction id is required")
			}
			if r.BidId == "" {
				return errors.New("bid id is required")
			}

			if err := l.OnDealProposalAccepted(
				ctx,
				auction.ID(r.AuctionId),
				auction.BidID(r.BidId),
				proposalCid); err != nil {
				return fmt.Errorf("calling deal-proposal-accepted handler: %s", err)
			}
			return nil
		}, opts...)
		if err != nil {
			return fmt.Errorf("registering handler for deal-proposal-accepted topic: %s", err)
		}
	}

	if l, ok := s.(ReadyToAuctionListener); ok {
		countRegistered++
		err := mb.RegisterTopicHandler(ReadyToAuctionTopic, func(ctx context.Context, data []byte) error {
			req := &pb.ReadyToAuction{}
			if err := proto.Unmarshal(data, req); err != nil {
				return fmt.Errorf("unmarshal ready-to-auction msg: %s", err)
			}
			if req.Id == "" {
				return errors.New("auction-id is empty")
			}
			if req.BatchId == "" {
				return errors.New("batch id is empty")
			}
			payloadCid, err := cid.Cast(req.PayloadCid)
			if err != nil {
				return errors.New("payload cid invalid")
			}
			if !payloadCid.Defined() {
				return errors.New("payload cid is undefined")
			}
			if req.DealSize == 0 {
				return errors.New("deal size must be greater than zero")
			}
			if req.DealReplication == 0 {
				return errors.New("deal replication must be greater than zero")
			}
			sources, err := sourcesFromPb(req.Sources)
			if err != nil {
				return fmt.Errorf("decoding sources: %v", err)
			}
			if err := l.OnReadyToAuction(
				ctx,
				auction.ID(req.Id),
				broker.BatchID(req.BatchId),
				payloadCid,
				req.DealSize,
				req.DealDuration,
				req.DealReplication,
				req.DealVerified,
				req.ExcludedStorageProviders,
				req.FilEpochDeadline,
				sources,
				req.ClientAddress,
				req.Providers,
			); err != nil {
				return fmt.Errorf("calling ready-to-auction handler: %s", err)
			}
			return nil
		}, opts...)
		if err != nil {
			return fmt.Errorf("registering handler for ready-to-auction topic: %s", err)
		}
	}

	if l, ok := s.(AuctionEventsListener); ok {
		countRegistered++
		if err := registerAuctionEventsListener(mb, l); err != nil {
			return fmt.Errorf("registering handler for auction events: %s", err)
		}
	} else if l, ok := s.(AuctionClosedListener); ok {
		countRegistered++
		err := mb.RegisterTopicHandler(AuctionClosedTopic, onAuctionClosedTopic(l), opts...)
		if err != nil {
			return fmt.Errorf("registering handler for auction-closed topic: %s", err)
		}
	}

	if countRegistered == 0 {
		return errors.New("no handlers were registered")
	}

	return nil
}

// PublishMsgReadyToBatch publishes a message to the ready-to-batch topic.
func PublishMsgReadyToBatch(ctx context.Context, mb MsgBroker, dataCids []ReadyToBatchData) error {
	if len(dataCids) == 0 {
		return errors.New("data cids is empty")
	}
	msg := &pb.ReadyToBatch{
		OperationId: uuid.New().String(),
		DataCids:    make([]*pb.ReadyToBatch_ReadyToBatchBR, len(dataCids)),
	}

	for i := range dataCids {
		if dataCids[i].StorageRequestID == "" {
			return errors.New("storage-request-id is empty")
		}
		if !dataCids[i].DataCid.Defined() {
			return errors.New("data-cid is undefined")
		}
		if dataCids[i].Origin == "" {
			return errors.New("origin is empty")
		}
		msg.DataCids[i] = &pb.ReadyToBatch_ReadyToBatchBR{
			StorageRequestId: string(dataCids[i].StorageRequestID),
			DataCid:          dataCids[i].DataCid.Bytes(),
			Origin:           dataCids[i].Origin,
		}
	}
	return marshalAndPublish(ctx, mb, ReadyToBatchTopic, msg)
}

// PublishMsgNewBatchCreated publishes a message to the new-batch-created topic.
func PublishMsgNewBatchCreated(
	ctx context.Context,
	mb MsgBroker,
	batchID broker.BatchID,
	batchCid cid.Cid,
	srIDs []broker.StorageRequestID,
	origin string,
	manifest []byte,
	carURL string) error {
	if batchID == "" {
		return errors.New("batch-id is empty")
	}
	if !batchCid.Defined() {
		return errors.New("batch-cid is undefined")
	}
	if origin == "" {
		return errors.New("origin is empty")
	}
	if len(manifest) == 0 {
		return errors.New("manifest is empty")
	}
	srStrIDs := make([]string, len(srIDs))
	for i, srID := range srIDs {
		if srID == "" {
			return errors.New("storage-request-id is empty")
		}
		srStrIDs[i] = string(srID)
	}
	if carURL == "" {
		return errors.New("car url is empty")
	}
	if _, err := url.ParseRequestURI(carURL); err != nil {
		return fmt.Errorf("car url %s is invalid: %s", carURL, err)
	}

	msg := &pb.NewBatchCreated{
		Id:                string(batchID),
		BatchCid:          batchCid.Bytes(),
		StorageRequestIds: srStrIDs,
		Origin:            origin,
		Manifest:          manifest,
		CarUrl:            carURL,
	}
	return marshalAndPublish(ctx, mb, NewBatchCreatedTopic, msg)
}

// PublishMsgNewBatchPrepared publishes a message to the new-batch-prepared topic.
func PublishMsgNewBatchPrepared(
	ctx context.Context,
	mb MsgBroker,
	baID broker.BatchID,
	pieceCid cid.Cid,
	pieceSize uint64) error {
	if baID == "" {
		return errors.New("batch-id is empty")
	}
	if !pieceCid.Defined() {
		return errors.New("piece-cid is undefined")
	}
	if pieceSize == 0 {
		return errors.New("piece-size is zero")
	}
	msg := &pb.NewBatchPrepared{
		Id:        string(baID),
		PieceCid:  pieceCid.Bytes(),
		PieceSize: pieceSize,
	}
	return marshalAndPublish(ctx, mb, NewBatchPreparedTopic, msg)
}

// PublishMsgReadyToCreateDeals publishes a message to the ready-to-create-deals topic.
func PublishMsgReadyToCreateDeals(
	ctx context.Context,
	mb MsgBroker,
	ads dealer.AuctionDeals) error {
	msg := &pb.ReadyToCreateDeals{
		Id:         ads.ID,
		BatchId:    string(ads.BatchID),
		PayloadCid: ads.PayloadCid.Bytes(),
		PieceCid:   ads.PieceCid.Bytes(),
		PieceSize:  ads.PieceSize,
		Duration:   ads.Duration,
		Proposals:  make([]*pb.ReadyToCreateDeals_Proposal, len(ads.Proposals)),
	}
	for i, t := range ads.Proposals {
		if t.StorageProviderID == "" {
			return errors.New("storage-provider-id is empty")
		}
		if t.StartEpoch == 0 {
			return errors.New("start-epoch is zero")
		}
		if t.AuctionID == "" {
			return errors.New("auction-id is empty")
		}
		if t.BidID == "" {
			return errors.New("bid-id is empty")
		}
		msg.Proposals[i] = &pb.ReadyToCreateDeals_Proposal{
			StorageProviderId:   t.StorageProviderID,
			PricePerGibPerEpoch: t.PricePerGiBPerEpoch,
			StartEpoch:          t.StartEpoch,
			Verified:            t.Verified,
			FastRetrieval:       t.FastRetrieval,
			AuctionId:           string(t.AuctionID),
			BidId:               string(t.BidID),
		}
	}
	if ads.RemoteWallet != nil {
		if err := ads.RemoteWallet.PeerID.Validate(); err != nil {
			return fmt.Errorf("remote wallet peer-id is invalid: %s", err)
		}
		if ads.RemoteWallet.AuthToken == "" {
			return errors.New("remote wallet auth-token is empty")
		}
		if ads.RemoteWallet.WalletAddr.Empty() {
			return errors.New("remote wallet wallet address is empty")
		}
		maddrs := make([]string, len(ads.RemoteWallet.Multiaddrs))
		for i, maddr := range ads.RemoteWallet.Multiaddrs {
			maddrs[i] = maddr.String()
		}

		msg.RemoteWallet = &pb.RemoteWallet{
			PeerId:     ads.RemoteWallet.PeerID.String(),
			AuthToken:  ads.RemoteWallet.AuthToken,
			WalletAddr: ads.RemoteWallet.WalletAddr.String(),
			Multiaddrs: maddrs,
		}
	}

	return marshalAndPublish(ctx, mb, ReadyToCreateDealsTopic, msg)
}

// PublishMsgFinalizedDeal publishes a message to the finalized-deal topic.
func PublishMsgFinalizedDeal(ctx context.Context, mb MsgBroker, fd broker.FinalizedDeal) error {
	msg := &pb.FinalizedDeal{
		OperationId:       uuid.New().String(),
		BatchId:           string(fd.BatchID),
		StorageProviderId: fd.StorageProviderID,
		DealId:            fd.DealID,
		DealExpiration:    fd.DealExpiration,
		ErrorCause:        fd.ErrorCause,
		AuctionId:         string(fd.AuctionID),
		BidId:             string(fd.BidID),
	}
	return marshalAndPublish(ctx, mb, FinalizedDealTopic, msg)
}

// PublishMsgDealProposalAccepted publishes a message to the deal-proposal-accepted topic.
func PublishMsgDealProposalAccepted(
	ctx context.Context,
	mb MsgBroker,
	sdID broker.BatchID,
	auctionID auction.ID,
	bidID auction.BidID,
	storageProviderID string,
	propCid cid.Cid) error {
	msg := &pb.DealProposalAccepted{
		BatchId:           string(sdID),
		StorageProviderId: storageProviderID,
		ProposalCid:       propCid.Bytes(),
		AuctionId:         string(auctionID),
		BidId:             string(bidID),
	}
	return marshalAndPublish(ctx, mb, DealProposalAcceptedTopic, msg)
}

// PublishMsgReadyToAuction publishes a message to the ready-to-auction topic.
func PublishMsgReadyToAuction(
	ctx context.Context,
	mb MsgBroker,
	id auction.ID,
	BatchID broker.BatchID,
	payloadCid cid.Cid,
	dealSize uint64,
	dealDuration,
	dealReplication int,
	dealVerified bool,
	excludedStorageProviders []string,
	filEpochDeadline uint64,
	sources auction.Sources,
	clientAddress string,
	providers []address.Address) error {
	msg := &pb.ReadyToAuction{
		Id:                       string(id),
		BatchId:                  string(BatchID),
		PayloadCid:               payloadCid.Bytes(),
		DealSize:                 dealSize,
		DealDuration:             uint64(dealDuration),
		DealReplication:          uint32(dealReplication),
		DealVerified:             dealVerified,
		ExcludedStorageProviders: excludedStorageProviders,
		FilEpochDeadline:         filEpochDeadline,
		Sources:                  sourcesToPb(sources),
		ClientAddress:            clientAddress,
		Providers:                common.StringifyAddrs(providers...),
	}
	return marshalAndPublish(ctx, mb, ReadyToAuctionTopic, msg)
}

// PublishMsgAuctionClosed publishes a message to the auction-closed topic.
func PublishMsgAuctionClosed(ctx context.Context, mb MsgBroker, au broker.ClosedAuction) error {
	msg, err := closedAuctionToPb(au)
	if err != nil {
		return fmt.Errorf("converting to pb: %s", err)
	}
	return marshalAndPublish(ctx, mb, AuctionClosedTopic, msg)
}

func sourcesToPb(sources auction.Sources) *pb.Sources {
	var carIPFS *pb.Sources_CARIPFS
	if sources.CARIPFS != nil {
		var multiaddrs []string
		for _, addr := range sources.CARIPFS.Multiaddrs {
			multiaddrs = append(multiaddrs, addr.String())
		}
		carIPFS = &pb.Sources_CARIPFS{
			Cid:        sources.CARIPFS.Cid.String(),
			Multiaddrs: multiaddrs,
		}
	}
	var carURL *pb.Sources_CARURL
	if sources.CARURL != nil {
		carURL = &pb.Sources_CARURL{
			Url: sources.CARURL.URL.String(),
		}
	}
	return &pb.Sources{
		CarUrl:  carURL,
		CarIpfs: carIPFS,
	}
}

func sourcesFromPb(pbs *pb.Sources) (sources auction.Sources, err error) {
	if pbs.CarUrl != nil {
		u, err := url.Parse(pbs.CarUrl.Url)
		if err != nil {
			return auction.Sources{}, fmt.Errorf("parsing %s: %s", pbs.CarUrl.Url, err)
		}
		sources.CARURL = &auction.CARURL{URL: *u}
	}

	if pbs.CarIpfs != nil {
		id, err := cid.Parse(pbs.CarIpfs.Cid)
		if err != nil {
			return auction.Sources{}, fmt.Errorf("parsing url %s: %s", pbs.CarIpfs.Cid, err)
		}
		var multiaddrs []multiaddr.Multiaddr
		for _, s := range pbs.CarIpfs.Multiaddrs {
			addr, err := multiaddr.NewMultiaddr(s)
			if err != nil {
				return auction.Sources{}, fmt.Errorf("parsing multiaddr %s: %s", s, err)
			}
			multiaddrs = append(multiaddrs, addr)
		}
		sources.CARIPFS = &auction.CARIPFS{Cid: id, Multiaddrs: multiaddrs}
	}
	return
}

func closedAuctionToPb(a broker.ClosedAuction) (*pb.AuctionClosed, error) {
	wb, err := auctionWinningBidsToPb(a.WinningBids)
	if err != nil {
		return nil, fmt.Errorf("converting winning bids to pb: %s", err)
	}
	status, err := auctionStatusToPb(a.Status)
	if err != nil {
		return nil, fmt.Errorf("converting status to pb: %s", err)
	}
	pba := &pb.AuctionClosed{
		OperationId:     uuid.New().String(),
		Id:              string(a.ID),
		BatchId:         string(a.BatchID),
		DealDuration:    a.DealDuration,
		DealReplication: a.DealReplication,
		DealVerified:    a.DealVerified,
		Status:          status,
		WinningBids:     wb,
		Error:           a.ErrorCause,
	}
	return pba, nil
}

func auctionStatusToPb(s broker.AuctionStatus) (pb.AuctionClosed_Status, error) {
	switch s {
	case broker.AuctionStatusUnspecified:
		return pb.AuctionClosed_STATUS_UNSPECIFIED, nil
	case broker.AuctionStatusQueued:
		return pb.AuctionClosed_STATUS_QUEUED, nil
	case broker.AuctionStatusStarted:
		return pb.AuctionClosed_STATUS_STARTED, nil
	case broker.AuctionStatusFinalized:
		return pb.AuctionClosed_STATUS_FINALIZED, nil
	default:
		return pb.AuctionClosed_STATUS_UNSPECIFIED, fmt.Errorf("unknown status %s", s)
	}
}

func auctionWinningBidsToPb(
	bids map[auction.BidID]broker.WinningBid,
) (map[string]*pb.AuctionClosed_WinningBid, error) {
	pbbids := make(map[string]*pb.AuctionClosed_WinningBid)
	for k, v := range bids {
		if v.StorageProviderID == "" {
			return nil, errors.New("storage-provider-id is empty")
		}
		if v.StartEpoch == 0 {
			return nil, errors.New("start epoch is zero")
		}
		pbbids[string(k)] = &pb.AuctionClosed_WinningBid{
			StorageProviderId: v.StorageProviderID,
			Price:             v.Price,
			StartEpoch:        v.StartEpoch,
			FastRetrieval:     v.FastRetrieval,
		}
	}
	return pbbids, nil
}

func closedAuctionFromPb(pba *pb.AuctionClosed) (broker.ClosedAuction, error) {
	wbids, err := auctionWinningBidsFromPb(pba.WinningBids)
	if err != nil {
		return broker.ClosedAuction{}, fmt.Errorf("decoding bids: %v", err)
	}

	if pba.Id == "" {
		return broker.ClosedAuction{}, errors.New("closed auction id is empty")
	}
	if pba.BatchId == "" {
		return broker.ClosedAuction{}, errors.New("batch id is empty")
	}
	if pba.DealDuration == 0 {
		return broker.ClosedAuction{}, errors.New("deal duration is zero")
	}
	if pba.DealReplication == 0 {
		return broker.ClosedAuction{}, errors.New("deal replication is zero")
	}
	a := broker.ClosedAuction{
		ID:              auction.ID(pba.Id),
		BatchID:         broker.BatchID(pba.BatchId),
		DealDuration:    pba.DealDuration,
		DealReplication: pba.DealReplication,
		DealVerified:    pba.DealVerified,
		Status:          auctionStatusFromPb(pba.Status),
		WinningBids:     wbids,
		ErrorCause:      pba.Error,
	}
	return a, nil
}

func auctionStatusFromPb(pbs pb.AuctionClosed_Status) broker.AuctionStatus {
	switch pbs {
	case pb.AuctionClosed_STATUS_UNSPECIFIED:
		return broker.AuctionStatusUnspecified
	case pb.AuctionClosed_STATUS_QUEUED:
		return broker.AuctionStatusQueued
	case pb.AuctionClosed_STATUS_STARTED:
		return broker.AuctionStatusStarted
	case pb.AuctionClosed_STATUS_FINALIZED:
		return broker.AuctionStatusFinalized
	default:
		return broker.AuctionStatusUnspecified
	}
}

func auctionWinningBidsFromPb(
	pbbids map[string]*pb.AuctionClosed_WinningBid,
) (map[auction.BidID]broker.WinningBid, error) {
	wbids := make(map[auction.BidID]broker.WinningBid)
	for k, v := range pbbids {
		if v.StorageProviderId == "" {
			return nil, errors.New("storage-provider-id is empty")
		}
		if v.Price < 0 {
			return nil, errors.New("price is negative")
		}
		if v.StartEpoch == 0 {
			return nil, errors.New("start-epoch is zero")
		}
		wbids[auction.BidID(k)] = broker.WinningBid{
			StorageProviderID: v.StorageProviderId,
			Price:             v.Price,
			StartEpoch:        v.StartEpoch,
			FastRetrieval:     v.FastRetrieval,
		}
	}
	return wbids, nil
}

func marshalAndPublish(ctx context.Context, mb MsgBroker, topic TopicName, m proto.Message) error {
	data, err := proto.Marshal(m)
	if err != nil {
		return fmt.Errorf("mashaling %s message: %s", topic, err)
	}
	if err := mb.PublishMsg(ctx, topic, data); err != nil {
		return fmt.Errorf("publishing %s message: %s", topic, err)
	}
	return nil
}
