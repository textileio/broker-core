package releaser

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/textileio/broker-core/cmd/chainapis/neard/contractclient"
	logging "github.com/textileio/go-log/v2"
)

var (
	log *logging.ZapEventLogger
)

// Releaser manages calling releaseDeposits on the contract.
type Releaser struct {
	cc      *contractclient.Client
	t       *time.Ticker
	timeout time.Duration
	close   chan struct{}
}

// New creates a new Releaser.
func New(cc *contractclient.Client, chainID string, freq, timeout time.Duration) (*Releaser, error) {
	if chainID == "" {
		return nil, errors.New("no chain id provided")
	}
	if freq <= 0 {
		return nil, fmt.Errorf("invalid freq: %v", freq)
	}
	if timeout <= 0 {
		return nil, fmt.Errorf("invalid timeout: %v", timeout)
	}

	log = logging.Logger(fmt.Sprintf("neard-releaser-%s", chainID))

	r := &Releaser{
		cc:      cc,
		t:       time.NewTicker(freq),
		timeout: timeout,
		close:   make(chan struct{}),
	}
	r.start()
	return r, nil
}

func (r *Releaser) start() {
	go func() {
		for {
			select {
			case <-r.t.C:
				ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
				state, err := r.cc.GetState(ctx)
				if err != nil {
					log.Errorf("calling get state: %v", err)
					cancel()
					continue
				}
				cancel()
				needsRelease := false
				for _, info := range state.DepositMap {
					if uint64(state.BlockHeight) > info.Deposit.Expiration {
						needsRelease = true
						break
					}
				}
				if needsRelease {
					log.Info("calling release deposits")
					ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
					if err := r.cc.ReleaseDeposits(ctx); err != nil {
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
