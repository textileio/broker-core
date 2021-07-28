package store

import (
	"context"
	"database/sql"
	"sort"
	"testing"

	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/dealerd/store/internal/db"
	"github.com/textileio/broker-core/tests"
)

func TestCreate(t *testing.T) {
	t.Parallel()
	u, err := tests.PostgresURL()
	require.NoError(t, err)
	s, err := New(u)
	require.NoError(t, err)

	ad := gad1
	aud := gaud1
	err = s.Create(context.Background(), &ad, []*AuctionDeal{&aud})
	require.NoError(t, err)
	deepCheckAuctionData(t, s, ad)
	deepCheckAuctionDeals(t, s, aud)
}

func TestCreateFail(t *testing.T) {
	t.Parallel()
	u, err := tests.PostgresURL()
	require.NoError(t, err)
	s, err := New(u)
	require.NoError(t, err)

	t.Run("auction-data duration 0", func(t *testing.T) {
		t.Parallel()
		ad := gad1
		ad.Duration = 0
		aud := gaud1
		err := s.Create(context.Background(), &ad, []*AuctionDeal{&aud})
		require.Error(t, err)
	})
	t.Run("auction-data undef id", func(t *testing.T) {
		t.Parallel()
		ad := gad1
		ad.ID = ""
		aud := gaud1
		err := s.Create(context.Background(), &ad, []*AuctionDeal{&aud})
		require.Error(t, err)
	})
	t.Run("auction-data undef batch id", func(t *testing.T) {
		t.Parallel()
		ad := gad1
		ad.BatchID = ""
		aud := gaud1
		err := s.Create(context.Background(), &ad, []*AuctionDeal{&aud})
		require.Error(t, err)
	})
	t.Run("auction-data undef payload cid", func(t *testing.T) {
		t.Parallel()
		ad := gad1
		ad.PayloadCid = cid.Undef
		aud := gaud1
		err := s.Create(context.Background(), &ad, []*AuctionDeal{&aud})
		require.Error(t, err)
	})
	t.Run("auction-data undef piece cid", func(t *testing.T) {
		t.Parallel()
		ad := gad1
		ad.PieceCid = cid.Undef
		aud := gaud1
		err := s.Create(context.Background(), &ad, []*AuctionDeal{&aud})
		require.Error(t, err)
	})
	t.Run("auction-data piece size 0", func(t *testing.T) {
		t.Parallel()
		ad := gad1
		ad.PieceSize = 0
		aud := gaud1
		err := s.Create(context.Background(), &ad, []*AuctionDeal{&aud})
		require.Error(t, err)
	})

	t.Run("auction-deal empty storage-provider", func(t *testing.T) {
		t.Parallel()
		ad := gad1
		aud := gaud1
		aud.StorageProviderID = ""
		err := s.Create(context.Background(), &ad, []*AuctionDeal{&aud})
		require.Error(t, err)
	})
	t.Run("auction-deal negative price", func(t *testing.T) {
		t.Parallel()
		ad := gad1
		aud := gaud1
		aud.PricePerGibPerEpoch = -1
		err := s.Create(context.Background(), &ad, []*AuctionDeal{&aud})
		require.Error(t, err)
	})
	t.Run("auction-deal start epoch 0", func(t *testing.T) {
		t.Parallel()
		ad := gad1
		aud := gaud1
		aud.StartEpoch = 0
		err = s.Create(context.Background(), &ad, []*AuctionDeal{&aud})
		require.Error(t, err)
	})
}

func TestSaveAuctionDeal(t *testing.T) {
	t.Parallel()
	u, err := tests.PostgresURL()
	require.NoError(t, err)
	s, err := New(u)
	require.NoError(t, err)

	ad := gad1
	aud := gaud1
	err = s.Create(context.Background(), &ad, []*AuctionDeal{&aud})
	require.NoError(t, err)

	auds, err := s.getAllPending()
	require.NoError(t, err)
	require.Len(t, auds, 1)

	auds[0].ErrorCause = "duke"
	auds[0].ProposalCid = "QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jZ1"
	auds[0].DealID = 1234
	auds[0].DealExpiration = 5678
	err = s.SaveAndMoveAuctionDeal(context.Background(), auds[0], StatusDealMaking)
	require.NoError(t, err)
	deepCheckAuctionDeals(t, s, auds[0])
}

