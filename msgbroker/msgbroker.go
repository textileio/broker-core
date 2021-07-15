package msgbroker

import (
	"context"
	"errors"
	"fmt"

	"github.com/ipfs/go-cid"
	"github.com/textileio/broker-core/broker"
	pbBroker "github.com/textileio/broker-core/gen/broker/v1"
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
)

// NewBatchCreatedListener is a handler for NewBatchCreated topic.
type NewBatchCreatedListener interface {
	OnNewBatchCreated(context.Context, broker.StorageDealID, cid.Cid, []broker.BrokerRequestID) error
}

// NewBatchPreparedListener is a handler for NewBatchPrepared topic.
type NewBatchPreparedListener interface {
	OnNewBatchPrepared(context.Context, broker.StorageDealID, broker.DataPreparationResult) error
}

// ReadyToBatchListener is a handler for ReadyToBatch topic.
type ReadyToBatchListener interface {
	OnReadyToBatch(context.Context, []ReadyToBatchData) error
}

// ReadyToBatchData contains broker request data information to be batched.
type ReadyToBatchData struct {
	BrokerRequestID broker.BrokerRequestID
	DataCid         cid.Cid
}

// RegisterHandlers automatically calls mb.RegisterTopicHandler in the methods that
// s might satisfy on known XXXListener interfaces. This allows to automatically wire
// s to receive messages from topics of implemented handlers.
func RegisterHandlers(mb MsgBroker, s interface{}, opts ...Option) error {
	var countRegistered int
	if l, ok := s.(NewBatchCreatedListener); ok {
		countRegistered++
		err := mb.RegisterTopicHandler(NewBatchCreatedTopic, func(ctx context.Context, data []byte) error {
			r := &pbBroker.NewBatchCreated{}
			if err := proto.Unmarshal(data, r); err != nil {
				return fmt.Errorf("unmarshal new batch created: %s", err)
			}
			if r.Id == "" {
				return errors.New("storage deal id is empty")
			}
			sdID := broker.StorageDealID(r.Id)

			batchCid, err := cid.Cast(r.BatchCid)
			if err != nil {
				return fmt.Errorf("decoding batch cid: %s", err)
			}
			if !batchCid.Defined() {
				return errors.New("data cid is undefined")
			}
			brids := make([]broker.BrokerRequestID, len(r.BrokerRequestIds))
			for i, id := range r.BrokerRequestIds {
				if id == "" {
					return fmt.Errorf("broker request id can't be empty")
				}
				brids[i] = broker.BrokerRequestID(id)
			}

			if err := l.OnNewBatchCreated(ctx, sdID, batchCid, brids); err != nil {
				return fmt.Errorf("calling on-new-batch-created handler: %s", err)
			}
			return nil
		}, opts...)
		if err != nil {
			return fmt.Errorf("registering handler for new-batch-created topic")
		}
	}

	if l, ok := s.(NewBatchPreparedListener); ok {
		countRegistered++
		err := mb.RegisterTopicHandler(NewBatchPreparedTopic, func(ctx context.Context, data []byte) error {
			r := &pbBroker.NewBatchPrepared{}
			if err := proto.Unmarshal(data, r); err != nil {
				return fmt.Errorf("unmarshal new batch prepared: %s", err)
			}
			pieceCid, err := cid.Cast(r.PieceCid)
			if err != nil {
				return fmt.Errorf("decoding piece cid: %s", err)
			}
			id := broker.StorageDealID(r.Id)
			pr := broker.DataPreparationResult{
				PieceCid:  pieceCid,
				PieceSize: r.PieceSize,
			}
			if err := l.OnNewBatchPrepared(ctx, id, pr); err != nil {
				return fmt.Errorf("calling on-new-batch-prepared handler: %s", err)
			}
			return nil
		}, opts...)
		if err != nil {
			return fmt.Errorf("registering handler for new-batch-prepared topic")
		}
	}

	if l, ok := s.(ReadyToBatchListener); ok {
		countRegistered++
		err := mb.RegisterTopicHandler(ReadyToBatchTopic, func(ctx context.Context, data []byte) error {
			r := &pbBroker.ReadyToBatch{}
			if err := proto.Unmarshal(data, r); err != nil {
				return fmt.Errorf("unmarshal ready to batch: %s", err)
			}
			if len(r.DataCids) == 0 {
				return errors.New("data cids list can't be empty")
			}

			rtb := make([]ReadyToBatchData, len(r.DataCids))
			for i := range r.DataCids {
				if r.DataCids[i].BrokerRequestId == "" {
					return fmt.Errorf("broker request id is empty")
				}
				brID := broker.BrokerRequestID(r.DataCids[i].BrokerRequestId)
				dataCid, err := cid.Cast(r.DataCids[i].DataCid)
				if err != nil {
					return fmt.Errorf("decoding data cid: %s", err)
				}
				rtb[i] = ReadyToBatchData{
					BrokerRequestID: brID,
					DataCid:         dataCid,
				}
			}

			if err := l.OnReadyToBatch(ctx, rtb); err != nil {
				return fmt.Errorf("calling ready-to-batch handler: %s", err)
			}
			return nil
		}, opts...)
		if err != nil {
			return fmt.Errorf("registering handler for new-batch-prepared topic")
		}
	}

	if countRegistered == 0 {
		return errors.New("no handlers were registered")
	}

	return nil
}

