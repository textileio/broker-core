package store

import (
	"testing"

	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/tests"
)

func TestFull(t *testing.T) {
	t.Parallel()

	ds := tests.NewTxMapDatastore()
	s, err := New(ds)
	require.NoError(t, err)

	br1 := BatchableBrokerRequest{
		BrokerRequestID: broker.BrokerRequestID("1"),
		DataCid:         castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jHA"),
	}
	br2 := BatchableBrokerRequest{
		BrokerRequestID: broker.BrokerRequestID("2"),
		DataCid:         castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jHB"),
	}
	br3 := BatchableBrokerRequest{
		BrokerRequestID: broker.BrokerRequestID("3"),
		DataCid:         castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jHC"),
	}

	// Enqueue three batchable broker requests.
	err = s.Enqueue(br1)
	require.NoError(t, err)
	err = s.Enqueue(br2)
	require.NoError(t, err)
	err = s.Enqueue(br3)
	require.NoError(t, err)
	require.Len(t, s.queue, 3)

	// Override br1, br2, br3 IDs with what was generated internally.
	// We don't have explicit "GET" APIs in this store.
	br1.ID = s.queue[0].ID
	br2.ID = s.queue[1].ID
	br3.ID = s.queue[2].ID

	// Now test Iterator, and dequeuing and verifying things work as expected.

	// Iterate and check that we see all three in order
	brs := []BatchableBrokerRequest{br1, br2, br3}
	assertIterator(t, s, brs)

	// Remove br2 in the middle.
	err = s.Dequeue([]BatchableBrokerRequest{br2})
	require.NoError(t, err)
	brs = []BatchableBrokerRequest{br1, br3}
	assertIterator(t, s, brs)

	// Remove br1 in the front.
	err = s.Dequeue([]BatchableBrokerRequest{br1})
	require.NoError(t, err)
	brs = []BatchableBrokerRequest{br3}
	assertIterator(t, s, brs)

	// Remove last one.
	err = s.Dequeue([]BatchableBrokerRequest{br3})
	require.NoError(t, err)
	assertIterator(t, s, nil)

	// Try to remove something that isn't there.
	err = s.Dequeue([]BatchableBrokerRequest{br3})
	require.Equal(t, ErrNotFound, err)
}

func TestPartialWrongDequeueing(t *testing.T) {
	t.Parallel()

	ds := tests.NewTxMapDatastore()
	s, err := New(ds)
	require.NoError(t, err)

	br1 := BatchableBrokerRequest{
		BrokerRequestID: broker.BrokerRequestID("1"),
		DataCid:         castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jHA"),
	}
	br2 := BatchableBrokerRequest{
		BrokerRequestID: broker.BrokerRequestID("2"),
		DataCid:         castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jHB"),
	}
	br3 := BatchableBrokerRequest{
		BrokerRequestID: broker.BrokerRequestID("3"),
		DataCid:         castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jHC"),
	}

	// Enqueue only the first two.
	err = s.Enqueue(br1)
	require.NoError(t, err)
	err = s.Enqueue(br2)
	require.NoError(t, err)

	// But try to dequeue br2 and br3. This is wrong and should error.
	// Note that we can remove br2, but not br3 (doesn't exist).
	err = s.Dequeue([]BatchableBrokerRequest{br2, br3})
	require.Equal(t, ErrNotFound, err)
	// This should error without removing br2 since we should do removals transactionally. All or nothing.
	// So let's check that.
	brs := []BatchableBrokerRequest{br1, br2}
	assertIterator(t, s, brs)
}

func TestRehydration(t *testing.T) {
	ds := tests.NewTxMapDatastore()
	s, err := New(ds)
	require.NoError(t, err)

	br1 := BatchableBrokerRequest{
		BrokerRequestID: broker.BrokerRequestID("1"),
		DataCid:         castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jHA"),
	}
	br2 := BatchableBrokerRequest{
		BrokerRequestID: broker.BrokerRequestID("2"),
		DataCid:         castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jHB"),
	}
	br3 := BatchableBrokerRequest{
		BrokerRequestID: broker.BrokerRequestID("3"),
		DataCid:         castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jHC"),
	}
	err = s.Enqueue(br1)
	require.NoError(t, err)
	err = s.Enqueue(br2)
	require.NoError(t, err)
	err = s.Enqueue(br3)
	require.NoError(t, err)
	// Check that things are all as expected.
	brs := []BatchableBrokerRequest{br1, br2, br3}
	assertIterator(t, s, brs)

	// Create new *Store, but with same `ds`. Cache should be populated.
	s, err = New(ds)
	require.NoError(t, err)
	// Check things again in this new *Store. Since the Iterator pulls things from
	// its internal in-memory cache, if the following works then it was re-populated correctly.
	assertIterator(t, s, brs)
}

func assertIterator(t *testing.T, s *Store, expectedResult []BatchableBrokerRequest) {
	t.Helper()
	var count int
	it := s.NewIterator()
	for {
		br, ok := it.Next()
		if !ok {
			break
		}
		// Check that we receive them in right order of enqueued.
		require.Equal(t, expectedResult[count].BrokerRequestID, br.BrokerRequestID)
		require.Equal(t, expectedResult[count].DataCid, br.DataCid)
		count++
	}
	require.Len(t, expectedResult, count) // Check we walked exactly all of them.
}

func castCid(cidStr string) cid.Cid {
	c, _ := cid.Decode(cidStr)
	return c
}
