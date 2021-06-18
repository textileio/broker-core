package client_test

import (
	"context"
	"crypto/rand"
	"net"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	util "github.com/ipfs/go-ipfs-util"
	format "github.com/ipfs/go-ipld-format"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	core "github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/auctioneerd/auctioneer"
	"github.com/textileio/broker-core/cmd/auctioneerd/client"
	"github.com/textileio/broker-core/cmd/auctioneerd/service"
	bidbotsrv "github.com/textileio/broker-core/cmd/bidbot/service"
	"github.com/textileio/broker-core/cmd/bidbot/service/datauri/apitest"
	bidstore "github.com/textileio/broker-core/cmd/bidbot/service/store"
	"github.com/textileio/broker-core/dshelper"
	"github.com/textileio/broker-core/finalizer"
	"github.com/textileio/broker-core/logging"
	"github.com/textileio/broker-core/marketpeer"
	brokermocks "github.com/textileio/broker-core/mocks/broker"
	auctioneermocks "github.com/textileio/broker-core/mocks/cmd/auctioneerd/auctioneer"
	golog "github.com/textileio/go-log/v2"
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
	c, dag := newClient(t, 1)
	gw := apitest.NewDataURIHTTPGateway(dag)
	t.Cleanup(gw.Close)

	_, dataURI, err := gw.CreateURI(true)
	require.NoError(t, err)

	id, err := c.ReadyToAuction(
		context.Background(),
		newDealID(),
		dataURI,
		oneGiB,
		sixMonthsEpochs,
		1,
		true,
		nil,
	)
	require.NoError(t, err)
	assert.NotEmpty(t, id)
}

func TestClient_GetAuction(t *testing.T) {
	c, dag := newClient(t, 1)
	gw := apitest.NewDataURIHTTPGateway(dag)
	t.Cleanup(gw.Close)

	_, dataURI, err := gw.CreateURI(true)
	require.NoError(t, err)

	id, err := c.ReadyToAuction(
		context.Background(),
		newDealID(),
		dataURI,
		oneGiB,
		sixMonthsEpochs,
		1,
		true,
		nil,
	)
	require.NoError(t, err)

	got, err := c.GetAuction(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
	assert.Equal(t, core.AuctionStatusStarted, got.Status)

	time.Sleep(time.Second * 15) // Allow to finish

	got, err = c.GetAuction(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, 1, int(got.Attempts))
	assert.Equal(t, core.AuctionStatusFinalized, got.Status) // no miners making bids
	assert.NotEmpty(t, got.ErrorCause)
}

func TestClient_RunAuction(t *testing.T) {
	c, dag := newClient(t, 2)
	bots := addBidbots(t, 10)
	gw := apitest.NewDataURIHTTPGateway(dag)
	t.Cleanup(gw.Close)

	time.Sleep(time.Second * 5) // Allow peers to boot

	_, dataURI, err := gw.CreateURI(true)
	require.NoError(t, err)

	id, err := c.ReadyToAuction(
		context.Background(),
		newDealID(),
		dataURI,
		oneGiB,
		sixMonthsEpochs,
		2,
		true,
		nil,
	)
	require.NoError(t, err)

	time.Sleep(time.Second * 15) // Allow to finish

	got, err := c.GetAuction(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
	assert.Equal(t, core.AuctionStatusFinalized, got.Status)
	assert.Empty(t, got.ErrorCause)
	require.Len(t, got.WinningBids, 2)

	for id, wb := range got.WinningBids {
		assert.NotNil(t, got.Bids[id])
		assert.True(t, wb.Acknowledged)

		// Set the proposal as accepted
		pcid := cid.NewCidV1(cid.Raw, util.Hash([]byte("howdy")))
		err = c.ProposalAccepted(context.Background(), got.ID, id, pcid)
		require.NoError(t, err)
	}

	time.Sleep(time.Second * 15) // Allow to finish

	// Check if the winning bids were able to fetch the data cid
	for _, wb := range got.WinningBids {
		bot := bots[wb.BidderID]
		require.NotNil(t, bot)

		bids, err := bot.ListBids(bidstore.Query{})
		require.NoError(t, err)
		require.Len(t, bids, 1)
		assert.Equal(t, bidstore.BidStatusFinalized, bids[0].Status)
		assert.Empty(t, bids[0].ErrorCause)
	}

	got, err = c.GetAuction(context.Background(), id)
	require.Error(t, err)
	assert.Contains(t, err.Error(), auctioneer.ErrAuctionNotFound.Error())
}

func newClient(t *testing.T, attempts uint32) (*client.Client, format.DAGService) {
	dir := t.TempDir()
	fin := finalizer.NewFinalizer()
	t.Cleanup(func() {
		require.NoError(t, fin.Cleanup(nil))
	})

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

func addBidbots(t *testing.T, n int) map[peer.ID]*bidbotsrv.Service {
	bots := make(map[peer.ID]*bidbotsrv.Service)
	fin := finalizer.NewFinalizer()
	t.Cleanup(func() {
		require.NoError(t, fin.Cleanup(nil))
	})
	for i := 0; i < n; i++ {
		dir := t.TempDir()

		priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
		require.NoError(t, err)

		store, err := dshelper.NewBadgerTxnDatastore(filepath.Join(dir, "bidstore"))
		require.NoError(t, err)
		fin.Add(store)

		config := bidbotsrv.Config{
			Peer: marketpeer.Config{
				PrivKey:    priv,
				RepoPath:   dir,
				EnableMDNS: true,
			},
			BidParams: bidbotsrv.BidParams{
				MinerAddr:             "foo",
				WalletAddrSig:         []byte("bar"),
				AskPrice:              100000000000,
				VerifiedAskPrice:      100000000000,
				FastRetrieval:         true,
				DealStartWindow:       oneDayEpochs,
				DealDataDirectory:     t.TempDir(),
				DealDataFetchAttempts: 3,
			},
			AuctionFilters: bidbotsrv.AuctionFilters{
				DealDuration: bidbotsrv.MinMaxFilter{
					Min: core.MinDealDuration,
					Max: core.MaxDealDuration,
				},
				DealSize: bidbotsrv.MinMaxFilter{
					Min: 56 * 1024,
					Max: 32 * 1000 * 1000 * 1000,
				},
			},
		}

		s, err := bidbotsrv.New(config, store, newFilClientMock())
		require.NoError(t, err)
		fin.Add(s)
		err = s.Subscribe(false)
		require.NoError(t, err)

		bots[s.ID()] = s
	}
	return bots
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
