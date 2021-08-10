package main

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/bidbot/lib/common"
	"github.com/textileio/broker-core/cmd/chainapis/neard/contractclient"
	"github.com/textileio/broker-core/cmd/chainapis/neard/metrics"
	"github.com/textileio/broker-core/cmd/chainapis/neard/nearclient"
	"github.com/textileio/broker-core/cmd/chainapis/neard/nearclient/keys"
	"github.com/textileio/broker-core/cmd/chainapis/neard/nearclient/types"
	"github.com/textileio/broker-core/cmd/chainapis/neard/releaser"
	"github.com/textileio/broker-core/cmd/chainapis/neard/service"
	logging "github.com/textileio/go-log/v2"
)

var (
	daemonName = "neard"
	log        = logging.Logger(daemonName)
	v          = viper.New()
	cv         = viper.New()
)

var flags = []common.Flag{
	{Name: "config-path", DefValue: "./neard.yaml", Description: "Path to the config file"},
	{Name: "listen-addr", DefValue: ":5000", Description: "gRPC listen address"},
	{Name: "log-debug", DefValue: false, Description: "Enable debug level logging"},
	{Name: "log-json", DefValue: false, Description: "Enable structured logging"},
}

func init() {
	common.ConfigureCLI(v, "NEAR", flags, rootCmd.Flags())

	_ = cv.BindPFlag("listen-addr", rootCmd.Flags().Lookup("listen-addr"))
	_ = cv.BindPFlag("log-debug", rootCmd.Flags().Lookup("log-debug"))
	_ = cv.BindPFlag("log-json", rootCmd.Flags().Lookup("log-json"))
}

var rootCmd = &cobra.Command{
	Use:   daemonName,
	Short: "neard provides an api to the near blockchain",
	Long:  `neard provides an api to the near blockchain`,
	PersistentPreRun: func(c *cobra.Command, args []string) {
		common.ExpandEnvVars(v, v.AllSettings())
		err := common.ConfigureLogging(v, nil)
		common.CheckErrf("setting log levels: %v", err)
	},
	Run: func(c *cobra.Command, args []string) {
		cv.SetConfigFile(v.GetString("config-path"))

		err := cv.ReadInConfig()
		common.CheckErrf("reading config: %v", err)

		// Obfuscating chain-apis since the nested map values contain private keys
		settings, err := common.MarshalConfig(cv, !cv.GetBool("log-json"), "chain-apis")
		common.CheckErrf("marshaling config: %v", err)
		log.Infof("loaded config: %s", string(settings))

		listenAddr := cv.GetString("listen-addr")
		metricsAddr := v.GetString("metrics-addr")
		chainApisMap := cv.GetStringMap("chain-apis")

		err = common.SetupInstrumentation(metricsAddr)
		common.CheckErrf("booting instrumentation: %v", err)

		rpcClients := []*rpc.Client{}
		contractClients := make(map[string]*contractclient.Client)
		releasers := []*releaser.Releaser{}
		ms := []*metrics.Metrics{}
		for chainID := range chainApisMap {
			sub := cv.Sub(fmt.Sprintf("chain-apis.%s", chainID))
			endpoint := sub.GetString("endpoint")
			timeout := sub.GetDuration("timeout")
			contractAddress := sub.GetString("contract-addr")
			clientAddress := sub.GetString("client-addr")
			clientPrivateKey := sub.GetString("client-private-key")
			releaseDepositsFreq := sub.GetDuration("release-deposits-freq")

			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			rpcClient, err := rpc.DialContext(ctx, endpoint)
			cancel()
			common.CheckErrf("dialing endpoint: %v", err)

			var signer keys.KeyPair
			if clientPrivateKey != "" {
				var err error
				signer, err = keys.NewKeyPairFromString(clientPrivateKey)
				common.CheckErrf("parsing private key: %v", err)
			}

			nearClient, err := nearclient.NewClient(&types.Config{
				RPCClient: rpcClient,
				Signer:    signer,
			})
			common.CheckErrf("creating near client: %v", err)

			contractClient, err := contractclient.NewClient(nearClient, contractAddress, clientAddress)
			common.CheckErrf("creating contract client: %v", err)

			releaser, err := releaser.New(contractClient, chainID, releaseDepositsFreq, timeout)
			common.CheckErrf("creating releaser: %v", err)

			m := metrics.New(contractClient, chainID)

			rpcClients = append(rpcClients, rpcClient)
			contractClients[chainID] = contractClient
			releasers = append(releasers, releaser)
			ms = append(ms, m)
		}

		if len(contractClients) == 0 {
			common.CheckErr(errors.New("no contract clients resolved"))
		}

		// Keep the linter happy
		_ = ms
		_ = releasers

		log.Info("Starting service...")
		listener, err := net.Listen("tcp", listenAddr)
		common.CheckErrf("creating listener: %v", err)

		service, err := service.NewService(listener, contractClients)
		common.CheckErrf("creating service: %v", err)

		common.HandleInterrupt(func() {
			for _, rpcClient := range rpcClients {
				rpcClient.Close()
			}
			log.Info("Gracefully stopping... (press Ctrl+C again to force)")
			if err := service.Close(); err != nil {
				log.Errorf("closing service: %v", err)
			}
			if err := listener.Close(); err != nil {
				log.Errorf("closing listener: %v", err)
			}
			log.Info("Closed.")
		})
	},
}

func main() {
	common.CheckErrf("executing root cmd: %v", rootCmd.Execute())
}
