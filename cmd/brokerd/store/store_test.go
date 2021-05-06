package store

import (
	"context"
	"testing"
	"time"

	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/require"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/tests"
)

func TestSaveAndGetBrokerRequest(t *testing.T) {
	ctx := context.Background()
	s := newStore(t)

	now := time.Now()
	br := broker.BrokerRequest{
		ID:      "BR1",
		DataCid: castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jH1"),
		Status:  broker.RequestBatching,
		Metadata: broker.Metadata{
			Region: "asia",
		},
		StorageDealID: broker.StorageDealID("SD1"),
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
	require.Equal(t, br.Metadata.Region, br2.Metadata.Region)
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
	require.Equal(t, br2.Metadata.Region, br3.Metadata.Region)
	require.Equal(t, br2.StorageDealID, br3.StorageDealID)
	require.Equal(t, now.Unix(), br3.CreatedAt.Unix())
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
