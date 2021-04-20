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
	"github.com/stretchr/testify/require"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/tests"
)

func TestPackAndRetrieve(t *testing.T) {
	ctx := context.Background()

	// 1- Create a Packer and have a go-ipfs client too.
	ipfsAPIMultiaddr := "/ip4/127.0.0.1/tcp/5001"
	ma, err := multiaddr.NewMultiaddr(ipfsAPIMultiaddr)
	require.NoError(t, err)
	ipfs, err := httpapi.NewApi(ma)
	require.NoError(t, err)

	brokerMock := &brokerMock{}
	packer := createPacker(t, ipfs, brokerMock)

	// 2- Add 100 random files and get their cids.
	dataCids := addRandomData(t, ipfs, 100)

	// 3- Signal ready to pack these cids to Packer
	for i, dataCid := range dataCids {
		err = packer.ReadyToPack(ctx, broker.BrokerRequestID(strconv.Itoa(i)), dataCid)
		require.NoError(t, err)
	}

	// 4- Force pack and inspect what was signaled to the broker
	err = packer.pack(ctx)
	require.NoError(t, err)

	require.Len(t, brokerMock.srids, 100)
	require.True(t, brokerMock.batchCid.Defined())

	_, pinned, err := ipfs.Pin().IsPinned(ctx, path.IpfsPath(brokerMock.batchCid))
	require.NoError(t, err)
	require.True(t, pinned)
	require.Len(t, packer.queue, 0)

	// TODO:
	// - Add test to check that three levels are < 1MiB in worst case scenario.
	// - Persistent queue, resumal.
	// - Test for collision behavior
}

func TestSelectorRetrieval(t *testing.T) {
	// TODO
}

func createPacker(t *testing.T, ipfsClient *httpapi.HttpApi, broker *brokerMock) *Packer {
	ds := tests.NewTxMapDatastore()
	// TODO: dockerize?
	packer, err := New(ds, ipfsClient, broker, WithFrequency(time.Hour))
	require.NoError(t, err)

	return packer
}

func addRandomData(t *testing.T, ipfs *httpapi.HttpApi, count int) []cid.Cid {
	t.Helper()
	r := rand.New(rand.NewSource(22))

	cids := make([]cid.Cid, count)
	for i := 0; i < count; i++ {
		data := make([]byte, 1000)
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
	batchCid cid.Cid
	srids    []broker.BrokerRequestID
}

func (bm *brokerMock) CreateStorageDeal(
	ctx context.Context,
	batchCid cid.Cid,
	srids []broker.BrokerRequestID) (broker.StorageDealID, error) {
	if bm.batchCid.Defined() {
		return "", fmt.Errorf("create storage deal called twice")
	}
	bm.batchCid = batchCid
	bm.srids = srids

	return broker.StorageDealID("DUKE"), nil
}

// StorageDealPrepared signals the broker that a StorageDeal was prepared and it's ready to auction.
func (bm *brokerMock) StorageDealPrepared(
	ctx context.Context,
	id broker.StorageDealID,
	pr broker.DataPreparationResult) error {
	panic("shouldn't be called")
}

func (bm *brokerMock) Create(ctx context.Context, c cid.Cid, meta broker.Metadata) (broker.BrokerRequest, error) {
	panic("shouldn't be called")
}

// Get returns a broker request from an id.
func (bm *brokerMock) Get(ctx context.Context, ID broker.BrokerRequestID) (broker.BrokerRequest, error) {
	panic("shouldn't be called")
}
