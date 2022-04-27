package filclient

import (
	"context"
	"encoding/json"
	"fmt"

	smtypes "github.com/filecoin-project/boost/storagemarket/types"
	"github.com/filecoin-project/boost/storagemarket/types/dealcheckpoints"
	"github.com/filecoin-project/go-address"
	cborutil "github.com/filecoin-project/go-cbor-util"
	"github.com/filecoin-project/go-fil-markets/storagemarket"
	"github.com/filecoin-project/go-fil-markets/storagemarket/network"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/google/uuid"
	"github.com/ipfs/go-cid"
	"github.com/jsign/go-filsigner/wallet"
	"github.com/textileio/broker-core/cmd/dealerd/store"
	"github.com/textileio/broker-core/metrics"
	"github.com/textileio/go-auctions-client/propsigner"
)

const dealStatusProtocolV110 = "/fil/storage/status/1.1.0"
const dealStatusProtocolV120 = "/fil/storage/status/1.2.0"

// CheckDealStatusWithStorageProvider checks a deal proposal status with a storage-provider.
// The caller should be aware that shouldn't fully trust data from storage-providers.
// To fully confirm the deal, a call to CheckChainDeal must be made after the storage-provider
// publishes the deal on-chain.
func (fc *FilClient) CheckDealStatusWithStorageProvider(
	ctx context.Context,
	storageProviderID string,
	dealIdentifier string,
	rw *store.RemoteWallet,
) (pcid *cid.Cid, ds storagemarket.StorageDealStatus, err error) {
	log.Debugf("checking status of proposal %s with storage-provider %s", dealIdentifier, storageProviderID)
	defer func() {
		metrics.MetricIncrCounter(ctx, err, fc.metricCheckDealStatusWithStorageProvider)
	}()
	sp, err := address.NewFromString(storageProviderID)
	if err != nil {
		return nil,
			storagemarket.StorageDealUnknown,
			fmt.Errorf("invalid storage-provider address %s: %s", storageProviderID, err)
	}

	if propCid, err := cid.Decode(dealIdentifier); err == nil {
		state, err := fc.checkDealStatusV110(ctx, sp, propCid, rw)
		if err != nil {
			return nil, storagemarket.StorageDealUnknown, fmt.Errorf("check deal status v1.1.0 for %s: %s", propCid, err)
		}
		return state.PublishCid, state.State, nil
	}
	dealUID, err := uuid.Parse(dealIdentifier)
	if err != nil {
		return nil, storagemarket.StorageDealUnknown, fmt.Errorf("parsing deal uuid %s: %s", dealUID, err)
	}
	publishCid, state, err := fc.checkDealStatusV120(ctx, sp, dealUID, rw)
	if err != nil {
		return nil, storagemarket.StorageDealUnknown, fmt.Errorf("check deal status v1.2.0 for %s: %s", dealUID, err)
	}
	return publishCid, state, nil
}

func (fc *FilClient) checkDealStatusV120(
	ctx context.Context,
	sp address.Address,
	dealUID uuid.UUID,
	rw *store.RemoteWallet) (*cid.Cid, storagemarket.StorageDealStatus, error) {
	sig, err := fc.signDealStatusRequest(ctx, cid.Undef, dealUID, rw)
	if err != nil {
		return nil, storagemarket.StorageDealUnknown, fmt.Errorf("creating deal status request: %s", err)
	}

	req := &smtypes.DealStatusRequest{
		DealUUID:  dealUID,
		Signature: *sig,
	}

	s, err := fc.streamToStorageProvider(ctx, sp, dealStatusProtocolV120)
	if err != nil {
		return nil, storagemarket.StorageDealUnknown, fmt.Errorf("opening stream with %s: %s", sp, err)
	}
	if err := cborutil.WriteCborRPC(s, req); err != nil {
		return nil, storagemarket.StorageDealUnknown, fmt.Errorf("failed to write status request: %w", err)
	}
	var resp smtypes.DealStatusResponse
	if err := cborutil.ReadCborRPC(s, &resp); err != nil {
		return nil, storagemarket.StorageDealUnknown, fmt.Errorf("reading response: %w", err)
	}

	respJSON, _ := json.MarshalIndent(resp, "", " ")
	log.Debugf("storage-provider %s replied dealuuid %s status check: %s (full: %s)",
		sp, dealUID, resp.DealStatus.Status, respJSON)

	if resp.Error != "" {
		log.Warnf("deal %s status error from %s: %s", dealUID, sp, resp.Error)
		return nil, storagemarket.StorageDealUnknown, fmt.Errorf("deal status v1.2.0 error: %s", resp.Error)
	}
	if resp.DealStatus == nil {
		return nil, storagemarket.StorageDealUnknown, fmt.Errorf("deal status is nil")
	}

	return resp.DealStatus.PublishCid, toLegacyDealStatus(resp.DealStatus), nil
}

