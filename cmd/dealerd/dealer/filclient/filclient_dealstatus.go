package filclient

import (
	"context"
	"fmt"

	"github.com/filecoin-project/go-address"
	cborutil "github.com/filecoin-project/go-cbor-util"
	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/filecoin-project/go-fil-markets/storagemarket/network"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/ipfs/go-cid"
	"github.com/jsign/go-filsigner/wallet"
	"github.com/textileio/broker-core/cmd/dealerd/store"
	"github.com/textileio/broker-core/metrics"
	"github.com/textileio/go-auctions-client/propsigner"
)

const dealStatusProtocol = "/fil/storage/status/1.1.0"

// CheckDealStatusWithStorageProvider checks a deal proposal status with a storage-provider.
// The caller should be aware that shouldn't fully trust data from storage-providers.
// To fully confirm the deal, a call to CheckChainDeal must be made after the storage-provider
// publishes the deal on-chain.
func (fc *FilClient) CheckDealStatusWithStorageProvider(
	ctx context.Context,
	storageProviderID string,
	propCid cid.Cid,
	rw *store.RemoteWallet,
) (status *storagemarket.ProviderDealState, err error) {
	log.Debugf("checking status of proposal %s with storage-provider %s", propCid, storageProviderID)
	defer func() {
		metrics.MetricIncrCounter(ctx, err, fc.metricCheckDealStatusWithStorageProvider)
	}()
	sp, err := address.NewFromString(storageProviderID)
	if err != nil {
		return nil, fmt.Errorf("invalid storage-provider address %s: %s", storageProviderID, err)
	}

	req, err := fc.createDealStatusRequest(ctx, propCid, rw)
	if err != nil {
		return nil, fmt.Errorf("creating deal status request: %s", err)
	}

	s, err := fc.streamToStorageProvider(ctx, sp, dealStatusProtocol)
	if err != nil {
		return nil, fmt.Errorf("opening stream with %s: %s", storageProviderID, err)
	}

	if err := cborutil.WriteCborRPC(s, req); err != nil {
		return nil, fmt.Errorf("failed to write status request: %w", err)
	}

	var resp network.DealStatusResponse
	if err := cborutil.ReadCborRPC(s, &resp); err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	log.Debugf("storage-provider %s replied proposal %s status check: %s",
		storageProviderID, propCid, storagemarket.DealStates[resp.DealState.State])

	if resp.DealState.State == storagemarket.StorageDealError {
		log.Warnf("deal %s error from %s: %s", resp.DealState.ProposalCid, storageProviderID, resp.DealState.Message)
	}

	return &resp.DealState, nil
}

func (fc *FilClient) createDealStatusRequest(
	ctx context.Context,
	propCid cid.Cid,
	rw *store.RemoteWallet) (*network.DealStatusRequest, error) {
	var sig *crypto.Signature
	if rw == nil {
		cidb, err := cborutil.Dump(propCid)
		if err != nil {
			return nil, fmt.Errorf("encoding proposal in cbor: %s", err)
		}
		sig, err = wallet.WalletSign(fc.conf.exportedHexKey, cidb)
		if err != nil {
			return nil, fmt.Errorf("signing status request failed: %w", err)
		}
	} else {
		peerID, err := fc.connectToRemoteWallet(ctx, rw)
		if err != nil {
			return nil, fmt.Errorf("connecting to remote wallet: %s", err)
		}

		log.Debugf("requesting proposal cid remote signature to %s", peerID)
		sig, err = propsigner.RequestDealStatusSignatureV1(ctx, fc.host, rw.AuthToken, rw.WalletAddr, propCid, peerID)
		if err != nil {
			return nil, fmt.Errorf("remote signing proposal: %s", err)
		}
		log.Debugf("remote proposal cid signature from %s is valid", peerID)
	}

	return &network.DealStatusRequest{
		Proposal:  propCid,
		Signature: *sig,
	}, nil
}
