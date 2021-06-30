package main

import (
	"encoding/json"
	_ "net/http/pprof"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/broker-core/cmd/dealerd/service"
	"github.com/textileio/broker-core/common"
	logging "github.com/textileio/go-log/v2"
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
		{Name: "allow-unverified-deals", DefValue: false, Description: "Allow unverified deals"},
		{
			Name:        "max-verified-price-per-gib-per-epoch",
			DefValue:    int64(0),
			Description: "Maximum price accepted for verified deals",
		},
		{
			Name:        "max-unverified-price-per-gib-per-epoch",
			DefValue:    int64(0),
			Description: "Maximum price accepted for unverified deals",
		},
		{Name: "mock", DefValue: false, Description: "Provides a mocked behavior"},
		{Name: "metrics-addr", DefValue: ":9090", Description: "Prometheus listen address"},
		{Name: "log-debug", DefValue: false, Description: "Enable debug level logging"},
		{Name: "log-json", DefValue: false, Description: "Enable structured logging"},
	}

	common.ConfigureCLI(v, "DEALER", flags, rootCmd.Flags())
}

var rootCmd = &cobra.Command{
	Use:   daemonName,
	Short: "dealerd executes deals for winning bids",
	Long:  "dealerd executes deals for winning bids",
	PersistentPreRun: func(c *cobra.Command, args []string) {
		common.ExpandEnvVars(v, v.AllSettings())
		err := common.ConfigureLogging(v, []string{
			daemonName,
			"dealer/service",
			"dealer",
			"dealermock",
			"dealer/filclient",
		})
		common.CheckErrf("setting log levels: %v", err)
	},
	Run: func(c *cobra.Command, args []string) {
		settings, err := marshalConfig(v)
		common.CheckErr(err)
		log.Infof("loaded config: %s", string(settings))

		if err := common.SetupInstrumentation(v.GetString("metrics-addr")); err != nil {
			log.Fatalf("booting instrumentation: %s", err)
		}

		config := service.Config{
			ListenAddr:    v.GetString("rpc-addr"),
			BrokerAPIAddr: v.GetString("broker-addr"),

			MongoURI:    v.GetString("mongo-uri"),
			MongoDBName: v.GetString("mongo-dbname"),

			LotusGatewayURL:         v.GetString("lotus-gateway-url"),
			LotusExportedWalletAddr: v.GetString("lotus-exported-wallet-address"),

			AllowUnverifiedDeals:             v.GetBool("allow-unverified-deals"),
			MaxVerifiedPricePerGiBPerEpoch:   v.GetInt64("max-verified-price-per-gib-per-epoch"),
			MaxUnverifiedPricePerGiBPerEpoch: v.GetInt64("max-unverified-price-per-gib-per-epoch"),

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

func marshalConfig(v *viper.Viper) ([]byte, error) {
	all := v.AllSettings()
	if all["lotus-exported-wallet-address"].(string) != "" {
		all["lotus-exported-wallet-address"] = "***"
	}
	return json.MarshalIndent(all, "", "  ")
}
