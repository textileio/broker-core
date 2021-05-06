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
	"github.com/textileio/broker-core/cmd/brokerd/store"
	"github.com/textileio/broker-core/dealer"
	"github.com/textileio/broker-core/packer"
	"github.com/textileio/broker-core/piecer"
	"github.com/textileio/broker-core/tests"
)

func TestCreateSuccess(t *testing.T) {
	t.Parallel()

	b, packer, _, _ := createBroker(t)
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
	require.Len(t, packer.readyToPackCalled, 1)
}

func TestCreateFail(t *testing.T) {
	t.Parallel()

	t.Run("invalid cid", func(t *testing.T) {
		t.Parallel()
		b, _, _, _ := createBroker(t)
		_, err := b.Create(context.Background(), cid.Undef, broker.Metadata{})
		require.Equal(t, ErrInvalidCid, err)
	})

	// TODO: create a failing test whenever we add
	// broker.Metadata() validation rules.
}

func TestCreateStorageDeal(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	b, _, piecer, _ := createBroker(t)

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
	sd, err := b.CreateStorageDeal(ctx, brgCid, []broker.BrokerRequestID{br1.ID, br2.ID})
	require.NoError(t, err)

	// Check that what Piecer was notified about matches
	// the BrokerRequest group to be prepared.
	require.Equal(t, brgCid, piecer.calledCid)
	require.Equal(t, sd, piecer.calledID)

	// Check that all broker request:
	// 1- Moved to StatusPreparing
	// 2- Are linked to the StorageDeal they are now part of.
	br1, err = b.Get(ctx, br1.ID)
	require.NoError(t, err)
	require.Equal(t, broker.RequestPreparing, br1.Status)
	require.Equal(t, sd, br1.StorageDealID)
	br2, err = b.Get(ctx, br2.ID)
	require.NoError(t, err)
	require.Equal(t, broker.RequestPreparing, br2.Status)
	require.Equal(t, sd, br2.StorageDealID)

	// Check that the StorageDeal was persisted correctly.
	sd2, err := b.GetStorageDeal(ctx, sd)
	require.NoError(t, err)
	require.Equal(t, sd, sd2.ID)
	require.True(t, sd2.PayloadCid.Defined())
	require.Equal(t, broker.StorageDealPreparing, sd2.Status)
	require.Len(t, sd2.BrokerRequestIDs, 2)
	require.True(t, time.Since(sd2.CreatedAt) < time.Minute)
	require.True(t, time.Since(sd2.UpdatedAt) < time.Minute)
}

func TestCreateStorageDealFail(t *testing.T) {
	t.Parallel()

	t.Run("invalid cid", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		b, _, _, _ := createBroker(t)
		_, err := b.CreateStorageDeal(ctx, cid.Undef, nil)
		require.Equal(t, ErrInvalidCid, err)
	})

	t.Run("empty group", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		b, _, _, _ := createBroker(t)
		brgCid := createCidFromString("StorageDeal")
		_, err := b.CreateStorageDeal(ctx, brgCid, nil)
		require.Equal(t, ErrEmptyGroup, err)
	})

	t.Run("group contains unknown broker request id", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		b, _, _, _ := createBroker(t)

		brgCid := createCidFromString("StorageDeal")
		_, err := b.CreateStorageDeal(ctx, brgCid, []broker.BrokerRequestID{broker.BrokerRequestID("invented")})
		require.True(t, errors.Is(err, store.ErrStorageDealContainsUnknownBrokerRequest))
	})
}

func TestStorageDealPrepared(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	b, _, _, auctioneer := createBroker(t)

	// 1- Create two broker requests and a corresponding storage deal.
	c := createCidFromString("BrokerRequest1")
	br1, err := b.Create(ctx, c, broker.Metadata{})
	require.NoError(t, err)
	c = createCidFromString("BrokerRequest2")
	br2, err := b.Create(ctx, c, broker.Metadata{})
	require.NoError(t, err)
	brgCid := createCidFromString("StorageDeal")
	sd, err := b.CreateStorageDeal(ctx, brgCid, []broker.BrokerRequestID{br1.ID, br2.ID})
	require.NoError(t, err)

	// 2- Call StorageDealPrepared as if the packer did.
	dpr := broker.DataPreparationResult{
		PieceSize: uint64(123456),
		PieceCid:  createCidFromString("piececid1"),
	}
	err = b.StorageDealPrepared(ctx, sd, dpr)
	require.NoError(t, err)

	// 3- Verify the storage deal moved to the correct status, and is linked
	//    to the auction id.
	sd2, err := b.GetStorageDeal(ctx, sd)
	require.NoError(t, err)
	require.Equal(t, broker.StorageDealAuctioning, sd2.Status)
	require.Equal(t, broker.AuctionID("AUCTION1"), sd2.Auction.ID)

	// 4- Verify that Auctioneer was called to prepare the data.
	require.Equal(t, broker.MaxDealEpochs, auctioneer.calledDealDuration)
	require.Equal(t, dpr.PieceSize, auctioneer.calledPieceSize)
	require.Equal(t, sd, auctioneer.calledStorageDealID)

	// 5- Verify that the underlying broker requests also moved to
	//    their correct statuses.
	mbr1, err := b.Get(ctx, br1.ID)
	require.NoError(t, err)
	require.Equal(t, broker.RequestAuctioning, mbr1.Status)
	mbr2, err := b.Get(ctx, br2.ID)
	require.NoError(t, err)
	require.Equal(t, broker.RequestAuctioning, mbr2.Status)
}

func createBroker(t *testing.T) (*Broker, *dumbPacker, *dumbPiecer, *dumbAuctioneer) {
	ds := tests.NewTxMapDatastore()
	packer := &dumbPacker{}
	piecer := &dumbPiecer{}
	auctioneer := &dumbAuctioneer{}
	dealer := &dumbDealer{}
	b, err := New(ds, packer, piecer, auctioneer, dealer, broker.MaxDealEpochs)
	require.NoError(t, err)

	return b, packer, piecer, auctioneer
}

type dumbPacker struct {
	readyToPackCalled []brc
}

type brc struct {
	id      broker.BrokerRequestID
	dataCid cid.Cid
}

var _ packer.Packer = (*dumbPacker)(nil)

func (dp *dumbPacker) ReadyToPack(ctx context.Context, id broker.BrokerRequestID, dataCid cid.Cid) error {
	dp.readyToPackCalled = append(dp.readyToPackCalled, brc{id: id, dataCid: dataCid})

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
	calledStorageDealID broker.StorageDealID
	calledPieceSize     uint64
	calledDealDuration  uint64
}

func (dp *dumbAuctioneer) GetAuction(ctx context.Context, id broker.AuctionID) (broker.Auction, error) {
	panic("shouldn't be called")
}

func (dp *dumbAuctioneer) ReadyToAuction(
	ctx context.Context,
	id broker.StorageDealID,
	dealSize, dealDuration uint64,
) (broker.AuctionID, error) {
	dp.calledStorageDealID = id
	dp.calledPieceSize = dealSize
	dp.calledDealDuration = dealDuration
	return broker.AuctionID("AUCTION1"), nil
}

type dumbDealer struct {
}

var _ dealer.Dealer = (*dumbDealer)(nil)

func (dd *dumbDealer) ReadyToCreateDeals(ctx context.Context, sdb dealer.AuctionDeals) error {
	panic("shouldn't be called")
}

func createCidFromString(s string) cid.Cid {
	mh, _ := multihash.Encode([]byte(s), multihash.SHA2_256)
	return cid.NewCidV1(cid.Raw, multihash.Multihash(mh))
}
