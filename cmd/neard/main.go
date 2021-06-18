package main

import (
	"context"
	"encoding/json"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/broker-core/cmd/common"
	"github.com/textileio/broker-core/cmd/neard/contractclient"
	"github.com/textileio/broker-core/cmd/neard/metrics"
	"github.com/textileio/broker-core/cmd/neard/nearclient"
	"github.com/textileio/broker-core/cmd/neard/nearclient/keys"
	"github.com/textileio/broker-core/cmd/neard/nearclient/types"
	"github.com/textileio/broker-core/cmd/neard/service"
	"github.com/textileio/broker-core/cmd/neard/statecache"
	"github.com/textileio/broker-core/cmd/neard/updater"
	logging "github.com/textileio/go-log/v2"
)

var (
	daemonName = "neard"
	log        = logging.Logger(daemonName)
	v          = viper.New()
)

var flags = []common.Flag{
	{Name: "rpc-addr", DefValue: "", Description: "gRPC listen address"},
	{Name: "endpoint-url", DefValue: "https://rpc.testnet.near.org", Description: "The NEAR enpoint URL to use"},
	{Name: "endpoint-timeout", DefValue: time.Second * 5, Description: "Timeout for initial connection to endpoint-url"},
	{
		Name:        "contract-account",
		DefValue:    "filecoin-bridge.testnet",
		Description: "The NEAR account id of the governance contract",
	},
	{
		Name:        "client-account",
		DefValue:    "filecoin-bridge.testnet",
		Description: "The NEAR account id of the user of this client",
	},
	{Name: "client-private-key", DefValue: "", Description: "The NEAR private key string of the client account"},
	{Name: "update-frequency", DefValue: time.Millisecond * 500, Description: "How often to query the contract state"},
	{Name: "request-timeout", DefValue: time.Minute, Description: "Timeout to use when calling endpoint-url API calls"},
	{Name: "metrics-addr", DefValue: ":9090", Description: "Prometheus listen address"},
	{Name: "log-debug", DefValue: false, Description: "Enable debug level logging"},
	{Name: "log-json", DefValue: false, Description: "Enable structured logging"},
}

func init() {
	common.ConfigureCLI(v, "NEAR", flags, rootCmd.Flags())
}

var rootCmd = &cobra.Command{
	Use:   daemonName,
	Short: "neard is provides an api to the near blockchain",
	Long:  `neard is provides an api to the near blockchain`,
	PersistentPreRun: func(c *cobra.Command, args []string) {
		common.ExpandEnvVars(v, v.AllSettings())
		err := common.ConfigureLogging(v, nil)
		common.CheckErrf("setting log levels: %v", err)
	},
	Run: func(c *cobra.Command, args []string) {
		settings, err := marshalConfig(v)
		common.CheckErr(err)
		log.Infof("loaded config: %s", string(settings))

		listenAddr := v.GetString("rpc-addr")
		metricsAddr := v.GetString("metrics-addr")
		endpointURL := v.GetString("endpoint-url")
		endpointTimeout := v.GetDuration("endpoint-timeout")
		contractAccountID := v.GetString("contract-account")
		clientAccountID := v.GetString("client-account")
		clientPrivateKey := v.GetString("client-private-key")
		updateFrequency := v.GetDuration("update-frequency")
		requestTimeout := v.GetDuration("request-timeout")

		err = common.SetupInstrumentation(metricsAddr)
		common.CheckErrf("booting instrumentation: %v", err)

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

		cc, err := contractclient.NewClient(nc, contractAccountID, clientAccountID)
		common.CheckErr(err)

		sc, err := statecache.NewStateCache()
		common.CheckErr(err)

		u := updater.NewUpdater(updater.Config{
			Contract:        cc,
			UpdateFrequency: updateFrequency,
			RequestTimeout:  requestTimeout,
			Delegate:        sc,
		})

		metrics.New(cc)

		log.Info("Starting service...")
		listener, err := net.Listen("tcp", listenAddr)
		common.CheckErr(err)

		service, err := service.NewService(listener, sc, cc)
		common.CheckErr(err)

		common.HandleInterrupt(func() {
			// common.CheckErr(u.Close())
			rpcClient.Close()
			log.Info("Gracefully stopping... (press Ctrl+C again to force)")
			common.CheckErr(service.Close())
			common.CheckErr(listener.Close())
			common.CheckErr(u.Close())
			log.Info("Closed.")
		})

	},
}

func main() {
	common.CheckErr(rootCmd.Execute())
}

func marshalConfig(v *viper.Viper) ([]byte, error) {
	all := v.AllSettings()
	if all["client-private-key"].(string) != "" {
		all["client-private-key"] = "***"
	}
	return json.MarshalIndent(all, "", "  ")
}
