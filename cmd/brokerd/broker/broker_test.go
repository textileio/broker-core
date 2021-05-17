package broker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multihash"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/textileio/broker-core/auctioneer"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/chainapi"
	"github.com/textileio/broker-core/cmd/brokerd/store"
	"github.com/textileio/broker-core/dealer"
	auctioneermocks "github.com/textileio/broker-core/mocks/auctioneer"
	chainapimocks "github.com/textileio/broker-core/mocks/chainapi"
	dealermocks "github.com/textileio/broker-core/mocks/dealer"
	packermocks "github.com/textileio/broker-core/mocks/packer"
	piecermocks "github.com/textileio/broker-core/mocks/piecer"
	"github.com/textileio/broker-core/packer"
	"github.com/textileio/broker-core/piecer"
	"github.com/textileio/broker-core/tests"
)

func TestCreateSuccess(t *testing.T) {
	t.Parallel()

	packer := defaultPackerMock(1)
	b := createBroker(
		t,
		packer,
		defaultPiecerMock(),
		defaultActioneerMock(),
		defaultDealerMock(),
		defaultChainAPIMock(),
	)

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
	packer.AssertExpectations(t)
}

func TestCreateFail(t *testing.T) {
	t.Parallel()

	t.Run("invalid cid", func(t *testing.T) {
		t.Parallel()
		b := createBroker(
			t,
			defaultPackerMock(1),
			defaultPiecerMock(),
			defaultActioneerMock(),
			defaultDealerMock(),
			defaultChainAPIMock(),
		)
		_, err := b.Create(context.Background(), cid.Undef, broker.Metadata{})
		require.Equal(t, ErrInvalidCid, err)
	})

	// TODO: create a failing test whenever we add
	// broker.Metadata() validation rules.
}

func TestCreateStorageDeal(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	piecer := defaultPiecerMock()
	b := createBroker(
		t,
		defaultPackerMock(2),
		piecer,
		defaultActioneerMock(),
		defaultDealerMock(),
		defaultChainAPIMock(),
	)

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
	piecer.AssertExpectations(t)

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
		b := createBroker(
			t,
			defaultPackerMock(1),
			defaultPiecerMock(),
			defaultActioneerMock(),
			defaultDealerMock(),
			defaultChainAPIMock(),
		)
		_, err := b.CreateStorageDeal(ctx, cid.Undef, nil)
		require.Equal(t, ErrInvalidCid, err)
	})

	t.Run("empty group", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		b := createBroker(
			t,
			defaultPackerMock(1),
			defaultPiecerMock(),
			defaultActioneerMock(),
			defaultDealerMock(),
			defaultChainAPIMock(),
		)
		brgCid := createCidFromString("StorageDeal")
		_, err := b.CreateStorageDeal(ctx, brgCid, nil)
		require.Equal(t, ErrEmptyGroup, err)
	})

	t.Run("group contains unknown broker request id", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		b := createBroker(
			t,
			defaultPackerMock(1),
			defaultPiecerMock(),
			defaultActioneerMock(),
			defaultDealerMock(),
			defaultChainAPIMock(),
		)

		brgCid := createCidFromString("StorageDeal")
		_, err := b.CreateStorageDeal(ctx, brgCid, []broker.BrokerRequestID{broker.BrokerRequestID("invented")})
		require.True(t, errors.Is(err, store.ErrStorageDealContainsUnknownBrokerRequest))
	})
}

