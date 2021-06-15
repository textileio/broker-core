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
	"github.com/textileio/broker-core/chainapi"
	"github.com/textileio/broker-core/cmd/brokerd/store"
	"github.com/textileio/broker-core/dealer"
	"github.com/textileio/broker-core/packer"
	"github.com/textileio/broker-core/piecer"
	"github.com/textileio/broker-core/tests"
)

func TestCreateSuccess(t *testing.T) {
	t.Parallel()

	b, packer, _, _, _, _ := createBroker(t)
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
	require.Len(t, packer.calledBrokerRequestIDs, 1)
}

func TestCreateFail(t *testing.T) {
	t.Parallel()

	t.Run("invalid cid", func(t *testing.T) {
		t.Parallel()
		b, _, _, _, _, _ := createBroker(t)
		_, err := b.Create(context.Background(), cid.Undef, broker.Metadata{})
		require.Equal(t, ErrInvalidCid, err)
	})

	// TODO: create a failing test whenever we add
	// broker.Metadata() validation rules.
}

func TestCreateStorageDeal(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	b, _, piecer, _, _, _ := createBroker(t)

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
		b, _, _, _, _, _ := createBroker(t)
		_, err := b.CreateStorageDeal(ctx, cid.Undef, nil)
		require.Equal(t, ErrInvalidCid, err)
	})

	t.Run("empty group", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		b, _, _, _, _, _ := createBroker(t)
		brgCid := createCidFromString("StorageDeal")
		_, err := b.CreateStorageDeal(ctx, brgCid, nil)
		require.Equal(t, ErrEmptyGroup, err)
	})

	t.Run("group contains unknown broker request id", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		b, _, _, _, _, _ := createBroker(t)

		brgCid := createCidFromString("StorageDeal")
		_, err := b.CreateStorageDeal(ctx, brgCid, []broker.BrokerRequestID{broker.BrokerRequestID("invented")})
		require.True(t, errors.Is(err, store.ErrStorageDealContainsUnknownBrokerRequest))
	})
}

