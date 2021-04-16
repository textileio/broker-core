package statecache

import (
	"sync"

	"github.com/textileio/broker-core/cmd/neard/lockboxclient"
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
			LockedFunds: make(map[string]lockboxclient.LockInfo),
		},
	}, nil
}

// HandleStateUpdate receives a new state.
func (sc *StateCache) HandleStateUpdate(state *lockboxclient.State) {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	sc.state = *state
}

// GetState returns the current state.
func (sc *StateCache) GetState() lockboxclient.State {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	return sc.state
}
