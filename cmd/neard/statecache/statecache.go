package statecache

import (
	"sync"

	"github.com/textileio/broker-core/cmd/neard/lockboxclient"
)

type StateCache struct {
	state lockboxclient.State
	lock  sync.Mutex
}

func NewStateCache() (*StateCache, error) {
	return &StateCache{
		state: lockboxclient.State{
			LockedFunds: make(map[string]lockboxclient.LockInfo),
		},
	}, nil
}

func (sc *StateCache) HandleStateUpdate(state *lockboxclient.State) {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	sc.state = *state
}

func (sc *StateCache) GetState() lockboxclient.State {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	return sc.state
}
