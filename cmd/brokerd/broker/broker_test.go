package broker

import (
	"context"
	"fmt"
	"net/url"
	"sort"
	"testing"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/multiformats/go-multiaddr"
	"github.com/multiformats/go-multihash"
	"github.com/stretchr/testify/require"
	"github.com/textileio/bidbot/lib/auction"
	"github.com/textileio/broker-core/auctioneer"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/chainapi"
	"github.com/textileio/broker-core/cmd/brokerd/store"
	"github.com/textileio/broker-core/dealer"
	pb "github.com/textileio/broker-core/gen/broker/v1"
	mbroker "github.com/textileio/broker-core/msgbroker"
	"github.com/textileio/broker-core/msgbroker/fakemsgbroker"
	"github.com/textileio/broker-core/tests"
	"google.golang.org/protobuf/proto"
)

func TestCreateBrokerRequestSuccess(t *testing.T) {
	t.Parallel()

	b, mb, _, _, _ := createBroker(t)
	c := createCidFromString("BrokerRequest1")

	br, err := b.Create(context.Background(), c)
	require.NoError(t, err)
	require.NotEmpty(t, br.ID)
	require.Equal(t, broker.RequestBatching, br.Status)
	require.True(t, time.Since(br.CreatedAt).Seconds() < 5)
	require.True(t, time.Since(br.UpdatedAt).Seconds() < 5)

	require.Equal(t, 1, mb.TotalPublished())
	require.Equal(t, 1, mb.TotalPublishedTopic(mbroker.ReadyToBatchTopic))
	data, err := mb.GetMsg(mbroker.ReadyToBatchTopic, 0)
	require.NoError(t, err)
	r := &pb.ReadyToBatch{}
	err = proto.Unmarshal(data, r)
	require.NoError(t, err)
	require.Len(t, r.DataCids, 1)
	require.NotEmpty(t, r.DataCids[0].BrokerRequestId)
	dataCid, err := cid.Cast(r.DataCids[0].DataCid)
	require.NoError(t, err)
	require.Equal(t, c.String(), dataCid.String())
}

func TestCreateBrokerRequestFail(t *testing.T) {
	t.Parallel()

	t.Run("invalid cid", func(t *testing.T) {
		t.Parallel()
		b, _, _, _, _ := createBroker(t)
		_, err := b.Create(context.Background(), cid.Undef)
		require.Equal(t, ErrInvalidCid, err)
	})
}

func TestCreateStorageDeal(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	b, _, _, _, _ := createBroker(t)

	// 1- Create two broker requests.
	c := createCidFromString("BrokerRequest1")
	br1, err := b.Create(ctx, c)
	require.NoError(t, err)
	require.Equal(t, broker.RequestBatching, br1.Status)

	c = createCidFromString("BrokerRequest2")
	br2, err := b.Create(ctx, c)
	require.NoError(t, err)
	require.Equal(t, broker.RequestBatching, br1.Status)

	// 2- Create a StorageDeal with both storage requests.
	brgCid := createCidFromString("StorageDeal")
	sd, err := b.CreateNewBatch(ctx, "SD1", brgCid, []broker.BrokerRequestID{br1.ID, br2.ID})
	require.NoError(t, err)

	// Check that all broker request:
	// 1- Moved to StatusPreparing
	// 2- Are linked to the StorageDeal they are now part of.
	bri1, err := b.GetBrokerRequestInfo(ctx, br1.ID)
	require.NoError(t, err)
	require.Equal(t, broker.RequestPreparing, bri1.BrokerRequest.Status)
	require.Equal(t, sd, bri1.BrokerRequest.StorageDealID)
	bri2, err := b.GetBrokerRequestInfo(ctx, br2.ID)
	require.NoError(t, err)
	require.Equal(t, broker.RequestPreparing, bri2.BrokerRequest.Status)
	require.Equal(t, sd, bri2.BrokerRequest.StorageDealID)

	// Check that the StorageDeal was persisted correctly.
	sd2, err := b.GetStorageDeal(ctx, sd)
	require.NoError(t, err)
	require.Equal(t, sd, sd2.ID)
	require.True(t, sd2.PayloadCid.Defined())
	require.Equal(t, broker.StorageDealPreparing, sd2.Status)
	brs, err := b.store.GetBrokerRequestIDs(ctx, sd)
	require.NoError(t, err)
	require.Len(t, brs, 2)
	require.True(t, time.Since(sd2.CreatedAt) < time.Minute)
	require.True(t, time.Since(sd2.UpdatedAt) < time.Minute)
	require.NotNil(t, sd2.Sources.CARURL)
	require.Equal(t, "http://duke.web3/car/"+sd2.PayloadCid.String(), sd2.Sources.CARURL.URL.String())
	require.Nil(t, sd2.Sources.CARIPFS)
	require.Zero(t, sd2.FilEpochDeadline)
	require.False(t, sd2.DisallowRebatching)
}

