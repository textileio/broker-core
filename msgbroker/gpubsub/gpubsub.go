package gpubsub

import (
	"context"
	"fmt"
	"os"
	"sync"
	"time"

	"cloud.google.com/go/pubsub"
	mbroker "github.com/textileio/broker-core/msgbroker"
	logger "github.com/textileio/go-log/v2"
	"google.golang.org/api/iterator"
)

var (
	log = logger.Logger("gpubsub")
)

type PubsubMsgBroker struct {
	subsPrefix  string
	topicPrefix string

	client          *pubsub.Client
	clientCtx       context.Context
	clientCtxCancel context.CancelFunc

	topicCacheLock sync.Mutex
	topicCache     map[string]*pubsub.Topic

	receivingHandlersWg sync.WaitGroup
}

var _ mbroker.MsgBroker = (*PubsubMsgBroker)(nil)

func New(projectID, apiKey, topicPrefix, subsPrefix string) (*PubsubMsgBroker, error) {
	_, usingEmulator := os.LookupEnv("PUBSUB_EMULATOR_HOST")
	if !usingEmulator && apiKey == "" {
		return nil, fmt.Errorf("api key is empty")
	}
	if !usingEmulator && projectID == "" {
		return nil, fmt.Errorf("project-id is empty")
	}

	ctx, cancel := context.WithCancel(context.Background())
	client, err := pubsub.NewClient(ctx, projectID)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("creating pubsub client: %s", err)
	}

	return &PubsubMsgBroker{
		topicPrefix: topicPrefix,
		subsPrefix:  subsPrefix,

		client:          client,
		clientCtx:       ctx,
		clientCtxCancel: cancel,

		topicCache: map[string]*pubsub.Topic{},
	}, nil
}

func (p *PubsubMsgBroker) RegisterTopicHandler(topicName mbroker.TopicName, handler mbroker.TopicHandler) error {
	topic, err := p.getTopic(topicName)
	if err != nil {
		return fmt.Errorf("get topic: %s", err)
	}

	subName := p.subsPrefix + "-" + string(topicName)
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
		if subi.ID() == subName {
			sub = subi
			break
		}
	}
	if sub == nil {
		log.Warnf("creating subscription %s for topic %s", subName, topicName)

		config := pubsub.SubscriptionConfig{
			Topic: topic,
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		sub, err = p.client.CreateSubscription(ctx, subName, config)
		if err != nil {
			return fmt.Errorf("creating subscription: %s", err)
		}
		log.Warnf("subscription %s for topic %s created", subName, topicName)
	}

	// TODO(jsign): tune ReceiveSettings
	p.receivingHandlersWg.Add(1)
	go func() {
		defer p.receivingHandlersWg.Done()
		err := sub.Receive(p.clientCtx, func(ctx context.Context, m *pubsub.Message) {
			if err := handler(m.Data); err != nil {
				log.Error(err)
				m.Nack()
				return
			}
			m.Ack()
		})
		if err != nil {
			log.Errorf("receive handler subscription %s, topic %s: %s", subName, topicName, err)
			return
		}
		log.Infof("handler for subscription %s, topic %s closed gracefully", subName, topicName)
	}()

	log.Debugf("registered handler for %s:%s", subName, topicName)
	return nil
}

func (p *PubsubMsgBroker) PublishMsg(ctx context.Context, topicName mbroker.TopicName, data []byte) error {
	topic, err := p.getTopic(topicName)
	if err != nil {
		return fmt.Errorf("get topic: %s", err)
	}
	msg := pubsub.Message{
		Data: data,
	}
	pr := topic.Publish(ctx, &msg)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	if _, err := pr.Get(ctx); err != nil {
		return fmt.Errorf("publishing to pubsub: %s", err)
	}

	return nil
}

func (p *PubsubMsgBroker) getTopic(tname mbroker.TopicName) (*pubsub.Topic, error) {
	name := p.topicPrefix + string(tname)
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

		// TODO(jsign): switch to CreatTopicWithConfig() with explicit config.
		topic, err = p.client.CreateTopic(ctx, name)
		if err != nil {
			return nil, fmt.Errorf("creating topic %s: %s", name, err)
		}
	}
	p.topicCache[name] = topic

	return topic, nil
}

func (p *PubsubMsgBroker) Close() error {
	log.Infof("closing pubsub msg broker...")

	p.clientCtxCancel()

	log.Infof("closing topics...")
	p.topicCacheLock.Lock()
	defer p.topicCacheLock.Unlock()
	for _, topic := range p.topicCache {
		topic.Stop()
	}
	log.Infof("topics closed")

	log.Infof("waiting for receiving handlers to close...")
	p.receivingHandlersWg.Wait()
	log.Infof("all receiving handlers to closed")

	log.Infof("pubsub msg broker closed")

	return nil
}
