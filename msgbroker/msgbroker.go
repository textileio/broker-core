package msgbroker

import "context"

// TopicHandler is function that processes a received message.
// If no error is returned, the message will be automatically acked.
// If an error is returned, the message will be automatically nacked.
type TopicHandler func([]byte) error

// MsgBroker is a message-broker for async message communication.
type MsgBroker interface {
	RegisterTopicHandler(subscriptionName, topicName string, handler TopicHandler) error
	PublishMsg(ctx context.Context, topicName string, data []byte) error
}
