package releaser

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/textileio/broker-core/cmd/chainapis/ethshared/contractclient"
	logging "github.com/textileio/go-log/v2"
)

var (
	log *logging.ZapEventLogger
)

// Releaser manages calling releaseDeposits on the contract.
type Releaser struct {
	cc         *contractclient.BridgeProvider
	clientAddr common.Address
	signer     bind.SignerFn
	t          *time.Ticker
	timeout    time.Duration
	close      chan struct{}
}

// New creates a new Releaser.
func New(
	cc *contractclient.BridgeProvider,
	chainID string,
	clientAddr common.Address,
	signer bind.SignerFn,
	freq, timeout time.Duration,
) (*Releaser, error) {
	if chainID == "" {
		return nil, errors.New("no chain id provided")
	}
	if signer == nil {
		return nil, errors.New("no signer provided")
	}
	if freq <= 0 {
		return nil, fmt.Errorf("invalid freq: %v", freq)
	}
	if timeout <= 0 {
		return nil, fmt.Errorf("invalid timeout: %v", timeout)
	}

	log = logging.Logger(fmt.Sprintf("ethd-releaser-%s", chainID))

	r := &Releaser{
		cc:         cc,
		clientAddr: clientAddr,
		signer:     signer,
		t:          time.NewTicker(freq),
		timeout:    timeout,
		close:      make(chan struct{}),
	}
	r.start()
	return r, nil
}

func (r *Releaser) start() {
	go func() {
		for {
			select {
			case <-r.t.C:
				needsRelease := true
				// TODO: Lookup in the Graph state if we need to release.
				if needsRelease {
					log.Info("calling release deposits")
					ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
					if _, err := r.cc.ReleaseDeposits(&bind.TransactOpts{
						Context: ctx,
						From:    r.clientAddr,
						Signer:  r.signer,
					}); err != nil {
						log.Errorf("calling release deposits: %v", err)
					}
					cancel()
				}
			case <-r.close:
				r.t.Stop()
				return
			}
		}
	}()
}
