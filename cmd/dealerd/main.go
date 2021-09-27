package main

import (
	_ "net/http/pprof"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/broker-core/cmd/dealerd/metrics"
	"github.com/textileio/broker-core/cmd/dealerd/service"
	"github.com/textileio/broker-core/common"
	"github.com/textileio/broker-core/msgbroker/gpubsub"
	"github.com/textileio/cli"
	logging "github.com/textileio/go-log/v2"
)

var (
	daemonName = "dealerd"
	log        = logging.Logger(daemonName)
	v          = viper.New()
)

func init() {
	flags := []cli.Flag{
		{Name: "postgres-uri", DefValue: "", Description: "PostgreSQL URI"},
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
		{Name: "relay-maddr", DefValue: "", Description: "Relay multiaddress for remote wallets"},
		{Name: "mock", DefValue: false, Description: "Provides a mocked behavior"},
		{Name: "gpubsub-project-id", DefValue: "", Description: "Google PubSub project id"},
		{Name: "gpubsub-api-key", DefValue: "", Description: "Google PubSub API key"},
		{Name: "msgbroker-topic-prefix", DefValue: "", Description: "Topic prefix to use for msg broker topics"},
		{Name: "metrics-addr", DefValue: ":9090", Description: "Prometheus listen address"},
		{Name: "log-debug", DefValue: false, Description: "Enable debug level logging"},
		{Name: "log-json", DefValue: false, Description: "Enable structured logging"},
	}

	cli.ConfigureCLI(v, "DEALER", flags, rootCmd.Flags())
}

var rootCmd = &cobra.Command{
	Use:   daemonName,
	Short: "dealerd executes deals for winning bids",
	Long:  "dealerd executes deals for winning bids",
	PersistentPreRun: func(c *cobra.Command, args []string) {
		cli.ExpandEnvVars(v, v.AllSettings())
		err := cli.ConfigureLogging(v, []string{
			daemonName,
			"dealer/service",
			"dealer",
			"dealermock",
			"dealer/filclient",
		})
		cli.CheckErrf("setting log levels: %v", err)
	},
	Run: func(c *cobra.Command, args []string) {
		settings, err := cli.MarshalConfig(v, !v.GetBool("log-json"), "gpubsub-api-key",
			"lotus-exported-wallet-address", "postgres-uri")
		cli.CheckErr(err)
		log.Infof("loaded config: %s", string(settings))

		if err := common.SetupInstrumentation(v.GetString("metrics-addr")); err != nil {
			log.Fatalf("booting instrumentation: %s", err)
		}

		config := service.Config{
			PostgresURI: v.GetString("postgres-uri"),

			LotusGatewayURL:         v.GetString("lotus-gateway-url"),
			LotusExportedWalletAddr: v.GetString("lotus-exported-wallet-address"),

			AllowUnverifiedDeals:             v.GetBool("allow-unverified-deals"),
			MaxVerifiedPricePerGiBPerEpoch:   v.GetInt64("max-verified-price-per-gib-per-epoch"),
			MaxUnverifiedPricePerGiBPerEpoch: v.GetInt64("max-unverified-price-per-gib-per-epoch"),
			RelayMaddr:                       v.GetString("relay-maddr"),

			Mock: v.GetBool("mock"),
		}
		projectID := v.GetString("gpubsub-project-id")
		apiKey := v.GetString("gpubsub-api-key")
		topicPrefix := v.GetString("msgbroker-topic-prefix")
		mb, err := gpubsub.NewMetered(projectID, apiKey, topicPrefix, "dealerd", metrics.Meter)
		cli.CheckErrf("creating google pubsub client: %s", err)

		serv, err := service.New(mb, config)
		cli.CheckErr(err)

		cli.HandleInterrupt(func() {
			if err := serv.Close(); err != nil {
				log.Errorf("closing service: %s", err)
			}
			if err := mb.Close(); err != nil {
				log.Errorf("closing message broker: %s", err)
			}
		})
	},
}

func main() {
	cli.CheckErr(rootCmd.Execute())
}
