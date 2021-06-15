package statecache

import (
	"sync"

	"github.com/textileio/broker-core/cmd/neard/contractclient"
	logging "github.com/textileio/go-log/v2"
)

var (
	log = logging.Logger("neard/statecache")
)

// StateCache holds and controls access to the state.
type StateCache struct {
	state contractclient.State
	lock  sync.Mutex
}

// NewStateCache creates a new StateCache.
func NewStateCache() (*StateCache, error) {
	return &StateCache{
		state: contractclient.State{
			LockedFunds: make(map[string]contractclient.DepositInfo),
		},
	}, nil
}

// HandleIntialStateUpdate receives a new state.
func (sc *StateCache) HandleIntialStateUpdate(state *contractclient.State) {
	log.Info("handling initial state update")
	sc.lock.Lock()
	defer sc.lock.Unlock()
	sc.state = *state
}

// HandleStateChanges handles state changes.
func (sc *StateCache) HandleStateChanges(changes []contractclient.Change, blockHash string, blockHeight int) {
	log.Info("handling state changes")
	sc.lock.Lock()
	defer sc.lock.Unlock()
	sc.state.BlockHash = blockHash
	sc.state.BlockHeight = blockHeight
	for _, change := range changes {
		switch change.Type {
		case contractclient.Update:
			sc.state.LockedFunds[change.Key] = *change.LockInfo
		case contractclient.Delete:
			delete(sc.state.LockedFunds, change.Key)
		}
	}
}

// HandleError handles errors.
func (sc *StateCache) HandleError(err error) {
	log.Infof("handling error: %v", err)
}

// GetState returns the current state.
func (sc *StateCache) GetState() contractclient.State {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	return sc.state
}
