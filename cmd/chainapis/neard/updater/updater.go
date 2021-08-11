package updater

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/textileio/broker-core/cmd/chainapis/neard/contractclient"
	logging "github.com/textileio/go-log/v2"
)

var (
	log = logging.Logger("neard/updater")
)

// UpdateDelegate describes an object that can be called back with a state update.
type UpdateDelegate interface {
	HandleIntialStateUpdate(*contractclient.State)
	HandleStateChanges([]contractclient.Change, string, int)
	HandleError(error)
}

// Updater updates the state to the delegate.
type Updater struct {
	config      Config
	mainCtx     context.Context
	cancel      context.CancelFunc
	blockHeight int
}

// Config holds the configuration for creating a new Updater.
type Config struct {
	Contract        *contractclient.Client
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
				initialState, err := u.config.Contract.GetState(ctx)
				cancel()
				if err != nil {
					u.config.Delegate.HandleError(fmt.Errorf("getting initial state: %v", err))
					updateFrequency *= 2
					continue
				}
				log.Errorf("initial state height: %v", initialState.BlockHeight)
				u.blockHeight = initialState.BlockHeight
				u.config.Delegate.HandleIntialStateUpdate(initialState)
				continue
			}
			ctx, cancel := context.WithTimeout(u.mainCtx, u.config.RequestTimeout)
			account, err := u.config.Contract.GetAccount(ctx)
			cancel()
			if err != nil {
				u.config.Delegate.HandleError(fmt.Errorf("getting node status: %v", err))
				updateFrequency *= 2
				continue
			}
			log.Errorf("account block height: %v", account.BlockHeight)
			if u.blockHeight >= account.BlockHeight {
				continue
			}
			time.Sleep(time.Millisecond * 2500)
			for i := u.blockHeight + 1; i <= account.BlockHeight; i++ {
				log.Errorf("getting changes for height: %v", i)
				ctx, cancel := context.WithTimeout(u.mainCtx, u.config.RequestTimeout)
				changes, blockHash, err := u.config.Contract.GetChanges(ctx, i)
				cancel()
				if err != nil {
					if strings.Contains(err.Error(), "Block not found") {
						// u.blockHeight = 0
						log.Warnf("%v -- resetting state", err)
						log.Errorf("RESETTING STATE! %v", err)
					} else {
						u.config.Delegate.HandleError(fmt.Errorf("getting changes: %v", err))
						updateFrequency *= 2
					}
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
