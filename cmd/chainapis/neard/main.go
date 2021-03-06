package main

import (
	"context"
	"errors"
	"fmt"
	"net"

	"github.com/ethereum/go-ethereum/rpc"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/broker-core/cmd/chainapis/neard/metrics"
	"github.com/textileio/broker-core/cmd/chainapis/neard/providerclient"
	"github.com/textileio/broker-core/cmd/chainapis/neard/registryclient"
	"github.com/textileio/broker-core/cmd/chainapis/neard/releaser"
	"github.com/textileio/broker-core/cmd/chainapis/neard/service"
	"github.com/textileio/broker-core/common"
	"github.com/textileio/cli"
	logging "github.com/textileio/go-log/v2"
	api "github.com/textileio/near-api-go"
	"github.com/textileio/near-api-go/keys"
	"github.com/textileio/near-api-go/types"
)

var (
	daemonName = "neard"
	log        = logging.Logger(daemonName)
	v          = viper.New()
	cv         = viper.New()
)

var flags = []cli.Flag{
	{Name: "config-path", DefValue: "./neard.yaml", Description: "Path to the config file"},
	{Name: "listen-addr", DefValue: ":5000", Description: "gRPC listen address"},
	{Name: "log-debug", DefValue: false, Description: "Enable debug level logging"},
	{Name: "log-json", DefValue: false, Description: "Enable structured logging"},
}

func init() {
	cli.ConfigureCLI(v, "NEAR", flags, rootCmd.Flags())

	_ = cv.BindPFlag("listen-addr", rootCmd.Flags().Lookup("listen-addr"))
	_ = cv.BindPFlag("log-debug", rootCmd.Flags().Lookup("log-debug"))
	_ = cv.BindPFlag("log-json", rootCmd.Flags().Lookup("log-json"))
}

var rootCmd = &cobra.Command{
	Use:   daemonName,
	Short: "neard provides an api to the near blockchain",
	Long:  `neard provides an api to the near blockchain`,
	PersistentPreRun: func(c *cobra.Command, args []string) {
		cli.ExpandEnvVars(v, v.AllSettings())
		err := cli.ConfigureLogging(v, nil)
		cli.CheckErrf("setting log levels: %v", err)
	},
	Run: func(c *cobra.Command, args []string) {
		cv.SetConfigFile(v.GetString("config-path"))

		err := cv.ReadInConfig()
		cli.CheckErrf("reading config: %v", err)

		// Obfuscating chain-apis since the nested map values contain private keys
		settings, err := cli.MarshalConfig(cv, !cv.GetBool("log-json"), "chain-apis")
		cli.CheckErrf("marshaling config: %v", err)
		log.Infof("loaded config: %s", string(settings))

		listenAddr := cv.GetString("listen-addr")
		metricsAddr := v.GetString("metrics-addr")
		chainApisMap := cv.GetStringMap("chain-apis")

		err = common.SetupInstrumentation(metricsAddr)
		cli.CheckErrf("booting instrumentation: %v", err)

		rpcClients := []*rpc.Client{}
		registryClients := make(map[string]*registryclient.Client)
		providerClients := make(map[string]*providerclient.Client)
		releasers := []*releaser.Releaser{}
		ms := []*metrics.Metrics{}
		for chainID := range chainApisMap {
			sub := cv.Sub(fmt.Sprintf("chain-apis.%s", chainID))
			endpoint := sub.GetString("endpoint")
			timeout := sub.GetDuration("timeout")
			registryContractAddress := sub.GetString("registry-contract-addr")
			providerContractAddress := sub.GetString("provider-contract-addr")
			clientAddress := sub.GetString("client-addr")
			clientPrivateKey := sub.GetString("client-private-key")
			releaseDepositsFreq := sub.GetDuration("release-deposits-freq")

			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			rpcClient, err := rpc.DialContext(ctx, endpoint)
			cancel()
			cli.CheckErrf("dialing endpoint: %v", err)

			var signer keys.KeyPair
			if clientPrivateKey != "" {
				var err error
				signer, err = keys.NewKeyPairFromString(clientPrivateKey)
				cli.CheckErrf("parsing private key: %v", err)
			}

			nearClient, err := api.NewClient(&types.Config{
				NetworkID: chainID,
				RPCClient: rpcClient,
				Signer:    signer,
			})
			cli.CheckErrf("creating near client: %v", err)

			registryClient, err := registryclient.NewClient(nearClient, registryContractAddress, clientAddress)
			cli.CheckErrf("creating registry contract client: %v", err)

			providerClient, err := providerclient.NewClient(nearClient, providerContractAddress, clientAddress)
			cli.CheckErrf("creating provider contract client: %v", err)

			releaser, err := releaser.New(providerClient, chainID, releaseDepositsFreq, timeout)
			cli.CheckErrf("creating releaser: %v", err)

			m := metrics.New(registryClient, providerClient, chainID)

			rpcClients = append(rpcClients, rpcClient)
			registryClients[chainID] = registryClient
			providerClients[chainID] = providerClient
			releasers = append(releasers, releaser)
			ms = append(ms, m)
		}

		if len(registryClients) == 0 {
			cli.CheckErr(errors.New("no registry contract clients resolved"))
		}
		if len(providerClients) == 0 {
			cli.CheckErr(errors.New("no provider contract clients resolved"))
		}

		// Keep the linter happy
		_ = ms
		_ = releasers

		log.Info("Starting service...")
		listener, err := net.Listen("tcp", listenAddr)
		cli.CheckErrf("creating listener: %v", err)

		service, err := service.NewService(listener, providerClients)
		cli.CheckErrf("creating service: %v", err)

		cli.HandleInterrupt(func() {
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
	cli.CheckErrf("executing root cmd: %v", rootCmd.Execute())
}
