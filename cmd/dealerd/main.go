package main

import (
	"encoding/json"
	_ "net/http/pprof"

	logging "github.com/ipfs/go-log/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/broker-core/cmd/common"
	"github.com/textileio/broker-core/cmd/dealerd/service"
)

var (
	daemonName = "dealerd"
	log        = logging.Logger(daemonName)
	v          = viper.New()
)

func init() {
	flags := []common.Flag{
		{Name: "rpc-addr", DefValue: ":5000", Description: "gRPC listen address"},
		{Name: "mongo-uri", DefValue: "", Description: "MongoDB URI backing go-datastore"},
		{Name: "mongo-dbname", DefValue: "", Description: "MongoDB database name backing go-datastore"},
		{Name: "broker-addr", DefValue: "", Description: "Broker API address"},
		{Name: "lotus-gateway-url", DefValue: "https://api.node.glif.io", Description: "Lotus gateway URL"},
		{
			Name:        "lotus-exported-wallet-address",
			DefValue:    "",
			Description: "Exported wallet address for deal making",
		},
		{Name: "mock", DefValue: false, Description: "Provides a mocked behavior"},
		{Name: "metrics-addr", DefValue: ":9090", Description: "Prometheus listen address"},
		{Name: "log-debug", DefValue: false, Description: "Enable debug level logging"},
		{Name: "log-json", DefValue: false, Description: "Enable structured logging"},
	}

	common.ConfigureCLI(v, "dealer", flags, rootCmd)
}

var rootCmd = &cobra.Command{
	Use:   daemonName,
	Short: "dealerd executes deals for winning bids",
	Long:  "dealerd executes deals for winning bids",
	PersistentPreRun: func(c *cobra.Command, args []string) {
		common.ExpandEnvVars(v, v.AllSettings())
		err := common.ConfigureLogging(v, nil)
		common.CheckErrf("setting log levels: %v", err)
	},
	Run: func(c *cobra.Command, args []string) {
		settings, err := json.MarshalIndent(v.AllSettings(), "", "  ")
		common.CheckErr(err)
		log.Infof("loaded config: %s", string(settings))

		if err := common.SetupInstrumentation(v.GetString("metrics.addr")); err != nil {
			log.Fatalf("booting instrumentation: %s", err)
		}

		config := service.Config{
			ListenAddr:    v.GetString("rpc-addr"),
			BrokerAPIAddr: v.GetString("broker-addr"),

			MongoURI:    v.GetString("mongo-uri"),
			MongoDBName: v.GetString("mongo-dbname"),

			LotusGatewayURL:         v.GetString("lotus-gateway-url"),
			LotusExportedWalletAddr: v.GetString("lotus-exported-wallet-addr"),

			Mock: v.GetBool("mock"),
		}
		serv, err := service.New(config)
		common.CheckErr(err)

		common.HandleInterrupt(func() {
			if err := serv.Close(); err != nil {
				log.Errorf("closing service: %s", err)
			}
		})
	},
}

func main() {
	common.CheckErr(rootCmd.Execute())
}
