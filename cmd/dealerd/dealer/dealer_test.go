package dealer

import (
	"context"
	"testing"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/ipfs/go-cid"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/multiformats/go-multiaddr"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/dealerd/store"
	"github.com/textileio/broker-core/dealer"
	dealeri "github.com/textileio/broker-core/dealer"
	pb "github.com/textileio/broker-core/gen/broker/v1"
	mbroker "github.com/textileio/broker-core/msgbroker"
	"github.com/textileio/broker-core/msgbroker/fakemsgbroker"
	"github.com/textileio/broker-core/tests"
	golog "github.com/textileio/go-log/v2"
	"google.golang.org/protobuf/proto"
)

func init() {
	if err := golog.SetLogLevels(map[string]golog.LogLevel{"dealer": golog.LevelDebug}); err != nil {
		panic(err)
	}
}

var (
	fakeProposalCid        = castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jHZ")
	fakePublishDealMessage = castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jZZ")
	fakeDealID             = int64(1337)
	fakeExpiration         = uint64(98765)
	auds                   = dealeri.AuctionDeals{
		ID:         "id-1",
		BatchID:    "SD1",
		PayloadCid: castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jH1"),
		PieceCid:   castCid("QmdKDf5nepPLXErXd1pYY8hA82yjMaW3fdkU8D8kiz3jH2"),
		Duration:   123,
		PieceSize:  456,
		Proposals: []dealeri.Proposal{
			{
				StorageProviderID:   "f0001",
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

	tests := []struct {
		name string
		auds dealer.AuctionDeals
	}{
		{name: "without remote wallet", auds: auds},
		{name: "with remote wallet", auds: makeRemoteWalletAuds(auds)},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			dealer, _ := newDealer(t)

			ctx := context.Background()
			err := dealer.ReadyToCreateDeals(ctx, test.auds)
			require.NoError(t, err)

			aud, ok, err := dealer.store.GetNextPending(ctx, store.StatusDealMaking)
			require.NoError(t, err)
			require.True(t, ok)

			// Check that the corresponding AuctionDeal has correct values.
			require.NotEmpty(t, aud.ID)
			require.NotEmpty(t, aud.AuctionDataID)
			require.Equal(t, test.auds.Proposals[0].StorageProviderID, aud.StorageProviderID)
			require.Equal(t, test.auds.Proposals[0].PricePerGiBPerEpoch, aud.PricePerGibPerEpoch)
			require.Equal(t, test.auds.Proposals[0].StartEpoch, aud.StartEpoch)
			require.Equal(t, test.auds.Proposals[0].Verified, aud.Verified)
			require.Equal(t, test.auds.Proposals[0].FastRetrieval, aud.FastRetrieval)
			require.Equal(t, test.auds.Proposals[0].AuctionID, aud.AuctionID)
			require.Equal(t, test.auds.Proposals[0].BidID, aud.BidID)
			require.Equal(t, store.StatusDealMaking, store.AuctionDealStatus(aud.Status))
			require.True(t, aud.Executing)
			require.Empty(t, aud.ErrorCause)
			require.True(t, time.Since(aud.CreatedAt) < time.Minute)
			require.Equal(t, "", aud.ProposalCid)
			require.Equal(t, int64(0), aud.DealID)
			require.Equal(t, uint64(0), aud.DealExpiration)

			rw, err := dealer.store.GetRemoteWallet(ctx, aud.AuctionDataID)
			if test.auds.RemoteWallet == nil {
				require.Nil(t, rw)
			} else {
				require.NoError(t, err)
				require.Equal(t, test.auds.RemoteWallet.PeerID.String(), rw.PeerID)
				require.Equal(t, test.auds.RemoteWallet.AuthToken, rw.AuthToken)
				require.Equal(t, test.auds.RemoteWallet.WalletAddr.String(), rw.WalletAddr)
				require.Len(t, rw.Multiaddrs, len(test.auds.RemoteWallet.Multiaddrs))
				for _, maddr := range test.auds.RemoteWallet.Multiaddrs {
					require.Contains(t, rw.Multiaddrs, maddr.String())
				}
			}

			// Check that the corresponding AuctionData has correct values.
			ad, err := dealer.store.GetAuctionData(ctx, aud.AuctionDataID)
			require.NoError(t, err)
			require.Equal(t, aud.AuctionDataID, ad.ID)
			require.Equal(t, auds.BatchID, ad.BatchID)
			require.Equal(t, auds.PayloadCid, ad.PayloadCid)
			require.Equal(t, auds.PieceCid, ad.PieceCid)
			require.Equal(t, auds.PieceSize, ad.PieceSize)
			require.Equal(t, auds.Duration, ad.Duration)
		})
	}
}

func TestReadyToCreateDealsIdempotency(t *testing.T) {
	t.Parallel()

	dealer, _ := newDealer(t)

	ctx := context.Background()
	err := dealer.ReadyToCreateDeals(ctx, auds)
	require.NoError(t, err)
	err = dealer.ReadyToCreateDeals(ctx, auds)
	require.ErrorIs(t, err, store.ErrAuctionDataExists)
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
	require.Equal(t, auds.Proposals[0].StorageProviderID, dpa.StorageProviderId)
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

	// There shouldn't be ANY auction deals in a state other than Finalized.
	statuses := []store.AuctionDealStatus{store.StatusDealMaking, store.StatusConfirmation, store.StatusReportFinalized}
	for _, ss := range statuses {
		_, ok, err := dealer.store.GetNextPending(ctx, ss)
		require.NoError(t, err)
		require.False(t, ok)
	}

	// There should be Finalized auction deals.
	_, ok, err := dealer.store.GetNextPending(ctx, store.StatusFinalized)
	require.NoError(t, err)
	require.True(t, ok)

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
	fc.On("ExecuteAuctionDeal",
		mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(fakeProposalCid, "", false, nil)

	cdswmCall := fc.On("CheckDealStatusWithStorageProvider", mock.Anything, mock.Anything, mock.Anything, mock.Anything)
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
	aud store.AuctionDeal,
	rw *store.RemoteWallet) (cid.Cid, string, bool, error) {
	args := fc.Called(ctx, ad, aud)
	return args.Get(0).(cid.Cid), args.String(1), args.Bool(2), args.Error(3)
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
func (fc *fcMock) CheckDealStatusWithStorageProvider(
	ctx context.Context,
	storageProviderID string,
	propCid cid.Cid,
	dealUUID string,
	rw *store.RemoteWallet) (*cid.Cid, storagemarket.StorageDealStatus, error) {
	args := fc.Called(ctx, storageProviderID, propCid)

	return args.Get(0).(*cid.Cid), args.Get(1).(storagemarket.StorageDealStatus), args.Error(2)
}

func makeRemoteWalletAuds(baseAuds dealeri.AuctionDeals) dealeri.AuctionDeals {
	waddr, err := address.NewFromString(
		"f3wmv7nhiqosmlr6mis2mr4xzupdhe3rtvw5ntis4x6yru7jhm35pfla2pkwgwfa3t62kdmoylssczmf74yika")
	panicIfErr(err)
	maddr1, err := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/1234")
	panicIfErr(err)
	maddr2, err := multiaddr.NewMultiaddr("/ip4/127.0.0.1/tcp/1235")
	panicIfErr(err)

	peerID, err := peer.Decode("Qmf12XE1bSB8SbngTc4Yy8KLNGKT2Nhp2xuj2DFEWX3N5H")
	panicIfErr(err)

	auds := baseAuds
	auds.RemoteWallet = &broker.RemoteWallet{
		PeerID:     peerID,
		WalletAddr: waddr,
		AuthToken:  "auth-token-1",
		Multiaddrs: []multiaddr.Multiaddr{maddr1, maddr2},
	}

	return auds
}

func panicIfErr(err error) {
	if err != nil {
		panic(err)
	}
}
