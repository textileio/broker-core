package packer

import (
	"bytes"
	"context"
	"fmt"
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
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/tests"
)

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

	brokerMock := &brokerMock{}
	packer := createPacker(t, ipfs, brokerMock)

	// 2- Add 100 random files and get their cids.
	numFiles := 100
	dataCids := addRandomData(t, ipfs, numFiles)

	// 3- Signal ready to pack these cids to Packer
	for i, dataCid := range dataCids {
		err = packer.ReadyToPack(ctx, broker.BrokerRequestID(strconv.Itoa(i)), dataCid)
		require.NoError(t, err)
	}

	// 4- Force pack and inspect what was signaled to the broker
	numBatchedCids, err := packer.pack(ctx)
	require.NoError(t, err)

	require.Len(t, brokerMock.srids, numFiles)
	require.Equal(t, int64(numFiles), numBatchedCids)
	require.True(t, brokerMock.batchCid.Defined())

	// Check that the batch cid was pinned in ipfs.
	_, pinned, err := ipfs.Pin().IsPinned(ctx, path.IpfsPath(brokerMock.batchCid))
	require.NoError(t, err)
	require.True(t, pinned)
}

func TestMultipleBrokerRequestWithSameCid(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ipfsDocker := launchIPFSContainer(t)
	ipfsAPIMultiaddr := "/ip4/127.0.0.1/tcp/" + ipfsDocker.GetPort("5001/tcp")

	ma, err := multiaddr.NewMultiaddr(ipfsAPIMultiaddr)
	require.NoError(t, err)
	ipfs, err := httpapi.NewApi(ma)
	require.NoError(t, err)

	brokerMock := &brokerMock{}
	packer := createPacker(t, ipfs, brokerMock)

	// 2- Create a single file
	dataCid := addRandomData(t, ipfs, 1)[0]

	// 3- Simulate multiple BrokerRequests but with the same data
	numRepeatedBrokerRequest := 100
	for i := 0; i < numRepeatedBrokerRequest; i++ {
		err = packer.ReadyToPack(ctx, broker.BrokerRequestID(strconv.Itoa(i)), dataCid)
		require.NoError(t, err)
	}

	// 4- Force pack and inspect what was signaled to the broker
	numCidsBatched, err := packer.pack(ctx)
	require.NoError(t, err)

	require.True(t, brokerMock.batchCid.Defined())
	// We fullfiled numRepeatedBrokerRequest, not only 1!
	require.Len(t, brokerMock.srids, numRepeatedBrokerRequest)
	// Despite we fulfilled multiple broker request, the batch only has one cid!
	require.Equal(t, int64(1), numCidsBatched)

	// Check that the batch cid was pinned in ipfs.
	_, pinned, err := ipfs.Pin().IsPinned(ctx, path.IpfsPath(brokerMock.batchCid))
	require.NoError(t, err)
	require.True(t, pinned)
}

func TestSizeLimit(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	ipfsDocker := launchIPFSContainer(t)
	ipfsAPIMultiaddr := "/ip4/127.0.0.1/tcp/" + ipfsDocker.GetPort("5001/tcp")

	ma, err := multiaddr.NewMultiaddr(ipfsAPIMultiaddr)
	require.NoError(t, err)
	ipfs, err := httpapi.NewApi(ma)
	require.NoError(t, err)

	brokerMock := &brokerMock{allowMultipleCalls: true}
	ds := tests.NewTxMapDatastore()
	// Configure max DAG size of 5KB
	packer, err := New(ds, ipfs, brokerMock, WithFrequency(time.Hour), WithSectorSize(50*100))
	require.NoError(t, err)

	// 2- Add 100 random files ready to batch, which is 10KiB. They can't fit all in a single dag.
	numFiles := 100
	dataCids := addRandomData(t, ipfs, numFiles)

	// 3- Signal that all of them are ready to pack.
	for i, dataCid := range dataCids {
		err = packer.ReadyToPack(ctx, broker.BrokerRequestID(strconv.Itoa(i)), dataCid)
		require.NoError(t, err)
	}
	numCidsBatched, err := packer.pack(ctx)
	require.NoError(t, err)

	require.True(t, brokerMock.batchCid.Defined())
	// The first batch should definitely considered less than 100 broker requests.
	require.Less(t, len(brokerMock.srids), numFiles)
	// There were no repetitions, so that means also 50 cids.
	require.Less(t, numCidsBatched, int64(numFiles))

	// Check that the batch cid was pinned in ipfs.
	_, pinned, err := ipfs.Pin().IsPinned(ctx, path.IpfsPath(brokerMock.batchCid))
	require.NoError(t, err)
	require.True(t, pinned)

	// Keep packing and counting...
	// If something is working wrong, this will be and endless loop.
	for numCidsBatched < int64(numFiles) {
		countBatched, err := packer.pack(ctx)
		require.NoError(t, err)
		numCidsBatched += countBatched
	}
	// Check that we got exactly numFiles batched in total, not a single extra one.
	require.Equal(t, int64(numFiles), numCidsBatched)
}

func TestSelectorRetrieval(t *testing.T) {
	t.Skipf("Not implemented yet, needs to wait a bit for some things to land in ipfs world")
}

func createPacker(t *testing.T, ipfsClient *httpapi.HttpApi, broker *brokerMock) *Packer {
	ds := tests.NewTxMapDatastore()
	packer, err := New(ds, ipfsClient, broker, WithFrequency(time.Hour))
	require.NoError(t, err)

	return packer
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

type brokerMock struct {
	batchCid           cid.Cid
	srids              []broker.BrokerRequestID
	allowMultipleCalls bool
}

func (bm *brokerMock) CreateStorageDeal(
	ctx context.Context,
	batchCid cid.Cid,
	srids []broker.BrokerRequestID) (broker.StorageDealID, error) {
	if !bm.allowMultipleCalls && bm.batchCid.Defined() {
		return "", fmt.Errorf("create storage deal called twice")
	}
	bm.batchCid = batchCid
	bm.srids = srids

	return broker.StorageDealID("DUKE"), nil
}

func (bm *brokerMock) StorageDealPrepared(context.Context, broker.StorageDealID, broker.DataPreparationResult) error {
	panic("shouldn't be called")
}

func (bm *brokerMock) StorageDealAuctioned(context.Context, broker.Auction) error {
	panic("shouldn't be called")
}

func (bm *brokerMock) StorageDealProposalAccepted(context.Context, broker.StorageDealID, string, cid.Cid) error {
	panic("shouldn't be called")
}

func (bm *brokerMock) Create(context.Context, cid.Cid, broker.Metadata) (broker.BrokerRequest, error) {
	panic("shouldn't be called")
}

func (bm *brokerMock) StorageDealFinalizedDeal(context.Context, broker.FinalizedAuctionDeal) error {
	panic("shouldn't be called")
}

func (bm *brokerMock) Get(context.Context, broker.BrokerRequestID) (broker.BrokerRequest, error) {
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
