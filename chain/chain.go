package chain

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-jsonrpc"
	"github.com/filecoin-project/go-state-types/crypto"
	"github.com/filecoin-project/lotus/api"
	"github.com/filecoin-project/lotus/api/apistruct"
	"github.com/libp2p/go-libp2p-core/peer"
	"github.com/textileio/broker-core/finalizer"
)

var (
	// FullNodeAddr is the URL used with the Lotus JSON RPC client.
	FullNodeAddr = "https://api.node.glif.io"

	requestTimeout = time.Second * 10
)

// Chain provides a limited set of Filecoin chain methods.
type Chain interface {
	io.Closer

	VerifyBidder(walletAddr string, bidderSig []byte, bidderID peer.ID) (bool, error)
	GetChainHeight() (uint64, error)
}

type chain struct {
	api api.FullNode

	ctx       context.Context
	finalizer *finalizer.Finalizer
}

// New returns a new Chain.
func New() (Chain, error) {
	fin := finalizer.NewFinalizer()
	ctx, cancel := context.WithCancel(context.Background())
	fin.Add(finalizer.NewContextCloser(cancel))

	var fn apistruct.FullNodeStruct
	closer, err := jsonrpc.NewClient(ctx, FullNodeAddr, "Filecoin", &fn.Internal, http.Header{})
	if err != nil {
		return nil, fmt.Errorf("creating json rpc client: %v", err)
	}
	fin.AddFn(closer)

	return &chain{
		api:       &fn,
		ctx:       ctx,
		finalizer: fin,
	}, nil
}

// Close the node.
func (c *chain) Close() error {
	return c.finalizer.Cleanup(nil)
}

// VerifyBidder ensures that the wallet address authorized the use of bidder peer.ID to make bids.
// Miner's authorize a bidding peer.ID by signing it with a wallet address private key.
func (c *chain) VerifyBidder(walletAddr string, bidderSig []byte, bidderID peer.ID) (bool, error) {
	pubkey, err := address.NewFromString(walletAddr)
	if err != nil {
		return false, fmt.Errorf("parsing wallet address: %v", err)
	}

	var sig crypto.Signature
	err = sig.UnmarshalBinary(bidderSig)
	if err != nil {
		return false, fmt.Errorf("unmarshaling signature: %v", err)
	}

	ctx, cancel := context.WithTimeout(c.ctx, requestTimeout)
	defer cancel()
	ok, err := c.api.WalletVerify(ctx, pubkey, []byte(bidderID), &sig)
	if err != nil {
		return false, fmt.Errorf("verifying signature: %v", err)
	}
	return ok, nil
}

// GetChainHeight returns the current chain height in epochs.
func (c *chain) GetChainHeight() (uint64, error) {
	ctx, cancel := context.WithTimeout(c.ctx, requestTimeout)
	defer cancel()
	ts, err := c.api.ChainHead(ctx)
	if err != nil {
		return 0, fmt.Errorf("getting chain head: %v", err)
	}
	return uint64(ts.Height()), nil
}
