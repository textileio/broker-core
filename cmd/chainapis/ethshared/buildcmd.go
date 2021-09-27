package ethshared

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"net"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/broker-core/cmd/chainapis/ethshared/contractclient"
	"github.com/textileio/broker-core/cmd/chainapis/ethshared/releaser"
	"github.com/textileio/broker-core/cmd/chainapis/ethshared/service"
	"github.com/textileio/cli"
	logging "github.com/textileio/go-log/v2"
)

var (
	log *logging.ZapEventLogger
	v   = viper.New()
	cv  = viper.New()
)

// BuildRootCmd builds the root command for the provided parameters.
func BuildRootCmd(daemonName, envPrefix, blockchainName string) *cobra.Command {
	log = logging.Logger(daemonName)

	rootCmd := buildRootCommand(daemonName, blockchainName)

	flags := []cli.Flag{
		{Name: "config-path", DefValue: fmt.Sprintf("./%s.yaml", daemonName), Description: "Path to the config file"},
		{Name: "listen-addr", DefValue: ":5000", Description: "gRPC listen address"},
		{Name: "log-debug", DefValue: false, Description: "Enable debug level logging"},
		{Name: "log-json", DefValue: false, Description: "Enable structured logging"},
	}

	cli.ConfigureCLI(v, envPrefix, flags, rootCmd.Flags())

	_ = cv.BindPFlag("listen-addr", rootCmd.Flags().Lookup("listen-addr"))
	_ = cv.BindPFlag("log-debug", rootCmd.Flags().Lookup("log-debug"))
	_ = cv.BindPFlag("log-json", rootCmd.Flags().Lookup("log-json"))

	return rootCmd
}

func buildRootCommand(daemonName, blockchainName string) *cobra.Command {
	return &cobra.Command{
		Use:   daemonName,
		Short: fmt.Sprintf("%s provides an api to the %s blockchain", daemonName, blockchainName),
		Long:  fmt.Sprintf("%s provides an api to the %s blockchain", daemonName, blockchainName),
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
			chainApisMap := cv.GetStringMap("chain-apis")

			ethClients := []*ethclient.Client{}
			contractClients := make(map[string]*contractclient.BridgeProvider)
			releasers := []*releaser.Releaser{}
			for chainID := range chainApisMap {
				sub := cv.Sub(fmt.Sprintf("chain-apis.%s", chainID))
				endpoint := sub.GetString("endpoint")
				timeout := sub.GetDuration("timeout")
				contractAddress := sub.GetString("contract-addr")
				clientAddress := sub.GetString("client-addr")
				clientPrivateKey := sub.GetString("client-private-key")
				releaseDepositsFreq := sub.GetDuration("release-deposits-freq")

				ctx, cancel := context.WithTimeout(context.Background(), timeout)
				ethClient, err := ethclient.DialContext(ctx, endpoint)
				cancel()
				cli.CheckErrf("dialing endpoint: %v", err)

				contractAddr := ethcommon.HexToAddress(contractAddress)
				clientAddr := ethcommon.HexToAddress(clientAddress)

				contractClient, err := contractclient.NewBridgeProvider(contractAddr, ethClient)
				cli.CheckErrf("creating contract client: %v", err)

				privateKey, err := crypto.HexToECDSA(clientPrivateKey)
				cli.CheckErrf("parsing private key: %v", err)

				signer := func(a ethcommon.Address, t *types.Transaction) (*types.Transaction, error) {
					chain, ok := (&big.Int{}).SetString(chainID, 10)
					if !ok {
						return nil, fmt.Errorf("parsing chain id %s to big int", chainID)
					}
					s := types.NewEIP155Signer(chain)
					return types.SignTx(t, s, privateKey)
				}

				releaser, err := releaser.New(contractClient, chainID, clientAddr, signer, releaseDepositsFreq, timeout)
				cli.CheckErrf("creating releaser: %v", err)

				ethClients = append(ethClients, ethClient)
				contractClients[chainID] = contractClient
				releasers = append(releasers, releaser)
			}
			_ = releasers

			if len(contractClients) == 0 {
				cli.CheckErr(errors.New("no contract clients resolved"))
			}

			log.Info("Starting service...")
			listener, err := net.Listen("tcp", listenAddr)
			cli.CheckErrf("creating listener: %v", err)

			service, err := service.NewService(listener, daemonName, contractClients)
			cli.CheckErrf("creating service: %v", err)

			cli.HandleInterrupt(func() {
				for _, c := range ethClients {
					c.Close()
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
