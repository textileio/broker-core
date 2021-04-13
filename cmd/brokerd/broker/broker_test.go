package broker

import (
	"context"
	"testing"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
	"github.com/stretchr/testify/require"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/packer"
	"github.com/textileio/broker-core/tests"
)

func TestCreateSuccess(t *testing.T) {
	t.Parallel()

	b, p := createBroker(t)
	mh, _ := multihash.Encode([]byte("AAA"), multihash.SHA2_256)
	c := cid.NewCidV1(cid.Raw, multihash.Multihash(mh))

	meta := broker.Metadata{Region: "Region1"}
	br, err := b.Create(context.Background(), c, meta)
	require.NoError(t, err)
	require.NotEmpty(t, br.ID)
	require.Equal(t, broker.StatusBatching, br.Status)
	require.Equal(t, meta, br.Metadata)

	// Check that the packer was notified.
	require.Len(t, p.readyToPackCalled, 1)
}

func TestCreateFail(t *testing.T) {
	t.Parallel()

	t.Run("invalid cid", func(t *testing.T) {
		t.Parallel()
		b, _ := createBroker(t)
		_, err := b.Create(context.Background(), cid.Undef, broker.Metadata{})
		require.Equal(t, ErrInvalidCid, err)
	})

	// TODO: create a failing test whenever we add
	// broker.Metadata() validation rules.
}

func createBroker(t *testing.T) (*Broker, *dumbPacker) {
	ds := tests.NewTxMapDatastore()
	packer := &dumbPacker{}
	b, err := New(ds, packer)
	require.NoError(t, err)

	return b, packer
}

type dumbPacker struct {
	err               error
	readyToPackCalled []broker.BrokerRequest
}

var _ packer.Packer = (*dumbPacker)(nil)

func (dp *dumbPacker) ReadyToPack(br broker.BrokerRequest) error {
	if dp.err != nil {
		return dp.err
	}
	dp.readyToPackCalled = append(dp.readyToPackCalled, br)

	return nil
}
