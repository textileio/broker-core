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
	"github.com/ipfs/go-cid"
	golog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/textileio/broker-core/broker"
	core "github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/auctioneerd/auctioneer"
	"github.com/textileio/broker-core/cmd/auctioneerd/cast"
	"github.com/textileio/broker-core/cmd/auctioneerd/client"
	"github.com/textileio/broker-core/cmd/auctioneerd/service"
	bidbotsrv "github.com/textileio/broker-core/cmd/bidbot/service"
	"github.com/textileio/broker-core/dshelper"
	"github.com/textileio/broker-core/finalizer"
	brokerpb "github.com/textileio/broker-core/gen/broker/v1"
	"github.com/textileio/broker-core/logging"
	"github.com/textileio/broker-core/marketpeer"
	mocks "github.com/textileio/broker-core/mocks/broker/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

const (
	bufConnSize = 1024 * 1024

	oneGiB    = 1024 * 1024 * 1024
	oneDay    = 60 * 24 * 2
	sixMonths = oneDay * 365 / 2
)

func init() {
	if err := logging.SetLogLevels(map[string]golog.LogLevel{
		"auctioneer":         golog.LevelDebug,
		"auctioneer/queue":   golog.LevelDebug,
		"auctioneer/service": golog.LevelDebug,
		"bidbot/service":     golog.LevelDebug,
		"mpeer":              golog.LevelDebug,
	}); err != nil {
		panic(err)
	}
}

func TestClient_ReadyToAuction(t *testing.T) {
	c := newClient(t)

	id, err := c.ReadyToAuction(context.Background(), newDealID(), oneGiB, sixMonths)
	require.NoError(t, err)
	assert.NotEmpty(t, id)
}

func TestClient_GetAuction(t *testing.T) {
	c := newClient(t)

	id, err := c.ReadyToAuction(context.Background(), newDealID(), oneGiB, sixMonths)
	require.NoError(t, err)

	got, err := c.GetAuction(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
	assert.Equal(t, core.AuctionStatusStarted, got.Status)

	time.Sleep(time.Second * 15) // Allow to finish

	got, err = c.GetAuction(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, core.AuctionStatusError, got.Status) // no miners making bids
}

func TestClient_RunAuction(t *testing.T) {
	c := newClient(t)
	addMiners(t, 10)

	time.Sleep(time.Second * 5) // Allow peers to boot

	id, err := c.ReadyToAuction(context.Background(), newDealID(), oneGiB, sixMonths)
	require.NoError(t, err)

	time.Sleep(time.Second * 15) // Allow to finish

	got, err := c.GetAuction(context.Background(), id)
	require.NoError(t, err)
	assert.Equal(t, id, got.ID)
	assert.Equal(t, core.AuctionStatusEnded, got.Status)
	assert.NotEmpty(t, got.WinningBids)
	if len(got.WinningBids) != 0 {
		assert.NotNil(t, got.Bids[got.WinningBids[0]])
	}
}

func newClient(t *testing.T) *client.Client {
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
		},
	}

	store, err := dshelper.NewBadgerTxnDatastore(filepath.Join(dir, "auctionq"))
	require.NoError(t, err)
	fin.Add(store)

	bm := &brokerMock{client: &mocks.APIServiceClient{}}
	bm.client.On(
		"StorageDealAuctioned",
		mock.Anything,
		mock.AnythingOfType("*broker.StorageDealAuctionedRequest"),
	).Return(&brokerpb.StorageDealAuctionedResponse{}, nil)

	s, err := service.New(config, store, bm, newChainMock())
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
	return client.New(conn)
}

func addMiners(t *testing.T, n int) {
	for i := 0; i < n; i++ {
		dir := t.TempDir()

		priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
		require.NoError(t, err)

		config := bidbotsrv.Config{
			RepoPath: dir,
			Peer: marketpeer.Config{
				PrivKey:    priv,
				RepoPath:   dir,
				EnableMDNS: true,
			},
			BidParams: bidbotsrv.BidParams{
				WalletAddr:       "foo",
				WalletAddrSig:    []byte("bar"),
				AskPrice:         100000000000,
				VerifiedAskPrice: 100000000000,
				FastRetrieval:    true,
				DealStartWindow:  oneDay,
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

		s, err := bidbotsrv.New(config, newChainMock())
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

type brokerMock struct {
	client *mocks.APIServiceClient
}

func (bm *brokerMock) CreateStorageDeal(context.Context, cid.Cid, []core.BrokerRequestID) (core.StorageDealID, error) {
	panic("shouldn't be called")
}

func (bm *brokerMock) StorageDealPrepared(context.Context, core.StorageDealID, core.DataPreparationResult) error {
	panic("shouldn't be called")
}

func (bm *brokerMock) StorageDealFinalizedDeals(context.Context, []broker.FinalizedAuctionDeal) error {
	panic("shouldn't be called")
}

func (bm *brokerMock) StorageDealAuctioned(ctx context.Context, auction core.Auction) error {
	_, err := bm.client.StorageDealAuctioned(ctx, &brokerpb.StorageDealAuctionedRequest{
		Auction: cast.AuctionToPb(auction),
	})
	return err
}

func (bm *brokerMock) Create(context.Context, cid.Cid, core.Metadata) (core.BrokerRequest, error) {
	panic("shouldn't be called")
}

func (bm *brokerMock) Get(context.Context, core.BrokerRequestID) (core.BrokerRequest, error) {
	panic("shouldn't be called")
}

func newChainMock() *chainMock {
	cm := &chainMock{}
	cm.On(
		"VerifyBidder",
		mock.Anything,
		mock.Anything,
		mock.Anything,
	).Return(true, nil)
	cm.On("GetChainHeight").Return(uint64(0), nil)
	cm.On("Close").Return(nil)
	return cm
}

type chainMock struct {
	mock.Mock
}

func (cm *chainMock) Close() error {
	args := cm.Called()
	return args.Error(0)
}

func (cm *chainMock) VerifyBidder(walletAddr string, bidderSig []byte, bidderID peer.ID) (bool, error) {
	args := cm.Called(walletAddr, bidderSig, bidderID)
	return args.Bool(0), args.Error(1)
}

func (cm *chainMock) GetChainHeight() (uint64, error) {
	args := cm.Called()
	return args.Get(0).(uint64), args.Error(1)
}
