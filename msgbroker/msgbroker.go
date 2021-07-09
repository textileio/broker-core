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
type TopicHandler func([]byte) error

// MsgBroker is a message-broker for async message communication.
type MsgBroker interface {
	// RegisterTopicHandler registers a handler to a topic, with a defined
	// subscription defined by the underlying implementation. Is highly recommended
	// to register handlers in a type-safe way using RegisterHandlers().
	RegisterTopicHandler(topic TopicName, handler TopicHandler) error

	// PublishMsg publishes a message to the desired topic.
	PublishMsg(ctx context.Context, topicName TopicName, data []byte) error
}

type TopicName string

const (
	NewBatchCreatedTopic  TopicName = "new-batch-created"
	NewBatchPreparedTopic           = "new-batch-prepared"
)

type NewBatchCreatedListener interface {
	OnNewBatchCreated(broker.StorageDealID, cid.Cid, []broker.BrokerRequestID) error
}

type NewBatchPreparedListener interface {
	OnNewBatchPrepared(broker.StorageDealID, broker.DataPreparationResult) error
}

func RegisterHandlers(mb MsgBroker, s interface{}) {
	if l, ok := s.(NewBatchCreatedListener); ok {
		mb.RegisterTopicHandler(NewBatchCreatedTopic, func(data []byte) error {
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

			if err := l.OnNewBatchCreated(sdID, batchCid, brids); err != nil {
				return fmt.Errorf("calling on-new-batch-created handler: %s", err)
			}
			return nil
		})
	}

	if l, ok := s.(NewBatchPreparedListener); ok {
		mb.RegisterTopicHandler(NewBatchPreparedTopic, func(data []byte) error {
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
			if err := l.OnNewBatchPrepared(id, pr); err != nil {
				return fmt.Errorf("calling on-new-batch-prepared handler: %s", err)
			}
			return nil
		})
	}
}
