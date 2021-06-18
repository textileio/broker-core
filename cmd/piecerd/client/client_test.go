package client_test

import (
	"bytes"
	"context"
	"io"
	"math/rand"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/ipfs/go-cid"
	ipfsfiles "github.com/ipfs/go-ipfs-files"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	"github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/multiformats/go-multiaddr"
	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/require"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/piecerd/client"
	"github.com/textileio/broker-core/cmd/piecerd/service"
	"github.com/textileio/broker-core/finalizer"
	"github.com/textileio/broker-core/tests"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

func TestClient_ReadyToPrepare(t *testing.T) {
	t.Parallel()
	c, bm, ipfs := newClient(t)

	dataCid := addRandomData(t, ipfs)
	sdID := broker.StorageDealID("SD1")
	err := c.ReadyToPrepare(context.Background(), sdID, dataCid)
	require.NoError(t, err)

	time.Sleep(time.Second * 3)

	bm.lock.Lock()
	require.NotEmpty(t, bm.sdpSDID)
	require.True(t, bm.sdpDPR.PieceCid.Defined())
	require.Greater(t, bm.sdpDPR.PieceSize, uint64(0))
	require.Equal(t, 1, bm.numCalls)
	bm.lock.Unlock()
}

func newClient(t *testing.T) (*client.Client, *brokerMock, *httpapi.HttpApi) {
	fin := finalizer.NewFinalizer()
	t.Cleanup(func() {
		require.NoError(t, fin.Cleanup(nil))
	})

	listener := bufconn.Listen(1024 * 1024)

	ipfsDocker := launchIPFSContainer(t)
	ipfsAPIMultiaddr := "/ip4/127.0.0.1/tcp/" + ipfsDocker.GetPort("5001/tcp")
	ma, err := multiaddr.NewMultiaddr(ipfsAPIMultiaddr)
	require.NoError(t, err)
	ipfs, err := httpapi.NewApi(ma)
	require.NoError(t, err)
	bm := &brokerMock{}

	config := service.Config{
		Listener:        listener,
		IpfsMultiaddrs:  []multiaddr.Multiaddr{ma},
		Broker:          bm,
		Datastore:       tests.NewTxMapDatastore(),
		DaemonFrequency: time.Millisecond * 200,
		RetryDelay:      time.Second,
	}
	s, err := service.New(config)
	require.NoError(t, err)
	fin.Add(s)

	dialer := func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}
	conn, err := grpc.Dial("bufnet", grpc.WithContextDialer(dialer), grpc.WithInsecure())
	require.NoError(t, err)
	fin.Add(conn)

	return client.NewClient(conn), bm, ipfs
}

func addRandomData(t *testing.T, ipfs *httpapi.HttpApi) cid.Cid {
	t.Helper()
	r := rand.New(rand.NewSource(22))

	data := make([]byte, 10000)
	_, err := io.ReadFull(r, data)
	require.NoError(t, err)

	node, err := ipfs.Unixfs().Add(
		context.Background(),
		ipfsfiles.NewReaderFile(bytes.NewReader(data)),
		options.Unixfs.CidVersion(1),
		options.Unixfs.Pin(true))
	require.NoError(t, err)

	return node.Cid()
}

type brokerMock struct {
	lock     sync.Mutex
	numCalls int
	sdpSDID  broker.StorageDealID
	sdpDPR   broker.DataPreparationResult
}

func (bm *brokerMock) CreateStorageDeal(
	ctx context.Context,
	batchCid cid.Cid,
	srids []broker.BrokerRequestID) (broker.StorageDealID, error) {
	panic("shouldn't be called")
}

func (bm *brokerMock) StorageDealPrepared(
	ctx context.Context,
	id broker.StorageDealID,
	dpr broker.DataPreparationResult) error {
	bm.lock.Lock()
	defer bm.lock.Unlock()
	bm.numCalls++
	bm.sdpSDID = id
	bm.sdpDPR = dpr
	return nil
}

func (bm *brokerMock) StorageDealAuctioned(context.Context, broker.Auction) error {
	panic("shouldn't be called")
}

func (bm *brokerMock) Create(context.Context, cid.Cid) (broker.BrokerRequest, error) {
	panic("shouldn't be called")
}

func (bm *brokerMock) CreatePrepared(context.Context, cid.Cid, broker.PreparedCAR) (broker.BrokerRequest, error) {
	panic("shouldn't be called")
}

func (bm *brokerMock) StorageDealFinalizedDeal(context.Context, broker.FinalizedAuctionDeal) error {
	panic("shouldn't be called")
}

func (bm *brokerMock) Get(context.Context, broker.BrokerRequestID) (broker.BrokerRequest, error) {
	panic("shouldn't be called")
}

func (bm *brokerMock) StorageDealProposalAccepted(context.Context, broker.StorageDealID, string, cid.Cid) error {
	panic("shouldn't be called")
}

func launchIPFSContainer(t *testing.T) *dockertest.Resource {
	pool, err := dockertest.NewPool("")
	require.NoError(t, err)

	ipfsDocker, err := pool.Run("ipfs/go-ipfs", "v0.8.0", []string{"IPFS_PROFILE=test"})
	require.NoError(t, err)

	err = ipfsDocker.Expire(180)
	require.NoError(t, err)

	time.Sleep(time.Second * 3)
	t.Cleanup(func() {
		err = pool.Purge(ipfsDocker)
		require.NoError(t, err)
	})

	return ipfsDocker
}
