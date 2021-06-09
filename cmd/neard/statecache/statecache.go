package statecache

import (
	"sync"

	"github.com/textileio/broker-core/cmd/neard/lockboxclient"
	logging "github.com/textileio/go-log/v2"
)

var (
	log = logging.Logger("neard/statecache")
)

// StateCache holds and controls access to the lock box state.
type StateCache struct {
	state lockboxclient.State
	lock  sync.Mutex
}

// NewStateCache creates a new StateCache.
func NewStateCache() (*StateCache, error) {
	return &StateCache{
		state: lockboxclient.State{
			LockedFunds: make(map[string]lockboxclient.DepositInfo),
		},
	}, nil
}

// HandleIntialStateUpdate receives a new state.
func (sc *StateCache) HandleIntialStateUpdate(state *lockboxclient.State) {
	log.Info("handling initial state update")
	sc.lock.Lock()
	defer sc.lock.Unlock()
	sc.state = *state
}

// HandleStateChanges handles state changes.
func (sc *StateCache) HandleStateChanges(changes []lockboxclient.Change, blockHash string, blockHeight int) {
	log.Info("handling state changes")
	sc.lock.Lock()
	defer sc.lock.Unlock()
	sc.state.BlockHash = blockHash
	sc.state.BlockHeight = blockHeight
	for _, change := range changes {
		switch change.Type {
		case lockboxclient.Update:
			sc.state.LockedFunds[change.Key] = *change.LockInfo
		case lockboxclient.Delete:
			delete(sc.state.LockedFunds, change.Key)
		}
	}
}

// HandleError handles errors.
func (sc *StateCache) HandleError(err error) {
	log.Infof("handling error: %v", err)
}

// GetState returns the current state.
func (sc *StateCache) GetState() lockboxclient.State {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	return sc.state
}