func TestStorageDealPrepared(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	auctioneer := defaultActioneerMock()

	b := createBroker(
		t,
		defaultPackerMock(2),
		defaultPiecerMock(),
		auctioneer,
		defaultDealerMock(),
		defaultChainAPIMock(),
	)

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
	auctioneer.AssertExpectations(t)

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

	dealer := defaultDealerMock()
	b := createBroker(
		t,
		defaultPackerMock(2),
		defaultPiecerMock(),
		defaultActioneerMock(),
		dealer,
		defaultChainAPIMock(),
	)

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
			WalletAddr:       "miner1",
			AskPrice:         100,
			VerifiedAskPrice: 200,
			StartEpoch:       300,
			FastRetrieval:    true,
		},
		broker.BidID("Bid2"): {
			MinerAddr:        "f02222",
			WalletAddr:       "miner2",
			AskPrice:         1100,
			VerifiedAskPrice: 1200,
			StartEpoch:       1300,
			FastRetrieval:    false,
		},
	}
	auction := broker.Auction{
		ID:            broker.AuctionID("AUCTION1"),
		StorageDealID: sd,
		DealSize:      dpr.PieceSize,
		DealDuration:  broker.MaxDealEpochs,
		Status:        broker.AuctionStatusEnded,
		Bids:          bids,
		WinningBids:   []broker.BidID{broker.BidID("Bid1"), broker.BidID("Bid2")},
	}
	err = b.StorageDealAuctioned(ctx, auction)
	require.NoError(t, err)

	// 3- Verify the storage deal moved to the correct status
	sd2, err := b.GetStorageDeal(ctx, sd)
	require.NoError(t, err)
	require.Equal(t, broker.StorageDealDealMaking, sd2.Status)

	// 4- Verify that Dealer was called to execute the winning bids.
	dealer.AssertExpectations(t)

	// TODO: verify arg values passed to dealer and other places of interest in all the tests

	// calledADS := dealer.calledAuctionDeals
	// require.Equal(t, sd, calledADS.StorageDealID)
	// require.Equal(t, brgCid, calledADS.PayloadCid)
	// require.Equal(t, dpr.PieceCid, calledADS.PieceCid)
	// require.Equal(t, dpr.PieceSize, calledADS.PieceSize)
	// require.Equal(t, broker.MaxDealEpochs, calledADS.Duration)
	// require.Len(t, calledADS.Targets, 2)

	// require.Equal(t, bids[broker.BidID("Bid1")].WalletAddr, calledADS.Targets[0].Miner)
	// require.Equal(t, bids[broker.BidID("Bid1")].AskPrice, calledADS.Targets[0].PricePerGiBPerEpoch)
	// require.Equal(t, bids[broker.BidID("Bid1")].StartEpoch, calledADS.Targets[0].StartEpoch)
	// require.True(t, calledADS.Targets[0].Verified)
	// require.Equal(t, bids[broker.BidID("Bid1")].FastRetrieval, calledADS.Targets[0].FastRetrieval)

	// require.Equal(t, bids[broker.BidID("Bid2")].WalletAddr, calledADS.Targets[1].Miner)
	// require.Equal(t, bids[broker.BidID("Bid2")].AskPrice, calledADS.Targets[1].PricePerGiBPerEpoch)
	// require.Equal(t, bids[broker.BidID("Bid2")].StartEpoch, calledADS.Targets[1].StartEpoch)
	// require.True(t, calledADS.Targets[1].Verified)
	// require.Equal(t, bids[broker.BidID("Bid2")].FastRetrieval, calledADS.Targets[1].FastRetrieval)

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
	packer := defaultPackerMock(4)
	b := createBroker(
		t,
		packer,
		defaultPiecerMock(),
		defaultActioneerMock(),
		defaultDealerMock(),
		defaultChainAPIMock(),
	)

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

	// Empty the call stack of the  mocks, since it has calls from before.
	// dealerd.calledAuctionDeals = dealer.AuctionDeals{}

	// 2- Call StorageDealAuctioned as if the auctioneer did.
	auction := broker.Auction{
		ID:            broker.AuctionID("AUCTION1"),
		StorageDealID: sd,
		DealSize:      dpr.PieceSize,
		DealDuration:  broker.MaxDealEpochs,
		Status:        broker.AuctionStatusError,
		Error:         "reached max retries",
	}
	err = b.StorageDealAuctioned(ctx, auction)
	require.NoError(t, err)

	// 3- Verify the storage deal moved to the correct status with error cause.
	sd2, err := b.GetStorageDeal(ctx, sd)
	require.NoError(t, err)
	require.Equal(t, broker.StorageDealError, sd2.Status)
	require.Equal(t, auction.Error, sd2.Error)

	// 4- Verify that Dealer was NOT called
	// require.Equal(t, broker.StorageDealID(""), dealerd.calledAuctionDeals.StorageDealID)

	// 5- Verify that the underlying broker requests were moved
	//    to Batching again.
	mbr1, err := b.Get(ctx, br1.ID)
	require.NoError(t, err)
	require.Equal(t, broker.RequestBatching, mbr1.Status)
	mbr2, err := b.Get(ctx, br2.ID)
	require.NoError(t, err)
	require.Equal(t, broker.RequestBatching, mbr2.Status)

	// 6- Verify that Packerd was called as expected.
	packer.AssertExpectations(t)
}

