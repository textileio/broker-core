package client_test

import (
	"bytes"
	"context"
	"math/rand"
	"net"
	"testing"
	"time"

	"github.com/ipfs/go-cid"
	ipfsfiles "github.com/ipfs/go-ipfs-files"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	"github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/multiformats/go-multiaddr"
	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/piecerd/client"
	"github.com/textileio/broker-core/cmd/piecerd/service"
	"github.com/textileio/broker-core/finalizer"
	brokermocks "github.com/textileio/broker-core/mocks/broker"
	"github.com/textileio/broker-core/tests"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

func TestClient_ReadyToPrepare(t *testing.T) {
	c, ipfs := newClient(t)

	dataCid := addRandomData(t, ipfs)
	sdID := broker.StorageDealID("SD1")
	err := c.ReadyToPrepare(context.Background(), sdID, dataCid)
	require.NoError(t, err)
}

func newClient(t *testing.T) (*client.Client, *httpapi.HttpApi) {
	fin := finalizer.NewFinalizer()
	t.Cleanup(func() {
		require.NoError(t, fin.Cleanup(nil))
	})

	listener := bufconn.Listen(1024 * 1024)
	fin.Add(listener)

	ipfsDocker := launchIPFSContainer(t)
	ipfsAPIMultiaddr := "/ip4/127.0.0.1/tcp/" + ipfsDocker.GetPort("5001/tcp")
	ma, err := multiaddr.NewMultiaddr(ipfsAPIMultiaddr)
	require.NoError(t, err)
	ipfs, err := httpapi.NewApi(ma)
	require.NoError(t, err)

	bm := &brokermocks.Broker{}
	bm.On(
		"StorageDealPrepared",
		mock.Anything,
		mock.Anything,
		mock.Anything,
	).Return(nil)

	config := service.Config{
		Listener:   listener,
		IpfsClient: ipfs,
		Broker:     bm,
		Datastore:  tests.NewTxMapDatastore(),
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

	return client.NewClient(conn), ipfs
}

func addRandomData(t *testing.T, ipfs *httpapi.HttpApi) cid.Cid {
	t.Helper()
	r := rand.New(rand.NewSource(22))

	data := make([]byte, 100)
	_, err := r.Read(data)
	require.NoError(t, err)

	node, err := ipfs.Unixfs().Add(
		context.Background(),
		ipfsfiles.NewReaderFile(bytes.NewReader(data)),
		options.Unixfs.CidVersion(1),
		options.Unixfs.Pin(true))
	require.NoError(t, err)

	return node.Cid()
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
