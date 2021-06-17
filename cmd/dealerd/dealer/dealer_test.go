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

var (
	fakeProposalCid        = castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jHZ")
	fakePublishDealMessage = castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jZZ")
	fakeDealID             = int64(1337)
	fakeExpiration         = uint64(98765)
	auds                   = dealeri.AuctionDeals{
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
)

func TestReadyToCreateDeals(t *testing.T) {
	t.Parallel()

	broker := &brokerMock{}
	dealer := newDealer(t, broker)

	err := dealer.ReadyToCreateDeals(context.Background(), auds)
	require.NoError(t, err)

	aud, ok, err := dealer.store.GetNext(store.PendingDealMaking)
	require.NoError(t, err)
	require.True(t, ok)

	// Check that the corresponding AuctionDeal has correct values.
	require.NotEmpty(t, aud.ID)
	require.NotEmpty(t, aud.AuctionDataID)
	require.Equal(t, auds.Targets[0].Miner, aud.Miner)
	require.Equal(t, auds.Targets[0].PricePerGiBPerEpoch, aud.PricePerGiBPerEpoch)
	require.Equal(t, auds.Targets[0].StartEpoch, aud.StartEpoch)
	require.Equal(t, auds.Targets[0].Verified, aud.Verified)
	require.Equal(t, auds.Targets[0].FastRetrieval, aud.FastRetrieval)
	require.Equal(t, store.ExecutingDealMaking, aud.Status)
	require.Empty(t, aud.ErrorCause)
	require.True(t, time.Since(aud.CreatedAt) < time.Minute)
	require.Equal(t, cid.Undef, aud.ProposalCid)
	require.Equal(t, int64(0), aud.DealID)
	require.Equal(t, uint64(0), aud.DealExpiration)

	// Check that the corresponding AuctionData has correct values.
	ad, err := dealer.store.GetAuctionData(aud.AuctionDataID)
	require.NoError(t, err)
	require.Equal(t, aud.AuctionDataID, ad.ID)
	require.Equal(t, auds.StorageDealID, ad.StorageDealID)
	require.Equal(t, auds.PayloadCid, ad.PayloadCid)
	require.Equal(t, auds.PieceCid, ad.PieceCid)
	require.Equal(t, auds.PieceSize, ad.PieceSize)
	require.Equal(t, auds.Duration, ad.Duration)
}

func TestStateMachineExecPending(t *testing.T) {
	t.Parallel()

	broker := &brokerMock{}
	dealer := newDealer(t, broker)
	err := dealer.ReadyToCreateDeals(context.Background(), auds)
	require.NoError(t, err)

	// Tick deal making.
	err = dealer.daemonDealMakerTick()
	require.NoError(t, err)

	// Check there're no more pending deals.
	_, ok, err := dealer.store.GetNext(store.PendingDealMaking)
	require.NoError(t, err)
	require.False(t, ok)

	// Check the deal moved to PendingConfirmation.
	wc, ok, err := dealer.store.GetNext(store.PendingConfirmation)
	require.NoError(t, err)
	require.True(t, ok)

	require.Equal(t, store.ExecutingConfirmation, wc.Status)
	require.Empty(t, wc.ErrorCause)
	require.True(t, time.Since(wc.UpdatedAt) < time.Minute)
	require.Equal(t, fakeProposalCid, wc.ProposalCid) // Crucial check.
	require.Equal(t, int64(0), wc.DealID)             // Can't be set at this stage.
	require.Equal(t, uint64(0), wc.DealExpiration)    // Can't be set at this stage.
}

func TestStateMachineExecWaitingConfirmation(t *testing.T) {
	t.Parallel()

	broker := &brokerMock{}
	dealer := newDealer(t, broker)

	err := dealer.ReadyToCreateDeals(context.Background(), auds)
	require.NoError(t, err)

	// Tick deal making.
	err = dealer.daemonDealMakerTick()
	require.NoError(t, err)

	// Tick deal monitoring.
	err = dealer.daemonDealMonitoringTick()
	require.NoError(t, err)

	// Check there're no more deals waiting for confirmation.
	_, ok, err := dealer.store.GetNext(store.PendingConfirmation)
	require.NoError(t, err)
	require.False(t, ok)

	// Check the deal moved to Success
	finalized, ok, err := dealer.store.GetNext(store.PendingReportFinalized)
	require.NoError(t, err)
	require.True(t, ok)

	require.Equal(t, store.ExecutingReportFinalized, finalized.Status)
	require.Empty(t, finalized.ErrorCause)
	require.True(t, time.Since(finalized.UpdatedAt) < time.Minute)
	require.Equal(t, fakeDealID, finalized.DealID)             // Crucial check
	require.Equal(t, fakeExpiration, finalized.DealExpiration) // Crucial check

	// Check that dealer has notified broker of accepted proposal.
	require.Equal(t, auds.StorageDealID, broker.callerPASdID)
	require.Equal(t, auds.Targets[0].Miner, broker.calledPAMiner)
	require.Equal(t, fakeProposalCid, broker.calledPAProposalCid)
}

func TestStateMachineExecReporting(t *testing.T) {
	broker := &brokerMock{}
	dealer := newDealer(t, broker)

	err := dealer.ReadyToCreateDeals(context.Background(), auds)
	require.NoError(t, err)

	// Tick deal making.
	err = dealer.daemonDealMakerTick()
	require.NoError(t, err)

	// Tick deal monitoring.
	err = dealer.daemonDealMonitoringTick()
	require.NoError(t, err)

	// Tick deal reporting.
	err = dealer.daemonDealReporterTick()
	require.NoError(t, err)

	// There shouldn't be ANY auction deal, since
	// things get removed from the datastore after being reported.
	statuses := []store.AuctionDealStatus{store.PendingDealMaking, store.PendingConfirmation, store.PendingReportFinalized}
	for _, ss := range statuses {
		_, ok, err := dealer.store.GetNext(ss)
		require.NoError(t, err)
		require.False(t, ok)
	}

	// Check that the broker was reported with the deal
	// results.
	report := broker.calledFAD
	require.Equal(t, auds.StorageDealID, report.StorageDealID)
	require.Equal(t, fakeDealID, report.DealID)
	require.Equal(t, fakeExpiration, report.DealExpiration)
	require.Empty(t, report.ErrorCause)
}

func newDealer(t *testing.T, broker broker.Broker) *Dealer {
	// Mock a happy-path filclient.
	fc := &fcMock{}
	fc.On("ExecuteAuctionDeal", mock.Anything, mock.Anything, mock.Anything).Return(fakeProposalCid, false, nil)

	cdswmCall := fc.On("CheckDealStatusWithMiner", mock.Anything, mock.Anything, mock.Anything)
	cdswmCall.Return(&storagemarket.ProviderDealState{
		PublishCid: &fakePublishDealMessage,
	}, nil)

	rdfmCall := fc.On("ResolveDealIDFromMessage", mock.Anything, fakeProposalCid, fakePublishDealMessage)
	rdfmCall.Return(fakeDealID, nil)

	fc.On("CheckChainDeal", mock.Anything, fakeDealID).Return(true, fakeExpiration, false, nil)

	fc.On("GetChainHeight", mock.Anything).Return(uint64(1111111), nil)

	ds := tests.NewTxMapDatastore()
	opts := []Option{
		WithDealMakingFreq(time.Hour),
		WithDealWatchingFreq(time.Hour),
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

func (fc *fcMock) ExecuteAuctionDeal(
	ctx context.Context,
	ad store.AuctionData,
	aud store.AuctionDeal) (cid.Cid, bool, error) {
	args := fc.Called(ctx, ad, aud)
	return args.Get(0).(cid.Cid), args.Bool(1), args.Error(2)
}

func (fc *fcMock) GetChainHeight(ctx context.Context) (uint64, error) {
	args := fc.Called(ctx)
	return args.Get(0).(uint64), args.Error(1)
}
func (fc *fcMock) ResolveDealIDFromMessage(
	ctx context.Context,
	proposalCid cid.Cid,
	publishDealMessage cid.Cid) (int64, error) {
	args := fc.Called(ctx, proposalCid, publishDealMessage)
	return args.Get(0).(int64), args.Error(1)
}

func (fc *fcMock) CheckChainDeal(ctx context.Context, dealID int64) (bool, uint64, bool, error) {
	args := fc.Called(ctx, dealID)
	return args.Bool(0), args.Get(1).(uint64), args.Bool(2), args.Error(3)
}
func (fc *fcMock) CheckDealStatusWithMiner(
	ctx context.Context,
	minerAddr string,
	propCid cid.Cid) (*storagemarket.ProviderDealState, error) {
	args := fc.Called(ctx, minerAddr, propCid)

	return args.Get(0).(*storagemarket.ProviderDealState), args.Error(1)
}

type brokerMock struct {
	calledFAD broker.FinalizedAuctionDeal

	callerPASdID        broker.StorageDealID
	calledPAMiner       string
	calledPAProposalCid cid.Cid
}

func (b *brokerMock) CreateStorageDeal(
	ctx context.Context,
	batchCid cid.Cid,
	srids []broker.BrokerRequestID) (broker.StorageDealID, error) {
	panic("shouldn't be called")
}

func (b *brokerMock) StorageDealPrepared(
	ctx context.Context,
	id broker.StorageDealID,
	pr broker.DataPreparationResult) error {
	panic("shouldn't be called")
}

func (b *brokerMock) StorageDealAuctioned(ctx context.Context, auction broker.Auction) error {
	panic("shouldn't be called")
}

func (b *brokerMock) StorageDealProposalAccepted(
	_ context.Context,
	sdID broker.StorageDealID,
	miner string,
	proposalCid cid.Cid) error {
	b.callerPASdID = sdID
	b.calledPAMiner = miner
	b.calledPAProposalCid = proposalCid
	return nil
}

func (b *brokerMock) StorageDealFinalizedDeal(ctx context.Context, res broker.FinalizedAuctionDeal) error {
	b.calledFAD = res
	return nil
}

func (b *brokerMock) Create(context.Context, cid.Cid) (broker.BrokerRequest, error) {
	panic("shouldn't be called")
}

func (b *brokerMock) CreatePrepared(context.Context, cid.Cid, broker.PreparedCAR) (broker.BrokerRequest, error) {
	panic("shouldn't be called")
}

func (b *brokerMock) Get(context.Context, broker.BrokerRequestID) (broker.BrokerRequest, error) {
	panic("shouldn't be called")
}