func PublishMsgReadyToBatch(ctx context.Context, mb MsgBroker, dataCids []ReadyToBatchData) error {
	if len(dataCids) == 0 {
		return errors.New("data cids is empty")
	}
	msg := &pbBroker.ReadyToBatch{
		DataCids: make([]*pbBroker.ReadyToBatch_ReadyToBatchBR, len(dataCids)),
	}

	for i := range dataCids {
		msg.DataCids[i] = &pbBroker.ReadyToBatch_ReadyToBatchBR{
			BrokerRequestId: string(dataCids[i].BrokerRequestID),
			DataCid:         dataCids[i].DataCid.Bytes(),
		}
	}
	data, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshaling ready-to-batch message: %s", err)
	}
	if err := mb.PublishMsg(ctx, ReadyToBatchTopic, data); err != nil {
		return fmt.Errorf("publishing ready-to-batch message: %s", err)
	}

	return nil
}

func PublishMsgNewBatchCreated(ctx context.Context, mb MsgBroker, batchID string, batchCid cid.Cid, brIDs []broker.BrokerRequestID) error {
	brids := make([]string, len(brIDs))
	for i, bbr := range brIDs {
		brids[i] = string(bbr)
	}
	msg := &pbBroker.NewBatchCreated{
		Id:               batchID,
		BatchCid:         batchCid.Bytes(),
		BrokerRequestIds: brids,
	}
	data, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("marshaling new-batch-created message: %s", err)
	}
	if err := mb.PublishMsg(ctx, NewBatchCreatedTopic, data); err != nil {
		return fmt.Errorf("publishing new-batch-created message: %s", err)
	}

	return nil
}

func PublishMsgNewBatchPrepared(ctx context.Context, mb MsgBroker, sdID broker.StorageDealID, pieceCid cid.Cid, pieceSize uint64) error {
	msg := &pbBroker.NewBatchPrepared{
		Id:        string(sdID),
		PieceCid:  pieceCid.Bytes(),
		PieceSize: pieceSize,
	}
	data, err := proto.Marshal(msg)
	if err != nil {
		return fmt.Errorf("signaling broker that storage deal is prepared: %s", err)
	}
	if err := mb.PublishMsg(ctx, NewBatchPreparedTopic, data); err != nil {
		return fmt.Errorf("publishing new-prepared-batch message: %s", err)
	}

	return nil
}
