package main

import (
	"context"
	"net"
	"time"

	ec "github.com/ethereum/go-ethereum/common"
	et "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/textileio/broker-core/cmd/chainapis/ethshared/contractclient"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/bidbot/lib/common"
	"github.com/textileio/broker-core/cmd/chainapis/ethshared/releaser"
	"github.com/textileio/broker-core/cmd/chainapis/ethshared/service"
	logging "github.com/textileio/go-log/v2"
)

var (
	daemonName = "polyd"
	log        = logging.Logger(daemonName)
	v          = viper.New()
)

var flags = []common.Flag{
	{Name: "rpc-addr", DefValue: "", Description: "gRPC listen address"},
	{
		Name:        "endpoint-url",
		DefValue:    "https://polygon-mumbai.infura.io/v3/92f6902cf1214401ae5b08a1e117eb91",
		Description: "The Polygon enpoint URL to use",
	},
	{Name: "endpoint-timeout", DefValue: time.Second * 5, Description: "Timeout for initial connection to endpoint-url"},
	{
		Name:        "contract-address",
		DefValue:    "0x8845A98EF6580d2a109f8FcfC10cc1d6007059fc",
		Description: "The Polygon address of the provider contract",
	},
	{
		Name:        "client-address",
		DefValue:    "", // TODO: Set a default with our actual address.
		Description: "The Polygon address of the user of this client",
	},
	{
		Name:        "client-private-key",
		DefValue:    "",
		Description: "The Polygon private key string of the client account",
	},
	{Name: "release-deposits-freq", DefValue: time.Minute * 10, Description: "How often to call releaseDeposits"},
	{
		Name:        "release-deposits-timeout",
		DefValue:    time.Second * 30,
		Description: "Request timeout when calling releaseDeposits",
	},
	{Name: "log-debug", DefValue: false, Description: "Enable debug level logging"},
	{Name: "log-json", DefValue: false, Description: "Enable structured logging"},
}

func init() {
	common.ConfigureCLI(v, "POLY", flags, rootCmd.Flags())
}

var rootCmd = &cobra.Command{
	Use:   daemonName,
	Short: "polyd is provides an api to the Polygon blockchain",
	Long:  `polyd is provides an api to the Polygon blockchain`,
	PersistentPreRun: func(c *cobra.Command, args []string) {
		common.ExpandEnvVars(v, v.AllSettings())
		err := common.ConfigureLogging(v, nil)
		common.CheckErrf("setting log levels: %v", err)
	},
	Run: func(c *cobra.Command, args []string) {
		settings, err := common.MarshalConfig(v, !v.GetBool("log-json"), "client-private-key")
		common.CheckErr(err)
		log.Infof("loaded config: %s", string(settings))

		listenAddr := v.GetString("rpc-addr")
		endpointURL := v.GetString("endpoint-url")
		endpointTimeout := v.GetDuration("endpoint-timeout")
		contractAddress := v.GetString("contract-address")
		clientAddress := v.GetString("client-address")
		clientPrivateKey := v.GetString("client-private-key")
		releaseDepositsFreq := v.GetDuration("release-deposits-freq")
		releaseDepositsTimeout := v.GetDuration("release-deposits-timeout")

		ctx, cancel := context.WithTimeout(context.Background(), endpointTimeout)
		defer cancel()

		ethClient, err := ethclient.DialContext(ctx, endpointURL)
		common.CheckErr(err)

		contractAddr := ec.HexToAddress(contractAddress)
		clientAddr := ec.HexToAddress(clientAddress)

		contractClient, err := contractclient.NewBridgeProvider(contractAddr, ethClient)
		common.CheckErr(err)

		privateKey, err := crypto.HexToECDSA(clientPrivateKey)
		common.CheckErr(err)

		signer := func(a ec.Address, t *et.Transaction) (*et.Transaction, error) {
			signer := et.HomesteadSigner{}
			return et.SignTx(t, signer, privateKey)
		}

		releaser := releaser.New(contractClient, clientAddr, signer, releaseDepositsFreq, releaseDepositsTimeout)

		log.Info("Starting service...")
		listener, err := net.Listen("tcp", listenAddr)
		common.CheckErr(err)

		service, err := service.NewService(listener, contractClient)
		common.CheckErr(err)

		common.HandleInterrupt(func() {
			// common.CheckErr(u.Close())
			ethClient.Close()
			common.CheckErr(releaser.Close())
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
