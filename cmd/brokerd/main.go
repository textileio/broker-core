package main

import (
	"encoding/json"
	_ "net/http/pprof"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/bidbot/lib/auction"
	"github.com/textileio/bidbot/lib/common"
	"github.com/textileio/broker-core/cmd/brokerd/service"
	logging "github.com/textileio/go-log/v2"
)

var (
	daemonName = "brokerd"
	log        = logging.Logger(daemonName)
	v          = viper.New()
)

func init() {
	flags := []common.Flag{
		{Name: "rpc-addr", DefValue: ":5000", Description: "gRPC listen address"},
		{Name: "mongo-uri", DefValue: "", Description: "MongoDB URI backing go-datastore"},
		{Name: "mongo-dbname", DefValue: "", Description: "MongoDB database name backing go-datastore"},
		{Name: "ipfs-api-multiaddr", DefValue: "", Description: "IPFS API multiaddress for unpinning data"},
		{Name: "piecer-addr", DefValue: "", Description: "Piecer API address"},
		{Name: "packer-addr", DefValue: "", Description: "Packer API address"},
		{Name: "auctioneer-addr", DefValue: "", Description: "Auctioneer API address"},
		{Name: "dealer-addr", DefValue: "", Description: "Dealer API address"},
		{Name: "reporter-addr", DefValue: "", Description: "Reporter API address"},
		{Name: "deal-duration", DefValue: auction.MaxDealDuration, Description: "Deal duration in Filecoin epochs"},
		{Name: "deal-replication", DefValue: auction.MinDealReplication, Description: "Deal replication factor"},
		{Name: "auction-max-retries", DefValue: "5", Description: "Maximum number of re-auctioning for a storage deal"},
		{Name: "verified-deals", DefValue: false, Description: "Make verified deals"},
		{Name: "metrics-addr", DefValue: ":9090", Description: "Prometheus listen address"},
		{Name: "car-export-url", DefValue: "", Description: "URL that generates CAR files for stored cids"},
		{Name: "log-debug", DefValue: false, Description: "Enable debug level logging"},
		{Name: "log-json", DefValue: false, Description: "Enable structured logging"},
	}

	common.ConfigureCLI(v, "BROKER", flags, rootCmd.Flags())
}

var rootCmd = &cobra.Command{
	Use:   daemonName,
	Short: "brokerd is a Broker to store data in Filecoin",
	Long:  `brokerd is a Broker to store data in Filecoin`,
	PersistentPreRun: func(c *cobra.Command, args []string) {
		common.ExpandEnvVars(v, v.AllSettings())
		err := common.ConfigureLogging(v, nil)
		common.CheckErrf("setting log levels: %v", err)
	},
	Run: func(c *cobra.Command, args []string) {
		settings, err := json.MarshalIndent(v.AllSettings(), "", "  ")
		common.CheckErr(err)
		log.Infof("loaded config: %s", string(settings))

		if err := common.SetupInstrumentation(v.GetString("metrics-addr")); err != nil {
			log.Fatalf("booting instrumentation: %s", err)
		}

		serviceConfig := service.Config{
			ListenAddr: v.GetString("rpc-addr"),

			PiecerAddr:     v.GetString("piecer-addr"),
			PackerAddr:     v.GetString("packer-addr"),
			AuctioneerAddr: v.GetString("auctioneer-addr"),
			DealerAddr:     v.GetString("dealer-addr"),
			ReporterAddr:   v.GetString("reporter-addr"),

			MongoURI:    v.GetString("mongo-uri"),
			MongoDBName: v.GetString("mongo-dbname"),

			IPFSAPIMultiaddr: v.GetString("ipfs-api-multiaddr"),

			DealDuration:    v.GetUint64("deal-duration"),
			DealReplication: v.GetUint32("deal-replication"),
			VerifiedDeals:   v.GetBool("verified-deals"),

			CARExportURL: v.GetString("car-export-url"),

			AuctionMaxRetries: v.GetInt("auction-max-retries"),
		}
		serv, err := service.New(serviceConfig)
		common.CheckErr(err)

		log.Info("listening to requests...")

		common.HandleInterrupt(func() {
			if err := serv.Close(); err != nil {
				log.Errorf("closing http endpoint: %s", err)
			}
		})
	},
}

func main() {
	common.CheckErr(rootCmd.Execute())
}
