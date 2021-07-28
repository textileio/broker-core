package store

import (
	"context"
	"testing"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/tests"
)

func TestSaveAndGetStorageRequest(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	s := newStore(t)

	br := broker.StorageRequest{
		ID:      "BR1",
		DataCid: castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jH1"),
		Status:  broker.RequestBatching,
		Origin:  "ORIGIN-1",
	}

	// Test not found.
	_, err := s.GetStorageRequest(ctx, "BR1")
	require.Equal(t, ErrNotFound, err)

	// Test creation
	err = s.CreateStorageRequest(ctx, br)
	require.NoError(t, err)
	br2, err := s.GetStorageRequest(ctx, "BR1")
	require.NoError(t, err)
	require.Equal(t, br.ID, br2.ID)
	require.Equal(t, br.DataCid, br2.DataCid)
	require.Equal(t, br.Status, br2.Status)
	require.Equal(t, br.Origin, br2.Origin)
	assert.True(t, time.Since(br2.CreatedAt) < 100*time.Millisecond, time.Since(br2.CreatedAt))
	assert.True(t, time.Since(br2.UpdatedAt) < 100*time.Millisecond, time.Since(br2.UpdatedAt))
}

func TestCreateBatch(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	s := newStore(t)

	// 0- Test retrieving a non-existent batch
	_, err := s.GetBatch(ctx, broker.BatchID("fake"))
	require.Error(t, ErrNotFound, err)

	// 1- Create two storage requests.
	br1 := broker.StorageRequest{
		ID:      "BR1",
		DataCid: castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jH2"),
		Status:  broker.RequestBatching,
	}
	br2 := broker.StorageRequest{
		ID:      "BR2",
		DataCid: castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jH1"),
		Status:  broker.RequestBatching,
	}
	err = s.CreateStorageRequest(ctx, br1)
	require.NoError(t, err)
	err = s.CreateStorageRequest(ctx, br2)
	require.NoError(t, err)

	// 2- Create a batch linked with those two storage request.
	ba := broker.Batch{
		ID:         "SD1",
		PayloadCid: castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jB1"),
		Status:     broker.BatchStatusPreparing,
		Origin:     "ORIGIN-1",
	}
	brIDs := []broker.StorageRequestID{br1.ID, br2.ID}
	err = s.CreateBatch(ctx, &ba, brIDs)
	require.NoError(t, err)

	// 3- Get the created batch by id, and check that fields are coherent.
	sd2, err := s.GetBatch(ctx, ba.ID)
	require.NoError(t, err)
	assert.Equal(t, ba.ID, sd2.ID)
	assert.Equal(t, ba.Status, sd2.Status)
	assert.Equal(t, "ORIGIN-1", sd2.Origin)
	assert.True(t, time.Since(sd2.CreatedAt) < 100*time.Millisecond, time.Since(sd2.CreatedAt))
	assert.True(t, time.Since(sd2.UpdatedAt) < 100*time.Millisecond, time.Since(sd2.CreatedAt))

	// 4- Check that both storage requests are associated to the batch.
	brs, err := s.db.GetStorageRequests(ctx, batchIDToSQL(ba.ID))
	require.NoError(t, err)
	assert.Equal(t, len(brIDs), len(brs))

	// 5- Check that the underlying storage request, moved to Preparing and are linked to the created
	//    batch!
	gbr1, err := s.GetStorageRequest(ctx, br1.ID)
	require.NoError(t, err)
	assert.Equal(t, broker.RequestPreparing, gbr1.Status)
	assert.Equal(t, ba.ID, gbr1.BatchID)
	assert.True(t, time.Since(gbr1.UpdatedAt) < 100*time.Millisecond, time.Since(gbr1.UpdatedAt))
	gbr2, err := s.GetStorageRequest(ctx, br2.ID)
	require.NoError(t, err)
	assert.Equal(t, broker.RequestPreparing, gbr2.Status)
	assert.Equal(t, ba.ID, gbr2.BatchID)
	assert.True(t, time.Since(gbr2.UpdatedAt) < 100*time.Millisecond, time.Since(gbr2.UpdatedAt))
}

func newStore(t *testing.T) *Store {
	u, err := tests.PostgresURL()
	require.NoError(t, err)
	s, err := New(u)
	require.NoError(t, err)
	return s
}

func castCid(cidStr string) cid.Cid {
	c, _ := cid.Decode(cidStr)
	return c
}