func TestCreatePrepared(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	b, _, auctioneer, _, _ := createBroker(t)

	// 1- Create some prepared data setup.
	deadline, _ := time.Parse(time.RFC3339, "2021-06-18T12:51:00+00:00")
	payloadCid := castCid("QmWc1T3ZMtAemjdt7Z87JmFVGjtxe4S6sNwn9zhvcNP1Fs")
	carURLStr := "https://duke.dog/car/" + payloadCid.String()
	carURL, _ := url.Parse(carURLStr)
	maddr, err := multiaddr.NewMultiaddr("/ip4/192.0.0.1/tcp/2020")
	require.NoError(t, err)
	pc := broker.PreparedCAR{
		PieceCid:  castCid("baga6ea4seaqofw2n4m4dagqbrrbmcbq3g7b5vzxlurpzxvvls4d5vk4skhdsuhq"),
		PieceSize: 1024,
		RepFactor: 3,
		Deadline:  deadline,
		Sources: auction.Sources{
			CARURL: &auction.CARURL{
				URL: *carURL,
			},
			CARIPFS: &auction.CARIPFS{
				Cid:        castCid("QmW2dMfxsd3YpS5MSMi5UUTbjMKUckZJjX5ouaQPuCjK8c"),
				Multiaddrs: []multiaddr.Multiaddr{maddr},
			},
		},
	}
	createdBr, err := b.CreatePrepared(ctx, payloadCid, pc)
	require.NoError(t, err)

	// 2- Check that the created BrokerRequest moved directly to Auctioning.
	br, err := b.store.GetBrokerRequest(ctx, createdBr.ID)
	require.NoError(t, err)
	require.Equal(t, broker.RequestAuctioning, br.Status)
	require.NotEmpty(t, br.StorageDealID)

	// 3- Check that the storage deal was created correctly, in particular:
	//    - The download sources URL and IPFS.
	//    - The FIL epoch deadline which should have been converted from time.Time to a FIL epoch.
	//    - The PayloadCId, PiceceCid and PieceSize which come from the prepared data parameters.
	sd, err := b.GetStorageDeal(ctx, br.StorageDealID)
	require.NoError(t, err)
	require.Equal(t, pc.RepFactor, sd.RepFactor)
	require.True(t, sd.DisallowRebatching)
	require.Equal(t, b.conf.dealDuration, uint64(sd.DealDuration))
	require.Equal(t, broker.StorageDealAuctioning, sd.Status)
	brs, err := b.store.GetBrokerRequestIDs(ctx, br.StorageDealID)
	require.NoError(t, err)
	require.Len(t, brs, 1)
	require.Contains(t, brs, br.ID)
	require.NotNil(t, sd.Sources.CARURL)
	require.Equal(t, carURLStr, sd.Sources.CARURL.URL.String())
	require.NotNil(t, sd.Sources.CARIPFS)
	require.Equal(t, pc.Sources.CARIPFS.Cid, sd.Sources.CARIPFS.Cid)
	require.Len(t, pc.Sources.CARIPFS.Multiaddrs, 1)
	require.Contains(t, sd.Sources.CARIPFS.Multiaddrs, pc.Sources.CARIPFS.Multiaddrs[0])
	require.Equal(t, uint64(857142), sd.FilEpochDeadline)
	require.Equal(t, payloadCid, sd.PayloadCid)
	require.Equal(t, pc.PieceCid, sd.PieceCid)
	require.Equal(t, pc.PieceSize, sd.PieceSize)

	// 4- Check that we made the call to create the auction.
	require.Equal(t, b.conf.dealDuration, uint64(auctioneer.calledDealDuration))
	require.Equal(t, payloadCid, auctioneer.calledPayloadCid)
	require.Equal(t, int(sd.PieceSize), auctioneer.calledPieceSize)
	require.Equal(t, sd.ID, auctioneer.calledStorageDealID)
	require.Equal(t, int(b.conf.dealDuration), auctioneer.calledDealDuration)
	require.Equal(t, pc.RepFactor, auctioneer.calledDealReplication)
	require.Equal(t, b.conf.verifiedDeals, auctioneer.calledDealVerified)
	require.NotNil(t, auctioneer.calledFilEpochDeadline)
	require.Equal(t, uint64(857142), auctioneer.calledFilEpochDeadline)
	require.NotNil(t, auctioneer.calledSources.CARURL)
	require.Equal(t, pc.Sources.CARIPFS.Cid, auctioneer.calledSources.CARIPFS.Cid)
	require.Len(t, auctioneer.calledSources.CARIPFS.Multiaddrs, 1)
}

