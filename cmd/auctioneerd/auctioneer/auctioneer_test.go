package auctioneer_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	util "github.com/ipfs/go-ipfs-util"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/textileio/bidbot/lib/auction"
	core "github.com/textileio/bidbot/lib/auction"
	"github.com/textileio/bidbot/lib/datauri/apitest"
	"github.com/textileio/bidbot/lib/dshelper"
	"github.com/textileio/bidbot/lib/logging"
	filclientmocks "github.com/textileio/bidbot/mocks/lib/filclient"
	lotusclientmocks "github.com/textileio/bidbot/mocks/service/lotusclient"
	bidbotsrv "github.com/textileio/bidbot/service"
	"github.com/textileio/bidbot/service/limiter"
	bidstore "github.com/textileio/bidbot/service/store"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/auctioneerd/auctioneer"
	"github.com/textileio/broker-core/cmd/auctioneerd/service"
	"github.com/textileio/broker-core/msgbroker/fakemsgbroker"
	"github.com/textileio/go-libp2p-pubsub-rpc/finalizer"
	rpcpeer "github.com/textileio/go-libp2p-pubsub-rpc/peer"
	golog "github.com/textileio/go-log/v2"
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
		"psrpc":              golog.LevelDebug,
		"psrpc/peer":         golog.LevelDebug,
	}); err != nil {
		panic(err)
	}
}

func TestClient_ReadyToAuction(t *testing.T) {
	s := newClient(t)
	gw := apitest.NewDataURIHTTPGateway(s.DAGService())
	t.Cleanup(gw.Close)

	payloadCid, sources, err := gw.CreateHTTPSources(true)
	require.NoError(t, err)

	err = s.OnReadyToAuction(
		context.Background(),
		"ID1",
		newBatchID(),
		payloadCid,
		oneGiB,
		sixMonthsEpochs,
		1,
		true,
		nil,
		0,
		sources,
	)
	require.NoError(t, err)
}

func TestClient_GetAuction(t *testing.T) {
	s := newClient(t)
	gw := apitest.NewDataURIHTTPGateway(s.DAGService())
	t.Cleanup(gw.Close)

	payloadCid, sources, err := gw.CreateHTTPSources(true)
	require.NoError(t, err)

	id := auction.ID("ID1")
	err = s.OnReadyToAuction(
		context.Background(),
		id,
		newBatchID(),
		payloadCid,
		oneGiB,
		sixMonthsEpochs,
		1,
		true,
		nil,
		0,
		sources,
	)
	require.NoError(t, err)

	got, err := s.GetAuction(id)
	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
	assert.Equal(t, broker.AuctionStatusStarted, got.Status)

	time.Sleep(time.Second * 15) // Allow to finish

	got, err = s.GetAuction(id)
	require.NoError(t, err)
	assert.Equal(t, broker.AuctionStatusFinalized, got.Status) // no miners making bids
	assert.NotEmpty(t, got.ErrorCause)
}

func TestClient_RunAuction(t *testing.T) {
	s := newClient(t)
	bots := addBidbots(t, 10)
	gw := apitest.NewDataURIHTTPGateway(s.DAGService())
	t.Cleanup(gw.Close)

	time.Sleep(time.Second * 5) // Allow peers to boot

	payloadCid, sources, err := gw.CreateHTTPSources(true)
	require.NoError(t, err)

	id := auction.ID("ID1")
	err = s.OnReadyToAuction(
		context.Background(),
		id,
		newBatchID(),
		payloadCid,
		oneGiB,
		sixMonthsEpochs,
		2,
		true,
		nil,
		0,
		sources,
	)
	require.NoError(t, err)

	time.Sleep(time.Second * 15) // Allow to finish

	got, err := s.GetAuction(id)
	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
	assert.Equal(t, broker.AuctionStatusFinalized, got.Status)
	assert.Empty(t, got.ErrorCause)
	assert.Len(t, got.Bids, 10)
	require.Len(t, got.WinningBids, 2)

	for id := range got.WinningBids {
		assert.NotNil(t, got.Bids[id])

		// Set the proposal as accepted
		pcid := cid.NewCidV1(cid.Raw, util.Hash([]byte("howdy")))
		err = s.OnDealProposalAccepted(context.Background(), got.ID, id, pcid)
		require.NoError(t, err)
	}

	time.Sleep(time.Second * 15) // Allow to finish

	// Re-get auction so we can check proposal cids
	got, err = s.GetAuction(id)
	require.NoError(t, err)

	for _, wb := range got.WinningBids {
		// Check if proposal cid was delivered without error
		assert.True(t, wb.ProposalCid.Defined())
		assert.Empty(t, wb.ErrorCause)

		// Check if the winning bids were able to fetch the data cid
		bot := bots[wb.BidderID]
		require.NotNil(t, bot)

		bids, err := bot.ListBids(bidstore.Query{})
		require.NoError(t, err)
		require.Len(t, bids, 1)
		assert.Equal(t, bidstore.BidStatusFinalized, bids[0].Status)
		assert.Empty(t, bids[0].ErrorCause)
	}
}

func newClient(t *testing.T) *service.Service {
	dir := t.TempDir()
	fin := finalizer.NewFinalizer()
	t.Cleanup(func() {
		require.NoError(t, fin.Cleanup(nil))
	})

	listener := bufconn.Listen(bufConnSize)
	fin.Add(listener)
	config := service.Config{
		Peer: rpcpeer.Config{
			RepoPath:   dir,
			EnableMDNS: true,
		},
		Auction: auctioneer.AuctionConfig{
			Duration: time.Second * 10,
		},
	}

	store, err := dshelper.NewBadgerTxnDatastore(filepath.Join(dir, "auctionq"))
	require.NoError(t, err)
	fin.Add(store)

	bm := fakemsgbroker.New()
	s, err := service.New(config, store, bm, newFilClientMock())
	require.NoError(t, err)
	fin.Add(s)
	err = s.Start(false)
	require.NoError(t, err)

	return s
}

func addBidbots(t *testing.T, n int) map[peer.ID]*bidbotsrv.Service {
	bots := make(map[peer.ID]*bidbotsrv.Service)
	fin := finalizer.NewFinalizer()
	t.Cleanup(func() {
		require.NoError(t, fin.Cleanup(nil))
	})
	for i := 0; i < n; i++ {
		dir := t.TempDir()

		store, err := dshelper.NewBadgerTxnDatastore(filepath.Join(dir, "bidstore"))
		require.NoError(t, err)
		fin.Add(store)

		config := bidbotsrv.Config{
			Peer: rpcpeer.Config{
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
			BytesLimiter: limiter.NopeLimiter{},
		}

		s, err := bidbotsrv.New(config, store, newLotusClientMock(), newFilClientMock())
		require.NoError(t, err)
		fin.Add(s)
		err = s.Subscribe(false)
		require.NoError(t, err)

		pi, err := s.PeerInfo()
		require.NoError(t, err)
		bots[pi.ID] = s
	}
	return bots
}

func newBatchID() broker.BatchID {
	return broker.BatchID(uuid.New().String())
}

func newLotusClientMock() *lotusclientmocks.LotusClient {
	lc := &lotusclientmocks.LotusClient{}
	lc.On("HealthCheck").Return(nil)
	lc.On(
		"ImportData",
		mock.Anything,
		mock.Anything,
	).Return(nil)
	lc.On("Close").Return(nil)
	return lc
}

func newFilClientMock() *filclientmocks.FilClient {
	fc := &filclientmocks.FilClient{}
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
