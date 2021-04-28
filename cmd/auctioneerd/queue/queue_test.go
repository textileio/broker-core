package queue_test

import (
	"context"
	"io/ioutil"
	"testing"
	"time"

	"github.com/google/uuid"
	golog "github.com/ipfs/go-log/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/textileio/broker-core/broker"
	. "github.com/textileio/broker-core/cmd/auctioneerd/queue"
	"github.com/textileio/broker-core/logging"
	badger "github.com/textileio/go-ds-badger3"
)

func init() {
	if err := logging.SetLogLevels(map[string]golog.LogLevel{
		"auctioneer/queue": golog.LevelDebug,
	}); err != nil {
		panic(err)
	}
}

func TestQueue_NewID(t *testing.T) {
	t.Parallel()
	q := newQueue(t)

	// Ensure monotonic
	var last broker.AuctionID
	for i := 0; i < 10000; i++ {
		id, err := q.NewID(time.Now())
		require.NoError(t, err)

		if i > 0 {
			assert.Greater(t, id, last)
		}
		last = id
	}
}

func TestQueue_ListRequests(t *testing.T) {
	t.Parallel()
	q := newQueue(t)

	t.Run("pagination", func(t *testing.T) {
		limit := 100
		now := time.Now()
		ids := make([]broker.AuctionID, limit)
		for i := 0; i < limit; i++ {
			now = now.Add(time.Millisecond)
			id, err := q.CreateAuction(uuid.NewString(), 0, 0, time.Second)
			require.NoError(t, err)
			ids[i] = id
		}

		// Allow all to finish
		time.Sleep(time.Second)

		// Empty query, should return newest 10 records
		l, err := q.ListAuctions(Query{})
		require.NoError(t, err)
		assert.Len(t, l, 10)
		assert.Equal(t, ids[limit-1], l[0].ID)
		assert.Equal(t, ids[limit-10], l[9].ID)

		// Get next page, should return next 10 records
		offset := l[len(l)-1].ID
		l, err = q.ListAuctions(Query{Offset: string(offset)})
		require.NoError(t, err)
		assert.Len(t, l, 10)
		assert.Equal(t, ids[limit-11], l[0].ID)
		assert.Equal(t, ids[limit-20], l[9].ID)

		// Get previous page, should return the first page in reverse order
		offset = l[0].ID
		l, err = q.ListAuctions(Query{Offset: string(offset), Order: OrderAscending})
		require.NoError(t, err)
		assert.Len(t, l, 10)
		assert.Equal(t, ids[limit-10], l[0].ID)
		assert.Equal(t, ids[limit-1], l[9].ID)
	})
}

func TestQueue_CreateAuction(t *testing.T) {
	t.Parallel()
	q := newQueue(t)

	id, err := q.CreateAuction(uuid.NewString(), 0, 0, time.Millisecond)
	require.NoError(t, err)

	// Allow to finish
	time.Sleep(time.Millisecond * 200)

	got, err := q.GetAuction(id)
	require.NoError(t, err)
	assert.Equal(t, broker.AuctionStatusEnded, got.Status)
}

func newQueue(t *testing.T) *Queue {
	dir, err := ioutil.TempDir("", "")
	require.NoError(t, err)
	s, err := badger.NewDatastore(dir, &badger.DefaultOptions)
	require.NoError(t, err)
	q, err := NewQueue(s, handler)
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, q.Close())
		require.NoError(t, s.Close())
	})
	return q
}

func handler(_ context.Context, _ *broker.Auction) error {
	time.Sleep(time.Millisecond * 100)
	return nil
}
