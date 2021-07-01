package store

import (
	"context"
	"testing"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"
	"github.com/textileio/bidbot/lib/auction"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/tests"
)

func TestSaveAndGetBrokerRequest(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	s := newStore(t)

	now := time.Now()
	br := broker.BrokerRequest{
		ID:            "BR1",
		DataCid:       castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jH1"),
		Status:        broker.RequestBatching,
		StorageDealID: auction.StorageDealID("SD1"),
		CreatedAt:     now,
		UpdatedAt:     now,
	}

	// Test not found.
	_, err := s.GetBrokerRequest(ctx, "BR1")
	require.Equal(t, ErrNotFound, err)

	// Test creation
	err = s.SaveBrokerRequest(ctx, br)
	require.NoError(t, err)
	br2, err := s.GetBrokerRequest(ctx, "BR1")
	require.NoError(t, err)
	require.Equal(t, br.ID, br2.ID)
	require.Equal(t, br.DataCid, br2.DataCid)
	require.Equal(t, br.Status, br2.Status)
	require.Equal(t, br.StorageDealID, br2.StorageDealID)
	require.Equal(t, now.Unix(), br2.CreatedAt.Unix())
	require.Equal(t, now.Unix(), br2.UpdatedAt.Unix())

	// Test modification.
	br2.Status = broker.RequestAuctioning
	err = s.SaveBrokerRequest(ctx, br2)
	require.NoError(t, err)
	br3, err := s.GetBrokerRequest(ctx, "BR1")
	require.NoError(t, err)
	require.Equal(t, br2.ID, br3.ID)
	require.Equal(t, br2.DataCid, br3.DataCid)
	require.Equal(t, br2.Status, br3.Status)
	require.Equal(t, br2.StorageDealID, br3.StorageDealID)
	require.Equal(t, now.Unix(), br3.CreatedAt.Unix())
}

func TestCreateStorageDeal(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	s := newStore(t)

	// 0- Test retrieving a non-existent storage deal
	_, err := s.GetStorageDeal(ctx, auction.StorageDealID("fake"))
	require.Error(t, ErrNotFound, err)

	// 1- Create two broker requests.
	now := time.Now()
	br1 := broker.BrokerRequest{
		ID:        "BR1",
		DataCid:   castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jH2"),
		Status:    broker.RequestBatching,
		CreatedAt: now,
		UpdatedAt: now,
	}
	br2 := broker.BrokerRequest{
		ID:        "BR1",
		DataCid:   castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jH1"),
		Status:    broker.RequestBatching,
		CreatedAt: now,
		UpdatedAt: now,
	}
	err = s.SaveBrokerRequest(ctx, br1)
	require.NoError(t, err)
	err = s.SaveBrokerRequest(ctx, br2)
	require.NoError(t, err)

	// 2- Create a storage deal linked with those two broker request.
	now = time.Now()
	sd := broker.StorageDeal{
		PayloadCid:       castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jB1"),
		Status:           broker.StorageDealPreparing,
		BrokerRequestIDs: []broker.BrokerRequestID{br1.ID, br2.ID},
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	err = s.CreateStorageDeal(ctx, &sd)
	require.NoError(t, err)

	// 3- Get the created storage deal by id, and check that fields are coherent.
	sd2, err := s.GetStorageDeal(ctx, sd.ID)
	require.NoError(t, err)
	require.Equal(t, sd.ID, sd2.ID)
	require.Equal(t, sd.Status, sd2.Status)
	require.Equal(t, len(sd.BrokerRequestIDs), len(sd2.BrokerRequestIDs))
	require.Equal(t, sd.CreatedAt.Unix(), sd2.CreatedAt.Unix())
	require.Equal(t, sd.UpdatedAt.Unix(), sd2.UpdatedAt.Unix())

	// 4- Check that the underlying storage request, moved to Preparing and are linked to the created
	//    storage deal!
	gbr1, err := s.GetBrokerRequest(ctx, br1.ID)
	require.NoError(t, err)
	require.Equal(t, broker.RequestPreparing, gbr1.Status)
	require.Equal(t, sd.ID, gbr1.StorageDealID)
	require.True(t, time.Since(gbr1.UpdatedAt) < time.Second)
	gbr2, err := s.GetBrokerRequest(ctx, br2.ID)
	require.NoError(t, err)
	require.Equal(t, broker.RequestPreparing, gbr2.Status)
	require.Equal(t, sd.ID, gbr2.StorageDealID)
	require.True(t, time.Since(gbr2.UpdatedAt) < time.Second)
}

func newStore(t *testing.T) *Store {
	ds := tests.NewTxMapDatastore()
	s, err := New(ds)
	require.NoError(t, err)
	return s
}

func castCid(cidStr string) cid.Cid {
	c, _ := cid.Decode(cidStr)
	return c
}