func TestCreateStorageDealFail(t *testing.T) {
	t.Parallel()

	t.Run("invalid cid", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		b, _, _, _, _ := createBroker(t)
		_, err := b.CreateNewBatch(ctx, "SD1", cid.Undef, nil)
		require.Equal(t, ErrInvalidCid, err)
	})

	t.Run("empty group", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		b, _, _, _, _ := createBroker(t)
		brgCid := createCidFromString("StorageDeal")
		_, err := b.CreateNewBatch(ctx, "SD1", brgCid, nil)
		require.Equal(t, ErrEmptyGroup, err)
	})

	t.Run("group contains unknown broker request id", func(t *testing.T) {
		t.Parallel()
		ctx := context.Background()
		b, _, _, _, _ := createBroker(t)

		brgCid := createCidFromString("StorageDeal")
		_, err := b.CreateNewBatch(ctx, "SD1", brgCid, []broker.BrokerRequestID{broker.BrokerRequestID("invented")})
		require.ErrorIs(t, err, store.ErrStorageDealContainsUnknownBrokerRequest)
	})
}

func TestStorageDealPrepared(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	b, _, auctioneer, _, _ := createBroker(t)

	// 1- Create two broker requests and a corresponding storage deal.
	c := createCidFromString("BrokerRequest1")
	br1, err := b.Create(ctx, c)
	require.NoError(t, err)
	c = createCidFromString("BrokerRequest2")
	br2, err := b.Create(ctx, c)
	require.NoError(t, err)
	brgCid := createCidFromString("StorageDeal")
	sd, err := b.CreateNewBatch(ctx, "SD1", brgCid, []broker.BrokerRequestID{br1.ID, br2.ID})
	require.NoError(t, err)

	// 2- Call StorageDealPrepared as if the piecer did.
	dpr := broker.DataPreparationResult{
		PieceSize: uint64(123456),
		PieceCid:  createCidFromString("piececid1"),
	}
	err = b.NewBatchPrepared(ctx, sd, dpr)
	require.NoError(t, err)

	// 3- Verify the storage deal moved to the correct status, and is linked
	//    to the auction id.
	sd2, err := b.GetStorageDeal(ctx, sd)
	require.NoError(t, err)
	require.Equal(t, broker.StorageDealAuctioning, sd2.Status)

	// 4- Verify that Auctioneer was called to auction the data.
	require.Equal(t, auction.MaxDealDuration, uint64(auctioneer.calledDealDuration))
	require.Equal(t, brgCid, auctioneer.calledPayloadCid)
	require.Equal(t, int(dpr.PieceSize), auctioneer.calledPieceSize)
	require.Equal(t, sd, auctioneer.calledStorageDealID)
	require.Equal(t, int(b.conf.dealDuration), auctioneer.calledDealDuration)
	require.Equal(t, int(b.conf.dealReplication), auctioneer.calledDealReplication)
	require.Equal(t, b.conf.verifiedDeals, auctioneer.calledDealVerified)
	require.Zero(t, auctioneer.calledFilEpochDeadline)
	require.NotNil(t, auctioneer.calledSources.CARURL)
	require.Equal(t, "http://duke.web3/car/"+sd2.PayloadCid.String(), auctioneer.calledSources.CARURL.URL.String())

	// 5- Verify that the underlying broker requests also moved to
	//    their correct statuses.
	mbr1, err := b.GetBrokerRequestInfo(ctx, br1.ID)
	require.NoError(t, err)
	require.Equal(t, broker.RequestAuctioning, mbr1.BrokerRequest.Status)
	mbr2, err := b.GetBrokerRequestInfo(ctx, br2.ID)
	require.NoError(t, err)
	require.Equal(t, broker.RequestAuctioning, mbr2.BrokerRequest.Status)
}

