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
	"github.com/textileio/broker-core/cmd/neard/nearclient/keys"
	"github.com/textileio/broker-core/cmd/neard/nearclient/types"
	"github.com/textileio/broker-core/cmd/neard/service"
	"github.com/textileio/broker-core/cmd/neard/statecache"
)

var (
	daemonName = "neard"
	log        = logging.Logger(daemonName)
	v          = viper.New()
)

var flags = []common.Flag{
	{Name: "rpc-addr", DefValue: "", Description: "The host and port the gRPC service should listen on."},
	{Name: "metrics-addr", DefValue: ":9090", Description: "Prometheus endpoint"},
	{Name: "endpoint-url", DefValue: "https://rpc.testnet.near.org", Description: "The NEAR enpoint URL to use."},
	{Name: "endpoint-timeout", DefValue: time.Second * 5, Description: "Timeout for initial connection to endpoint-url."},
	{Name: "lockbox-account", DefValue: "lock-box.testnet", Description: "The NEAR account id of the Lock Box contract."},
	{Name: "client-account", DefValue: "lock-box.testnet", Description: "The NEAR account id of the user of this client."},
	{Name: "client-private-key", DefValue: "", Description: "The NEAR private key string of the client account."},
	{Name: "update-frequency", DefValue: time.Millisecond * 500, Description: "How often to query the contract state."},
	{Name: "request-timeout", DefValue: time.Minute, Description: "Timeout to use when calling endpoint-url API calls."},
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

		listenAddr := v.GetString("rpc-addr")
		metricsAddr := v.GetString("metrics-addr")
		endpointURL := v.GetString("endpoint-url")
		endpointTimeout := v.GetDuration("endpoint-timeout")
		lockboxAccountID := v.GetString("lockbox-account")
		clientAccountID := v.GetString("client-account")
		clientPrivateKey := v.GetString("client-private-key")
		// updateFrequency := v.GetDuration("update-frequency")
		// requestTimeout := v.GetDuration("request-timeout")

		if err := common.SetupInstrumentation(metricsAddr); err != nil {
			log.Fatalf("booting instrumentation: %s", err)
		}

		ctx, cancel := context.WithTimeout(context.Background(), endpointTimeout)
		defer cancel()

		rpcClient, err := rpc.DialContext(ctx, endpointURL)
		common.CheckErr(err)

		var signer keys.KeyPair
		if clientPrivateKey != "" {
			var err error
			signer, err = keys.NewKeyPairFromString(clientPrivateKey)
			common.CheckErr(err)
		}

		nc, err := nearclient.NewClient(&types.Config{
			RPCClient: rpcClient,
			Signer:    signer,
		})
		common.CheckErr(err)

		lc, err := lockboxclient.NewClient(nc, lockboxAccountID, clientAccountID)
		common.CheckErr(err)

		sc, err := statecache.NewStateCache()
		common.CheckErr(err)

		// u := updater.NewUpdater(updater.Config{
		// 	Lbc:             lc,
		// 	UpdateFrequency: updateFrequency,
		// 	RequestTimeout:  requestTimeout,
		// 	Delegate:        sc,
		// })

		log.Info("Starting service...")
		listener, err := net.Listen("tcp", listenAddr)
		common.CheckErr(err)

		service, err := service.NewService(listener, sc, lc)
		common.CheckErr(err)

		common.HandleInterrupt(func() {
			// common.CheckErr(u.Close())
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
