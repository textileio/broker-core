package statecache

import (
	"encoding/json"
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
			DepositMap: make(map[string]contractclient.DepositInfo),
			BrokerMap:  make(map[string]contractclient.BrokerInfo),
		},
	}, nil
}

// HandleIntialStateUpdate receives a new state.
func (sc *StateCache) HandleIntialStateUpdate(state *contractclient.State) {
	log.Info("handling initial state update")
	sc.lock.Lock()
	defer sc.lock.Unlock()
	sc.state = *state
	sc.logState()
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
			if change.BrokerValue != nil {
				sc.state.BrokerMap[change.BrokerValue.Key] = *change.BrokerValue.Value
			} else if change.DepositValue != nil {
				sc.state.DepositMap[change.DepositValue.Key] = *change.DepositValue.Value
			}
		case contractclient.Delete:
			if change.BrokerValue != nil {
				delete(sc.state.BrokerMap, change.BrokerValue.Key)
			} else if change.DepositValue != nil {
				delete(sc.state.DepositMap, change.DepositValue.Key)
			}
		}
	}
	if len(changes) > 0 {
		sc.logState()
	}
}

// HandleError handles errors.
func (sc *StateCache) HandleError(err error) {
	log.Errorf("handling error: %v", err)
}

// GetState returns the current state.
func (sc *StateCache) GetState() contractclient.State {
	sc.lock.Lock()
	defer sc.lock.Unlock()
	return sc.state
}

func (sc *StateCache) logState() {
	bytes, _ := json.MarshalIndent(sc.state, "", "  ")
	log.Errorf("\n%s", string(bytes))
}
