package packer

import (
	"bytes"
	"context"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/ipfs/go-cid"
	ipfsfiles "github.com/ipfs/go-ipfs-files"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	"github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/multiformats/go-multiaddr"
	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/require"
	"github.com/textileio/bidbot/lib/logging"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/packerd/store"
	pb "github.com/textileio/broker-core/gen/broker/v1"
	mbroker "github.com/textileio/broker-core/msgbroker"
	"github.com/textileio/broker-core/msgbroker/fakemsgbroker"
	"github.com/textileio/broker-core/tests"
	golog "github.com/textileio/go-log/v2"
	"google.golang.org/protobuf/proto"
)

func init() {
	if err := logging.SetLogLevels(map[string]golog.LogLevel{
		"packer": golog.LevelDebug,
		"store":  golog.LevelDebug,
	}); err != nil {
		panic(err)
	}
}

func TestPack(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ipfsDocker := launchIPFSContainer(t)
	ipfsAPIMultiaddr := "/ip4/127.0.0.1/tcp/" + ipfsDocker.GetPort("5001/tcp")

	// 1- Create a Packer and have a go-ipfs client too.
	ma, err := multiaddr.NewMultiaddr(ipfsAPIMultiaddr)
	require.NoError(t, err)
	ipfs, err := httpapi.NewApi(ma)
	require.NoError(t, err)

	packer, mb := createPacker(t, ipfs)

	numBatchedCids, err := packer.pack(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, numBatchedCids)

	// 2- Add 100 random files and get their cids.
	numFiles := 100
	dataCids := addRandomData(t, ipfs, numFiles)

	// 3- Signal ready to pack these cids to Packer
	for i, dataCid := range dataCids {
		err = packer.ReadyToBatch(
			ctx,
			mbroker.OperationID(strconv.Itoa(i)),
			[]mbroker.ReadyToBatchData{
				{StorageRequestID: broker.StorageRequestID(strconv.Itoa(i)),
					DataCid: dataCid,
					Origin:  "OR",
				}},
		)
		require.NoError(t, err)
	}

	// 4- Force pack and inspect what was signaled to the broker
	numBatchedCids, err = packer.pack(ctx)
	require.NoError(t, err)

	require.Equal(t, 1, mb.TotalPublished())
	require.Equal(t, 1, mb.TotalPublishedTopic(mbroker.NewBatchCreatedTopic))

	msgb, err := mb.GetMsg(mbroker.NewBatchCreatedTopic, 0)
	require.NoError(t, err)
	msg := &pb.NewBatchCreated{}
	err = proto.Unmarshal(msgb, msg)
	require.NoError(t, err)

	require.Len(t, msg.StorageRequestIds, numFiles)
	require.Equal(t, numFiles, numBatchedCids)
	require.NotEmpty(t, msg.BatchCid)
	bcid, err := cid.Cast(msg.BatchCid)
	require.NoError(t, err)
	require.True(t, bcid.Defined())

	// Check that the batch cid was pinned in ipfs.
	_, pinned, err := ipfs.Pin().IsPinned(ctx, path.IpfsPath(bcid))
	require.NoError(t, err)
	require.True(t, pinned)
}

func TestPackBatch(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ipfsDocker := launchIPFSContainer(t)
	ipfsAPIMultiaddr := "/ip4/127.0.0.1/tcp/" + ipfsDocker.GetPort("5001/tcp")

	// 1- Create a Packer and have a go-ipfs client too.
	ma, err := multiaddr.NewMultiaddr(ipfsAPIMultiaddr)
	require.NoError(t, err)
	ipfs, err := httpapi.NewApi(ma)
	require.NoError(t, err)

	packer, mb := createPacker(t, ipfs)

	// 2- Add 100 random files and get their cids.
	numFiles := 100
	dataCids := addRandomData(t, ipfs, numFiles)

	// 3- Batch the 100 files in a single call
	rtbs := make([]mbroker.ReadyToBatchData, len(dataCids))
	for i, dataCid := range dataCids {
		rtbs[i] = mbroker.ReadyToBatchData{
			StorageRequestID: broker.StorageRequestID(strconv.Itoa(i)),
			DataCid:          dataCid,
			Origin:           "OR",
		}
	}
	err = packer.ReadyToBatch(ctx, "op-1", rtbs)
	require.NoError(t, err)

	// 4- Force pack and inspect what was signaled to the broker
	numBatchedCids, err := packer.pack(ctx)
	require.NoError(t, err)

	require.Equal(t, 1, mb.TotalPublished())
	require.Equal(t, 1, mb.TotalPublishedTopic(mbroker.NewBatchCreatedTopic))

	msgb, err := mb.GetMsg(mbroker.NewBatchCreatedTopic, 0)
	require.NoError(t, err)
	msg := &pb.NewBatchCreated{}
	err = proto.Unmarshal(msgb, msg)
	require.NoError(t, err)

	require.Len(t, msg.StorageRequestIds, numFiles)
	require.Equal(t, numFiles, numBatchedCids)
	require.NotEmpty(t, msg.BatchCid)
	bcid, err := cid.Cast(msg.BatchCid)
	require.NoError(t, err)
	require.True(t, bcid.Defined())

	// Check that the batch cid was pinned in ipfs.
	_, pinned, err := ipfs.Pin().IsPinned(ctx, path.IpfsPath(bcid))
	require.NoError(t, err)
	require.True(t, pinned)
}
func TestStats(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ipfsDocker := launchIPFSContainer(t)
	ipfsAPIMultiaddr := "/ip4/127.0.0.1/tcp/" + ipfsDocker.GetPort("5001/tcp")

	// 1- Create a Packer and have a go-ipfs client too.
	ma, err := multiaddr.NewMultiaddr(ipfsAPIMultiaddr)
	require.NoError(t, err)
	ipfs, err := httpapi.NewApi(ma)
	require.NoError(t, err)

	packer, _ := createPacker(t, ipfs)
	_, err = packer.store.GetStats(ctx)
	require.NoError(t, err)
}

