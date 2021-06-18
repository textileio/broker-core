package datauri_test

import (
	"context"
	"testing"

	format "github.com/ipfs/go-ipld-format"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	. "github.com/textileio/broker-core/cmd/bidbot/service/datauri"
	"github.com/textileio/broker-core/cmd/bidbot/service/datauri/apitest"
	"github.com/textileio/broker-core/finalizer"
	"github.com/textileio/broker-core/marketpeer"
)

const testCid = "bafybeic6xu6afw5lg6a6h6uk27twq3bmzxjg346nhsyenuhxwzfv6yhu5y"

func TestNewURI(t *testing.T) {
	// http
	_, err := NewURI("http://foo.com/cid/" + testCid)
	require.NoError(t, err)
	// https
	_, err = NewURI("https://foo.com/cid/" + testCid)
	require.NoError(t, err)
	// not supported
	_, err = NewURI("s3://foo.com/notsupported")
	require.ErrorIs(t, err, ErrSchemeNotSupported)
	// bad cid
	_, err = NewURI("https://foo.com/cid/123")
	require.Error(t, err)
}

func TestURI_Cid(t *testing.T) {
	u, err := NewURI("https://foo.com/cid/" + testCid)
	require.NoError(t, err)
	assert.Equal(t, testCid, u.Cid().String())
}

func TestURI_Validate(t *testing.T) {
	gw := apitest.NewDataURIHTTPGateway(createDagService(t))
	t.Cleanup(gw.Close)

	// Validate good car file
	_, dataURI, err := gw.CreateURI(true)
	require.NoError(t, err)
	u, err := NewURI(dataURI)
	require.NoError(t, err)
	err = u.Validate(context.Background())
	require.NoError(t, err)

	// Validate car file not found
	_, dataURI, err = gw.CreateURI(false)
	require.NoError(t, err)
	u, err = NewURI(dataURI)
	require.NoError(t, err)
	err = u.Validate(context.Background())
	require.ErrorIs(t, err, ErrCarFileUnavailable)

	// Validate bad car file
	_, dataURI, err = gw.CreateURIWithWrongRoot()
	require.NoError(t, err)
	u, err = NewURI(dataURI)
	require.NoError(t, err)
	err = u.Validate(context.Background())
	require.ErrorIs(t, err, ErrInvalidCarFile)
}

func createDagService(t *testing.T) format.DAGService {
	dir := t.TempDir()
	fin := finalizer.NewFinalizer()
	t.Cleanup(func() {
		require.NoError(t, fin.Cleanup(nil))
	})
	p, err := marketpeer.New(marketpeer.Config{RepoPath: dir})
	require.NoError(t, err)
	fin.Add(p)
	return p.DAGService()
}
