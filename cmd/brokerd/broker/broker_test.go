package broker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
	"github.com/stretchr/testify/require"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/brokerd/srstore"
	"github.com/textileio/broker-core/packer"
	"github.com/textileio/broker-core/piecer"
	"github.com/textileio/broker-core/tests"
)

func TestCreateSuccess(t *testing.T) {
	t.Parallel()

	b, p, _ := createBroker(t)
	c := createCidFromString("BrokerRequest1")

	meta := broker.Metadata{Region: "Region1"}
	br, err := b.Create(context.Background(), c, meta)
	require.NoError(t, err)
	require.NotEmpty(t, br.ID)
	require.Equal(t, broker.RequestBatching, br.Status)
	require.Equal(t, meta, br.Metadata)
	require.True(t, time.Since(br.CreatedAt).Seconds() < 5)
	require.True(t, time.Since(br.UpdatedAt).Seconds() < 5)

	// Check that the packer was notified.
	require.Len(t, p.readyToPackCalled, 1)
}

func TestCreateFail(t *testing.T) {
	t.Parallel()

	t.Run("invalid cid", func(t *testing.T) {
		t.Parallel()
		b, _, _ := createBroker(t)
		_, err := b.Create(context.Background(), cid.Undef, broker.Metadata{})
		require.Equal(t, ErrInvalidCid, err)
	})

	// TODO: create a failing test whenever we add
	// broker.Metadata() validation rules.
}

func TestCreateStorageDeal(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	b, _, p := createBroker(t)

	// 1- Create two broker requests.
	c := createCidFromString("BrokerRequest1")
	br1, err := b.Create(ctx, c, broker.Metadata{})
	require.NoError(t, err)
	require.Equal(t, broker.RequestBatching, br1.Status)

	c = createCidFromString("BrokerRequest2")
	br2, err := b.Create(ctx, c, broker.Metadata{})
	require.NoError(t, err)
	require.Equal(t, broker.RequestBatching, br1.Status)

	// 2- Create a StorageDeal with both storage requests.
	brgCid := createCidFromString("StorageDeal")
	srg := broker.BrokerRequestGroup{
		Cid:                    brgCid,
		GroupedStorageRequests: []broker.BrokerRequestID{br1.ID, br2.ID},
	}
	sd, err := b.CreateStorageDeal(ctx, srg)
	require.NoError(t, err)

	// Check that what Piecer was notified about matches
	// the BrokerRequest group to be prepared.
	require.Equal(t, brgCid, p.calledCid)
	require.Equal(t, sd.ID, p.calledID)

	// Check that all broker request:
	// 1- Moved to StatusPreparing
	// 2- Are linked to the StorageDeal they are now part of.
	br1, err = b.Get(ctx, br1.ID)
	require.NoError(t, err)
	require.Equal(t, broker.RequestPreparing, br1.Status)
	require.Equal(t, sd.ID, br1.StorageDealID)
	br2, err = b.Get(ctx, br2.ID)
	require.NoError(t, err)
	require.Equal(t, broker.RequestPreparing, br2.Status)
	require.Equal(t, sd.ID, br2.StorageDealID)

	// Check that the StorageDeal was persisted correctly.
	sd2, err := b.GetStorageDeal(ctx, sd.ID)
	require.NoError(t, err)
	require.Equal(t, sd.ID, sd2.ID)
	require.Equal(t, sd.Cid, sd2.Cid)
	require.Equal(t, sd.Status, sd2.Status)
	require.Equal(t, len(sd.BrokerRequestIDs), len(sd2.BrokerRequestIDs))
	require.Equal(t, sd.CreatedAt.Unix(), sd2.CreatedAt.Unix())
	require.Equal(t, sd.UpdatedAt.Unix(), sd2.UpdatedAt.Unix())
}

func TestCreateStorageDealFail(t *testing.T) {
	t.Parallel()

	t.Run("invalid cid", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		b, _, _ := createBroker(t)
		srb := broker.BrokerRequestGroup{Cid: cid.Undef}
		_, err := b.CreateStorageDeal(ctx, srb)
		require.Equal(t, ErrInvalidCid, err)
	})

	t.Run("empty group", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		b, _, _ := createBroker(t)
		brgCid := createCidFromString("StorageDeal")
		srb := broker.BrokerRequestGroup{Cid: brgCid}
		_, err := b.CreateStorageDeal(ctx, srb)
		require.Equal(t, ErrEmptyGroup, err)
	})

	t.Run("group contains unknown broker request id", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		b, _, _ := createBroker(t)

		brgCid := createCidFromString("StorageDeal")
		srb := broker.BrokerRequestGroup{
			Cid:                    brgCid,
			GroupedStorageRequests: []broker.BrokerRequestID{broker.BrokerRequestID("INVENTED")},
		}
		_, err := b.CreateStorageDeal(ctx, srb)
		require.True(t, errors.Is(err, srstore.ErrStorageDealContainsUnknownBrokerRequest))
	})
}

func createBroker(t *testing.T) (*Broker, *dumbPacker, *dumbPiecer) {
	ds := tests.NewTxMapDatastore()
	packer := &dumbPacker{}
	piecer := &dumbPiecer{}
	auctioneer := &dumbAuctioneer{}
	b, err := New(ds, packer, piecer, auctioneer)
	require.NoError(t, err)

	return b, packer, piecer
}

type dumbPacker struct {
	readyToPackCalled []broker.BrokerRequest
}

var _ packer.Packer = (*dumbPacker)(nil)

func (dp *dumbPacker) ReadyToPack(ctx context.Context, br broker.BrokerRequest) error {
	dp.readyToPackCalled = append(dp.readyToPackCalled, br)

	return nil
}

type dumbPiecer struct {
	calledCid cid.Cid
	calledID  broker.StorageDealID
}

var _ piecer.Piecer = (*dumbPiecer)(nil)

func (dp *dumbPiecer) ReadyToPrepare(ctx context.Context, id broker.StorageDealID, c cid.Cid) error {
	dp.calledCid = c
	dp.calledID = id

	return nil
}

type dumbAuctioneer struct {
}

func (dp *dumbAuctioneer) ReadyToAuction(ctx context.Context, sd broker.StorageDeal) error {
	return nil
}

func createCidFromString(s string) cid.Cid {
	mh, _ := multihash.Encode([]byte(s), multihash.SHA2_256)
	return cid.NewCidV1(cid.Raw, multihash.Multihash(mh))
}
