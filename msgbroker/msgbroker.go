package msgbroker

import "context"

// AckMessageFunc is a function that ACK a received message.
type AckMessageFunc func()

// NackMessageFunc is a function that NACKs a received message.
type NackMessageFunc func()

// TopicHandler is function that processes a received message. The function
// is responsible for calling Ack() or Nack() functions to properly signal
// message processing.
type TopicHandler func([]byte, AckMessageFunc, NackMessageFunc)

// MsgBroker is a message-broker for async message communication.
type MsgBroker interface {
	RegisterTopicHandler(subscriptionName, topicName string, handler TopicHandler) error
	PublishMsg(ctx context.Context, topicName string, data []byte) error
}
