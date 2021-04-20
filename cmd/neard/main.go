package main

import (
	"context"
	"encoding/json"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
	logging "github.com/ipfs/go-log/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/broker-core/cmd/common"
	"github.com/textileio/broker-core/cmd/neard/lockboxclient"
	"github.com/textileio/broker-core/cmd/neard/nearclient"
	"github.com/textileio/broker-core/cmd/neard/service"
	"github.com/textileio/broker-core/cmd/neard/statecache"
	"github.com/textileio/broker-core/cmd/neard/updater"
)

var (
	daemonName = "neard"
	log        = logging.Logger(daemonName)
	v          = viper.New()
)

var flags = []common.Flag{
	{Name: "listen.addr", DefValue: ":5000", Description: "The host and port the gRPC service should listen on."},
	{Name: "metrics.addr", DefValue: ":9090", Description: "Prometheus endpoint"},
	{Name: "endpoint.url", DefValue: "https://rpc.testnet.near.org", Description: "The NEAR enpoint URL to use."},
	{Name: "endpoint.timeout", DefValue: time.Second * 5, Description: "Timeout for initial connection to endpoint-url."},
	{Name: "account.id", DefValue: "lock-box.testnet", Description: "The NEAR account id of the Lock Box smart contract."},
	{Name: "update.frequency", DefValue: time.Millisecond * 500, Description: "How often to query the smart contract state."},
	{Name: "request.timeout", DefValue: time.Minute, Description: "Timeout to use when calling endpoint-url API calls."},
	{Name: "debug", DefValue: false, Description: "Enable debug level logs."},
}

func init() {
	common.ConfigureCLI(v, "NEAR", flags, rootCmd)
}

var rootCmd = &cobra.Command{
	Use:   daemonName,
	Short: "neard is provides an api to the near blockchain",
	Long:  `neard is provides an api to the near blockchain`,
	PersistentPreRun: func(c *cobra.Command, args []string) {
		logging.SetAllLoggers(logging.LevelInfo)
		if v.GetBool("debug") {
			logging.SetAllLoggers(logging.LevelDebug)
		}
	},
	Run: func(c *cobra.Command, args []string) {
		settings, err := json.MarshalIndent(v.AllSettings(), "", "  ")
		common.CheckErr(err)
		log.Infof("loaded config: %s", string(settings))

		listenAddr := v.GetString("listen.addr")
		metricsAddr := v.GetString("metrics.addr")
		endpointURL := v.GetString("endpoint.url")
		endpointTimeout := v.GetDuration("endpoint.timeout")
		accountID := v.GetString("account.id")
		updateFrequency := v.GetDuration("update.frequency")
		requestTimeout := v.GetDuration("request.timeout")

		if err := common.SetupInstrumentation(metricsAddr); err != nil {
			log.Fatalf("booting instrumentation: %s", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), endpointTimeout)
		defer cancel()

		rpcClient, err := rpc.DialContext(ctx, endpointURL)
		common.CheckErr(err)

		nc, err := nearclient.NewClient(rpcClient)
		common.CheckErr(err)

		lc, err := lockboxclient.NewClient(nc, accountID)
		common.CheckErr(err)

		sc, err := statecache.NewStateCache()
		common.CheckErr(err)

		u := updater.NewUpdater(updater.Config{
			Lbc:             lc,
			UpdateFrequency: updateFrequency,
			RequestTimeout:  requestTimeout,
			Delegate:        sc,
		})

		log.Info("Starting service...")
		listener, err := net.Listen("tcp", listenAddr)
		common.CheckErr(err)

		service, err := service.NewService(listener, sc)
		common.CheckErr(err)

		common.HandleInterrupt(func() {
			common.CheckErr(u.Close())
			rpcClient.Close()
			log.Info("Gracefully stopping... (press Ctrl+C again to force)")
			common.CheckErr(service.Close())
			common.CheckErr(listener.Close())
			log.Info("Closed.")
		})

	},
}

func main() {
	common.CheckErr(rootCmd.Execute())
}
