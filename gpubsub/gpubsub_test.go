package gpubsub

import (
	"bytes"
	"context"
	"os"
	"testing"
	"time"

	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/require"
	logger "github.com/textileio/go-log/v2"
)

func init() {
	logger.SetAllLoggers(logger.LevelDebug)
}

func TestE2E(t *testing.T) {
	t.Parallel()

	// 1. Launch dockerized pubsub emulator.
	launchPubsubEmulator(t)
	ps, err := New("test", "test", "test-")
	require.NoError(t, err)

	// 2. Register a handler for topic-1.
	var receivedID MessageID
	var receivedData []byte
	waitChan := make(chan struct{})
	ps.RegisterTopicHandler("sub-1", "topic-1", func(id MessageID, data []byte, ack AckMessageFunc, nack NackMessageFunc) {
		receivedID = id
		receivedData = data
		ack()
		close(waitChan)
	})

	// 3. Send a message to topic-1
	sentData := []byte("duke-ftw")
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	sentID, err := ps.PublishMsg(ctx, "topic-1", sentData)
	require.NoError(t, err)
	require.NotEmpty(t, sentID)

	select {
	case <-time.After(time.Second * 5):
		t.Fatalf("timed out waiting for handler call")
	case <-waitChan:
	}
	require.Equal(t, sentID, receivedID)
	require.True(t, bytes.Equal(sentData, receivedData))

	// 4. Make topic-1 handler send another message to topic-2.
	// 5. See if topic-2 message was received.

	// Run all again to see if can reuse topics.
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
