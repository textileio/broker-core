package packer

import (
	"bytes"
	"context"
	"math/rand"
	"strconv"
	"testing"

	"github.com/ipfs/go-cid"
	ipfsfiles "github.com/ipfs/go-ipfs-files"
	httpapi "github.com/ipfs/go-ipfs-http-client"
	"github.com/ipfs/interface-go-ipfs-core/options"
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

	packer := createPacker(t, ipfsAPIMultiaddr)

	// 2- Add 100 random files and get their cids.
	dataCids := addRandomData(t, ipfs, 100)

	// 3- Signal ready to pack these cids to Packer, and force pack.
	for i, dataCid := range dataCids {
		err = packer.ReadyToPack(ctx, broker.BrokerRequestID(strconv.Itoa(i)), dataCid)
		require.NoError(t, err)
	}

	// 4- Test retrieval with go-ipfs with selector

	// TODO:
	// - Customize cut frequency
	// - Custome max batch size
	// - Add test to check that three levels are < 1MiB in worst case scenario.
	// - Persistent queue, resumal.
	// - Test for collision behavior
}

func createPacker(t *testing.T, ipfsMultiaddr string) *Packer {
	ds := tests.NewTxMapDatastore()
	// TODO: dockerize?
	packer, err := New(ds, ipfsMultiaddr, "")
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
			options.Unixfs.Pin(false))
		require.NoError(t, err)

		cids[i] = node.Cid()
	}
	return cids
}
