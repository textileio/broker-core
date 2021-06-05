package queue

import (
	"context"
	"crypto/rand"
	"strings"
	"testing"
	"time"

	"github.com/ipfs/go-cid"
	util "github.com/ipfs/go-ipfs-util"
	golog "github.com/ipfs/go-log/v2"
	"github.com/oklog/ulid/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/textileio/broker-core/broker"
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

func TestQueue_newID(t *testing.T) {
	t.Parallel()
	q := newQueue(t)

	// Ensure monotonic
	var last broker.AuctionID
	for i := 0; i < 10000; i++ {
		id, err := q.newID(time.Now())
		require.NoError(t, err)

		if i > 0 {
			assert.Greater(t, id, last)
		}
		last = id
	}
}

func TestQueue_ListAuctions(t *testing.T) {
	t.Parallel()
	q := newQueue(t)

	t.Run("pagination", func(t *testing.T) {
		limit := 100
		now := time.Now()
		ids := make([]broker.AuctionID, limit)
		for i := 0; i < limit; i++ {
			now = now.Add(time.Millisecond)
			id, err := q.CreateAuction(broker.Auction{
				StorageDealID:   broker.StorageDealID(strings.ToLower(ulid.MustNew(ulid.Now(), rand.Reader).String())),
				DataCid:         cid.NewCidV1(cid.Raw, util.Hash([]byte("howdy"))),
				DealSize:        1024,
				DealDuration:    1,
				DealReplication: 1,
				Duration:        time.Second,
			})
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

	id, err := q.CreateAuction(broker.Auction{
		StorageDealID:   broker.StorageDealID(strings.ToLower(ulid.MustNew(ulid.Now(), rand.Reader).String())),
		DataCid:         cid.NewCidV1(cid.Raw, util.Hash([]byte("howdy"))),
		DealSize:        1024,
		DealDuration:    1,
		DealReplication: 1,
		Duration:        time.Millisecond,
	})
	require.NoError(t, err)

	// Allow to finish
	time.Sleep(time.Millisecond * 200)

	got, err := q.GetAuction(id)
	require.NoError(t, err)
	assert.NotEmpty(t, got.ID)
	assert.NotEmpty(t, got.StorageDealID)
	assert.Equal(t, broker.AuctionStatusEnded, got.Status)
	assert.True(t, got.DataCid.Defined())
	assert.Equal(t, 1024, int(got.DealSize))
	assert.Equal(t, 1, int(got.DealDuration))
	assert.Equal(t, 1, int(got.DealReplication))
	assert.False(t, got.DealVerified)
	assert.Empty(t, got.Bids)
	assert.Empty(t, got.WinningBids)
	assert.Equal(t, 0, int(got.Attempts))
	assert.Empty(t, got.ErrorCause)
	assert.False(t, got.StartedAt.IsZero())
	assert.False(t, got.UpdatedAt.IsZero())
}

func newQueue(t *testing.T) *Queue {
	s, err := badger.NewDatastore(t.TempDir(), &badger.DefaultOptions)
	require.NoError(t, err)
	q := NewQueue(s, runner, finalizer, 2)
	t.Cleanup(func() {
		require.NoError(t, q.Close())
		require.NoError(t, s.Close())
	})
	return q
}

func runner(
	_ context.Context,
	_ broker.Auction,
	_ func(bid broker.Bid) (broker.BidID, error),
) (map[broker.BidID]broker.WinningBid, error) {
	time.Sleep(time.Millisecond * 100)
	return nil, nil
}

func finalizer(_ context.Context, _ broker.Auction) error {
	return nil
}