func TestStorageDealAuctionedExactRepFactor(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	b, _, auctioneer, dealer, _ := createBroker(t)
	b.conf.dealReplication = 2

	// 1- Create two broker requests and a corresponding storage deal, and
	//    pass through prepared.
	c := createCidFromString("BrokerRequest1")
	br1, err := b.Create(ctx, c)
	require.NoError(t, err)
	c = createCidFromString("BrokerRequest2")
	br2, err := b.Create(ctx, c)
	require.NoError(t, err)
	brgCid := createCidFromString("StorageDeal")
	sd, err := b.CreateNewBatch(ctx, "SD1", brgCid, []broker.BrokerRequestID{br1.ID, br2.ID})
	require.NoError(t, err)
	dpr := broker.DataPreparationResult{
		PieceSize: uint64(123456),
		PieceCid:  createCidFromString("piececid1"),
	}
	err = b.NewBatchPrepared(ctx, sd, dpr)
	require.NoError(t, err)

	// 2- Call StorageDealAuctioned as if the auctioneer did.
	winningBids := map[auction.BidID]broker.WinningBid{
		auction.BidID("Bid1"): {
			MinerAddr:     "f01111",
			Price:         200,
			StartEpoch:    300,
			FastRetrieval: true,
		},
		auction.BidID("Bid2"): {
			MinerAddr:     "f02222",
			Price:         1200,
			StartEpoch:    1300,
			FastRetrieval: false,
		},
	}

	a := broker.ClosedAuction{
		ID:              auction.AuctionID("AUCTION1"),
		StorageDealID:   sd,
		DealDuration:    auction.MaxDealDuration,
		DealReplication: 2,
		DealVerified:    true,
		Status:          broker.AuctionStatusFinalized,
		WinningBids:     winningBids,
	}
	err = b.StorageDealAuctioned(ctx, a)
	require.NoError(t, err)

	// 3- Verify the storage deal moved to the correct status
	sd2, err := b.GetStorageDeal(ctx, sd)
	require.NoError(t, err)
	require.Equal(t, broker.StorageDealDealMaking, sd2.Status)
	require.NotEmpty(t, sd2.ID)
	deals, err := b.store.GetMinerDeals(ctx, sd)
	require.NoError(t, err)
	require.Len(t, deals, 2)
	require.Equal(t, int(a.DealReplication), sd2.RepFactor)
	require.Greater(t, sd2.DealDuration, 0)
	require.Greater(t, sd2.CreatedAt.Unix(), int64(0))
	require.Greater(t, sd2.UpdatedAt.Unix(), int64(0))
	require.Equal(t, sd2.AuctionRetries, 0)
	require.Empty(t, sd2.Error)
	require.Equal(t, brgCid, sd2.PayloadCid)
	require.Equal(t, dpr.PieceCid, sd2.PieceCid)
	require.Equal(t, dpr.PieceSize, sd2.PieceSize)
	for bidID, wb := range winningBids {
		var found bool
		for _, deal := range deals {
			if deal.BidID == bidID {
				require.Equal(t, sd, deal.StorageDealID)
				require.Equal(t, a.ID, deal.AuctionID)
				require.Greater(t, deal.CreatedAt.Unix(), int64(0))
				require.Greater(t, deal.UpdatedAt.Unix(), int64(0))
				require.Equal(t, wb.MinerAddr, deal.MinerAddr)
				require.Equal(t, int64(0), deal.DealID)
				require.Equal(t, uint64(0), deal.DealExpiration)
				require.Empty(t, deal.ErrorCause)
				found = true
				break
			}
		}
		require.True(t, found)
	}

	// 4- Verify that Dealer was called to execute the winning bids.
	calledADS := dealer.calledAuctionDeals
	require.Equal(t, sd, calledADS.StorageDealID)
	require.Equal(t, brgCid, calledADS.PayloadCid)
	require.Equal(t, dpr.PieceCid, calledADS.PieceCid)
	require.Equal(t, dpr.PieceSize, calledADS.PieceSize)
	require.Equal(t, auction.MaxDealDuration, calledADS.Duration)
	require.Len(t, calledADS.Proposals, 2)

	for _, tr := range calledADS.Proposals {
		var bid broker.WinningBid
		for _, b := range winningBids {
			if tr.Miner == b.MinerAddr {
				bid = b
				break
			}
		}
		if bid.MinerAddr == "" {
			t.Errorf("AuctionDealsTarget has no corresponding Bid")
		}
		require.Equal(t, bid.Price, tr.PricePerGiBPerEpoch)
		require.Equal(t, bid.StartEpoch, tr.StartEpoch)
		require.True(t, tr.Verified)
		require.Equal(t, bid.FastRetrieval, tr.FastRetrieval)
	}

	// 5- Verify that the underlying broker requests also moved to
	//    their correct statuses.
	mbr1, err := b.GetBrokerRequestInfo(ctx, br1.ID)
	require.NoError(t, err)
	require.Equal(t, broker.RequestDealMaking, mbr1.BrokerRequest.Status)
	mbr2, err := b.GetBrokerRequestInfo(ctx, br2.ID)
	require.NoError(t, err)
	require.Equal(t, broker.RequestDealMaking, mbr2.BrokerRequest.Status)

	// 6- Verify that the auctioneer wasn't called again for a new auction.
	//    The replication factor was 2, and we had 2 winning bids so a new auction
	//    isn't necessary.
	require.Equal(t, 1, auctioneer.calledCount)
}

