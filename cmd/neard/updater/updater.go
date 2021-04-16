package updater

import (
	"context"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"github.com/textileio/broker-core/cmd/neard/lockboxclient"
)

var (
	log = logging.Logger("updater")
)

// UpdateDelegate describes an object that can be called back with a state update.
type UpdateDelegate interface {
	HandleStateUpdate(*lockboxclient.State)
}

// Updater updates the lock box state to the delegate.
type Updater struct {
	config  Config
	mainCtx context.Context
	cancel  context.CancelFunc
}

// Config holds the configuration for creating a new Updater.
type Config struct {
	Lbc             *lockboxclient.Client
	UpdateFrequency time.Duration
	RequestTimeout  time.Duration
	Delegate        UpdateDelegate
}

// NewUpdater creates a new Updater.
func NewUpdater(config Config) *Updater {
	ctx, cancel := context.WithCancel(context.Background())
	u := &Updater{
		config:  config,
		mainCtx: ctx,
		cancel:  cancel,
	}
	go u.run()
	return u
}

// Close closes the update, canceling all current work.
func (u *Updater) Close() error {
	u.cancel()
	return nil
}

func (u *Updater) run() {
	for {
		select {
		case <-time.After(u.config.UpdateFrequency):
			ctx, cancel := context.WithTimeout(u.mainCtx, u.config.RequestTimeout)
			// ToDo: Detect fatal vs recoverable error, backoff maybe.
			state, err := u.config.Lbc.GetState(ctx)
			if err != nil {
				log.Errorf("getting state: %v", err)
				cancel()
				continue
			}
			u.config.Delegate.HandleStateUpdate(state)
			cancel()
		case <-u.mainCtx.Done():
			return
		}
	}
}
