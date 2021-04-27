package client_test

import (
	"context"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/google/uuid"
	golog "github.com/ipfs/go-log/v2"
	"github.com/phayes/freeport"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/auctioneerd/auctioneer"
	"github.com/textileio/broker-core/cmd/auctioneerd/client"
	"github.com/textileio/broker-core/cmd/auctioneerd/service"
	minersrv "github.com/textileio/broker-core/cmd/minerd/service"
	pb "github.com/textileio/broker-core/gen/broker/auctioneer/v1"
	"github.com/textileio/broker-core/logging"
	"github.com/textileio/broker-core/marketpeer"
	"github.com/textileio/broker-core/rpc"
)

const (
	oneGiB    = 1024 * 1024 * 1024
	twoMonths = 60 * 24 * 2 * 60
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

	res, err := c.CreateAuction(context.Background(), newDealID(), oneGiB, twoMonths)
	require.NoError(t, err)
	assert.NotEmpty(t, res.Id)
}

func TestClient_GetAuction(t *testing.T) {
	c := newClient(t)

	res, err := c.CreateAuction(context.Background(), newDealID(), oneGiB, twoMonths)
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

	time.Sleep(time.Second * 5) // Allow peers to boot

	res, err := c.CreateAuction(context.Background(), newDealID(), oneGiB, twoMonths)
	require.NoError(t, err)

	time.Sleep(time.Second * 15) // Allow to finish

	got, err := c.GetAuction(context.Background(), res.Id)
	require.NoError(t, err)
	assert.Equal(t, res.Id, got.Auction.Id)
	assert.Equal(t, pb.Auction_STATUS_ENDED, got.Auction.Status)
	assert.NotEmpty(t, got.Auction.WinningBid)
	assert.NotNil(t, got.Auction.Bids[got.Auction.WinningBid])
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
		// TODO: Add mocked broker client.
		Peer: marketpeer.Config{
			RepoPath: dir,
		},
		Auction: auctioneer.AuctionConfig{
			Duration: time.Second * 10,
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
		dir := t.TempDir()

		config := minersrv.Config{
			RepoPath: dir,
			Peer: marketpeer.Config{
				RepoPath: dir,
			},
			BidParams: minersrv.BidParams{
				AskPrice: 100,
			},
			AuctionFilters: minersrv.AuctionFilters{
				DealDuration: minersrv.MinMaxFilter{
					Min: broker.MinDealEpochs,
					Max: broker.MaxDealEpochs,
				},
				DealSize: minersrv.MinMaxFilter{
					Min: 56 * 1024,
					Max: 32 * 1000 * 1000 * 1000,
				},
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

func newDealID() broker.StorageDealID {
	return broker.StorageDealID(uuid.New().String())
}