func TestStorageDealAuctionedLessRepFactor(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	b, _, auctioneer, _, _ := createBroker(t)
	b.conf.dealReplication = 3

	// 1- Create two broker requests and a corresponding storage deal, and
	//    pass through prepared.
	c := createCidFromString("BrokerRequest1")
	br1, err := b.Create(ctx, c)
	require.NoError(t, err)
	c = createCidFromString("BrokerRequest2")
	br2, err := b.Create(ctx, c)
	require.NoError(t, err)
	brgCid := createCidFromString("StorageDeal")
	sd, err := b.CreateNewBatch(ctx, "SD1", brgCid, []broker.BrokerRequestID{br1.ID, br2.ID})
	require.NoError(t, err)
	dpr := broker.DataPreparationResult{
		PieceSize: uint64(123456),
		PieceCid:  createCidFromString("piececid1"),
	}
	err = b.NewBatchPrepared(ctx, sd, dpr)
	require.NoError(t, err)

	// 2- Call StorageDealAuctioned as if the auctioneer did.
	winningBids := map[auction.BidID]broker.WinningBid{
		auction.BidID("Bid1"): {
			MinerAddr:     "f01111",
			Price:         200,
			StartEpoch:    300,
			FastRetrieval: true,
		},
		auction.BidID("Bid2"): {
			MinerAddr:     "f02222",
			Price:         1200,
			StartEpoch:    1300,
			FastRetrieval: false,
		},
	}
	a := broker.ClosedAuction{
		ID:              auction.AuctionID("AUCTION1"),
		StorageDealID:   sd,
		DealDuration:    auction.MaxDealDuration,
		DealReplication: 2,
		DealVerified:    true,
		Status:          broker.AuctionStatusFinalized,
		WinningBids:     winningBids,
	}
	err = b.StorageDealAuctioned(ctx, a)
	require.NoError(t, err)

	// 3- Check that the broker deal has bumped the auction retry counter.
	sd2, err := b.GetStorageDeal(ctx, sd)
	require.NoError(t, err)
	require.Equal(t, 1, sd2.AuctionRetries)

	// 4- We received two winning bids instead of three. Check that the Auctioneer
	//    was called to create a new auction with rep factor 1.
	require.Equal(t, 2, auctioneer.calledCount)
	require.Equal(t, auction.MaxDealDuration, uint64(auctioneer.calledDealDuration))
	require.Equal(t, brgCid, auctioneer.calledPayloadCid)
	require.Equal(t, int(dpr.PieceSize), auctioneer.calledPieceSize)
	require.Equal(t, sd, auctioneer.calledStorageDealID)
	require.Equal(t, int(b.conf.dealDuration), auctioneer.calledDealDuration)
	require.Equal(t, 1, auctioneer.calledDealReplication)
	require.Equal(t, b.conf.verifiedDeals, auctioneer.calledDealVerified)
	require.Contains(t, auctioneer.calledExcludedMiners, "f01111")
	require.Contains(t, auctioneer.calledExcludedMiners, "f02222")
}

