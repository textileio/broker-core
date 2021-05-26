package pubsub

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	util "github.com/ipfs/go-ipfs-util"
	golog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
	"github.com/textileio/broker-core/broker"
)

var (
	log = golog.Logger("mpeer/pubsub")

	// ErrAckNotReceived indicates a published message was not acknowledged by any recipients.
	ErrAckNotReceived = errors.New("message was not acknowledged")
)

// Handler is used to receive topic peer events and messages.
type Handler func(from peer.ID, topic string, msg []byte)

// Topic provides a nice interface to a libp2p pubsub topic.
type Topic struct {
	ps             *pubsub.PubSub
	host           peer.ID
	eventHandler   Handler
	messageHandler Handler

	acks      map[cid.Cid]chan struct{}
	acksTopic *Topic

	t *pubsub.Topic
	h *pubsub.TopicEventHandler
	s *pubsub.Subscription

	ctx    context.Context
	cancel context.CancelFunc

	lk sync.Mutex
}

// NewTopic returns a new topic for the host.
func NewTopic(ctx context.Context, ps *pubsub.PubSub, host peer.ID, topic string, subscribe bool) (*Topic, error) {
	t, err := newTopic(ctx, ps, host, topic, subscribe)
	if err != nil {
		return nil, fmt.Errorf("creating topic: %v", err)
	}
	t.acksTopic, err = newTopic(ctx, ps, host, broker.AcksTopic(topic, host), true)
	if err != nil {
		return nil, fmt.Errorf("creating ack topic: %v", err)
	}
	t.acksTopic.eventHandler = t.ackEventHandler
	t.acksTopic.messageHandler = t.ackMessageHandler
	t.acks = make(map[cid.Cid]chan struct{})
	return t, nil
}

func newTopic(ctx context.Context, ps *pubsub.PubSub, host peer.ID, topic string, subscribe bool) (*Topic, error) {
	top, err := ps.Join(topic)
	if err != nil {
		return nil, fmt.Errorf("joining topic: %v", err)
	}

	handler, err := top.EventHandler()
	if err != nil {
		return nil, fmt.Errorf("getting topic handler: %v", err)
	}

	var sub *pubsub.Subscription
	if subscribe {
		sub, err = top.Subscribe()
		if err != nil {
			return nil, fmt.Errorf("subscribing to topic: %v", err)
		}
	}

	t := &Topic{
		ps:   ps,
		host: host,
		t:    top,
		h:    handler,
		s:    sub,
	}
	t.ctx, t.cancel = context.WithCancel(ctx)

	go t.watch()
	if t.s != nil {
		go t.listen()
	}

	return t, nil
}

// Close the topic.
func (t *Topic) Close() error {
	t.lk.Lock()
	defer t.lk.Unlock()
	t.h.Cancel()
	if t.s != nil {
		t.s.Cancel()
	}
	if err := t.t.Close(); err != nil {
		return err
	}
	t.cancel()
	if t.acksTopic != nil {
		return t.acksTopic.Close()
	}
	return nil
}

// SetEventHandler sets a handler func that will be called with peer events.
func (t *Topic) SetEventHandler(handler Handler) {
	t.lk.Lock()
	defer t.lk.Unlock()
	t.eventHandler = handler
}

// SetMessageHandler sets a handler func that will be called with topic messages.
// A subscription is required for the handler to be called.
func (t *Topic) SetMessageHandler(handler Handler) {
	t.lk.Lock()
	defer t.lk.Unlock()
	t.messageHandler = handler
}

// Publish data.
// ackTimeout indicates how long Publish will wait for a receiver ack.
// A zero or negative ackTimeout will cause Publish to not wait for an ack.
// A positive ackTimeout will cause Publish to block until a single ack is received or ackTimout is reached.
func (t *Topic) Publish(ctx context.Context, data []byte, ackTimeout time.Duration, opts ...pubsub.PubOpt) error {
	var ackCh chan struct{}
	if ackTimeout > 0 && t.acksTopic != nil {
		msgID := cid.NewCidV1(cid.Raw, util.Hash(data))
		ackCh = make(chan struct{})
		t.lk.Lock()
		t.acks[msgID] = ackCh
		t.lk.Unlock()

		defer func() {
			t.lk.Lock()
			delete(t.acks, msgID)
			t.lk.Unlock()
		}()
	}

	if err := t.t.Publish(ctx, data, opts...); err != nil {
		return fmt.Errorf("publishing to main topic: %v", err)
	}

	if ackCh != nil {
		timer := time.NewTimer(ackTimeout)
		select {
		case <-timer.C:
			return ErrAckNotReceived
		case <-ackCh:
			timer.Stop()
		}
	}
	return nil
}

func (t *Topic) watch() {
	for {
		e, err := t.h.NextPeerEvent(t.ctx)
		if err != nil {
			break
		}
		var msg string
		switch e.Type {
		case pubsub.PeerJoin:
			msg = "JOINED"
		case pubsub.PeerLeave:
			msg = "LEFT"
		default:
			continue
		}
		t.lk.Lock()
		if t.eventHandler != nil {
			t.eventHandler(e.Peer, t.t.String(), []byte(msg))
		}
		t.lk.Unlock()
	}
}

func (t *Topic) listen() {
	for {
		msg, err := t.s.Next(t.ctx)
		if err != nil {
			break
		}
		if msg.ReceivedFrom.String() == t.host.String() {
			continue
		}
		t.lk.Lock()

		if t.messageHandler != nil {
			if !strings.Contains(t.t.String(), "/ack") {
				// This is a normal message; respond with ACK
				go func() {
					msgID := cid.NewCidV1(cid.Raw, util.Hash(msg.Data))
					t.publishAck(msg.ReceivedFrom, msgID)
				}()
			}
			t.messageHandler(msg.ReceivedFrom, t.t.String(), msg.Data)
		}
		t.lk.Unlock()
	}
}

func (t *Topic) publishAck(from peer.ID, msgID cid.Cid) {
	acks, err := newTopic(t.ctx, t.ps, t.host, broker.AcksTopic(t.t.String(), from), false)
	if err != nil {
		log.Errorf("creating ack topic: %v", err)
		return
	}
	defer func() { _ = acks.Close() }()
	acks.SetEventHandler(t.ackEventHandler)

	if err := acks.t.Publish(t.ctx, msgID.Bytes()); err != nil {
		log.Errorf("publishing ack: %v", err)
	}
}

func (t *Topic) ackEventHandler(from peer.ID, topic string, msg []byte) {
	log.Debugf("%s ack peer event: %s %s", topic, from, msg)
}

func (t *Topic) ackMessageHandler(from peer.ID, topic string, msg []byte) {
	msgID, err := cid.Cast(msg)
	if err != nil {
		log.Errorf("parsing cid in ack handler: %v", err)
		return
	}
	log.Debugf("%s ack from %s: %s", topic, from, msgID)
	t.lk.Lock()
	ch := t.acks[msgID]
	t.lk.Unlock()
	if ch != nil {
		select {
		case ch <- struct{}{}:
		default:
		}
	}
}
