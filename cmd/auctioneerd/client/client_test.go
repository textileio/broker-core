package client_test

// @todo: on save file, run "goimports" to clean up imports

import (
	"context"
	"fmt"
	"testing"

	golog "github.com/ipfs/go-log/v2"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/require"
	"github.com/textileio/broker-core/cmd/authd/client"
	"github.com/textileio/broker-core/cmd/authd/service"
	"github.com/textileio/broker-core/logging"
	"github.com/textileio/broker-core/rpc"
)

func init() {
	if err := logging.SetLogLevels(map[string]golog.LogLevel{
		"auth/service": golog.LevelDebug,
	}); err != nil {
		panic(err)
	}
}

func TestClient_Create(t *testing.T) {
	c := newClient(t)

	res, err := c.Auth(context.Background())
	require.NoError(t, err)
	require.NotNil(t, res)
}

func newClient(t *testing.T) *client.Client {
	listenPort, err := freeport.GetFreePort()
	require.NoError(t, err)
	listenAddr := fmt.Sprintf("127.0.0.1:%d", listenPort)
	s, err := service.New(listenAddr)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, s.Close())
	})

	c, err := client.NewClient(listenAddr, rpc.GetClientOpts(listenAddr)...)
	require.NoError(t, err)

	t.Cleanup(func() {
		require.NoError(t, c.Close())
	})

	return c
}
