package packer

import (
	"bytes"
	"context"
	"io"
	"io/ioutil"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"github.com/ipfs/go-cid"
	files "github.com/ipfs/go-ipfs-files"
	ipfsfiles "github.com/ipfs/go-ipfs-files"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	ipld "github.com/ipfs/go-ipld-format"
	unixfile "github.com/ipfs/go-unixfs/file"
	"github.com/ipfs/interface-go-ipfs-core/options"
	"github.com/ipfs/interface-go-ipfs-core/path"
	"github.com/multiformats/go-multiaddr"
	"github.com/ory/dockertest/v3"
	"github.com/stretchr/testify/require"

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
	if err := golog.SetLogLevels(map[string]golog.LogLevel{
		"packer": golog.LevelDebug,
		"store":  golog.LevelDebug,
	}); err != nil {
		panic(err)
	}
}

func TestPack(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	packer, ipfs, mb := createPacker(t)

	numBatchedCids, err := packer.pack(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, numBatchedCids)

	// 1- Add 100 random files and get their cids.
	numFiles := 100
	dataCids := addRandomData(t, ipfs, numFiles)

	// 2- Signal ready to pack these cids to Packer
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

	// 3- Force pack and inspect what was signaled to the broker
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
	require.Equal(t, "OR", msg.Origin)
	require.NotEmpty(t, msg.BatchCid)
	require.Greater(t, msg.BatchSize, int64(0))
	bcid, err := cid.Cast(msg.BatchCid)
	require.NoError(t, err)
	require.True(t, bcid.Defined())
	require.NotEmpty(t, msg.Manifest)
	dagManifest := getBatchManifestFromDAG(t, bcid, ipfs.Dag())
	require.True(t, bytes.Equal(dagManifest, msg.Manifest))
	require.Equal(t, "http://duke.web3/car/"+bcid.String(), msg.CarUrl)

	// Check that the batch cid was pinned in ipfs.
	_, pinned, err := ipfs.Pin().IsPinned(ctx, path.IpfsPath(bcid))
	require.NoError(t, err)
	require.True(t, pinned)
}

func TestTimeBasedBatchClosing(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	closingTime := time.Second * 3
	packer, ipfs, _ := createPackerCustom(t, nil, 1<<30, closingTime)

	dataCids := addRandomDataWithSize(t, ipfs, 1, 1<<20)

	for i, dataCid := range dataCids {
		err := packer.ReadyToBatch(
			ctx,
			mbroker.OperationID(strconv.Itoa(i)),
			[]mbroker.ReadyToBatchData{
				{StorageRequestID: broker.StorageRequestID(strconv.Itoa(i)),
					DataCid: dataCid,
					Origin:  "Textile",
				}},
		)
		require.NoError(t, err)
	}

	numBatchedCids, err := packer.pack(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, numBatchedCids)

	time.Sleep(closingTime * 2)

	numBatchedCids, err = packer.pack(ctx)
	require.NoError(t, err)
	require.Equal(t, 1, numBatchedCids)
}

func TestPackWithCARUploader(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	fu := &fakeCARUploader{dummyURL: "http://fake.dev/car/jorge.car"}
	packer, ipfs, mb := createPackerCustom(t, fu, 100*100, time.Second*3)

	numBatchedCids, err := packer.pack(ctx)
	require.NoError(t, err)
	require.Equal(t, 0, numBatchedCids)

	numFiles := 100
	dataCids := addRandomData(t, ipfs, numFiles)

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

	numBatchedCids, err = packer.pack(ctx)
	require.NoError(t, err)
	require.Equal(t, numFiles, numBatchedCids)

	require.Equal(t, 1, mb.TotalPublished())
	require.Equal(t, 1, mb.TotalPublishedTopic(mbroker.NewBatchCreatedTopic))

	msgb, err := mb.GetMsg(mbroker.NewBatchCreatedTopic, 0)
	require.NoError(t, err)
	msg := &pb.NewBatchCreated{}
	err = proto.Unmarshal(msgb, msg)
	require.NoError(t, err)

	require.Equal(t, fu.dummyURL, msg.CarUrl)
}