func TestStorageDealFailedAuction(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	b, mb, _, dealerd, _ := createBroker(t)

	// 1- Create two broker requests and a corresponding storage deal, and
	//    pass through prepared.
	c := createCidFromString("BrokerRequest1")
	br1, err := b.Create(ctx, c)
	require.NoError(t, err)
	c = createCidFromString("BrokerRequest2")
	br2, err := b.Create(ctx, c)
	require.NoError(t, err)
	brgCid := createCidFromString("StorageDeal")
	sd, err := b.CreateNewBatch(ctx, "SD1", brgCid, []broker.BrokerRequestID{br1.ID, br2.ID})
	require.NoError(t, err)
	dpr := broker.DataPreparationResult{
		PieceSize: uint64(123456),
		PieceCid:  createCidFromString("piececid1"),
	}
	err = b.NewBatchPrepared(ctx, sd, dpr)
	require.NoError(t, err)

	// Empty the call stack of the mocks, since it has calls from before.
	dealerd.calledAuctionDeals = dealer.AuctionDeals{}

	// 2- Call StorageDealAuctioned as if the auctioneer did.
	a := broker.ClosedAuction{
		ID:              auction.AuctionID("AUCTION1"),
		StorageDealID:   sd,
		DealDuration:    auction.MaxDealDuration,
		DealReplication: 1,
		DealVerified:    true,
		Status:          broker.AuctionStatusFinalized,
		ErrorCause:      "reached max retries",
	}
	err = b.StorageDealAuctioned(ctx, a)
	require.NoError(t, err)

	// 3- Verify the storage deal moved to the correct status with error cause.
	sd2, err := b.GetStorageDeal(ctx, sd)
	require.NoError(t, err)
	require.Equal(t, broker.StorageDealError, sd2.Status)
	require.Equal(t, a.ErrorCause, sd2.Error)

	// 4- Verify that Dealer was NOT called
	require.Equal(t, broker.StorageDealID(""), dealerd.calledAuctionDeals.StorageDealID)

	// 5- Verify that the underlying broker requests were moved
	//    to Batching again.
	mbr1, err := b.GetBrokerRequestInfo(ctx, br1.ID)
	require.NoError(t, err)
	require.Equal(t, broker.RequestBatching, mbr1.BrokerRequest.Status)
	mbr2, err := b.GetBrokerRequestInfo(ctx, br2.ID)
	require.NoError(t, err)
	require.Equal(t, broker.RequestBatching, mbr2.BrokerRequest.Status)

	// 6- Verify that there were 3 published messages.
	//    Two of them are from `b.Create` for creating the first two BrokerRequests.
	//    The third one is the important one for this test, which is a signal that the failed auction
	//    sent to be re-batched both storage deals due to the failing auction.
	require.Equal(t, 3, mb.TotalPublished())
	data, err := mb.GetMsg(mbroker.ReadyToBatchTopic, 2) // Take the third msg in the topic (0-based idx).
	require.NoError(t, err)
	r := &pb.ReadyToBatch{}
	err = proto.Unmarshal(data, r)
	require.NoError(t, err)
	require.Len(t, r.DataCids, 2)

	require.Equal(t, string(br1.ID), r.DataCids[0].BrokerRequestId)
	dataCid0, err := cid.Cast(r.DataCids[0].DataCid)
	require.NoError(t, err)
	require.Equal(t, br1.DataCid, dataCid0)
	require.Equal(t, string(br2.ID), r.DataCids[1].BrokerRequestId)
	dataCid1, err := cid.Cast(r.DataCids[1].DataCid)
	require.NoError(t, err)
	require.Equal(t, br2.DataCid, dataCid1)
}

