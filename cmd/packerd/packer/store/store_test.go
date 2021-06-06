package store

import (
	"testing"

	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/logging"
	"github.com/textileio/broker-core/tests"

	golog "github.com/ipfs/go-log/v2"
)

func init() {
	if err := logging.SetLogLevels(map[string]golog.LogLevel{
		"packer/store": golog.LevelDebug,
	}); err != nil {
		panic(err)
	}
}

func TestFull(t *testing.T) {
	t.Parallel()

	ds := tests.NewTxMapDatastore()
	s, err := New(ds, 100, 4)
	require.NoError(t, err)

	br1 := BatchableBrokerRequest{
		BrokerRequestID: broker.BrokerRequestID("1"),
		DataCid:         castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jHA"),
		Size:            1,
	}
	br2 := BatchableBrokerRequest{
		BrokerRequestID: broker.BrokerRequestID("2"),
		DataCid:         castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jHB"),
		Size:            2,
	}
	br3 := BatchableBrokerRequest{
		BrokerRequestID: broker.BrokerRequestID("3"),
		DataCid:         castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jHC"),
		Size:            3,
	}

	// Enqueue three batchable broker requests.
	err = s.Enqueue(br1)
	require.NoError(t, err)
	_, _, ok, err := s.GetNextReadyBatch()
	require.NoError(t, err)
	require.False(t, ok)

	err = s.Enqueue(br2)
	require.NoError(t, err)
	_, _, ok, err = s.GetNextReadyBatch()
	require.NoError(t, err)
	require.False(t, ok)

	err = s.Enqueue(br3)
	require.NoError(t, err)
	batch, bbrs, ok, err := s.GetNextReadyBatch()
	require.NoError(t, err)
	require.True(t, ok)
	require.Len(t, bbrs, 3)
	require.NotEmpty(t, batch.ID)
	require.Equal(t, int64(len(bbrs)), batch.Count)
	require.Equal(t, int64(6), batch.Size)

	// The only batch that exists is Executing now.
	// Check that getting a next one shouldn't return a result.
	_, _, ok, err = s.GetNextReadyBatch()
	require.NoError(t, err)
	require.False(t, ok)

	// Test deleting a prepared batch.
	err = s.DeleteBatch(batch.ID)
	require.NoError(t, err)
}

func TestTriggerNewBatch(t *testing.T) {
	t.Parallel()

	ds := tests.NewTxMapDatastore()

	// Ask for a minimum size of 4, but maximum of 5.
	s, err := New(ds, 5, 4)
	require.NoError(t, err)

	br1 := BatchableBrokerRequest{
		BrokerRequestID: broker.BrokerRequestID("1"),
		DataCid:         castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jHA"),
		Size:            1,
	}
	br2 := BatchableBrokerRequest{
		BrokerRequestID: broker.BrokerRequestID("2"),
		DataCid:         castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jHB"),
		Size:            2,
	}
	br3 := BatchableBrokerRequest{
		BrokerRequestID: broker.BrokerRequestID("3"),
		DataCid:         castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jHC"),
		Size:            3,
	}

	// With the first and second  isn't enough...
	err = s.Enqueue(br1)
	require.NoError(t, err)
	_, _, ok, err := s.GetNextReadyBatch()
	require.NoError(t, err)
	require.False(t, ok)
	err = s.Enqueue(br2)
	require.NoError(t, err)
	_, _, ok, err = s.GetNextReadyBatch()
	require.NoError(t, err)
	require.False(t, ok)

	// The third one shouldn't fit in the open batch. It should trigger
	// a new open batch.
	err = s.Enqueue(br3)
	require.NoError(t, err)
	_, _, ok, err = s.GetNextReadyBatch()
	require.NoError(t, err)
	require.False(t, ok)

	// At this point, we should have two open batches:
	// - Batch A (size 3): with the first two enqueued broker requests.
	// - Batch B (size 3): with the third one.

	// Adding b4 should close the first one. Let's check that.
	br4 := BatchableBrokerRequest{
		BrokerRequestID: broker.BrokerRequestID("4"),
		DataCid:         castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jHD"),
		Size:            1,
	}
	err = s.Enqueue(br4)
	require.NoError(t, err)
	batch, bbrs, ok, err := s.GetNextReadyBatch()
	require.NoError(t, err)
	require.True(t, ok)
	require.Len(t, bbrs, 3)
	require.NotEmpty(t, batch.ID)
	require.Equal(t, int64(len(bbrs)), batch.Count)
	require.Equal(t, int64(4), batch.Size)

	// Adding b4 should close the second one. Let's check that.
	br5 := BatchableBrokerRequest{
		BrokerRequestID: broker.BrokerRequestID("4"),
		DataCid:         castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jHE"),
		Size:            2,
	}
	err = s.Enqueue(br5)
	require.NoError(t, err)
	batch, bbrs, ok, err = s.GetNextReadyBatch()
	require.NoError(t, err)
	require.True(t, ok)
	require.Len(t, bbrs, 2)
	require.NotEmpty(t, batch.ID)
	require.Equal(t, int64(len(bbrs)), batch.Count)
	require.Equal(t, int64(5), batch.Size)

	// There should be no remaining open batches.
	_, _, ok, err = s.GetNextReadyBatch()
	require.NoError(t, err)
	require.False(t, ok)
}

func TestViolateMaxSize(t *testing.T) {
	t.Parallel()

	ds := tests.NewTxMapDatastore()

	s, err := New(ds, 1, 1)
	require.NoError(t, err)

	br1 := BatchableBrokerRequest{
		BrokerRequestID: broker.BrokerRequestID("1"),
		DataCid:         castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jHA"),
		Size:            10,
	}

	err = s.Enqueue(br1)
	require.Error(t, err)
}

func castCid(cidStr string) cid.Cid {
	c, _ := cid.Decode(cidStr)
	return c
}
