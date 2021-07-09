package fakemsgbroker

import (
	"context"
	"fmt"
	"sync"

	mbroker "github.com/textileio/broker-core/msgbroker"
)

type FakeMsgBroker struct {
	lock          sync.Mutex
	topicMessages map[string][][]byte
}

func New() *FakeMsgBroker {
	return &FakeMsgBroker{
		topicMessages: map[string][][]byte{},
	}
}

func (b *FakeMsgBroker) RegisterTopicHandler(topicName mbroker.TopicName, handler mbroker.TopicHandler) error {
	panic("not implemented")
}
func (b *FakeMsgBroker) PublishMsg(ctx context.Context, topicName mbroker.TopicName, data []byte) error {
	b.lock.Lock()
	defer b.lock.Unlock()

	b.topicMessages[string(topicName)] = append(b.topicMessages[string(topicName)], data)

	return nil
}

// Helpers for tests

func (b *FakeMsgBroker) TotalPublished() int {
	b.lock.Lock()
	defer b.lock.Unlock()

	var count int
	for _, msgs := range b.topicMessages {
		count += len(msgs)
	}

	return count
}

func (b *FakeMsgBroker) TotalPublishedTopic(name string) int {
	b.lock.Lock()
	defer b.lock.Unlock()

	return len(b.topicMessages[name])
}

func (b *FakeMsgBroker) GetMsg(name string, idx int) ([]byte, error) {
	b.lock.Lock()
	defer b.lock.Unlock()

	topic := b.topicMessages[name]
	if idx >= len(topic) {
		return nil, fmt.Errorf("topic queue has length %d smaller than idx access %d", len(topic), idx)
	}

	return topic[idx], nil
}