func toLegacyDealStatus(ds *smtypes.DealStatus) storagemarket.StorageDealStatus {
	if ds.Error != "" {
		return storagemarket.StorageDealError
	}

	switch ds.Status {
	case dealcheckpoints.Accepted.String():
		return storagemarket.StorageDealWaitingForData
	case dealcheckpoints.Transferred.String():
		return storagemarket.StorageDealVerifyData
	case dealcheckpoints.Published.String():
		return storagemarket.StorageDealPublishing
	case dealcheckpoints.PublishConfirmed.String():
		return storagemarket.StorageDealStaged
	case dealcheckpoints.AddedPiece.String():
		return storagemarket.StorageDealAwaitingPreCommit
	case dealcheckpoints.IndexedAndAnnounced.String():
		return storagemarket.StorageDealAwaitingPreCommit
	case dealcheckpoints.Complete.String():
		return storagemarket.StorageDealSealing
	}

	return storagemarket.StorageDealUnknown
}

func (fc *FilClient) checkDealStatusV110(
	ctx context.Context,
	sp address.Address,
	propCid cid.Cid,
	rw *store.RemoteWallet) (*storagemarket.ProviderDealState, error) {
	sig, err := fc.signDealStatusRequest(ctx, propCid, uuid.Nil, rw)
	if err != nil {
		return nil, fmt.Errorf("creating deal status request: %s", err)
	}
	req := &network.DealStatusRequest{
		Proposal:  propCid,
		Signature: *sig,
	}

	s, err := fc.streamToStorageProvider(ctx, sp, dealStatusProtocolV110)
	if err != nil {
		return nil, fmt.Errorf("opening stream with %s: %s", sp, err)
	}

	if err := cborutil.WriteCborRPC(s, req); err != nil {
		return nil, fmt.Errorf("failed to write status request: %w", err)
	}

	var resp network.DealStatusResponse
	if err := cborutil.ReadCborRPC(s, &resp); err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}
	log.Debugf("storage-provider %s replied proposal %s status check: %s",
		sp, propCid, storagemarket.DealStates[resp.DealState.State])

	if resp.DealState.State == storagemarket.StorageDealError {
		log.Warnf("deal %s error from %s: %s", resp.DealState.ProposalCid, sp, resp.DealState.Message)
	}

	return &resp.DealState, nil
}

func (fc *FilClient) signDealStatusRequest(
	ctx context.Context,
	propCid cid.Cid,
	dealUUID uuid.UUID,
	rw *store.RemoteWallet) (*crypto.Signature, error) {
	var err error
	var sig *crypto.Signature
	if rw == nil {
		var payload []byte
		if dealUUID == uuid.Nil {
			payload, err = cborutil.Dump(propCid)
			if err != nil {
				return nil, fmt.Errorf("encoding proposalcid in cbor: %s", err)
			}
		} else {
			payload, err = dealUUID.MarshalBinary()
			if err != nil {
				return nil, fmt.Errorf("binary marshaling dealuuid %s: %s", propCid, err)
			}
		}

		sig, err = wallet.WalletSign(fc.conf.exportedHexKey, payload)
		if err != nil {
			return nil, fmt.Errorf("signing status request failed: %w", err)
		}
	} else {
		peerID, err := fc.connectToRemoteWallet(ctx, rw)
		if err != nil {
			return nil, fmt.Errorf("connecting to remote wallet: %s", err)
		}

		log.Debugf("requesting proposal cid (%s/%s) remote signature to %s", propCid, dealUUID, peerID)
		var signaturePayload []byte
		if dealUUID == uuid.Nil {
			signaturePayload, err = propCid.MarshalBinary()
			if err != nil {
				return nil, fmt.Errorf("binary marshaling proposal cid %s: %s", propCid, err)
			}
		} else {
			signaturePayload, err = dealUUID.MarshalBinary()
			if err != nil {
				return nil, fmt.Errorf("getting uuid bytes: %w", err)
			}
		}
		sig, err = propsigner.RequestDealStatusSignatureV1(
			ctx, fc.host, rw.AuthToken, rw.WalletAddr, signaturePayload, peerID)
		if err != nil {
			return nil, fmt.Errorf("remote signing proposal: %s", err)
		}
		log.Debugf("remote proposal cid (%s/%s) signature from %s is valid", peerID)
	}

	return sig, nil
}