func TestStorageDealFinalizedDeals(t *testing.T) {
	t.Parallel()
	ctx := context.Background()

	chainAPI := defaultChainAPIMock()
	b := createBroker(
		t,
		defaultPackerMock(2),
		defaultPiecerMock(),
		defaultActioneerMock(),
		defaultDealerMock(),
		chainAPI,
	)

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
			WalletAddr:       "miner1",
			AskPrice:         100,
			VerifiedAskPrice: 200,
			StartEpoch:       300,
			FastRetrieval:    true,
		},
		broker.BidID("Bid2"): {
			MinerAddr:        "f2",
			WalletAddr:       "miner2",
			AskPrice:         1100,
			VerifiedAskPrice: 1200,
			StartEpoch:       1300,
			FastRetrieval:    false,
		},
	}
	auction := broker.Auction{
		ID:            broker.AuctionID("AUCTION1"),
		StorageDealID: sd,
		DealSize:      dpr.PieceSize,
		DealDuration:  broker.MaxDealEpochs,
		Status:        broker.AuctionStatusEnded,
		Bids:          bids,
		WinningBids:   []broker.BidID{broker.BidID("Bid1"), broker.BidID("Bid2")},
	}
	err = b.StorageDealAuctioned(ctx, auction)
	require.NoError(t, err)

	// 2- Call StorageDealFinalizedDeals with the first deal having
	//    success. We'll make a further call to the second one.
	fads := []broker.FinalizedAuctionDeal{{
		StorageDealID:  auction.StorageDealID,
		DealID:         100,
		DealExpiration: 200,
		Miner:          "f0011",
	}}
	err = b.StorageDealFinalizedDeals(ctx, fads)
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

	// 4- Let's finalize the other one but with error. This results in a storage deal
	//    that had two winning bids, one of them succeeded and othe other failed deal making.
	fads = []broker.FinalizedAuctionDeal{{
		StorageDealID: auction.StorageDealID,
		Miner:         "f0012",
		ErrorCause:    "the miner rejected our proposal",
	}}
	err = b.StorageDealFinalizedDeals(ctx, fads)
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

	// 6- Verify that the chainAPI was called only once to report results on-chain.
	chainAPI.AssertExpectations(t)
}

func createBroker(
	t *testing.T,
	packer packer.Packer,
	piecer piecer.Piecer,
	auctioneer auctioneer.Auctioneer,
	dealer dealer.Dealer,
	chainAPI chainapi.ChainAPI,
) *Broker {
	ds := tests.NewTxMapDatastore()
	b, err := New(ds, packer, piecer, auctioneer, dealer, chainAPI, broker.MaxDealEpochs)
	require.NoError(t, err)

	return b
}

func defaultPackerMock(times int) *packermocks.Packer {
	p := &packermocks.Packer{}
	p.On(
		"ReadyToPack",
		mock.Anything,
		mock.AnythingOfType("broker.BrokerRequestID"),
		mock.AnythingOfType("cid.Cid"),
	).Return(nil).Times(times)
	return p
}

func defaultPiecerMock() *piecermocks.Piecer {
	p := &piecermocks.Piecer{}
	p.On(
		"ReadyToPrepare",
		mock.Anything,
		mock.AnythingOfType("broker.StorageDealID"),
		mock.AnythingOfType("cid.Cid"),
	).Return(nil).Once()
	return p
}

func defaultActioneerMock() *auctioneermocks.Auctioneer {
	a := &auctioneermocks.Auctioneer{}
	a.On(
		"ReadyToAuction",
		mock.Anything,
		mock.AnythingOfType("broker.StorageDealID"),
		mock.AnythingOfType("uint64"),
		mock.AnythingOfType("uint64"),
	).Return(broker.AuctionID("AUCTION1"), nil).Once()
	return a
}

func defaultDealerMock() *dealermocks.Dealer {
	d := &dealermocks.Dealer{}
	d.On(
		"ReadyToCreateDeals",
		mock.Anything,
		mock.AnythingOfType("dealer.AuctionDeals"),
	).Return(nil).Once()
	return d
}

func defaultChainAPIMock() *chainapimocks.ChainAPI {
	c := &chainapimocks.ChainAPI{}
	c.On(
		"UpdatePayload",
		mock.Anything,
		mock.AnythingOfType("cid.Cid"),
		mock.AnythingOfType("chainapi.UpdatePayloadOption"),
		mock.AnythingOfType("chainapi.UpdatePayloadOption"),
		mock.AnythingOfType("chainapi.UpdatePayloadOption"),
	).Return(nil).Once()
	return c
}

func createCidFromString(s string) cid.Cid {
	mh, _ := multihash.Encode([]byte(s), multihash.SHA2_256)
	return cid.NewCidV1(cid.Raw, multihash.Multihash(mh))
}
