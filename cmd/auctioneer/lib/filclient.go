package lib

import (
	"io"

	"github.com/libp2p/go-libp2p-core/peer"
)

// FilClient provides functionalities to verify bidders.
type FilClient interface {
	io.Closer

	VerifyBidder(walletAddr string, bidderSig []byte, bidderID peer.ID) (bool, error)
	GetChainHeight() (uint64, error)
}
