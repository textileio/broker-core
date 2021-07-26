package dealer

import (
	"context"
	"testing"
	"time"

	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/ipfs/go-cid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/textileio/bidbot/lib/logging"
	"github.com/textileio/broker-core/cmd/dealerd/store"
	dealeri "github.com/textileio/broker-core/dealer"
	pb "github.com/textileio/broker-core/gen/broker/v1"
	mbroker "github.com/textileio/broker-core/msgbroker"
	"github.com/textileio/broker-core/msgbroker/fakemsgbroker"
	"github.com/textileio/broker-core/tests"
	golog "github.com/textileio/go-log/v2"
	"google.golang.org/protobuf/proto"
)

func init() {
	if err := logging.SetLogLevels(map[string]golog.LogLevel{"dealer": golog.LevelDebug}); err != nil {
		panic(err)
	}
}

var (
	fakeProposalCid        = castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jHZ")
	fakePublishDealMessage = castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jZZ")
	fakeDealID             = int64(1337)
	fakeExpiration         = uint64(98765)
	auds                   = dealeri.AuctionDeals{
		BatchID:    "SD1",
		PayloadCid: castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jH1"),
		PieceCid:   castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jH2"),
		Duration:   123,
		PieceSize:  456,
		Proposals: []dealeri.Proposal{
			{
				MinerID:             "f0001",
				FastRetrieval:       true,
				PricePerGiBPerEpoch: 100,
				StartEpoch:          200,
				Verified:            true,
				AuctionID:           "auction-1",
				BidID:               "bid-1",
			},
		},
	}
)

func TestReadyToCreateDeals(t *testing.T) {
	t.Parallel()

	dealer, _ := newDealer(t)

	ctx := context.Background()
	err := dealer.ReadyToCreateDeals(ctx, auds)
	require.NoError(t, err)

	aud, ok, err := dealer.store.GetNextPending(ctx, store.StatusDealMaking)
	require.NoError(t, err)
	require.True(t, ok)

	// Check that the corresponding AuctionDeal has correct values.
	require.NotEmpty(t, aud.ID)
	require.NotEmpty(t, aud.AuctionDataID)
	require.Equal(t, auds.Proposals[0].MinerID, aud.MinerID)
	require.Equal(t, auds.Proposals[0].PricePerGiBPerEpoch, aud.PricePerGibPerEpoch)
	require.Equal(t, auds.Proposals[0].StartEpoch, aud.StartEpoch)
	require.Equal(t, auds.Proposals[0].Verified, aud.Verified)
	require.Equal(t, auds.Proposals[0].FastRetrieval, aud.FastRetrieval)
	require.Equal(t, auds.Proposals[0].AuctionID, aud.AuctionID)
	require.Equal(t, auds.Proposals[0].BidID, aud.BidID)
	require.Equal(t, store.StatusDealMaking, store.AuctionDealStatus(aud.Status))
	require.True(t, aud.Executing)
	require.Empty(t, aud.ErrorCause)
	require.True(t, time.Since(aud.CreatedAt) < time.Minute)
	require.Equal(t, "", aud.ProposalCid)
	require.Equal(t, int64(0), aud.DealID)
	require.Equal(t, uint64(0), aud.DealExpiration)

	// Check that the corresponding AuctionData has correct values.
	ad, err := dealer.store.GetAuctionData(ctx, aud.AuctionDataID)
	require.NoError(t, err)
	require.Equal(t, aud.AuctionDataID, ad.ID)
	require.Equal(t, auds.BatchID, ad.BatchID)
	require.Equal(t, auds.PayloadCid, ad.PayloadCid)
	require.Equal(t, auds.PieceCid, ad.PieceCid)
	require.Equal(t, auds.PieceSize, ad.PieceSize)
	require.Equal(t, auds.Duration, ad.Duration)
}

func TestStateMachineExecPending(t *testing.T) {
	t.Parallel()

	dealer, _ := newDealer(t)
	ctx := context.Background()
	err := dealer.ReadyToCreateDeals(ctx, auds)
	require.NoError(t, err)

	// Tick deal making.
	err = dealer.daemonDealMakerTick()
	require.NoError(t, err)

	// Check there're no more pending deals.
	_, ok, err := dealer.store.GetNextPending(ctx, store.StatusDealMaking)
	require.NoError(t, err)
	require.False(t, ok)

	// Check the deal moved to PendingConfirmation.
	wc, ok, err := dealer.store.GetNextPending(ctx, store.StatusConfirmation)
	require.NoError(t, err)
	require.True(t, ok)

	require.Equal(t, store.StatusConfirmation, store.AuctionDealStatus(wc.Status))
	require.True(t, wc.Executing)
	require.Empty(t, wc.ErrorCause)
	require.True(t, time.Since(wc.UpdatedAt) < time.Minute)
	require.Equal(t, fakeProposalCid.String(), wc.ProposalCid) // Crucial check.
	require.Equal(t, int64(0), wc.DealID)                      // Can't be set at this stage.
	require.Equal(t, uint64(0), wc.DealExpiration)             // Can't be set at this stage.
}

