package store

import (
	"bytes"
	"encoding/gob"
	"testing"

	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/tests"
)

func TestCreate(t *testing.T) {
	t.Parallel()
	s, err := New(tests.NewTxMapDatastore())
	require.NoError(t, err)

	err = s.Create(gad1, []AuctionDeal{gaud1})
	require.NoError(t, err)
	deepCheckAuctionData(t, s, gad1)
	deepCheckAuctionDeals(t, s, gaud1)
}

func TestCreateFail(t *testing.T) {
	t.Parallel()
	s, err := New(tests.NewTxMapDatastore())
	require.NoError(t, err)

	t.Run("auction-data duration 0", func(t *testing.T) {
		t.Parallel()
		ad := gad1
		ad.Duration = 0
		err := s.Create(ad, []AuctionDeal{gaud1})
		require.Error(t, err)
	})
	t.Run("auction-data undef storage deal id", func(t *testing.T) {
		t.Parallel()
		ad := gad1
		ad.StorageDealID = ""
		err := s.Create(ad, []AuctionDeal{gaud1})
		require.Error(t, err)
	})
	t.Run("auction-data undef payload cid", func(t *testing.T) {
		t.Parallel()
		ad := gad1
		ad.PayloadCid = cid.Undef
		err := s.Create(ad, []AuctionDeal{gaud1})
		require.Error(t, err)
	})
	t.Run("auction-data undef piece cid", func(t *testing.T) {
		t.Parallel()
		ad := gad1
		ad.PieceCid = cid.Undef
		err := s.Create(ad, []AuctionDeal{gaud1})
		require.Error(t, err)
	})
	t.Run("auction-data piece size 0", func(t *testing.T) {
		t.Parallel()
		ad := gad1
		ad.PieceSize = 0
		err := s.Create(ad, []AuctionDeal{gaud1})
		require.Error(t, err)
	})

	t.Run("auction-deal empty miner", func(t *testing.T) {
		t.Parallel()
		aud := gaud1
		aud.Miner = ""
		err := s.Create(gad1, []AuctionDeal{aud})
		require.Error(t, err)
	})
	t.Run("auction-deal negative price", func(t *testing.T) {
		t.Parallel()
		aud := gaud1
		aud.PricePerGiBPerEpoch = -1
		err := s.Create(gad1, []AuctionDeal{aud})
		require.Error(t, err)
	})
	t.Run("auction-deal start epoch 0", func(t *testing.T) {
		t.Parallel()
		aud := gaud1
		aud.StartEpoch = 0
		err = s.Create(gad1, []AuctionDeal{aud})
		require.Error(t, err)
	})
}

func TestRehydrate(t *testing.T) {
	t.Parallel()
	ds := tests.NewTxMapDatastore()
	s, err := New(ds)
	require.NoError(t, err)

	err = s.Create(gad1, []AuctionDeal{gaud1})
	require.NoError(t, err)

	// Recreate store with same ds.
	s, err = New(ds)
	require.NoError(t, err)
	deepCheckAuctionData(t, s, gad1)
	deepCheckAuctionDeals(t, s, gaud1)
}

func TestSaveAuctionDeal(t *testing.T) {
	t.Parallel()

	s, err := New(tests.NewTxMapDatastore())
	require.NoError(t, err)

	err = s.Create(gad1, []AuctionDeal{gaud1})
	require.NoError(t, err)

	auds, err := s.GetAllAuctionDeals(Pending)
	require.NoError(t, err)

	auds[0].Status = WaitingConfirmation
	auds[0].ErrorCause = "duke"
	auds[0].ProposalCid = castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jZ1")
	auds[0].DealID = 1234
	auds[0].DealExpiration = 5678
	err = s.SaveAuctionDeal(auds[0])
	require.NoError(t, err)
	deepCheckAuctionDeals(t, s, auds[0])
}

func TestSaveAuctionDealFail(t *testing.T) {
	t.Parallel()

	t.Run("not found", func(t *testing.T) {
		t.Parallel()

		s, err := New(tests.NewTxMapDatastore())
		require.NoError(t, err)
		aud := gaud1
		aud.ID = "invented"
		err = s.SaveAuctionDeal(aud)
		require.Equal(t, ErrNotFound, err)
	})
	t.Run("wrong status transition", func(t *testing.T) {
		t.Parallel()

		s, err := New(tests.NewTxMapDatastore())
		require.NoError(t, err)
		err = s.Create(gad1, []AuctionDeal{gaud1})
		require.NoError(t, err)
		auds, err := s.GetAllAuctionDeals(Pending)
		require.NoError(t, err)

		auds[0].Status = Success
		err = s.SaveAuctionDeal(auds[0])
		require.Error(t, err)
	})
}

func TestGetAllAuctionDeals(t *testing.T) {
	t.Parallel()

	s, err := New(tests.NewTxMapDatastore())
	require.NoError(t, err)

	err = s.Create(gad1, []AuctionDeal{gaud1, gaud2})
	require.NoError(t, err)
	deepCheckAuctionData(t, s, gad1)
	deepCheckAuctionDeals(t, s, gaud1, gaud2)

	auds, err := s.GetAllAuctionDeals(Pending)
	require.NoError(t, err)
	require.Len(t, auds, 2)
	cmpAuctionDeals(t, gaud1, auds[0])
	cmpAuctionDeals(t, gaud2, auds[1])
}

func TestGetAuctionData(t *testing.T) {
	t.Parallel()

	s, err := New(tests.NewTxMapDatastore())
	require.NoError(t, err)
	err = s.Create(gad1, []AuctionDeal{gaud1})
	require.NoError(t, err)
	auds, err := s.GetAllAuctionDeals(Pending)
	require.NoError(t, err)

	ad, err := s.GetAuctionData(auds[0].AuctionDataID)
	require.NoError(t, err)
	cmpAuctionData(t, gad1, ad)
}

