package metrics

import (
	"context"
	"math"
	"math/big"
	"time"

	"github.com/textileio/broker-core/cmd/neard/contractclient"
	"github.com/textileio/broker-core/cmd/neard/nearclient"
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
	nc *nearclient.Client
}

// New creates a new Metrics.
func New(cc *contractclient.Client, nc *nearclient.Client) *Metrics {
	m := &Metrics{cc: cc, nc: nc}
	m.initMetrics()
	return m
}

func (m *Metrics) initMetrics() {
	var (
		providerCount     metric.Int64ValueObserver
		depositsCount     metric.Int64ValueObserver
		depositsSum       metric.Float64ValueObserver
		accountBal        metric.Float64ValueObserver
		lockedAccountBal  metric.Float64ValueObserver
		storageUsage      metric.Int64ValueObserver
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
			sumDeposits := big.NewFloat(0)
			for _, deposit := range state.DepositMap {
				sumDeposits.Add(sumDeposits, (&big.Float{}).SetInt(deposit.Deposit.Amount))
			}
			sumDeposits.Mul(sumDeposits, big.NewFloat(math.Pow(10, -24)))
			sumDepositsF, _ := sumDeposits.Float64()
			obs = append(
				obs,
				providerCount.Observation(int64(len(state.BrokerMap))),
				depositsCount.Observation(int64(len(state.DepositMap))),
				depositsSum.Observation(sumDepositsF),
			)
		}

		// Account info metrics.
		acc, err := m.cc.GetAccount(ctx)
		if err != nil {
			log.Errorf("getting account info: %v", err)
		} else {
			// Parse account balance.
			bal, ok := (&big.Float{}).SetString(acc.Amount)
			if !ok {
				log.Errorf("unable to parse account balance: %s", acc.Amount)
			} else {
				bal.Mul(bal, big.NewFloat(math.Pow(10, -24)))
				balF, _ := bal.Float64()
				obs = append(obs, accountBal.Observation(balF))
			}

			// Parse account locked balance.
			lockedBal, ok := (&big.Float{}).SetString(acc.Locked)
			if !ok {
				log.Errorf("unable to parse locked account balance: %s", acc.Locked)
			} else {
				lockedBal.Mul(lockedBal, big.NewFloat(math.Pow(10, -24)))
				lockedBalF, _ := lockedBal.Float64()
				obs = append(obs, lockedAccountBal.Observation(lockedBalF))
			}

			obs = append(obs, storageUsage.Observation(int64(acc.StorageUsage)))
		}

		// Node status metrics.
		nodeStatus, err := m.nc.NodeStatus(ctx)
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
	depositsCount = batchObs.NewInt64ValueObserver(prefix + ".deposits_count")
	depositsSum = batchObs.NewFloat64ValueObserver(prefix + ".sum_deposits")
	accountBal = batchObs.NewFloat64ValueObserver(prefix + ".account_bal")
	lockedAccountBal = batchObs.NewFloat64ValueObserver(prefix + ".locked_account_bal")
	storageUsage = batchObs.NewInt64ValueObserver(prefix + ".storage_usage")
	latestBlocktime = batchObs.NewInt64ValueObserver(prefix + ".latest_block_time")
	latestBlockHeight = batchObs.NewInt64ValueObserver(prefix + ".latest_block_height")
}
