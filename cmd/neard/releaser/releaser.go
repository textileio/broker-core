package releaser

import (
	"context"
	"time"

	"github.com/textileio/broker-core/cmd/neard/contractclient"
	logging "github.com/textileio/go-log/v2"
)

var (
	log = logging.Logger("neard/releaser")
)

// Releaser manages calling releaseDeposits on the contract.
type Releaser struct {
	cc      *contractclient.Client
	t       *time.Ticker
	timeout time.Duration
	close   chan struct{}
}

// New creates a new Releaser.
func New(cc *contractclient.Client, freq, timeout time.Duration) *Releaser {
	r := &Releaser{
		cc:      cc,
		t:       time.NewTicker(freq),
		timeout: timeout,
		close:   make(chan struct{}),
	}
	r.start()
	return r
}

func (r *Releaser) start() {
	go func() {
		for {
			select {
			case <-r.t.C:
				ctx, cancel := context.WithTimeout(context.Background(), r.timeout)
				if err := r.cc.ReleaseDeposits(ctx); err != nil {
					log.Errorf("calling release deposits: %v", err)
				}
				cancel()
			case <-r.close:
				r.t.Stop()
				return
			}
		}
	}()
}

// Close shuts down the Releaser.
func (r *Releaser) Close() error {
	close(r.close)
	return nil
}
