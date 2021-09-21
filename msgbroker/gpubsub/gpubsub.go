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
	"go.opentelemetry.io/otel/metric"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

var (
	log = logger.Logger("gpubsub")
)

// PubsubMsgBroker is an implementation of MsgBroker for Google PubSub.
type PubsubMsgBroker struct {
	subsName    string
	topicPrefix string

	client          *pubsub.Client
	clientCtx       context.Context
	clientCtxCancel context.CancelFunc

	topicCacheLock sync.Mutex
	topicCache     map[string]*pubsub.Topic

	receivingHandlersWg sync.WaitGroup
	metrics             metricsCollector
}

var _ mbroker.MsgBroker = (*PubsubMsgBroker)(nil)

// New returns a new *PubsubMsgBroker.
func New(projectID, apiKey, topicPrefix, subsName string) (*PubsubMsgBroker, error) {
	_, usingEmulator := os.LookupEnv("PUBSUB_EMULATOR_HOST")
	if !usingEmulator && apiKey == "" {
		return nil, fmt.Errorf("api key is empty")
	}
	if !usingEmulator && projectID == "" {
		return nil, fmt.Errorf("project-id is empty")
	}
	if !usingEmulator && topicPrefix == "" {
		return nil, fmt.Errorf("topic-prefix is empty")
	}

	ctx, cancel := context.WithCancel(context.Background())
	client, err := pubsub.NewClient(ctx, projectID, option.WithCredentialsJSON([]byte(apiKey)))
	if err != nil {
		cancel()
		return nil, fmt.Errorf("creating pubsub client: %s", err)
	}

	return &PubsubMsgBroker{
		topicPrefix: topicPrefix + "-",
		subsName:    subsName,

		client:          client,
		clientCtx:       ctx,
		clientCtxCancel: cancel,

		topicCache: map[string]*pubsub.Topic{},

		metrics: noopMetricsCollector{},
	}, nil
}

// NewMetered is the same as New but exporting some metrics.
func NewMetered(projectID, apiKey, topicPrefix, subsName string, meter metric.MeterMust) (*PubsubMsgBroker, error) {
	p, err := New(projectID, apiKey, topicPrefix, subsName)
	if err == nil {
		p.initMetrics(meter)
	}
	return p, err
}

// RegisterTopicHandler registers a handler for a topic.
// - If the topic doesn't exist in PubSub, it's automatically created. The topic name is defined
//   as '<topic-prefix>-<topic-name>'.
// - If a subscription doesn't exist in PubSub it's automatically created. The subscription name is
//   defined as '<topic-prefix>-<topic-name>-<subs-name>' (e.g: "staging-new-batch-created-brokerd")
func (p *PubsubMsgBroker) RegisterTopicHandler(
	tname mbroker.TopicName,
	handler mbroker.TopicHandler,
	opts ...mbroker.Option) error {
	config, err := mbroker.ApplyRegisterHandlerOptions(opts...)
	if err != nil {
		return fmt.Errorf("applying options: %s", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	topicName := p.topicPrefix + string(tname)
	topic, err := p.getTopic(ctx, topicName)
	if err != nil {
		return fmt.Errorf("get topic: %s", err)
	}

	subName := topicName + "-" + p.subsName
	var sub *pubsub.Subscription
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
			Topic:                 topic,
			PushConfig:            pubsub.PushConfig{},
			AckDeadline:           config.AckDeadline,
			RetainAckedMessages:   false,
			RetentionDuration:     time.Hour * 24 * 7,
			ExpirationPolicy:      time.Duration(0),
			Labels:                nil,
			EnableMessageOrdering: false,
			DeadLetterPolicy:      nil,
			Filter:                "",
			RetryPolicy: &pubsub.RetryPolicy{
				MinimumBackoff: time.Second * 10,
				MaximumBackoff: time.Minute * 10,
			},
			Detached: false,
		}
		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		sub, err = p.client.CreateSubscription(ctx, subName, config)
		if err != nil {
			return fmt.Errorf("creating subscription: %s", err)
		}
		log.Warnf("subscription %s for topic %s created", subName, topicName)
	}

	sub.ReceiveSettings = pubsub.ReceiveSettings{
		MaxExtension:           -1,
		MaxExtensionPeriod:     0,
		MaxOutstandingMessages: 1000,
		MaxOutstandingBytes:    100e6, // 100MB
		NumGoroutines:          10,
	}
	p.receivingHandlersWg.Add(1)
	go func() {
		defer p.receivingHandlersWg.Done()
		err := sub.Receive(p.clientCtx, func(ctx context.Context, m *pubsub.Message) {
			start := time.Now()
			tctx, cancel := context.WithTimeout(ctx, config.AckDeadline)
			defer cancel()
			if err := handler(tctx, m.Data); err != nil {
				log.Errorf("handling %s message: %v", topicName, err)
				m.Nack()
				p.metrics.onHandle(ctx, topicName, time.Since(start), err)
				return
			}
			m.Ack()
			p.metrics.onHandle(ctx, topicName, time.Since(start), nil)
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

// PublishMsg publishes a payload to a topic.
// If the topic doesn't exist, it's created. The topic name is the same as described in
// RegisterTopicHandler method documentation.
func (p *PubsubMsgBroker) PublishMsg(ctx context.Context, tname mbroker.TopicName, data []byte) (err error) {
	defer func() { p.metrics.onPublish(ctx, string(tname), err) }()
	topicName := p.topicPrefix + string(tname)
	topic, err := p.getTopic(ctx, topicName)
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

func (p *PubsubMsgBroker) getTopic(ctx context.Context, name string) (*pubsub.Topic, error) {
	p.topicCacheLock.Lock()
	defer p.topicCacheLock.Unlock()
	topic, ok := p.topicCache[name]
	if ok {
		return topic, nil
	}

	topic = p.client.Topic(name)
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
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
	}
	p.topicCache[name] = topic

	return topic, nil
}

// Close closes the client.
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
