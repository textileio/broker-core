package filclient

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/apistruct"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/textileio/broker-core/finalizer"
)

var requestTimeout = time.Second * 10

// FilClient provides functionalities to verify bidders.
type FilClient struct {
	api api.FullNode

	ctx       context.Context
	finalizer *finalizer.Finalizer
}

// New returns a new FilClient.
func New(lotusGatewayURL string) (*FilClient, error) {
	fin := finalizer.NewFinalizer()
	ctx, cancel := context.WithCancel(context.Background())
	fin.Add(finalizer.NewContextCloser(cancel))

	var fn apistruct.FullNodeStruct
	closer, err := jsonrpc.NewClient(ctx, lotusGatewayURL, "Filecoin", &fn.Internal, http.Header{})
	if err != nil {
		return nil, fmt.Errorf("creating json rpc client: %v", err)
	}
	fin.AddFn(closer)

	return &FilClient{
		api:       &fn,
		ctx:       ctx,
		finalizer: fin,
	}, nil
}

// Close the client.
func (fc *FilClient) Close() error {
	return fc.finalizer.Cleanup(nil)
}

// VerifyBidder ensures that the wallet address authorized the use of bidder peer.ID to make bids.
// Miner's authorize a bidding peer.ID by signing it with a wallet address private key.
func (fc *FilClient) VerifyBidder(
	walletAddr string,
	bidderSig []byte,
	bidderID peer.ID,
	minerAddrStr string) (bool, error) {
	pubkey, err := address.NewFromString(walletAddr)
	if err != nil {
		return false, fmt.Errorf("parsing wallet address: %v", err)
	}

	var sig crypto.Signature
	err = sig.UnmarshalBinary(bidderSig)
	if err != nil {
		return false, fmt.Errorf("unmarshaling signature: %v", err)
	}

	minerAddr, err := address.NewFromString(minerAddrStr)
	if err != nil {
		return false, fmt.Errorf("parsing miner address: %s", err)
	}
	ctx, cancel := context.WithTimeout(fc.ctx, requestTimeout)
	defer cancel()
	mi, err := fc.api.StateMinerInfo(ctx, minerAddr, types.EmptyTSK)
	if err != nil {
		return false, fmt.Errorf("getting on-chain miner info: %s", err)
	}
	ownerWalletAddr, err := fc.api.StateAccountKey(ctx, mi.Owner, types.EmptyTSK)
	if err != nil {
		return false, fmt.Errorf("get owner walleta ddr: %s", err)
	}

	if ownerWalletAddr.String() != walletAddr {
		return false,
			fmt.Errorf("the owner wallet addr %s doesn't match with the provided addr %s", ownerWalletAddr, walletAddr)
	}

	ctx, cancel = context.WithTimeout(fc.ctx, requestTimeout)
	defer cancel()
	ok, err := fc.api.WalletVerify(ctx, pubkey, []byte(bidderID), &sig)
	if err != nil {
		return false, fmt.Errorf("verifying signature: %v", err)
	}
	return ok, nil
}

// GetChainHeight returns the current chain height in epochs.
func (fc *FilClient) GetChainHeight() (uint64, error) {
	ctx, cancel := context.WithTimeout(fc.ctx, requestTimeout)
	defer cancel()
	ts, err := fc.api.ChainHead(ctx)
	if err != nil {
		return 0, fmt.Errorf("getting chain head: %v", err)
	}
	return uint64(ts.Height()), nil
}
