package client_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	golog "github.com/ipfs/go-log/v2"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/textileio/broker-core/cmd/auctioneerd/client"
	"github.com/textileio/broker-core/cmd/auctioneerd/service"
	minersrv "github.com/textileio/broker-core/cmd/minerd/service"
	pb "github.com/textileio/broker-core/gen/broker/auctioneer/v1"
	"github.com/textileio/broker-core/logging"
	"github.com/textileio/broker-core/marketpeer"
	"github.com/textileio/broker-core/rpc"
)

func init() {
	if err := logging.SetLogLevels(map[string]golog.LogLevel{
		"auctioneer":         golog.LevelDebug,
		"auctioneer/queue":   golog.LevelDebug,
		"auctioneer/service": golog.LevelDebug,
		"miner/service":      golog.LevelDebug,
		"mpeer":              golog.LevelDebug,
	}); err != nil {
		panic(err)
	}
}

func TestClient_CreateAuction(t *testing.T) {
	c := newClient(t)

	res, err := c.CreateAuction(context.Background())
	require.NoError(t, err)
	assert.NotEmpty(t, res.Id)
}

func TestClient_GetAuction(t *testing.T) {
	c := newClient(t)

	res, err := c.CreateAuction(context.Background())
	require.NoError(t, err)

	got, err := c.GetAuction(context.Background(), res.Id)
	require.NoError(t, err)
	assert.Equal(t, res.Id, got.Auction.Id)
	assert.Equal(t, pb.Auction_STATUS_STARTED, got.Auction.Status)

	time.Sleep(time.Second * 10) // Allow to finish

	got, err = c.GetAuction(context.Background(), res.Id)
	require.NoError(t, err)
	assert.Equal(t, pb.Auction_STATUS_ERROR, got.Auction.Status) // no miners making bids
}

func TestClient_RunAuction(t *testing.T) {
	c := newClient(t)
	addMiners(t, 10)

	time.Sleep(time.Second) // Allow peers to boot

	res, err := c.CreateAuction(context.Background())
	require.NoError(t, err)

	time.Sleep(time.Second * 11) // Allow to finish

	got, err := c.GetAuction(context.Background(), res.Id)
	require.NoError(t, err)
	assert.Equal(t, res.Id, got.Auction.Id)
	assert.Equal(t, pb.Auction_STATUS_ENDED, got.Auction.Status)
	assert.Len(t, got.Auction.Bids, 10)
	assert.NotEmpty(t, got.Auction.Winner)
	assert.NotNil(t, got.Auction.Bids[got.Auction.Winner])
}

func newClient(t *testing.T) *client.Client {
	dir, err := ioutil.TempDir("", "")
	require.NoError(t, err)

	listenPort, err := freeport.GetFreePort()
	require.NoError(t, err)
	listenAddr := fmt.Sprintf("127.0.0.1:%d", listenPort)

	config := service.Config{
		RepoPath:   dir,
		ListenAddr: listenAddr,
		Peer: marketpeer.Config{
			RepoPath: dir,
		},
	}
	s, err := service.New(config)
	require.NoError(t, err)
	err = s.EnableMDNS(1)
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

func addMiners(t *testing.T, n int) {
	for i := 0; i < n; i++ {
		dir, err := ioutil.TempDir("", "")
		require.NoError(t, err)

		config := minersrv.Config{
			RepoPath: dir,
			Peer: marketpeer.Config{
				RepoPath: dir,
			},
		}
		s, err := minersrv.New(config)
		require.NoError(t, err)
		err = s.EnableMDNS(1)
		require.NoError(t, err)

		t.Cleanup(func() {
			require.NoError(t, s.Close())
		})
	}
}