func TestStorageDealFinalizedDeals(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	b, _, _, _, chainAPI := createBroker(t)
	b.conf.dealReplication = 2

	// 1- Create two broker requests and a corresponding storage deal, and
	//    pass through prepared, auctioned, and deal making.
	c1 := createCidFromString("BrokerRequest1")
	br1, err := b.Create(ctx, c1)
	require.NoError(t, err)
	c2 := createCidFromString("BrokerRequest2")
	br2, err := b.Create(ctx, c2)
	require.NoError(t, err)
	brgCid := createCidFromString("StorageDeal")
	sd, err := b.CreateNewBatch(ctx, "SD1", brgCid, []broker.BrokerRequestID{br1.ID, br2.ID})
	require.NoError(t, err)
	dpr := broker.DataPreparationResult{
		PieceSize: uint64(123456),
		PieceCid:  createCidFromString("piececid1"),
	}
	err = b.NewBatchPrepared(ctx, sd, dpr)
	require.NoError(t, err)

	winningBids := map[auction.BidID]broker.WinningBid{
		auction.BidID("Bid1"): {
			MinerAddr:     "f0011",
			Price:         200,
			StartEpoch:    300,
			FastRetrieval: true,
		},
		auction.BidID("Bid2"): {
			MinerAddr:     "f0012",
			Price:         1200,
			StartEpoch:    1300,
			FastRetrieval: false,
		},
	}

	auction := broker.ClosedAuction{
		ID:              auction.AuctionID("AUCTION1"),
		StorageDealID:   sd,
		DealDuration:    auction.MaxDealDuration,
		DealReplication: 2,
		Status:          broker.AuctionStatusFinalized,
		WinningBids:     winningBids,
	}
	err = b.StorageDealAuctioned(ctx, auction)
	require.NoError(t, err)

	// 2- Call StorageDealFinalizedDeals with the first deal having
	//    success. We'll make a further call to the second one.
	fad1 := broker.FinalizedAuctionDeal{
		StorageDealID:  auction.StorageDealID,
		DealID:         100,
		DealExpiration: 200,
		Miner:          "f0011",
	}
	err = b.StorageDealFinalizedDeal(ctx, fad1)
	require.NoError(t, err)

	// 3- Verify the storage deal and underlying broker request are still in deal making,
	//    since there's another pending deal to be reported.
	sd2, err := b.GetStorageDeal(ctx, sd)
	require.NoError(t, err)
	require.Equal(t, broker.StorageDealDealMaking, sd2.Status)
	mbr1, err := b.GetBrokerRequestInfo(ctx, br1.ID)
	require.NoError(t, err)
	require.Equal(t, broker.RequestDealMaking, mbr1.BrokerRequest.Status)
	mbr2, err := b.GetBrokerRequestInfo(ctx, br2.ID)
	require.NoError(t, err)
	require.Equal(t, broker.RequestDealMaking, mbr2.BrokerRequest.Status)

	// 4- Let's finalize the other one.
	chainAPI.clean() // clean the previous call stack
	fad2 := broker.FinalizedAuctionDeal{
		StorageDealID:  auction.StorageDealID,
		Miner:          "f0012",
		DealID:         101,
		DealExpiration: 201,
	}
	err = b.StorageDealFinalizedDeal(ctx, fad2)
	require.NoError(t, err)

	// 5- Verify that the storage deal switched to Success, since at least one of the winning bids
	//    succeeded.
	sd2, err = b.GetStorageDeal(ctx, sd)
	require.NoError(t, err)
	require.Equal(t, broker.StorageDealSuccess, sd2.Status)
	mbr1, err = b.GetBrokerRequestInfo(ctx, br1.ID)
	require.NoError(t, err)
	require.Equal(t, broker.RequestSuccess, mbr1.BrokerRequest.Status)
	mbr2, err = b.GetBrokerRequestInfo(ctx, br2.ID)
	require.NoError(t, err)
	require.Equal(t, broker.RequestSuccess, mbr2.BrokerRequest.Status)

	// 6- Verify that we have 3 unpin jobs: the batch cid, and the two data cids in the batch.
	for i := 0; i < 3; i++ {
		_, ok, err := b.store.UnpinJobGetNext(ctx)
		require.NoError(t, err)
		require.True(t, ok, fmt.Sprintf("should have got unpin job in round %d", i))
	}
	_, ok, err := b.store.UnpinJobGetNext(ctx)
	require.NoError(t, err)
	require.False(t, ok)

	// 7- Verify that getting the broker request information shows the deals.
	brs := []broker.BrokerRequest{br1, br2}
	for _, br := range brs {
		bri, err := b.GetBrokerRequestInfo(ctx, br.ID)
		require.NoError(t, err)
		require.Len(t, bri.Deals, 2)
		sort.Slice(bri.Deals, func(i, j int) bool { return bri.Deals[i].Miner < bri.Deals[j].Miner })
		require.Equal(t, bri.Deals[0].Miner, fad1.Miner)
		require.Equal(t, bri.Deals[0].DealID, fad1.DealID)
		require.Equal(t, bri.Deals[0].Expiration, fad1.DealExpiration)
		require.Equal(t, bri.Deals[1].Miner, fad2.Miner)
		require.Equal(t, bri.Deals[1].DealID, fad2.DealID)
		require.Equal(t, bri.Deals[1].Expiration, fad2.DealExpiration)
	}
}