func TestStorageDealPrepared(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	b, _, _, auctioneer, _, _ := createBroker(t)

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

	// 2- Call StorageDealPrepared as if the piecer did.
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
	require.Equal(t, broker.MaxDealDuration, uint64(auctioneer.calledDealDuration))
	require.Equal(t, dpr.PieceSize, uint64(auctioneer.calledPieceSize))
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

func TestStorageDealAuctioned(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	b, _, _, _, dealer, _ := createBroker(t)

	// 1- Create two broker requests and a corresponding storage deal, and
	//    pass through prepared.
	c := createCidFromString("BrokerRequest1")
	br1, err := b.Create(ctx, c, broker.Metadata{})
	require.NoError(t, err)
	c = createCidFromString("BrokerRequest2")
	br2, err := b.Create(ctx, c, broker.Metadata{})
	require.NoError(t, err)
	brgCid := createCidFromString("StorageDeal")
	sd, err := b.CreateStorageDeal(ctx, brgCid, []broker.BrokerRequestID{br1.ID, br2.ID})
	require.NoError(t, err)
	dpr := broker.DataPreparationResult{
		PieceSize: uint64(123456),
		PieceCid:  createCidFromString("piececid1"),
	}
	err = b.StorageDealPrepared(ctx, sd, dpr)
	require.NoError(t, err)

	// 2- Call StorageDealAuctioned as if the auctioneer did.
	bids := map[broker.BidID]broker.Bid{
		broker.BidID("Bid1"): {
			MinerAddr:        "f01111",
			AskPrice:         100,
			VerifiedAskPrice: 200,
			StartEpoch:       300,
			FastRetrieval:    true,
		},
		broker.BidID("Bid2"): {
			MinerAddr:        "f02222",
			AskPrice:         1100,
			VerifiedAskPrice: 1200,
			StartEpoch:       1300,
			FastRetrieval:    false,
		},
	}

	auction := broker.Auction{
		ID:              broker.AuctionID("AUCTION1"),
		StorageDealID:   sd,
		DealSize:        dpr.PieceSize,
		DealDuration:    broker.MaxDealDuration,
		DealReplication: 2,
		DealVerified:    true,
		Status:          broker.AuctionStatusFinalized,
		Bids:            bids,
		WinningBids: map[broker.BidID]broker.WinningBid{
			broker.BidID("Bid1"): {},
			broker.BidID("Bid2"): {},
		},
	}
	err = b.StorageDealAuctioned(ctx, auction)
	require.NoError(t, err)

	// 3- Verify the storage deal moved to the correct status
	sd2, err := b.GetStorageDeal(ctx, sd)
	require.NoError(t, err)
	require.Equal(t, broker.StorageDealDealMaking, sd2.Status)

	// 4- Verify that Dealer was called to execute the winning bids.
	calledADS := dealer.calledAuctionDeals
	require.Equal(t, sd, calledADS.StorageDealID)
	require.Equal(t, brgCid, calledADS.PayloadCid)
	require.Equal(t, dpr.PieceCid, calledADS.PieceCid)
	require.Equal(t, dpr.PieceSize, calledADS.PieceSize)
	require.Equal(t, broker.MaxDealDuration, calledADS.Duration)
	require.Len(t, calledADS.Targets, 2)

	for _, tr := range calledADS.Targets {
		var bid broker.Bid
		for _, b := range bids {
			if tr.Miner == b.MinerAddr {
				bid = b
				break
			}
		}
		if bid.MinerAddr == "" {
			t.Errorf("AuctionDealsTarget has no corresponding Bid")
		}
		require.Equal(t, bid.AskPrice, tr.PricePerGiBPerEpoch)
		require.Equal(t, bid.StartEpoch, tr.StartEpoch)
		require.True(t, tr.Verified)
		require.Equal(t, bid.FastRetrieval, tr.FastRetrieval)
	}

	// 5- Verify that the underlying broker requests also moved to
	//    their correct statuses.
	mbr1, err := b.Get(ctx, br1.ID)
	require.NoError(t, err)
	require.Equal(t, broker.RequestDealMaking, mbr1.Status)
	mbr2, err := b.Get(ctx, br2.ID)
	require.NoError(t, err)
	require.Equal(t, broker.RequestDealMaking, mbr2.Status)
}

func TestStorageDealFailedAuction(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	b, packer, _, _, dealerd, _ := createBroker(t)

	// 1- Create two broker requests and a corresponding storage deal, and
	//    pass through prepared.
	c := createCidFromString("BrokerRequest1")
	br1, err := b.Create(ctx, c, broker.Metadata{})
	require.NoError(t, err)
	c = createCidFromString("BrokerRequest2")
	br2, err := b.Create(ctx, c, broker.Metadata{})
	require.NoError(t, err)
	brgCid := createCidFromString("StorageDeal")
	sd, err := b.CreateStorageDeal(ctx, brgCid, []broker.BrokerRequestID{br1.ID, br2.ID})
	require.NoError(t, err)
	dpr := broker.DataPreparationResult{
		PieceSize: uint64(123456),
		PieceCid:  createCidFromString("piececid1"),
	}
	err = b.StorageDealPrepared(ctx, sd, dpr)
	require.NoError(t, err)

	// Empty the call stack of the mocks, since it has calls from before.
	dealerd.calledAuctionDeals = dealer.AuctionDeals{}
	packer.calledBrokerRequestIDs = nil

	// 2- Call StorageDealAuctioned as if the auctioneer did.
	auction := broker.Auction{
		ID:              broker.AuctionID("AUCTION1"),
		StorageDealID:   sd,
		DealSize:        dpr.PieceSize,
		DealDuration:    broker.MaxDealDuration,
		DealReplication: 1,
		DealVerified:    true,
		Status:          broker.AuctionStatusFinalized,
		ErrorCause:      "reached max retries",
	}
	err = b.StorageDealAuctioned(ctx, auction)
	require.NoError(t, err)

	// 3- Verify the storage deal moved to the correct status with error cause.
	sd2, err := b.GetStorageDeal(ctx, sd)
	require.NoError(t, err)
	require.Equal(t, broker.StorageDealError, sd2.Status)
	require.Equal(t, auction.ErrorCause, sd2.Error)

	// 4- Verify that Dealer was NOT called
	require.Equal(t, broker.StorageDealID(""), dealerd.calledAuctionDeals.StorageDealID)

	// 5- Verify that the underlying broker requests were moved
	//    to Batching again.
	mbr1, err := b.Get(ctx, br1.ID)
	require.NoError(t, err)
	require.Equal(t, broker.RequestBatching, mbr1.Status)
	mbr2, err := b.Get(ctx, br2.ID)
	require.NoError(t, err)
	require.Equal(t, broker.RequestBatching, mbr2.Status)

	// 6- Verify that Packerd was called again with the new liberated broker requests
	//    that can be batched again.
	require.Len(t, packer.calledBrokerRequestIDs, 2)
	require.Equal(t, br1.ID, packer.calledBrokerRequestIDs[0].id)
	require.Equal(t, br1.DataCid, packer.calledBrokerRequestIDs[0].dataCid)
	require.Equal(t, br2.ID, packer.calledBrokerRequestIDs[1].id)
	require.Equal(t, br2.DataCid, packer.calledBrokerRequestIDs[1].dataCid)
}

func TestStorageDealFinalizedDeals(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	b, _, _, _, _, chainAPI := createBroker(t)

	// 1- Create two broker requests and a corresponding storage deal, and
	//    pass through prepared, auctioned, and deal making.
	c1 := createCidFromString("BrokerRequest1")
	br1, err := b.Create(ctx, c1, broker.Metadata{})
	require.NoError(t, err)
	c2 := createCidFromString("BrokerRequest2")
	br2, err := b.Create(ctx, c2, broker.Metadata{})
	require.NoError(t, err)
	brgCid := createCidFromString("StorageDeal")
	sd, err := b.CreateStorageDeal(ctx, brgCid, []broker.BrokerRequestID{br1.ID, br2.ID})
	require.NoError(t, err)
	dpr := broker.DataPreparationResult{
		PieceSize: uint64(123456),
		PieceCid:  createCidFromString("piececid1"),
	}
	err = b.StorageDealPrepared(ctx, sd, dpr)
	require.NoError(t, err)

	bids := map[broker.BidID]broker.Bid{
		broker.BidID("Bid1"): {
			MinerAddr:        "f1",
			AskPrice:         100,
			VerifiedAskPrice: 200,
			StartEpoch:       300,
			FastRetrieval:    true,
		},
		broker.BidID("Bid2"): {
			MinerAddr:        "f2",
			AskPrice:         1100,
			VerifiedAskPrice: 1200,
			StartEpoch:       1300,
			FastRetrieval:    false,
		},
	}
	auction := broker.Auction{
		ID:              broker.AuctionID("AUCTION1"),
		StorageDealID:   sd,
		DealSize:        dpr.PieceSize,
		DealDuration:    broker.MaxDealDuration,
		DealReplication: 2,
		Status:          broker.AuctionStatusFinalized,
		Bids:            bids,
		WinningBids: map[broker.BidID]broker.WinningBid{
			broker.BidID("Bid1"): {},
			broker.BidID("Bid2"): {},
		},
	}
	err = b.StorageDealAuctioned(ctx, auction)
	require.NoError(t, err)

	// 2- Call StorageDealFinalizedDeals with the first deal having
	//    success. We'll make a further call to the second one.
	fad := broker.FinalizedAuctionDeal{
		StorageDealID:  auction.StorageDealID,
		DealID:         100,
		DealExpiration: 200,
		Miner:          "f0011",
	}
	err = b.StorageDealFinalizedDeal(ctx, fad)
	require.NoError(t, err)

	// 3- Verify the storage deal and underlying broker request are still in deal making,
	//    since there's another pending deal to be reported.
	sd2, err := b.GetStorageDeal(ctx, sd)
	require.NoError(t, err)
	require.Equal(t, broker.StorageDealDealMaking, sd2.Status)
	mbr1, err := b.Get(ctx, br1.ID)
	require.NoError(t, err)
	require.Equal(t, broker.RequestDealMaking, mbr1.Status)
	mbr2, err := b.Get(ctx, br2.ID)
	require.NoError(t, err)
	require.Equal(t, broker.RequestDealMaking, mbr2.Status)

	chainAPI.clean() // clean the previous call stack
	// 4- Let's finalize the other one but with error. This results in a storage deal
	//    that had two winning bids, one of them succeeded and othe other failed deal making.
	fad = broker.FinalizedAuctionDeal{
		StorageDealID: auction.StorageDealID,
		Miner:         "f0012",
		ErrorCause:    "the miner rejected our proposal",
	}
	err = b.StorageDealFinalizedDeal(ctx, fad)
	require.NoError(t, err)

	// 5- Verify that the storage deal switched to Success, since at least one of the winning bids
	//    succeeded.
	sd2, err = b.GetStorageDeal(ctx, sd)
	require.NoError(t, err)
	require.Equal(t, broker.StorageDealSuccess, sd2.Status)
	mbr1, err = b.Get(ctx, br1.ID)
	require.NoError(t, err)
	require.Equal(t, broker.RequestSuccess, mbr1.Status)
	mbr2, err = b.Get(ctx, br2.ID)
	require.NoError(t, err)
	require.Equal(t, broker.RequestSuccess, mbr2.Status)
}

func createBroker(t *testing.T) (
	*Broker,
	*dumbPacker,
	*dumbPiecer,
	*dumbAuctioneer,
	*dumbDealer,
	*dumbChainAPI) {
	ds := tests.NewTxMapDatastore()
	packer := &dumbPacker{}
	piecer := &dumbPiecer{}
	auctioneer := &dumbAuctioneer{}
	dealer := &dumbDealer{}
	chainAPI := &dumbChainAPI{}
	b, err := New(
		ds,
		packer,
		piecer,
		auctioneer,
		dealer,
		chainAPI,
		broker.MaxDealDuration,
		broker.MinDealReplication,
		true,
	)
	require.NoError(t, err)

	return b, packer, piecer, auctioneer, dealer, chainAPI
}

type dumbPacker struct {
	calledBrokerRequestIDs []brc
}

type brc struct {
	id      broker.BrokerRequestID
	dataCid cid.Cid
}

var _ packer.Packer = (*dumbPacker)(nil)

func (dp *dumbPacker) ReadyToPack(ctx context.Context, id broker.BrokerRequestID, dataCid cid.Cid) error {
	dp.calledBrokerRequestIDs = append(dp.calledBrokerRequestIDs, brc{id: id, dataCid: dataCid})

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
	calledStorageDealID   broker.StorageDealID
	calledDataCid         cid.Cid
	calledPieceSize       int
	calledDealDuration    int
	calledDealReplication int
	calledDealVerified    bool
}

func (dp *dumbAuctioneer) ReadyToAuction(
	ctx context.Context,
	id broker.StorageDealID,
	dataCid cid.Cid,
	dealSize, dealDuration, dealReplication int,
	dealVerified bool,
) (broker.AuctionID, error) {
	dp.calledStorageDealID = id
	dp.calledDataCid = dataCid
	dp.calledPieceSize = dealSize
	dp.calledDealDuration = dealDuration
	dp.calledDealReplication = dealReplication
	dp.calledDealVerified = dealVerified
	return broker.AuctionID("AUCTION1"), nil
}

func (dp *dumbAuctioneer) GetAuction(ctx context.Context, id broker.AuctionID) (broker.Auction, error) {
	panic("shouldn't be called")
}

func (dp *dumbAuctioneer) ProposalAccepted(context.Context, broker.AuctionID, broker.BidID, cid.Cid) error {
	panic("shouldn't be called")
}

type dumbDealer struct {
	calledAuctionDeals dealer.AuctionDeals
}

var _ dealer.Dealer = (*dumbDealer)(nil)

func (dd *dumbDealer) ReadyToCreateDeals(ctx context.Context, ads dealer.AuctionDeals) error {
	dd.calledAuctionDeals = ads
	return nil
}

func createCidFromString(s string) cid.Cid {
	mh, _ := multihash.Encode([]byte(s), multihash.SHA2_256)
	return cid.NewCidV1(cid.Raw, multihash.Multihash(mh))
}

type dumbChainAPI struct {
}

var _ chainapi.ChainAPI = (*dumbChainAPI)(nil)

func (dr *dumbChainAPI) HasDeposit(ctx context.Context, brokerID, accountID string) (bool, error) {
	return true, nil
}

func (dr *dumbChainAPI) clean() {
}
