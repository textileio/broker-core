package dealer

import (
	"context"
	"testing"
	"time"

	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/dealerd/dealer/store"
	dealeri "github.com/textileio/broker-core/dealer"
	"github.com/textileio/broker-core/tests"
)

func TestReadyToCreateDeals(t *testing.T) {
	dealer := newDealer(t, nil, nil)

	auds := dealeri.AuctionDeals{
		StorageDealID: "SD1",
		PayloadCid:    castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jH1"),
		PieceCid:      castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jH2"),
		Duration:      123,
		PieceSize:     456,
		Targets: []dealeri.AuctionDealsTarget{
			{
				Miner:               "f0001",
				FastRetrieval:       true,
				PricePerGiBPerEpoch: 100,
				StartEpoch:          200,
				Verified:            true,
			},
		},
	}
	err := dealer.ReadyToCreateDeals(context.Background(), auds)
	require.NoError(t, err)

	dsAuds, err := dealer.store.GetAllAuctionDeals(store.Pending)
	require.NoError(t, err)
	require.Len(t, dsAuds, 1)

	// Check that the corresponding AuctionDeal has correct values.
	target := dsAuds[0]
	require.NotEmpty(t, target.ID)
	require.NotEmpty(t, target.AuctionDataID)
	require.Equal(t, auds.Targets[0].Miner, target.Miner)
	require.Equal(t, auds.Targets[0].PricePerGiBPerEpoch, target.PricePerGiBPerEpoch)
	require.Equal(t, auds.Targets[0].StartEpoch, target.StartEpoch)
	require.Equal(t, auds.Targets[0].Verified, target.Verified)
	require.Equal(t, auds.Targets[0].FastRetrieval, target.FastRetrieval)
	require.Equal(t, store.Pending, target.Status)
	require.Empty(t, target.ErrorCause)
	require.True(t, time.Since(target.CreatedAt) < time.Minute)
	require.Equal(t, cid.Undef, target.ProposalCid)
	require.Equal(t, int64(0), target.DealID)
	require.Equal(t, uint64(0), target.DealExpiration)

	// Check that the corresponding AuctionData has correct values.
	ad, err := dealer.store.GetAuctionData(target.AuctionDataID)
	require.NoError(t, err)
	require.Equal(t, target.AuctionDataID, ad.ID)
	require.Equal(t, auds.StorageDealID, ad.StorageDealID)
	require.Equal(t, auds.PayloadCid, ad.PayloadCid)
	require.Equal(t, auds.PieceCid, ad.PieceCid)
	require.Equal(t, auds.PieceSize, ad.PieceSize)
	require.Equal(t, auds.Duration, ad.Duration)
}

func TestStateMachineExecPending(t *testing.T) {
	fakeProposalCid := castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jHZ")
	fc := &fcMock{}
	fc.On("ExecuteAuctionDeal", mock.Anything, mock.Anything, mock.Anything).Return(fakeProposalCid, nil).Times(1)

	dealer := newDealer(t, nil, fc)
	auds := dealeri.AuctionDeals{
		StorageDealID: "SD1",
		PayloadCid:    castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jH1"),
		PieceCid:      castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jH2"),
		Duration:      123,
		PieceSize:     456,
		Targets: []dealeri.AuctionDealsTarget{
			{
				Miner:               "f0001",
				FastRetrieval:       true,
				PricePerGiBPerEpoch: 100,
				StartEpoch:          200,
				Verified:            true,
			},
		},
	}
	err := dealer.ReadyToCreateDeals(context.Background(), auds)
	require.NoError(t, err)

	err = dealer.daemonDealMakerTick()
	require.NoError(t, err)

	// Check there're no more pending deals.
	pendings, err := dealer.store.GetAllAuctionDeals(store.Pending)
	require.Len(t, pendings, 0)

	// Check the deal moved to WaitingConfirmation
	waitingConfirm, err := dealer.store.GetAllAuctionDeals(store.WaitingConfirmation)
	require.NoError(t, err)
	require.Len(t, waitingConfirm, 1)

	wc := waitingConfirm[0]
	require.Equal(t, store.WaitingConfirmation, wc.Status)
	require.Empty(t, wc.ErrorCause)
	require.True(t, time.Since(wc.UpdatedAt) < time.Minute)
	require.Equal(t, fakeProposalCid, wc.ProposalCid) // Crucial check.
	require.Equal(t, int64(0), wc.DealID)             // Can't be set at this stage.
	require.Equal(t, uint64(0), wc.DealExpiration)    // Can't be set at this stage.

	fc.AssertExpectations(t)
}

func newDealer(t *testing.T, broker broker.Broker, fc Filclient) *Dealer {
	ds := tests.NewTxMapDatastore()
	opts := []Option{
		WithDealMakingFreq(time.Hour),
		WithDealMonitoringFreq(time.Hour),
		WithDealReportingFreq(time.Hour),
	}
	dealer, err := New(ds, broker, fc, opts...)
	require.NoError(t, err)
	return dealer
}

func castCid(cidStr string) cid.Cid {
	c, _ := cid.Decode(cidStr)
	return c
}

type fcMock struct {
	mock.Mock
}

func (fc *fcMock) ExecuteAuctionDeal(ctx context.Context, ad store.AuctionData, aud store.AuctionDeal) (cid.Cid, error) {
	args := fc.Called(ctx, ad, aud)

	return args.Get(0).(cid.Cid), args.Error(1)

}
func (fc *fcMock) GetChainHeight(ctx context.Context) (uint64, error) {
	args := fc.Called(ctx)

	return args.Get(0).(uint64), args.Error(1)
}
func (fc *fcMock) ResolveDealIDFromMessage(ctx context.Context, proposalCid cid.Cid, publishDealMessage cid.Cid) (int64, error) {
	args := fc.Called(ctx, proposalCid, publishDealMessage)

	return args.Get(0).(int64), args.Error(1)
}

func (fc *fcMock) CheckChainDeal(ctx context.Context, dealID int64) (bool, uint64, error) {
	args := fc.Called(ctx, dealID)

	return args.Bool(0), args.Get(1).(uint64), args.Error(2)

}
func (fc *fcMock) CheckDealStatusWithMiner(
	ctx context.Context,
	minerAddr string,
	propCid cid.Cid) (*storagemarket.ProviderDealState, error) {
	args := fc.Called(ctx, minerAddr, propCid)

	return args.Get(0).(*storagemarket.ProviderDealState), args.Error(1)
}