func TestStateMachineExecWaitingConfirmation(t *testing.T) {
	t.Parallel()

	dealer, mb := newDealer(t)

	ctx := context.Background()
	err := dealer.ReadyToCreateDeals(ctx, auds)
	require.NoError(t, err)

	// Tick deal making.
	err = dealer.daemonDealMakerTick()
	require.NoError(t, err)

	// Tick deal monitoring.
	err = dealer.daemonDealMonitoringTick()
	require.NoError(t, err)

	// Check there're no more deals waiting for confirmation.
	_, ok, err := dealer.store.GetNextPending(ctx, store.StatusConfirmation)
	require.NoError(t, err)
	require.False(t, ok)

	// Check the deal moved to Success
	finalized, ok, err := dealer.store.GetNextPending(ctx, store.StatusReportFinalized)
	require.NoError(t, err)
	require.True(t, ok)

	require.Equal(t, store.StatusReportFinalized, store.AuctionDealStatus(finalized.Status))
	require.True(t, finalized.Executing)
	require.Empty(t, finalized.ErrorCause)
	require.True(t, time.Since(finalized.UpdatedAt) < time.Minute)
	require.Equal(t, fakeDealID, finalized.DealID)             // Crucial check
	require.Equal(t, fakeExpiration, finalized.DealExpiration) // Crucial check

	// Check that dealer has notified broker of accepted proposal.
	require.Equal(t, 1, mb.TotalPublished())
	require.Equal(t, 1, mb.TotalPublishedTopic(mbroker.DealProposalAcceptedTopic))
	data, err := mb.GetMsg(mbroker.DealProposalAcceptedTopic, 0)
	require.NoError(t, err)
	dpa := &pb.DealProposalAccepted{}
	err = proto.Unmarshal(data, dpa)
	require.NoError(t, err)
	require.Equal(t, string(auds.Proposals[0].AuctionID), dpa.AuctionId)
	require.Equal(t, string(auds.Proposals[0].BidID), dpa.BidId)
	require.Equal(t, string(auds.BatchID), dpa.BatchId)
	require.Equal(t, auds.Proposals[0].MinerID, dpa.MinerId)
	require.Equal(t, fakeProposalCid.Bytes(), dpa.ProposalCid)
}

func TestStateMachineExecReporting(t *testing.T) {
	dealer, mb := newDealer(t)

	ctx := context.Background()
	err := dealer.ReadyToCreateDeals(ctx, auds)
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
	statuses := []store.AuctionDealStatus{store.StatusDealMaking, store.StatusConfirmation, store.StatusReportFinalized}
	for _, ss := range statuses {
		_, ok, err := dealer.store.GetNextPending(ctx, ss)
		require.NoError(t, err)
		require.False(t, ok)
	}

	// Msg 1: Proposal accepted, Msg 2: report finalized.
	require.Equal(t, 2, mb.TotalPublished())
	require.Equal(t, 1, mb.TotalPublishedTopic(mbroker.FinalizedDealTopic))
	data, err := mb.GetMsg(mbroker.FinalizedDealTopic, 0)
	require.NoError(t, err)
	fd := &pb.FinalizedDeal{}
	err = proto.Unmarshal(data, fd)
	require.NoError(t, err)

	require.Equal(t, string(auds.BatchID), fd.BatchId)
	require.Equal(t, fakeDealID, fd.DealId)
	require.Equal(t, fakeExpiration, fd.DealExpiration)
	require.Empty(t, fd.ErrorCause)
}

func newDealer(t *testing.T) (*Dealer, *fakemsgbroker.FakeMsgBroker) {
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

	u, err := tests.PostgresURL()
	require.NoError(t, err)
	opts := []Option{
		WithDealMakingFreq(time.Hour),
		WithDealWatchingFreq(time.Hour),
		WithDealReportingFreq(time.Hour),
	}

	mb := fakemsgbroker.New()
	dealer, err := New(u, mb, fc, opts...)
	require.NoError(t, err)

	return dealer, mb
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
	minerID string,
	propCid cid.Cid) (*storagemarket.ProviderDealState, error) {
	args := fc.Called(ctx, MinerID, propCid)

	return args.Get(0).(*storagemarket.ProviderDealState), args.Error(1)
}
