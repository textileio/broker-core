package piecer

import (
	"bytes"
	"context"
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
	pbBroker "github.com/textileio/broker-core/gen/broker/v1"
	"github.com/textileio/broker-core/msgbroker"
	"github.com/textileio/broker-core/msgbroker/fakemsgbroker"
	"google.golang.org/protobuf/proto"
)

func TestPrepare(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	ipfsDocker := launchIPFSContainer(t)
	ipfsAPI := "/ip4/127.0.0.1/tcp/" + ipfsDocker.GetPort("5001/tcp")

	ma, err := multiaddr.NewMultiaddr(ipfsAPI)
	require.NoError(t, err)
	ipfs, err := httpapi.NewApi(ma)
	require.NoError(t, err)

	mb := fakemsgbroker.New()

	p, err := New([]multiaddr.Multiaddr{ma}, mb)
	require.NoError(t, err)

	id := broker.StorageDealID("invented-id")
	dataCid := addRandomData(t, ipfs)

	err = p.ReadyToPrepare(ctx, id, dataCid)
	require.NoError(t, err)
	require.Equal(t, 1, mb.TotalPublished())
	require.Equal(t, 1, mb.TotalPublishedTopic(msgbroker.NewBatchPreparedTopic))

	msgPayload, err := mb.GetMsg(msgbroker.NewBatchPreparedTopic, 0)
	require.NoError(t, err)
	r := &pbBroker.NewBatchPrepared{}
	err = proto.Unmarshal(msgPayload, r)
	require.NoError(t, err)

	require.Equal(t, string(id), r.Id)
	pieceCid, err := cid.Cast(r.PieceCid)
	require.NoError(t, err)
	require.Equal(t, "baga6ea4seaqngpsnpeoyfwegexd2dp6npdvffsigbxybt6lzsiyd3jbfmcji4ga", pieceCid.String())
	require.Equal(t, uint64(1024*1024*4), r.PieceSize)
}

func TestCanceledPrepare(t *testing.T) {
	t.Parallel()

	ipfsDocker := launchIPFSContainer(t)
	ipfsAPI := "/ip4/127.0.0.1/tcp/" + ipfsDocker.GetPort("5001/tcp")

	ma, err := multiaddr.NewMultiaddr(ipfsAPI)
	require.NoError(t, err)
	ipfs, err := httpapi.NewApi(ma)
	require.NoError(t, err)

	mb := fakemsgbroker.New()

	p, err := New([]multiaddr.Multiaddr{ma}, mb)
	require.NoError(t, err)

	id := broker.StorageDealID("invented-id")
	dataCid := addRandomData(t, ipfs)

	ctx, cls := context.WithCancel(context.Background())
	go func() {
		time.Sleep(time.Nanosecond * 100)
		cls()
	}()
	err = p.ReadyToPrepare(ctx, id, dataCid)
	require.Error(t, err)
	require.Contains(t, err.Error(), "context canceled")
	require.Equal(t, 0, mb.TotalPublished())
}

func addRandomData(t *testing.T, ipfs *httpapi.HttpApi) cid.Cid {
	t.Helper()
	r := rand.New(rand.NewSource(22))

	data := make([]byte, 1024*1024*3) // 3MiB would result in a 4MiB piece size
	_, _ = r.Read(data)

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
