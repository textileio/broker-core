package fakemsgbroker

import (
	"context"
	"fmt"
	"sync"

	mbroker "github.com/textileio/broker-core/msgbroker"
)

// FakeMsgBroker is an in-memory implementation of a msgbroker. It's only useful for tests.
type FakeMsgBroker struct {
	lock          sync.Mutex
	topicMessages map[string][][]byte
}

// New returns a new FakeMsgBroker.
func New() *FakeMsgBroker {
	return &FakeMsgBroker{
		topicMessages: map[string][][]byte{},
	}
}

// RegisterTopicHandler registers a handler for a topic.
func (b *FakeMsgBroker) RegisterTopicHandler(
	topicName mbroker.TopicName,
	handler mbroker.TopicHandler,
	opts ...mbroker.Option) error {
	panic("not implemented")
}

// PublishMsg publishes a payload to a topic.
func (b *FakeMsgBroker) PublishMsg(ctx context.Context, topicName mbroker.TopicName, data []byte) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.topicMessages[string(topicName)] = append(b.topicMessages[string(topicName)], data)

	return nil
}

// Helpers for tests

// TotalPublished returns the amount of all published messages.
func (b *FakeMsgBroker) TotalPublished() int {
	b.lock.Lock()
	defer b.lock.Unlock()

	var count int
	for _, msgs := range b.topicMessages {
		count += len(msgs)
	}

	return count
}

// TotalPublishedTopic returns the total amount of published messages in a topic.
func (b *FakeMsgBroker) TotalPublishedTopic(name string) int {
	b.lock.Lock()
	defer b.lock.Unlock()

	return len(b.topicMessages[name])
}

// GetMsg returns the idx-th (0-based index) message present in a particular topic.
func (b *FakeMsgBroker) GetMsg(name string, idx int) ([]byte, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	topic := b.topicMessages[name]
	if idx >= len(topic) {
		return nil, fmt.Errorf("topic queue has length %d smaller than idx access %d", len(topic), idx)
	}

	return topic[idx], nil
}