func TestSaveAuctionDealFail(t *testing.T) {
	t.Parallel()

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		u, err := tests.PostgresURL()
		require.NoError(t, err)
		s, err := New(u)
		require.NoError(t, err)
		aud := gaud1
		aud.ID = "invented"
		aud.Status = db.StatusDealMaking
		err = s.SaveAndMoveAuctionDeal(context.Background(), aud, StatusDealMaking)
		require.Equal(t, ErrNotFound, err)
	})
	t.Run("wrong status transition", func(t *testing.T) {
		t.Parallel()

		u, err := tests.PostgresURL()
		require.NoError(t, err)
		s, err := New(u)
		require.NoError(t, err)
		ad := gad1
		aud := gaud1
		err = s.Create(context.Background(), &ad, []*AuctionDeal{&aud})
		require.NoError(t, err)
		auds, err := s.getAllPending()
		require.NoError(t, err)

		auds[0].Status = db.StatusReportFinalized
		err = s.SaveAndMoveAuctionDeal(context.Background(), auds[0], StatusReportFinalized)
		require.Error(t, err)
	})
}

func TestGetNext(t *testing.T) {
	t.Parallel()
	u, err := tests.PostgresURL()
	require.NoError(t, err)
	s, err := New(u)
	require.NoError(t, err)
	ctx := context.Background()
	ad := gad1
	aud := gaud1
	err = s.Create(ctx, &ad, []*AuctionDeal{&aud})
	require.NoError(t, err)

	testsCases := []AuctionDealStatus{
		StatusDealMaking,
		StatusConfirmation,
		StatusReportFinalized}

	for _, tt := range testsCases {
		err = s.SaveAndMoveAuctionDeal(context.Background(), aud, tt)
		require.NoError(t, err)

		// Call get next
		aud2, ok, err := s.GetNextPending(ctx, tt)
		require.NoError(t, err)

		// 1. Verify that we have a result.
		require.True(t, ok)
		require.Equal(t, aud.ID, aud2.ID)

		// 2. Verify that the returned element changed status
		//    to the next appropriate status.
		require.Equal(t, tt, AuctionDealStatus(aud2.Status))
		require.True(t, aud2.Executing)

		// 3. Verify that calling GetNextPending again returns no results.
		_, ok, err = s.GetNextPending(ctx, tt)
		require.NoError(t, err)
		require.False(t, ok)
	}
}

func TestGetAllAuctionDeals(t *testing.T) {
	t.Parallel()

	u, err := tests.PostgresURL()
	require.NoError(t, err)
	s, err := New(u)
	require.NoError(t, err)

	ad := gad1
	aud1 := gaud1
	aud2 := gaud2
	err = s.Create(context.Background(), &ad, []*AuctionDeal{&aud1, &aud2})
	require.NoError(t, err)
	deepCheckAuctionData(t, s, ad)
	deepCheckAuctionDeals(t, s, aud1, aud2)

	auds, err := s.getAllPending()
	require.NoError(t, err)
	require.Len(t, auds, 2)
	sort.Slice(auds, func(i, j int) bool { return auds[i].StorageProviderID < auds[j].StorageProviderID })
	cmpAuctionDeals(t, aud1, auds[0])
	cmpAuctionDeals(t, aud2, auds[1])
}

func TestGetAuctionData(t *testing.T) {
	t.Parallel()

	u, err := tests.PostgresURL()
	require.NoError(t, err)
	s, err := New(u)
	require.NoError(t, err)
	ad := gad1
	aud := gaud1
	err = s.Create(context.Background(), &ad, []*AuctionDeal{&aud})
	require.NoError(t, err)
	auds, err := s.getAllPending()
	require.NoError(t, err)

	ad2, err := s.GetAuctionData(context.Background(), auds[0].AuctionDataID)
	require.NoError(t, err)
	cmpAuctionData(t, ad, ad2)
}

func TestGetAuctionNotFound(t *testing.T) {
	t.Parallel()

	u, err := tests.PostgresURL()
	require.NoError(t, err)
	s, err := New(u)
	require.NoError(t, err)
	ad := gad1
	aud := gaud1
	err = s.Create(context.Background(), &ad, []*AuctionDeal{&aud})
	require.NoError(t, err)

	_, err = s.GetAuctionData(context.Background(), "invented")
	require.Equal(t, sql.ErrNoRows, err)
}

