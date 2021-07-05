package gpubsub

import (
	"bytes"
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/require"
	logger "github.com/textileio/go-log/v2"
)

func init() {
	logger.SetAllLoggers(logger.LevelDebug)
}

// This test does some e2e testing of gpubsub library:
// 1. Registers a handler for subscription sub-1 and topic topic-1.
//    This handler receives a message, and publish another message in topic-2.
// 2. The test sends a message to topic-1.
// 3. The test checks that:
//    3.1. The message sent in 2. was received by handler registered in 1.
//    3.2. The message that handler 1. should have sent in topic-2 really was sent.
//
// All topics and subscriptions doesn't exist when the test runs, so the code is also testing the internal
// logic of gpubsub regarding creating topics and subscriptions that doesn't exist.
// These tests covers then:
// - Creating a topic.
// - Creating a subscription.
// - Sending a message to a topic.
// - Receiving a message from a topic.
// - Sending messages to a topic through a message handler (not entirely relevant, but covered).
func TestE2E(t *testing.T) {
	t.Parallel()
	var lock sync.Mutex // We use shared vars, so to be safe.

	var sentIDTopic1 MessageID
	sentDataTopic1 := []byte("duke-ftw")

	waitChan := make(chan struct{})
	var sentIDTopic2 MessageID
	sentDataTopic2 := []byte("duke-ftw-2")

	// 1. Launch dockerized pubsub emulator.
	launchPubsubEmulator(t)
	ps, err := New("test", "test", "test-")
	require.NoError(t, err)

	// 2. Register a handler for topic-1.
	ps.RegisterTopicHandler("sub-1", "topic-1", func(id MessageID, data []byte, ack AckMessageFunc, nack NackMessageFunc) {
		lock.Lock()
		defer lock.Unlock()
		require.Equal(t, sentIDTopic1, id)
		require.True(t, bytes.Equal(sentDataTopic1, data))

		ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
		defer cancel()
		sentIDTopic2, err = ps.PublishMsg(ctx, "topic-2", sentDataTopic2)
		require.NoError(t, err)
		require.NotEmpty(t, sentIDTopic2)

		ack()
	})

	ps.RegisterTopicHandler("sub-2", "topic-2", func(id MessageID, data []byte, ack AckMessageFunc, nack NackMessageFunc) {
		lock.Lock()
		defer lock.Unlock()

		require.Equal(t, sentIDTopic2, id)
		require.True(t, bytes.Equal(sentDataTopic2, data))

		ack()
		close(waitChan)
	})

	// 3. Send a message to topic-1
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	lock.Lock()
	sentIDTopic1, err = ps.PublishMsg(ctx, "topic-1", sentDataTopic1)
	lock.Unlock()
	require.NoError(t, err)
	require.NotEmpty(t, sentIDTopic1)

	// Wait for expected things to happen; 5s timeout max.
	select {
	case <-time.After(time.Second * 5):
		t.Fatalf("timed out waiting for handler call")
	case <-waitChan:
	}
}

func launchPubsubEmulator(t *testing.T) {
	pool, err := dockertest.NewPool("")
	require.NoError(t, err)

	container, err := pool.Run("textile/pubsub-emulator", "latest", []string{})
	require.NoError(t, err)

	err = container.Expire(180)
	require.NoError(t, err)

	time.Sleep(time.Second * 2)
	t.Cleanup(func() {
		err = pool.Purge(container)
		require.NoError(t, err)
	})

	pubsubHost := "127.0.0.1:" + container.GetPort("8085/tcp")
	err = os.Setenv("PUBSUB_EMULATOR_HOST", pubsubHost)
	require.NoError(t, err)
}
