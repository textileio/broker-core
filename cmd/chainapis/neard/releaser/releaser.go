package releaser

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/textileio/broker-core/cmd/chainapis/neard/providerclient"
	logging "github.com/textileio/go-log/v2"
)

// Releaser manages calling releaseDeposits on the contract.
type Releaser struct {
	cc      *providerclient.Client
	t       *time.Ticker
	timeout time.Duration
	close   chan struct{}

	log *logging.ZapEventLogger
}

// New creates a new Releaser.
func New(cc *providerclient.Client, chainID string, freq, timeout time.Duration) (*Releaser, error) {
	if chainID == "" {
		return nil, errors.New("no chain id provided")
	}
	if freq <= 0 {
		return nil, fmt.Errorf("invalid freq: %v", freq)
	}
	if timeout <= 0 {
		return nil, fmt.Errorf("invalid timeout: %v", timeout)
	}

	r := &Releaser{
		cc:      cc,
		t:       time.NewTicker(freq),
		timeout: timeout,
		close:   make(chan struct{}),
		log:     logging.Logger(fmt.Sprintf("neard-releaser-%s", chainID)),
	}

	_ = logging.SetLogLevel(fmt.Sprintf("neard-releaser-%s", chainID), "INFO")

	r.start()
	return r, nil
}

func (r *Releaser) start() {
	go func() {
		for {
			select {
			case <-r.t.C:
				r.log.Info("checking state for expired deposits")
				ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
				state, err := r.cc.GetState(ctx)
				if err != nil {
					r.log.Errorf("calling get state: %v", err)
					cancel()
					continue
				}
				cancel()
				needsRelease := false
				for _, deposit := range state.DepositMap {
					// value > 0 && currentTimestamp <= DepositTimestamp + value / sessionDivisor
					zero := (&big.Int{}).SetInt64(0)
					sessionLength := (&big.Int{}).Div(deposit.Value, state.SessionDivisor)
					if deposit.Value.Cmp(zero) > 0 && time.Now().UnixNano() <= deposit.Timestamp+sessionLength.Int64() {
						needsRelease = true
						break
					}
				}
				if needsRelease {
					r.log.Info("calling release deposits")
					ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
					if err := r.cc.ReleaseDeposits(ctx); err != nil {
						r.log.Errorf("calling release deposits: %v", err)
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
