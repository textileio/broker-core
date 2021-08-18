package metrics

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"time"

	"github.com/textileio/broker-core/cmd/chainapis/neard/providerclient"
	"github.com/textileio/broker-core/cmd/chainapis/neard/registryclient"
	logging "github.com/textileio/go-log/v2"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/metric/global"
)

const prefix = "neard"

var meter = metric.Must(global.Meter(prefix))

// Metrics creates metrics about the NEAR smart contract and API node.
type Metrics struct {
	rc      *registryclient.Client
	pc      *providerclient.Client
	chainID string

	log *logging.ZapEventLogger
}

// New creates a new Metrics.
func New(rc *registryclient.Client, pc *providerclient.Client, chainID string) *Metrics {
	m := &Metrics{
		rc:      rc,
		pc:      pc,
		chainID: chainID,
		log:     logging.Logger(fmt.Sprintf("neard-metrics-%s", chainID)),
	}

	_ = logging.SetLogLevel(fmt.Sprintf("neard-metrics-%s", chainID), "INFO")

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

		// Registry contract State metrics.
		registryState, err := m.rc.GetState(ctx)
		if err != nil {
			m.log.Errorf("getting registry contract state: %v", err)
		} else {
			obs = append(
				obs,
				providerCount.Observation(int64(len(registryState.Providers))),
			)
		}

		// Provider contract State metrics.
		providerState, err := m.pc.GetState(ctx)
		if err != nil {
			m.log.Errorf("getting provider contract state: %v", err)
		} else {
			// Calc sum of deposits.
			sumDeposits := big.NewFloat(0)
			for _, deposit := range providerState.DepositMap {
				sumDeposits.Add(sumDeposits, (&big.Float{}).SetInt(deposit.Value))
			}
			sumDeposits.Mul(sumDeposits, big.NewFloat(math.Pow(10, -24)))
			sumDepositsF, _ := sumDeposits.Float64()
			obs = append(
				obs,
				depositsCount.Observation(int64(len(providerState.DepositMap))),
				depositsSum.Observation(sumDepositsF),
			)
		}

		// Account info metrics.
		acc, err := m.pc.GetAccount(ctx)
		if err != nil {
			m.log.Errorf("getting account info: %v", err)
		} else {
			// Parse account balance.
			bal, ok := (&big.Float{}).SetString(acc.Amount)
			if !ok {
				m.log.Errorf("unable to parse account balance: %s", acc.Amount)
			} else {
				bal.Mul(bal, big.NewFloat(math.Pow(10, -24)))
				balF, _ := bal.Float64()
				obs = append(obs, accountBal.Observation(balF))
			}

			// Parse account locked balance.
			lockedBal, ok := (&big.Float{}).SetString(acc.Locked)
			if !ok {
				m.log.Errorf("unable to parse locked account balance: %s", acc.Locked)
			} else {
				lockedBal.Mul(lockedBal, big.NewFloat(math.Pow(10, -24)))
				lockedBalF, _ := lockedBal.Float64()
				obs = append(obs, lockedAccountBal.Observation(lockedBalF))
			}

			obs = append(obs, storageUsage.Observation(int64(acc.StorageUsage)))
		}

		// Node status metrics.
		nodeStatus, err := m.pc.NearClient.NodeStatus(ctx)
		if err != nil {
			m.log.Errorf("getting node status: %v", err)
		} else {
			// Parse latest block time
			latestBlockTime, err := time.Parse(time.RFC3339Nano, nodeStatus.SyncInfo.LatestBlockTime)
			if err != nil {
				m.log.Errorf("parsing latest block time: %v", err)
			} else {
				obs = append(obs, latestBlocktime.Observation(latestBlockTime.Unix()))
			}
			obs = append(obs, latestBlockHeight.Observation(int64(nodeStatus.SyncInfo.LatestBlockHeight)))
		}

		result.Observe([]attribute.KeyValue{{Key: "chainId", Value: attribute.StringValue(m.chainID)}}, obs...)
	})
	providerCount = batchObs.NewInt64ValueObserver(prefix + ".provider_count")
	depositsCount = batchObs.NewInt64ValueObserver(prefix + ".deposits_count")
	depositsSum = batchObs.NewFloat64ValueObserver(prefix + ".deposits_sum")
	accountBal = batchObs.NewFloat64ValueObserver(prefix + ".account_bal")
	lockedAccountBal = batchObs.NewFloat64ValueObserver(prefix + ".locked_account_bal")
	storageUsage = batchObs.NewInt64ValueObserver(prefix + ".storage_usage")
	latestBlocktime = batchObs.NewInt64ValueObserver(prefix + ".latest_block_time")
	latestBlockHeight = batchObs.NewInt64ValueObserver(prefix + ".latest_block_height")
}
