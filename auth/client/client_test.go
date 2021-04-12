package client_test

// @todo: on save file, run "goimports" to clean up imports

import (
	"context"
	"fmt"
	"testing"

	"github.com/phayes/freeport"
	"github.com/stretchr/testify/require"
	"github.com/textileio/broker-core/auth/client"
	"github.com/textileio/broker-core/auth/common"
)

func TestClient_Create(t *testing.T) {
	c := newClient(t)

	res, err := c.Auth(context.Background())
	require.NoError(t, err)
	require.NotNil(t, res)
}

func newService(t *testing.T) (listenAddr string) {
	// @todo
	// err := tutil.SetLogLevels(map[string]logging.LogLevel{
	// 	"auth":      logging.LevelDebug,
	// })
	// require.NoError(t, err)

	listenPort, err := freeport.GetFreePort()
	require.NoError(t, err)
	listenAddr = fmt.Sprintf("127.0.0.1:%d", listenPort)
	server, err := common.GetServerAndProxy(listenAddr, "127.0.0.1:0")
	require.NoError(t, err)

	t.Cleanup(func() {
		server.Stop()
	})

	return listenAddr
}

func newClient(t *testing.T) *client.Client {
	listenAddr := newService(t)
	c, err := client.NewClient(listenAddr, common.GetClientRPCOpts(listenAddr)...)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, c.Close())
	})
	return c
}
