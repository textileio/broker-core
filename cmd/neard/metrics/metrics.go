package metrics

import (
	"context"
	"time"

	"github.com/filecoin-project/go-state-types/big"
	"github.com/textileio/broker-core/cmd/neard/contractclient"
	logging "github.com/textileio/go-log/v2"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
)

var (
	log = logging.Logger("neard/metrics")
)

const prefix = "neard"

var meter = metric.Must(global.Meter(prefix))

// Metrics creates metrics about the NEAR smart contract and API node.
type Metrics struct {
	cc *contractclient.Client
}

// New creates a new Metrics.
func New(cc *contractclient.Client) *Metrics {
	m := &Metrics{cc: cc}
	m.initMetrics()
	return m
}

func (m *Metrics) initMetrics() {
	var (
		providerCount     metric.Int64ValueObserver
		depositCount      metric.Int64ValueObserver
		depositSum        metric.Int64SumObserver
		accountBal        metric.Int64ValueObserver
		lockedAccountBal  metric.Int64ValueObserver
		latestBlocktime   metric.Int64ValueObserver
		latestBlockHeight metric.Int64ValueObserver
	)
	batchObs := meter.NewBatchObserver(func(ctx context.Context, result metric.BatchObserverResult) {
		var obs []metric.Observation

		// Contract state metrics.
		state, err := m.cc.GetState(ctx)
		if err != nil {
			log.Errorf("getting contract state: %v", err)
		} else {
			// Calc sum of deposits.
			sumDeposits := big.Zero().Int
			for _, deposit := range state.DepositMap {
				sumDeposits.Add(sumDeposits, deposit.Deposit.Amount)
			}
			obs = append(
				obs,
				providerCount.Observation(int64(len(state.BrokerMap))),
				depositCount.Observation(int64(len(state.DepositMap))),
				depositSum.Observation(sumDeposits.Int64()),
			)
		}

		// Account info metrics.
		acc, err := m.cc.NearClient.Account("filecoin-bridge.testnet").State(ctx)
		if err != nil {
			log.Errorf("getting account info: %v", err)
		} else {
			// Parse account balance.
			bal, ok := (&big.Int{}).SetString(acc.Amount, 10)
			if !ok {
				log.Errorf("unable to parse account balance: %s", acc.Amount)
			} else {
				obs = append(obs, accountBal.Observation(bal.Int64()))
			}

			// Parse account locked balance.
			lockedBal, ok := (&big.Int{}).SetString(acc.Locked, 10)
			if !ok {
				log.Errorf("unable to parse locked account balance: %s", acc.Locked)
			} else {
				obs = append(obs, lockedAccountBal.Observation(lockedBal.Int64()))
			}
		}

		// Node status metrics.
		nodeStatus, err := m.cc.NearClient.NodeStatus(ctx)
		if err != nil {
			log.Errorf("getting node status: %v", err)
		} else {
			// Parse latest block time
			latestBlockTime, err := time.Parse(time.RFC3339Nano, nodeStatus.SyncInfo.LatestBlockTime)
			if err != nil {
				log.Errorf("parsing latest block time: %v", err)
			} else {
				obs = append(obs, latestBlocktime.Observation(latestBlockTime.Unix()))
			}

			obs = append(obs, latestBlockHeight.Observation(int64(nodeStatus.SyncInfo.LatestBlockHeight)))
		}

		result.Observe(nil, obs...)
	})
	providerCount = batchObs.NewInt64ValueObserver(prefix + ".provider_count")
	depositCount = batchObs.NewInt64ValueObserver(prefix + ".deposit_count")
	depositSum = batchObs.NewInt64SumObserver(prefix + "deposits_sum")
	accountBal = batchObs.NewInt64ValueObserver(prefix + "account_bal")
	lockedAccountBal = batchObs.NewInt64ValueObserver(prefix + "locked_account_bal")
	latestBlocktime = batchObs.NewInt64ValueObserver(prefix + "latest_block_time")
	latestBlockHeight = batchObs.NewInt64ValueObserver(prefix + "latest_block_height")
}
