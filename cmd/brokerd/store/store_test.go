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

func TestSaveAndGetBrokerRequest(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	s := newStore(t)

	br := broker.BrokerRequest{
		ID:      "BR1",
		DataCid: castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jH1"),
		Status:  broker.RequestBatching,
	}

	// Test not found.
	_, err := s.GetBrokerRequest(ctx, "BR1")
	require.Equal(t, ErrNotFound, err)

	// Test creation
	err = s.CreateBrokerRequest(ctx, br)
	require.NoError(t, err)
	br2, err := s.GetBrokerRequest(ctx, "BR1")
	require.NoError(t, err)
	require.Equal(t, br.ID, br2.ID)
	require.Equal(t, br.DataCid, br2.DataCid)
	require.Equal(t, br.Status, br2.Status)
	assert.True(t, time.Since(br2.CreatedAt) < 100*time.Millisecond, time.Since(br2.CreatedAt))
	assert.True(t, time.Since(br2.UpdatedAt) < 100*time.Millisecond, time.Since(br2.UpdatedAt))
}

func TestCreateBatch(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	s := newStore(t)

	// 0- Test retrieving a non-existent storage deal
	_, err := s.GetBatch(ctx, broker.BatchID("fake"))
	require.Error(t, ErrNotFound, err)

	// 1- Create two broker requests.
	br1 := broker.BrokerRequest{
		ID:      "BR1",
		DataCid: castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jH2"),
		Status:  broker.RequestBatching,
	}
	br2 := broker.BrokerRequest{
		ID:      "BR2",
		DataCid: castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jH1"),
		Status:  broker.RequestBatching,
	}
	err = s.CreateBrokerRequest(ctx, br1)
	require.NoError(t, err)
	err = s.CreateBrokerRequest(ctx, br2)
	require.NoError(t, err)

	// 2- Create a storage deal linked with those two broker request.
	sd := broker.Batch{
		ID:         "SD1",
		PayloadCid: castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jB1"),
		Status:     broker.BatchStatusPreparing,
	}
	brIDs := []broker.BrokerRequestID{br1.ID, br2.ID}
	err = s.CreateBatch(ctx, &sd, brIDs)
	require.NoError(t, err)

	// 3- Get the created storage deal by id, and check that fields are coherent.
	sd2, err := s.GetBatch(ctx, sd.ID)
	require.NoError(t, err)
	assert.Equal(t, sd.ID, sd2.ID)
	assert.Equal(t, sd.Status, sd2.Status)
	assert.True(t, time.Since(sd2.CreatedAt) < 100*time.Millisecond, time.Since(sd2.CreatedAt))
	assert.True(t, time.Since(sd2.UpdatedAt) < 100*time.Millisecond, time.Since(sd2.CreatedAt))

	// 4- Check that both broker requests are associated to the storage deal.
	brs, err := s.db.GetBrokerRequests(ctx, BatchIDToSQL(sd.ID))
	require.NoError(t, err)
	assert.Equal(t, len(brIDs), len(brs))

	// 5- Check that the underlying storage request, moved to Preparing and are linked to the created
	//    storage deal!
	gbr1, err := s.GetBrokerRequest(ctx, br1.ID)
	require.NoError(t, err)
	assert.Equal(t, broker.RequestPreparing, gbr1.Status)
	assert.Equal(t, sd.ID, gbr1.BatchID)
	assert.True(t, time.Since(gbr1.UpdatedAt) < 100*time.Millisecond, time.Since(gbr1.UpdatedAt))
	gbr2, err := s.GetBrokerRequest(ctx, br2.ID)
	require.NoError(t, err)
	assert.Equal(t, broker.RequestPreparing, gbr2.Status)
	assert.Equal(t, sd.ID, gbr2.BatchID)
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
