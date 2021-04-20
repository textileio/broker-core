package statecache

import (
	"sync"

	logging "github.com/ipfs/go-log/v2"
	"github.com/textileio/broker-core/cmd/neard/lockboxclient"
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
			LockedFunds: make(map[string]lockboxclient.LockInfo),
		},
	}, nil
}

// HandleIntialStateUpdate receives a new state.
func (sc *StateCache) HandleIntialStateUpdate(state *lockboxclient.State) {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	sc.state = *state
}

func (sc *StateCache) HandleStateChanges(changes []lockboxclient.Change, blockHash string, blockHeight int) {
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

// GetState returns the current state.
func (sc *StateCache) GetState() lockboxclient.State {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	return sc.state
}