func createBroker(t *testing.T) (
	*Broker,
	*fakemsgbroker.FakeMsgBroker,
	*dumbAuctioneer,
	*dumbDealer,
	*dumbChainAPI) {
	auctioneer := &dumbAuctioneer{}
	dealer := &dumbDealer{}
	chainAPI := &dumbChainAPI{}
	u, err := tests.PostgresURL()
	require.NoError(t, err)

	fmb := fakemsgbroker.New()
	b, err := New(
		u,
		auctioneer,
		dealer,
		chainAPI,
		nil,
		fmb,
		WithCARExportURL("http://duke.web3/car/"),
	)
	require.NoError(t, err)

	return b, fmb, auctioneer, dealer, chainAPI
}

type dumbAuctioneer struct {
	calledStorageDealID    broker.StorageDealID
	calledPayloadCid       cid.Cid
	calledPieceSize        int
	calledDealDuration     int
	calledDealReplication  int
	calledDealVerified     bool
	calledExcludedMiners   []string
	calledCount            int
	calledSources          auction.Sources
	calledFilEpochDeadline uint64
}

func (dp *dumbAuctioneer) ReadyToAuction(
	ctx context.Context,
	id broker.StorageDealID,
	payloadCid cid.Cid,
	dealSize, dealDuration, dealReplication int,
	dealVerified bool,
	excludedMiners []string,
	filEpochDeadline uint64,
	sources auction.Sources,
) (auction.AuctionID, error) {
	dp.calledStorageDealID = id
	dp.calledPayloadCid = payloadCid
	dp.calledPieceSize = dealSize
	dp.calledDealDuration = dealDuration
	dp.calledDealReplication = dealReplication
	dp.calledDealVerified = dealVerified
	dp.calledExcludedMiners = excludedMiners
	dp.calledCount++
	dp.calledSources = sources
	dp.calledFilEpochDeadline = filEpochDeadline
	return auction.AuctionID("AUCTION1"), nil
}

func (dp *dumbAuctioneer) GetAuction(ctx context.Context, id auction.AuctionID) (auctioneer.Auction, error) {
	panic("shouldn't be called")
}

func (dp *dumbAuctioneer) ProposalAccepted(context.Context, auction.AuctionID, auction.BidID, cid.Cid) error {
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
	return cid.NewCidV1(cid.Raw, mh)
}

type dumbChainAPI struct {
}

var _ chainapi.ChainAPI = (*dumbChainAPI)(nil)

func (dr *dumbChainAPI) HasDeposit(ctx context.Context, brokerID, accountID string) (bool, error) {
	return true, nil
}

func (dr *dumbChainAPI) clean() {
}

func castCid(cidStr string) cid.Cid {
	c, _ := cid.Decode(cidStr)
	return c
}
