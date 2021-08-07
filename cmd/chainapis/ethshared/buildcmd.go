package ethshared

import (
	"context"
	"fmt"
	"net"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/bidbot/lib/common"
	"github.com/textileio/broker-core/cmd/chainapis/ethshared/contractclient"
	"github.com/textileio/broker-core/cmd/chainapis/ethshared/releaser"
	"github.com/textileio/broker-core/cmd/chainapis/ethshared/service"
	logging "github.com/textileio/go-log/v2"
)

var (
	log *logging.ZapEventLogger
	v   = viper.New()
	cv  = viper.New()
)

var flags = []common.Flag{
	{Name: "config-path", DefValue: "", Description: "Path to the config file"},
	{Name: "listen-addr", DefValue: ":5000", Description: "gRPC listen address"},
	{Name: "log-debug", DefValue: false, Description: "Enable debug level logging"},
	{Name: "log-json", DefValue: false, Description: "Enable structured logging"},
}

// BuildRootCmd builds the root command for the provided parameters.
func BuildRootCmd(daemonName, envPrefix, chainName string) *cobra.Command {
	log = logging.Logger(daemonName)

	rootCmd := buildRootCommand(daemonName, chainName)

	common.ConfigureCLI(v, envPrefix, flags, rootCmd.Flags())

	_ = cv.BindPFlag("listen-addr", rootCmd.Flags().Lookup("listen-addr"))
	_ = cv.BindPFlag("log-debug", rootCmd.Flags().Lookup("log-debug"))
	_ = cv.BindPFlag("log-json", rootCmd.Flags().Lookup("log-json"))

	return rootCmd
}

func buildRootCommand(daemonName, chainName string) *cobra.Command {
	return &cobra.Command{
		Use:   daemonName,
		Short: fmt.Sprintf("%s provides an api to the %s blockchain", daemonName, chainName),
		Long:  fmt.Sprintf("%s provides an api to the %s blockchain", daemonName, chainName),
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
			chainApisMap := cv.GetStringMap("chain-apis")

			ethClients := []*ethclient.Client{}
			contractClients := make(map[string]*contractclient.BridgeProvider)
			releasers := []*releaser.Releaser{}
			for key := range chainApisMap {
				sub := cv.Sub(fmt.Sprintf("chain-apis.%s", key))
				endpoint := sub.GetString("endpoint")
				timeout := sub.GetDuration("timeout")
				contractAddress := sub.GetString("contract-addr")
				clientAddress := sub.GetString("client-addr")
				clientPrivateKey := sub.GetString("client-private-key")
				releaseDepositsFreq := sub.GetDuration("release-deposits-freq")

				ctx, cancel := context.WithTimeout(context.Background(), timeout)
				ethClient, err := ethclient.DialContext(ctx, endpoint)
				cancel()
				common.CheckErrf("dialing endpoint: %v", err)

				contractAddr := ethcommon.HexToAddress(contractAddress)
				clientAddr := ethcommon.HexToAddress(clientAddress)

				contractClient, err := contractclient.NewBridgeProvider(contractAddr, ethClient)
				common.CheckErrf("creating contract client: %v", err)

				privateKey, err := crypto.HexToECDSA(clientPrivateKey)
				common.CheckErrf("parsing private key: %v", err)

				signer := func(a ethcommon.Address, t *types.Transaction) (*types.Transaction, error) {
					signer := types.HomesteadSigner{}
					return types.SignTx(t, signer, privateKey)
				}

				ethClients = append(ethClients, ethClient)
				contractClients[key] = contractClient
				releasers = append(releasers, releaser.New(contractClient, clientAddr, signer, releaseDepositsFreq, timeout))
			}

			log.Info("Starting service...")
			listener, err := net.Listen("tcp", listenAddr)
			common.CheckErrf("creating listener: %v", err)

			service, err := service.NewService(listener, daemonName, contractClients)
			common.CheckErrf("creating service: %v", err)

			common.HandleInterrupt(func() {
				for _, c := range ethClients {
					c.Close()
				}
				for _, r := range releasers {
					if err := r.Close(); err != nil {
						log.Errorf("closing releaser: %v", err)
					}
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
}