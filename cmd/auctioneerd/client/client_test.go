package client_test

import (
	"context"
	"crypto/rand"
	"io/ioutil"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	cbor "github.com/ipfs/go-ipld-cbor"
	format "github.com/ipfs/go-ipld-format"
	golog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/multiformats/go-multihash"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	core "github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/auctioneerd/auctioneer"
	"github.com/textileio/broker-core/cmd/auctioneerd/client"
	"github.com/textileio/broker-core/cmd/auctioneerd/service"
	bidbotsrv "github.com/textileio/broker-core/cmd/bidbot/service"
	"github.com/textileio/broker-core/dshelper"
	"github.com/textileio/broker-core/finalizer"
	"github.com/textileio/broker-core/logging"
	"github.com/textileio/broker-core/marketpeer"
	brokermocks "github.com/textileio/broker-core/mocks/broker"
	auctioneermocks "github.com/textileio/broker-core/mocks/cmd/auctioneerd/auctioneer"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const (
	bufConnSize = 1024 * 1024

	oneGiB          = 1024 * 1024 * 1024
	oneDayEpochs    = 60 * 24 * 2
	sixMonthsEpochs = oneDayEpochs * 365 / 2
)

func init() {
	if err := logging.SetLogLevels(map[string]golog.LogLevel{
		"auctioneer":         golog.LevelDebug,
		"auctioneer/queue":   golog.LevelDebug,
		"auctioneer/service": golog.LevelDebug,
		"bidbot/service":     golog.LevelDebug,
		"bidbot/store":       golog.LevelDebug,
		"mpeer":              golog.LevelDebug,
		"mpeer/pubsub":       golog.LevelDebug,
	}); err != nil {
		panic(err)
	}
}

func TestClient_ReadyToAuction(t *testing.T) {
	c, _ := newClient(t, 1)

	id, err := c.ReadyToAuction(context.Background(), newDealID(), oneGiB, sixMonthsEpochs, 1, true)
	require.NoError(t, err)
	assert.NotEmpty(t, id)
}

func TestClient_GetAuction(t *testing.T) {
	c, _ := newClient(t, 1)

	id, err := c.ReadyToAuction(context.Background(), newDealID(), oneGiB, sixMonthsEpochs, 1, true)
	require.NoError(t, err)

	got, err := c.GetAuction(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
	assert.Equal(t, core.AuctionStatusStarted, got.Status)

	time.Sleep(time.Second * 15) // Allow to finish

	got, err = c.GetAuction(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, 1, int(got.Attempts))
	assert.Equal(t, core.AuctionStatusError, got.Status) // no miners making bids
}

func TestClient_RunAuction(t *testing.T) {
	c, dag := newClient(t, 2)
	addMiners(t, 10)

	time.Sleep(time.Second * 5) // Allow peers to boot

	id, err := c.ReadyToAuction(context.Background(), newDealID(), oneGiB, sixMonthsEpochs, 2, true)
	require.NoError(t, err)

	time.Sleep(time.Second * 15) // Allow to finish

	got, err := c.GetAuction(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
	assert.Equal(t, core.AuctionStatusEnded, got.Status)
	require.Len(t, got.WinningBids, 2)

	pnode, err := cbor.WrapObject([]byte("lets make a deal"), multihash.SHA2_256, -1)
	require.NoError(t, err)
	err = dag.Add(context.Background(), pnode)
	require.NoError(t, err)

	for id, wb := range got.WinningBids {
		assert.NotNil(t, got.Bids[id])
		assert.True(t, wb.Acknowledged)

		// Set the proposal as accepted
		err = c.ProposalAccepted(context.Background(), got.ID, id, pnode.Cid())
		require.NoError(t, err)
	}

	time.Sleep(time.Second * 15) // Allow to finish

	got, err = c.GetAuction(context.Background(), id)
	require.Error(t, err)
	assert.Contains(t, err.Error(), auctioneer.ErrAuctionNotFound.Error())
}

func newClient(t *testing.T, attempts uint32) (*client.Client, format.DAGService) {
	fin := finalizer.NewFinalizer()
	t.Cleanup(func() {
		require.NoError(t, fin.Cleanup(nil))
	})

	dir, err := ioutil.TempDir("", "")
	require.NoError(t, err)

	listener := bufconn.Listen(bufConnSize)
	fin.Add(listener)
	config := service.Config{
		Listener: listener,
		Peer: marketpeer.Config{
			RepoPath:   dir,
			EnableMDNS: true,
		},
		Auction: auctioneer.AuctionConfig{
			Duration: time.Second * 10,
			Attempts: attempts,
		},
	}

	store, err := dshelper.NewBadgerTxnDatastore(filepath.Join(dir, "auctionq"))
	require.NoError(t, err)
	fin.Add(store)

	bm := &brokermocks.Broker{}
	bm.On(
		"StorageDealAuctioned",
		mock.Anything,
		mock.AnythingOfType("broker.Auction"),
	).Return(nil)

	s, err := service.New(config, store, bm, newFilClientMock())
	require.NoError(t, err)
	fin.Add(s)
	err = s.Start(false)
	require.NoError(t, err)

	dialer := func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}
	conn, err := grpc.Dial("bufnet", grpc.WithContextDialer(dialer), grpc.WithInsecure())
	require.NoError(t, err)
	fin.Add(conn)
	return client.New(conn), s.DAGService()
}

func addMiners(t *testing.T, n int) {
	for i := 0; i < n; i++ {
		dir := t.TempDir()

		priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
		require.NoError(t, err)

		store, err := dshelper.NewBadgerTxnDatastore(filepath.Join(dir, "bidstore"))
		require.NoError(t, err)
		t.Cleanup(func() { _ = store.Close() })

		config := bidbotsrv.Config{
			Peer: marketpeer.Config{
				PrivKey:    priv,
				RepoPath:   dir,
				EnableMDNS: true,
			},
			BidParams: bidbotsrv.BidParams{
				WalletAddrSig:            []byte("bar"),
				AskPrice:                 100000000000,
				VerifiedAskPrice:         100000000000,
				FastRetrieval:            true,
				DealStartWindow:          oneDayEpochs,
				ProposalCidFetchAttempts: 3,
			},
			AuctionFilters: bidbotsrv.AuctionFilters{
				DealDuration: bidbotsrv.MinMaxFilter{
					Min: core.MinDealEpochs,
					Max: core.MaxDealEpochs,
				},
				DealSize: bidbotsrv.MinMaxFilter{
					Min: 56 * 1024,
					Max: 32 * 1000 * 1000 * 1000,
				},
			},
		}

		s, err := bidbotsrv.New(config, store, newFilClientMock())
		require.NoError(t, err)
		err = s.Subscribe(false)
		require.NoError(t, err)

		t.Cleanup(func() {
			require.NoError(t, s.Close())
		})
	}
}

func newDealID() core.StorageDealID {
	return core.StorageDealID(uuid.New().String())
}

func newFilClientMock() *auctioneermocks.FilClient {
	fc := &auctioneermocks.FilClient{}
	fc.On(
		"VerifyBidder",
		mock.Anything,
		mock.Anything,
		mock.Anything,
	).Return(true, nil)
	fc.On("GetChainHeight").Return(uint64(0), nil)
	fc.On("Close").Return(nil)
	return fc
}