func TestRemoveAuctionDeals(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	u, err := tests.PostgresURL()
	require.NoError(t, err)
	s, err := New(u)
	require.NoError(t, err)
	ad := gad1
	aud1 := gaud1
	aud2 := gaud2
	err = s.Create(ctx, &ad, []*AuctionDeal{&aud1, &aud2})
	require.NoError(t, err)

	all, err := s.getAllPending()
	require.NoError(t, err)
	err = s.RemoveAuctionDeal(ctx, all[0])
	require.Error(t, err, "should not remove non-final status auction deal")
	all[0].Status = db.StatusReportFinalized
	err = s.RemoveAuctionDeal(ctx, all[0])
	require.Error(t, err, "should not remove pending auction deal")
	all[0].Executing = true
	err = s.RemoveAuctionDeal(ctx, all[0])
	require.NoError(t, err)

	// Check the corresponding AuctionData wasn't removed from the store,
	// since the second auction data is still linking to it.
	_, err = s.GetAuctionData(ctx, ad.ID)
	require.NoError(t, err)

	// Remove the second.
	all[1].Status = db.StatusReportFinalized
	all[1].Executing = true
	all[1].ErrorCause = "failed because something happened"
	err = s.RemoveAuctionDeal(ctx, all[1])
	require.NoError(t, err)

	// Check the corresponding AuctionData was also removed.
	// No more auction datas linked to it.
	_, err = s.GetAuctionData(ctx, ad.ID)
	require.Equal(t, sql.ErrNoRows, err)
}

func castCid(cidStr string) cid.Cid {
	c, err := cid.Decode(cidStr)
	if err != nil {
		panic(err)
	}
	return c
}

func deepCheckAuctionData(t *testing.T, s *Store, ad AuctionData) {
	t.Helper()

	// Check value in datastore.
	dsAd, err := s.GetAuctionData(context.Background(), ad.ID)
	require.NoError(t, err)
	cmpAuctionData(t, ad, dsAd)
}

func cmpAuctionData(t *testing.T, ad1, ad2 AuctionData) {
	require.Equal(t, ad1.BatchID, ad2.BatchID)
	require.Equal(t, ad1.PayloadCid, ad2.PayloadCid)
	require.Equal(t, ad1.PieceCid, ad2.PieceCid)
	require.Equal(t, ad1.Duration, ad2.Duration)
	require.Equal(t, ad1.PieceSize, ad2.PieceSize)
}

func deepCheckAuctionDeals(t *testing.T, s *Store, auds ...AuctionDeal) {
	t.Helper()

	for _, aud := range auds {
		// Check value in datastore.
		dsAud, err := s.db.GetAuctionDeal(context.Background(), aud.ID)
		require.NoError(t, err)
		require.NotEmpty(t, dsAud.ID)
		require.NotEmpty(t, dsAud.CreatedAt)
		require.NotEmpty(t, dsAud.AuctionDataID)
		cmpAuctionDeals(t, aud, AuctionDeal(dsAud))
	}
}

func cmpAuctionDeals(t *testing.T, aud1, aud2 AuctionDeal) {
	require.Equal(t, aud1.StorageProviderID, aud2.StorageProviderID)
	require.Equal(t, aud1.PricePerGibPerEpoch, aud2.PricePerGibPerEpoch)
	require.Equal(t, aud1.StartEpoch, aud2.StartEpoch)
	require.Equal(t, aud1.Verified, aud2.Verified)
	require.Equal(t, aud1.FastRetrieval, aud2.FastRetrieval)
	require.Equal(t, aud1.Status, aud2.Status)
	require.Equal(t, aud1.Executing, aud2.Executing)
	require.Equal(t, aud1.ErrorCause, aud2.ErrorCause)
	require.Equal(t, aud1.ProposalCid, aud2.ProposalCid)
	require.Equal(t, aud1.DealID, aud2.DealID)
	require.Equal(t, aud1.DealExpiration, aud2.DealExpiration)
}

var (
	gad1 = AuctionData{
		ID:         "id-1",
		BatchID:    broker.BatchID("1"),
		PayloadCid: castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jP1"),
		PieceCid:   castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jPA"),
		Duration:   10,
		PieceSize:  100,
	}

	gaud1 = AuctionDeal{
		StorageProviderID:   "f011001",
		PricePerGibPerEpoch: 10,
		StartEpoch:          20,
		Verified:            true,
		FastRetrieval:       true,
	}
	gaud2 = AuctionDeal{
		StorageProviderID:   "f011002",
		PricePerGibPerEpoch: 11,
		StartEpoch:          21,
		Verified:            true,
		FastRetrieval:       true,
	}
)