func TestPackMultiple(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	packer, ipfs, mb := createPacker(t)

	// 1- Add 100 random files and get their cids.
	numFiles := 100
	dataCids := addRandomData(t, ipfs, numFiles)

	// 2- Batch the 100 files in a single call
	rtbs := make([]mbroker.ReadyToBatchData, len(dataCids))
	for i, dataCid := range dataCids {
		rtbs[i] = mbroker.ReadyToBatchData{
			StorageRequestID: broker.StorageRequestID(strconv.Itoa(i)),
			DataCid:          dataCid,
			Origin:           "OR",
		}
	}
	err := packer.ReadyToBatch(ctx, "op-1", rtbs)
	require.NoError(t, err)

	// 3- Force pack and inspect what was signaled to the broker
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
	require.NotEmpty(t, msg.Manifest)
	dagManifest := getBatchManifestFromDAG(t, bcid, ipfs.Dag())
	require.True(t, bytes.Equal(dagManifest, msg.Manifest))

	// Check that the batch cid was pinned in ipfs.
	_, pinned, err := ipfs.Pin().IsPinned(ctx, path.IpfsPath(bcid))
	require.NoError(t, err)
	require.True(t, pinned)
}

func TestStats(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	packer, _, _ := createPacker(t)
	_, _, err := packer.store.GetStats(ctx)
	require.NoError(t, err)
}

func TestPackIdempotency(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	packer, ipfs, _ := createPacker(t)

	// 1- Add one file to go-ipfs.
	dataCids := addRandomData(t, ipfs, 1)

	// 2- Make first call.
	err := packer.ReadyToBatch(
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

	packer, ipfs, mb := createPacker(t)

	// 2- Create a single file
	dataCid := addRandomData(t, ipfs, 1)[0]

	// 3- Simulate multiple StorageRequests but with the same data
	numRepeatedStorageRequest := 100
	for i := 0; i < numRepeatedStorageRequest; i++ {
		err := packer.ReadyToBatch(
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

func createPacker(t *testing.T) (*Packer, *httpapi.HttpApi, *fakemsgbroker.FakeMsgBroker) {
	return createPackerCustom(t, nil, 100*100, time.Second*3)
}

func createPackerCustom(
	t *testing.T,
	u CARUploader,
	minSize int64,
	closingTime time.Duration) (*Packer, *httpapi.HttpApi, *fakemsgbroker.FakeMsgBroker) {
	mb := fakemsgbroker.New()
	postgresURL, err := tests.PostgresURL()
	require.NoError(t, err)

	ipfsDocker := launchIPFSContainer(t)
	ipfsAPIMultiaddr := "/ip4/127.0.0.1/tcp/" + ipfsDocker.GetPort("5001/tcp")

	ma, err := multiaddr.NewMultiaddr(ipfsAPIMultiaddr)
	require.NoError(t, err)
	pinnerClient, err := httpapi.NewApi(ma)
	require.NoError(t, err)

	opts := []Option{
		WithDaemonFrequency(time.Hour),
		WithBatchMinSize(minSize),
		WithBatchMinWaiting(closingTime),
		WithBatchWaitScalingFactor(1),
		WithCARExportURL("http://duke.web3/car/")}
	if u != nil {
		opts = append(opts, WithCARUploader(u))
	}
	packer, err := New(postgresURL, pinnerClient, []multiaddr.Multiaddr{ma}, mb, opts...)
	require.NoError(t, err)

	return packer, pinnerClient, mb
}

func addRandomData(t *testing.T, ipfs *httpapi.HttpApi, count int) []cid.Cid {
	t.Helper()
	return addRandomDataWithSize(t, ipfs, count, 100)
}

func addRandomDataWithSize(t *testing.T, ipfs *httpapi.HttpApi, count int, size int) []cid.Cid {
	t.Helper()
	r := rand.New(rand.NewSource(22))

	cids := make([]cid.Cid, count)
	for i := 0; i < count; i++ {
		data := make([]byte, size)
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

func getBatchManifestFromDAG(t *testing.T, payloadCid cid.Cid, dagService ipld.DAGService) []byte {
	ctx := context.Background()
	rootNd, err := dagService.Get(ctx, payloadCid)
	require.NoError(t, err)
	manifestLnk := rootNd.Links()[0]
	manifestNd, err := dagService.Get(ctx, manifestLnk.Cid)
	require.NoError(t, err)
	manifestF, err := unixfile.NewUnixfsFile(context.Background(), dagService, manifestNd)
	require.NoError(t, err)
	manifest := manifestF.(files.File)

	dagManifest, err := ioutil.ReadAll(manifest)
	require.NoError(t, err)

	return dagManifest
}

type fakeCARUploader struct {
	dummyURL string
}

func (fu *fakeCARUploader) Store(_ context.Context, _ string, _ io.Reader) (string, error) {
	return fu.dummyURL, nil
}