func TestGetAuctionNotFound(t *testing.T) {
	t.Parallel()

	s, err := New(tests.NewTxMapDatastore())
	require.NoError(t, err)
	err = s.Create(gad1, []AuctionDeal{gaud1})
	require.NoError(t, err)

	_, err = s.GetAuctionData("invented")
	require.Equal(t, ErrNotFound, err)
}

func TestRemoveAuctionDeals(t *testing.T) {
	t.Parallel()

	s, err := New(tests.NewTxMapDatastore())
	require.NoError(t, err)
	err = s.Create(gad1, []AuctionDeal{gaud1, gaud2})
	require.NoError(t, err)

	all, err := s.GetAllAuctionDeals(Pending)
	require.NoError(t, err)
	err = s.RemoveAuctionDeals([]AuctionDeal{all[0]})
	require.Error(t, err) // Can't remove non-final status (i.e: Pending)

	// Remove the first auction deal.
	all[0].Status = Success
	err = s.RemoveAuctionDeals([]AuctionDeal{all[0]})
	require.NoError(t, err)

	// Check the corresponding AuctionData wasn't removed from the store,
	// since the second auction data is still linking to it.
	_, err = s.GetAuctionData(all[1].AuctionDataID)
	require.NoError(t, err)

	// Remove the second.
	all[1].Status = Error
	err = s.RemoveAuctionDeals([]AuctionDeal{all[1]})
	require.NoError(t, err)

	// Check the corresponding AuctionData was also removed.
	// No more auction datas linked to it.
	_, err = s.GetAuctionData(all[1].AuctionDataID)
	require.Equal(t, ErrNotFound, err)
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

	// Check value in cache.
	var cacheAd *AuctionData
	for _, v := range s.auctionData {
		if v.StorageDealID == ad.StorageDealID {
			cacheAd = &v
			break
		}
	}
	require.NotNil(t, cacheAd)
	require.NotEmpty(t, cacheAd.ID)
	require.NotEmpty(t, cacheAd.CreatedAt)
	cmpAuctionData(t, ad, *cacheAd)

	// Check value in datastore.
	buf, err := s.ds.Get(makeAuctionDataKey(cacheAd.ID))
	require.NoError(t, err)
	var dsAd AuctionData
	d := gob.NewDecoder(bytes.NewReader(buf))
	err = d.Decode(&dsAd)
	require.NoError(t, err)
	cmpAuctionData(t, ad, *cacheAd)
}

func cmpAuctionData(t *testing.T, ad1, ad2 AuctionData) {
	require.Equal(t, ad1.StorageDealID, ad2.StorageDealID)
	require.Equal(t, ad1.PayloadCid, ad2.PayloadCid)
	require.Equal(t, ad1.PieceCid, ad2.PieceCid)
	require.Equal(t, ad1.Duration, ad2.Duration)
	require.Equal(t, ad1.PieceSize, ad2.PieceSize)
}

func deepCheckAuctionDeals(t *testing.T, s *Store, auds ...AuctionDeal) {
	t.Helper()

	for _, aud := range auds {
		// Check value in cache.
		var cacheAud *AuctionDeal
		for _, iaud := range s.auctionDeals {
			if iaud.Miner == aud.Miner {
				cacheAud = &iaud
				break
			}
		}
		require.NotNil(t, cacheAud)
		require.NotEmpty(t, cacheAud.ID)
		require.NotEmpty(t, cacheAud.CreatedAt)
		cmpAuctionDeals(t, aud, *cacheAud)

		// Check value in datastore.
		buf, err := s.ds.Get(makeAuctionDealKey(cacheAud.ID))
		require.NoError(t, err)
		var dsAud AuctionDeal
		d := gob.NewDecoder(bytes.NewReader(buf))
		err = d.Decode(&dsAud)
		require.NoError(t, err)
		require.NotEmpty(t, cacheAud.ID)
		require.NotEmpty(t, cacheAud.CreatedAt)
		require.NotEmpty(t, cacheAud.AuctionDataID)
		cmpAuctionDeals(t, aud, dsAud)
	}
}

func cmpAuctionDeals(t *testing.T, aud1, aud2 AuctionDeal) {
	require.Equal(t, aud1.Miner, aud2.Miner)
	require.Equal(t, aud1.PricePerGiBPerEpoch, aud2.PricePerGiBPerEpoch)
	require.Equal(t, aud1.StartEpoch, aud2.StartEpoch)
	require.Equal(t, aud1.Verified, aud2.Verified)
	require.Equal(t, aud1.FastRetrieval, aud2.FastRetrieval)
	require.Equal(t, aud1.Status, aud2.Status)
	require.Equal(t, aud1.ErrorCause, aud2.ErrorCause)
	require.Equal(t, aud1.ProposalCid, aud2.ProposalCid)
	require.Equal(t, aud1.DealID, aud2.DealID)
	require.Equal(t, aud1.DealExpiration, aud2.DealExpiration)
}

var (
	gad1 = AuctionData{
		StorageDealID: broker.StorageDealID("1"),
		PayloadCid:    castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jP1"),
		PieceCid:      castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jPA"),
		Duration:      10,
		PieceSize:     100,
	}

	gaud1 = AuctionDeal{
		Miner:               "f011001",
		PricePerGiBPerEpoch: 10,
		StartEpoch:          20,
		Verified:            true,
		FastRetrieval:       true,
	}
	gaud2 = AuctionDeal{
		Miner:               "f011002",
		PricePerGiBPerEpoch: 11,
		StartEpoch:          21,
		Verified:            true,
		FastRetrieval:       true,
	}
)