func TestPackIdempotency(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ipfsDocker := launchIPFSContainer(t)
	ipfsAPIMultiaddr := "/ip4/127.0.0.1/tcp/" + ipfsDocker.GetPort("5001/tcp")

	// 1- Create a Packer and have a go-ipfs client too.
	ma, err := multiaddr.NewMultiaddr(ipfsAPIMultiaddr)
	require.NoError(t, err)
	ipfs, err := httpapi.NewApi(ma)
	require.NoError(t, err)

	packer, _ := createPacker(t, ipfs)

	// 2- Add one file to go-ipfs.
	dataCids := addRandomData(t, ipfs, 1)

	// 3- Make first call.
	err = packer.ReadyToBatch(
		ctx,
		mbroker.OperationID("op-1"),
		[]mbroker.ReadyToBatchData{{StorageRequestID: broker.StorageRequestID("br-1"), DataCid: dataCids[0]}},
	)
	require.NoError(t, err)

	// 3- Make second call with the same operation id.
	err = packer.ReadyToBatch(
		ctx,
		mbroker.OperationID("op-1"),
		[]mbroker.ReadyToBatchData{{StorageRequestID: broker.StorageRequestID("br-1"), DataCid: dataCids[0]}},
	)
	require.ErrorIs(t, err, store.ErrOperationIDExists)
}

func TestMultipleStorageRequestWithSameCid(t *testing.T) {
	t.Parallel()
	t.SkipNow()
	ctx := context.Background()

	ipfsDocker := launchIPFSContainer(t)
	ipfsAPIMultiaddr := "/ip4/127.0.0.1/tcp/" + ipfsDocker.GetPort("5001/tcp")

	ma, err := multiaddr.NewMultiaddr(ipfsAPIMultiaddr)
	require.NoError(t, err)
	ipfs, err := httpapi.NewApi(ma)
	require.NoError(t, err)

	packer, mb := createPacker(t, ipfs)

	// 2- Create a single file
	dataCid := addRandomData(t, ipfs, 1)[0]

	// 3- Simulate multiple StorageRequests but with the same data
	numRepeatedStorageRequest := 100
	for i := 0; i < numRepeatedStorageRequest; i++ {
		err = packer.ReadyToBatch(
			ctx,
			mbroker.OperationID(strconv.Itoa(1)),
			[]mbroker.ReadyToBatchData{{StorageRequestID: broker.StorageRequestID(strconv.Itoa(i)), DataCid: dataCid}},
		)
		require.NoError(t, err)
	}

	// 4- Force pack and inspect what was signaled to the broker
	numCidsBatched, err := packer.pack(ctx)
	require.NoError(t, err)

	require.Equal(t, mb.TotalPublished(), 1)
	require.Equal(t, 1, mb.TotalPublishedTopic(mbroker.NewBatchCreatedTopic))
	msgb, err := mb.GetMsg(mbroker.NewBatchCreatedTopic, 0)
	require.NoError(t, err)
	msg := &pb.NewBatchCreated{}
	err = proto.Unmarshal(msgb, msg)
	require.NoError(t, err)
	bcid, err := cid.Cast(msg.BatchCid)
	require.NoError(t, err)

	require.True(t, bcid.Defined())
	// We fullfiled numRepeatedStorageRequest, not only 1!
	require.Len(t, msg.StorageRequestIds, numRepeatedStorageRequest)
	// Despite we fulfilled multiple storage request, the batch only has one cid!
	require.Equal(t, 1, numCidsBatched)

	// Check that the batch cid was pinned in ipfs.
	_, pinned, err := ipfs.Pin().IsPinned(ctx, path.IpfsPath(bcid))
	require.NoError(t, err)
	require.True(t, pinned)
}

func createPacker(t *testing.T, ipfsClient *httpapi.HttpApi) (*Packer, *fakemsgbroker.FakeMsgBroker) {
	mb := fakemsgbroker.New()
	postgresURL, err := tests.PostgresURL()
	require.NoError(t, err)
	packer, err := New(postgresURL, ipfsClient, mb, WithDaemonFrequency(time.Hour), WithBatchMinSize(100*100))
	require.NoError(t, err)

	return packer, mb
}

func addRandomData(t *testing.T, ipfs *httpapi.HttpApi, count int) []cid.Cid {
	t.Helper()
	r := rand.New(rand.NewSource(22))

	cids := make([]cid.Cid, count)
	for i := 0; i < count; i++ {
		data := make([]byte, 100)
		_, _ = r.Read(data)

		node, err := ipfs.Unixfs().Add(
			context.Background(),
			ipfsfiles.NewReaderFile(bytes.NewReader(data)),
			options.Unixfs.CidVersion(1),
			options.Unixfs.Pin(true))
		require.NoError(t, err)

		cids[i] = node.Cid()
	}
	return cids
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
