package pubsub

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/ipfs/go-cid"
	util "github.com/ipfs/go-ipfs-util"
	cbor "github.com/ipfs/go-ipld-cbor"
	golog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/peer"
	pubsub "github.com/libp2p/go-libp2p-pubsub"
)

var (
	log = golog.Logger("mpeer/pubsub")

	// ErrResponseNotReceived indicates a response was not received after publishing a message.
	ErrResponseNotReceived = errors.New("response not received")
)

// EventHandler is used to receive topic peer events.
type EventHandler func(from peer.ID, topic string, msg []byte)

// MessageHandler is used to receive topic messages.
type MessageHandler func(from peer.ID, topic string, msg []byte) ([]byte, error)

// Response wraps a message response.
type Response struct {
	ID   string // The cid of the received message
	Data []byte
	Err  error
}

func init() {
	cbor.RegisterCborType(Response{})
}

func responseTopic(base string, pid peer.ID) string {
	return path.Join(base, pid.String(), "_response")
}

// Topic provides a nice interface to a libp2p pubsub topic.
type Topic struct {
	ps             *pubsub.PubSub
	host           peer.ID
	eventHandler   EventHandler
	messageHandler MessageHandler

	resChs   map[cid.Cid]chan Response
	resTopic *Topic

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
	t.resTopic, err = newTopic(ctx, ps, host, responseTopic(topic, host), true)
	if err != nil {
		return nil, fmt.Errorf("creating response topic: %v", err)
	}
	t.resTopic.eventHandler = t.resEventHandler
	t.resTopic.messageHandler = t.resMessageHandler
	t.resChs = make(map[cid.Cid]chan Response)
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
	if t.resTopic != nil {
		return t.resTopic.Close()
	}
	return nil
}

// SetEventHandler sets a handler func that will be called with peer events.
func (t *Topic) SetEventHandler(handler EventHandler) {
	t.lk.Lock()
	defer t.lk.Unlock()
	t.eventHandler = handler
}

// SetMessageHandler sets a handler func that will be called with topic messages.
// A subscription is required for the handler to be called.
func (t *Topic) SetMessageHandler(handler MessageHandler) {
	t.lk.Lock()
	defer t.lk.Unlock()
	t.messageHandler = handler
}

// Publish data.
// resTimeout indicates how long Publish will wait for a response.
// A zero or negative resTimeout will cause Publish to not wait for a response.
// A positive resTimeout will cause Publish to block until a single response is received or resTimout is reached.
func (t *Topic) Publish(
	ctx context.Context,
	data []byte,
	resTimeout time.Duration,
	opts ...pubsub.PubOpt,
) ([]byte, error) {
	var resCh chan Response
	if resTimeout > 0 && t.resTopic != nil {
		msgID := cid.NewCidV1(cid.Raw, util.Hash(data))
		resCh = make(chan Response)
		t.lk.Lock()
		t.resChs[msgID] = resCh
		t.lk.Unlock()

		defer func() {
			t.lk.Lock()
			delete(t.resChs, msgID)
			t.lk.Unlock()
		}()
	}

	if err := t.t.Publish(ctx, data, opts...); err != nil {
		return nil, fmt.Errorf("publishing to main topic: %v", err)
	}

	if resCh != nil {
		timer := time.NewTimer(resTimeout)
		select {
		case <-timer.C:
			return nil, ErrResponseNotReceived
		case res := <-resCh:
			timer.Stop()
			return res.Data, res.Err
		}
	}
	return nil, nil
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
			data, err := t.messageHandler(msg.ReceivedFrom, t.t.String(), msg.Data)
			if !strings.Contains(t.t.String(), "/_response") {
				// This is a normal message; respond with data and error
				go func() {
					msgID := cid.NewCidV1(cid.Raw, util.Hash(msg.Data))
					t.publishResponse(msg.ReceivedFrom, msgID, data, err)
				}()
			}
		}
		t.lk.Unlock()
	}
}

func (t *Topic) publishResponse(from peer.ID, id cid.Cid, data []byte, e error) {
	topic, err := newTopic(t.ctx, t.ps, t.host, responseTopic(t.t.String(), from), false)
	if err != nil {
		log.Errorf("creating response topic: %v", err)
		return
	}
	defer func() { _ = topic.Close() }()
	topic.SetEventHandler(t.resEventHandler)

	res := Response{
		ID:   id.String(),
		Data: data,
		Err:  e,
	}
	msg, err := cbor.DumpObject(&res)
	if err != nil {
		log.Errorf("encoding response: %v", err)
		return
	}

	if err := topic.t.Publish(t.ctx, msg); err != nil {
		log.Errorf("publishing response: %v", err)
	}
}

func (t *Topic) resEventHandler(from peer.ID, topic string, msg []byte) {
	log.Debugf("%s response peer event: %s %s", topic, from, msg)
}

func (t *Topic) resMessageHandler(from peer.ID, topic string, msg []byte) ([]byte, error) {
	var res Response
	if err := cbor.DecodeInto(msg, &res); err != nil {
		return nil, fmt.Errorf("decoding response: %v", err)
	}
	id, err := cid.Decode(res.ID)
	if err != nil {
		return nil, fmt.Errorf("decoding response id: %v", err)
	}

	log.Debugf("%s response from %s: %s", topic, from, res.ID)
	t.lk.Lock()
	ch := t.resChs[id]
	t.lk.Unlock()
	if ch != nil {
		select {
		case ch <- res:
		default:
		}
	}
	return nil, nil // no response to a response
}
