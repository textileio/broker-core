package main

import (
	_ "net/http/pprof"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/bidbot/lib/auction"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/brokerd/metrics"
	"github.com/textileio/broker-core/cmd/brokerd/service"
	"github.com/textileio/broker-core/common"
	"github.com/textileio/broker-core/msgbroker/gpubsub"
	"github.com/textileio/cli"
	logging "github.com/textileio/go-log/v2"
)

var (
	daemonName = "brokerd"
	log        = logging.Logger(daemonName)
	v          = viper.New()
)

func init() {
	flags := []cli.Flag{
		{Name: "listen-addr", DefValue: ":5000", Description: "gRPC listen address"},
		{Name: "postgres-uri", DefValue: "", Description: "PostgreSQL URI"},
		{Name: "ipfs-api-multiaddr", DefValue: "", Description: "IPFS API multiaddress for unpinning data"},
		{Name: "deal-duration", DefValue: auction.MaxDealDuration, Description: "Deal duration in Filecoin epochs"},
		{Name: "deal-replication", DefValue: broker.MinDealReplication, Description: "Deal replication factor"},
		{Name: "default-wallet-address", DefValue: "",
			Description: "The wallet address by which the deals are signed, unless the storage request requires remote signing"},
		{Name: "auction-max-retries", DefValue: "5", Description: "Maximum number of re-auctioning for a storage deal"},
		{Name: "auction-deadline-duration", DefValue: "240h",
			Description: "Auction duration for creating auctions in batches with defined deadlines"},
		{Name: "auction-proposal-start-offset", DefValue: "72h",
			Description: "The default duration to calculate DealStartEpoch in fired auctions"},
		{Name: "verified-deals", DefValue: false, Description: "Make verified deals"},
		{Name: "gpubsub-project-id", DefValue: "", Description: "Google PubSub project id"},
		{Name: "gpubsub-api-key", DefValue: "", Description: "Google PubSub API key"},
		{Name: "msgbroker-topic-prefix", DefValue: "", Description: "Topic prefix to use for msg broker topics"},
		{Name: "metrics-addr", DefValue: ":9090", Description: "Prometheus listen address"},
		{Name: "log-debug", DefValue: false, Description: "Enable debug level logging"},
		{Name: "log-json", DefValue: false, Description: "Enable structured logging"},
	}

	cli.ConfigureCLI(v, "BROKER", flags, rootCmd.Flags())
}

var rootCmd = &cobra.Command{
	Use:   daemonName,
	Short: "brokerd is a Broker to store data in Filecoin",
	Long:  `brokerd is a Broker to store data in Filecoin`,
	PersistentPreRun: func(c *cobra.Command, args []string) {
		cli.ExpandEnvVars(v, v.AllSettings())
		err := cli.ConfigureLogging(v, nil)
		cli.CheckErrf("setting log levels: %v", err)
	},
	Run: func(c *cobra.Command, args []string) {
		settings, err := cli.MarshalConfig(v, !v.GetBool("log-json"), "gpubsub-api-key", "postgres-uri")
		cli.CheckErr(err)
		log.Infof("loaded config: %s", string(settings))

		if err := common.SetupInstrumentation(v.GetString("metrics-addr")); err != nil {
			log.Fatalf("booting instrumentation: %s", err)
		}

		serviceConfig := service.Config{
			ListenAddr: v.GetString("listen-addr"),

			PostgresURI: v.GetString("postgres-uri"),

			IPFSAPIMultiaddr: v.GetString("ipfs-api-multiaddr"),

			DealDuration:         v.GetUint64("deal-duration"),
			DealReplication:      v.GetUint32("deal-replication"),
			DefaultWalletAddress: v.GetString("default-wallet-address"),
			VerifiedDeals:        v.GetBool("verified-deals"),

			AuctionMaxRetries:            v.GetInt("auction-max-retries"),
			DefaultBatchDeadlineDuration: v.GetDuration("auction-deadline-duration"),
			DefaultProposalStartOffset:   v.GetDuration("auction-proposal-start-offset"),
		}

		projectID := v.GetString("gpubsub-project-id")
		apiKey := v.GetString("gpubsub-api-key")
		topicPrefix := v.GetString("msgbroker-topic-prefix")
		mb, err := gpubsub.NewMetered(projectID, apiKey, topicPrefix, "brokerd", metrics.Meter)
		cli.CheckErr(err)

		serv, err := service.New(mb, serviceConfig)
		cli.CheckErr(err)

		log.Info("listening to requests...")

		cli.HandleInterrupt(func() {
			if err := serv.Close(); err != nil {
				log.Errorf("closing http endpoint: %s", err)
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
