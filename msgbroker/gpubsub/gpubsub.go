package gpubsub

import (
	"context"
	"fmt"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
	"github.com/labstack/gommon/log"
	"github.com/textileio/broker-core/msgbroker"
	"google.golang.org/api/iterator"
)

type PubsubMsgBroker struct {
	topicPrefix string

	client          *pubsub.Client
	clientCtx       context.Context
	clientCtxCancel context.CancelFunc

	topicCacheLock sync.Mutex
	topicCache     map[string]*pubsub.Topic
}

func New(projectID, apiKey, topicPrefix string) (*PubsubMsgBroker, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("api key is empty")
	}
	if projectID == "" {
		return nil, fmt.Errorf("project-id is empty")
	}
	if topicPrefix == "" {
		return nil, fmt.Errorf("topic-prefix is empty")
	}

	ctx, cancel := context.WithCancel(context.Background())
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("creating pubsub client: %s", err)
	}

	return &PubsubMsgBroker{
		client:          client,
		clientCtx:       ctx,
		clientCtxCancel: cancel,

		topicCache: map[string]*pubsub.Topic{},
	}, nil
}

// TODO(jsign): move to base package and create subfolders
func (p *PubsubMsgBroker) RegisterTopicHandler(subscriptionName, topicName string, handler msgbroker.TopicHandler) error {
	topic, err := p.getTopic(topicName)
	// TODO(jsign): tune topic.PublishSettings?

	var sub *pubsub.Subscription
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	it := topic.Subscriptions(ctx)
	for {
		subi, err := it.Next()
		if err == iterator.Done {
			break
		}
		if err != nil {
			return fmt.Errorf("looking for subscription: %s", err)
		}
		if subi.ID() == subscriptionName {
			sub = subi
			break
		}
	}
	if sub == nil {
		log.Warnf("creating subscription %s for topic %s", subscriptionName, topicName)

		config := pubsub.SubscriptionConfig{
			Topic: topic,
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		sub, err = p.client.CreateSubscription(ctx, topicName, config)
		if err != nil {
			return fmt.Errorf("creating subscription: %s", err)
		}
	}

	// TODO(jsign): tune ReceiveSettings
	// TODO(jsign) handle receive 100% failure etc.
	go func() {
		err := sub.Receive(p.clientCtx, func(ctx context.Context, m *pubsub.Message) {
			handler(msgbroker.MessageID(m.ID), m.Data, m.Ack, m.Nack)
		})
		if err != nil {
			log.Errorf("receive handler subscription %s, topic %s: %s", subscriptionName, topicName, err)
		}
	}()

	log.Debugf("registered handler for %s:%s", subscriptionName, topicName)
	return nil
}

func (p *PubsubMsgBroker) PublishMsg(ctx context.Context, topicName string, data []byte) (msgbroker.MessageID, error) {
	topic, err := p.getTopic(topicName)
	if err != nil {
		return "", fmt.Errorf("get topic: %s", err)
	}
	msg := pubsub.Message{
		Data: data,
	}
	pr := topic.Publish(ctx, &msg)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	id, err := pr.Get(ctx)
	if err != nil {
		return "", fmt.Errorf("publishing to pubsub: %s", err)
	}

	return msgbroker.MessageID(id), nil
}

func (p *PubsubMsgBroker) getTopic(name string) (*pubsub.Topic, error) {
	p.topicCacheLock.Lock()
	defer p.topicCacheLock.Unlock()
	topic, ok := p.topicCache[name]
	if ok {
		return topic, nil
	}

	topic = p.client.Topic(name)
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	exist, err := topic.Exists(ctx)
	if err != nil {
		return nil, fmt.Errorf("check topic exists: %s", err)
	}
	if !exist {
		log.Warnf("creating topic %s", name)

		topic, err = p.client.CreateTopic(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("creating topic %s: %s", name, err)
		}
		// TODO(jsign): set publishsettings
	}
	p.topicCache[name] = topic

	// TODO(jsign): when topic.Stop()?
	return topic, nil
}

func (p *PubsubMsgBroker) Close() {
	p.clientCtxCancel()
	// TODO(jsign): wait for things.
}
