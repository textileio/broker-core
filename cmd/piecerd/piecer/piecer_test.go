package piecer

import (
	"bytes"
	"context"
	"io"
	"math/rand"
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
	"github.com/textileio/broker-core/cmd/piecerd/store"
	pb "github.com/textileio/broker-core/gen/broker/v1"
	"github.com/textileio/broker-core/msgbroker"
	"github.com/textileio/broker-core/msgbroker/fakemsgbroker"
	"github.com/textileio/broker-core/tests"
	"github.com/textileio/go-libp2p-pubsub-rpc/finalizer"
	"google.golang.org/protobuf/proto"
)

func TestE2E(t *testing.T) {
	t.Parallel()
	p, mb, ipfs := newClient(t)

	dataCid := addRandomData(t, ipfs)
	sdID := broker.BatchID("SD1")
	err := p.ReadyToPrepare(context.Background(), sdID, dataCid)
	require.NoError(t, err)

	time.Sleep(time.Second * 1)

	// Assert the prepared batch msg was published to the topic.
	require.Equal(t, 1, mb.TotalPublished())
	require.Equal(t, 1, mb.TotalPublishedTopic(msgbroker.NewBatchPreparedTopic))
	data, err := mb.GetMsg(msgbroker.NewBatchPreparedTopic, 0)
	require.NoError(t, err)
	nbc := &pb.NewBatchPrepared{}
	err = proto.Unmarshal(data, nbc)
	require.NoError(t, err)

	require.Equal(t, string(sdID), nbc.Id)
	pieceCid, err := cid.Cast(nbc.PieceCid)
	require.NoError(t, err)
	require.Equal(t, "baga6ea4seaqabf42l52koxdzu4prvzf55a4dxnfbahilkyiqfp4yxgznifvscmy", pieceCid.String())
	require.Equal(t, uint64(16384), nbc.PieceSize)

	_, ok, err := p.store.GetNextPending(context.Background())
	require.NoError(t, err)
	require.False(t, ok)
}

func TestVirtualPadding(t *testing.T) {
	t.Parallel()
	p, mb, ipfs := newClientWithPadding(t, 64<<10)

	dataCid := addRandomData(t, ipfs)
	sdID := broker.BatchID("SD1")
	err := p.ReadyToPrepare(context.Background(), sdID, dataCid)
	require.NoError(t, err)

	time.Sleep(time.Second * 1)

	// Assert the prepared batch msg was published to the topic.
	require.Equal(t, 1, mb.TotalPublished())
	require.Equal(t, 1, mb.TotalPublishedTopic(msgbroker.NewBatchPreparedTopic))
	data, err := mb.GetMsg(msgbroker.NewBatchPreparedTopic, 0)
	require.NoError(t, err)
	nbc := &pb.NewBatchPrepared{}
	err = proto.Unmarshal(data, nbc)
	require.NoError(t, err)

	require.Equal(t, string(sdID), nbc.Id)
	pieceCid, err := cid.Cast(nbc.PieceCid)
	require.NoError(t, err)
	require.Equal(t, "baga6ea4seaqanldgxtddjpceoskqhd6y5qehsitn5pi4c5no4fgrqjd6mkkfkoq", pieceCid.String())
	require.Equal(t, uint64(64<<10), nbc.PieceSize)

	_, ok, err := p.store.GetNextPending(context.Background())
	require.NoError(t, err)
	require.False(t, ok)
}

func TestIdempotency(t *testing.T) {
	t.Parallel()
	p, _, ipfs := newClient(t)

	dataCid := addRandomData(t, ipfs)
	sdID := broker.BatchID("SD1")
	err := p.ReadyToPrepare(context.Background(), sdID, dataCid)
	require.NoError(t, err)

	err = p.ReadyToPrepare(context.Background(), sdID, dataCid)
	require.ErrorIs(t, err, store.ErrBatchExists)
}

func TestE2EFail(t *testing.T) {
	t.Parallel()
	p, mb, _ := newClient(t)

	// Fake a cid that doesn't exist in the IPFS node, so the preparation can fail.
	dataCid, err := cid.Decode("baga6ea4seaqabf42l52koxdzu4prvzf55a4dxnfbahilkyiqfp4yxgznifvscmj")
	require.NoError(t, err)
	sdID := broker.BatchID("SD1")
	err = p.ReadyToPrepare(context.Background(), sdID, dataCid)
	require.NoError(t, err)

	time.Sleep(time.Second * 1)

	// Assert the prepared batch msg wasn't published to the topic.
	require.Equal(t, 0, mb.TotalPublished())
	require.Equal(t, 0, mb.TotalPublishedTopic(msgbroker.NewBatchPreparedTopic))

	// Assert that the same entry is available again to process.
	_, ok, err := p.store.GetNextPending(context.Background())
	require.NoError(t, err)
	require.True(t, ok)
}

func newClient(t *testing.T) (*Piecer, *fakemsgbroker.FakeMsgBroker, *httpapi.HttpApi) {
	return newClientWithPadding(t, 0)
}

func newClientWithPadding(t *testing.T, padToSize uint64) (*Piecer, *fakemsgbroker.FakeMsgBroker, *httpapi.HttpApi) {
	fin := finalizer.NewFinalizer()
	t.Cleanup(func() {
		require.NoError(t, fin.Cleanup(nil))
	})

	ipfsDocker := launchIPFSContainer(t)
	ipfsAPIMultiaddr := "/ip4/127.0.0.1/tcp/" + ipfsDocker.GetPort("5001/tcp")
	ma, err := multiaddr.NewMultiaddr(ipfsAPIMultiaddr)
	require.NoError(t, err)
	ipfs, err := httpapi.NewApi(ma)
	require.NoError(t, err)

	u, err := tests.PostgresURL()
	require.NoError(t, err)

	mb := fakemsgbroker.New()

	p, err := New(u, []multiaddr.Multiaddr{ma}, mb, time.Hour, time.Millisecond, padToSize)
	require.NoError(t, err)
	fin.Add(p)

	return p, mb, ipfs
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
