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
	HandleIntialStateUpdate(*lockboxclient.State)
	HandleStateChanges([]lockboxclient.Change, string, int)
	HandleError(error)
}

// Updater updates the lock box state to the delegate.
type Updater struct {
	config      Config
	mainCtx     context.Context
	cancel      context.CancelFunc
	blockHeight int
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
	updateFrequency := u.config.UpdateFrequency
	for {
	Loop:
		select {
		case <-time.After(updateFrequency):
			if u.blockHeight == 0 {
				ctx, cancel := context.WithTimeout(u.mainCtx, u.config.RequestTimeout)
				initialState, err := u.config.Lbc.GetState(ctx)
				cancel()
				if err != nil {
					u.config.Delegate.HandleError(err)
					updateFrequency *= 2
					continue
				}
				u.blockHeight = initialState.BlockHeight
				u.config.Delegate.HandleIntialStateUpdate(initialState)
				continue
			}
			ctx, cancel := context.WithTimeout(u.mainCtx, u.config.RequestTimeout)
			account, err := u.config.Lbc.GetAccount(ctx)
			cancel()
			if err != nil {
				u.config.Delegate.HandleError(err)
				updateFrequency *= 2
				continue
			}
			if u.blockHeight >= account.BlockHeight {
				continue
			}
			for i := u.blockHeight + 1; i <= account.BlockHeight; i++ {
				ctx, cancel := context.WithTimeout(u.mainCtx, u.config.RequestTimeout)
				changes, blockHash, err := u.config.Lbc.GetChanges(ctx, i)
				cancel()
				if err != nil {
					u.config.Delegate.HandleError(err)
					updateFrequency *= 2
					break Loop
				}
				u.blockHeight = i
				u.config.Delegate.HandleStateChanges(changes, blockHash, i)
				updateFrequency = u.config.UpdateFrequency
			}
		case <-u.mainCtx.Done():
			return
		}
	}
}
