package main

import (
	"encoding/json"
	_ "net/http/pprof"

	logging "github.com/ipfs/go-log/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/brokerd/service"
	"github.com/textileio/broker-core/cmd/common"
)

var (
	daemonName = "brokerd"
	log        = logging.Logger(daemonName)
	v          = viper.New()
)

func init() {
	flags := []common.Flag{
		{Name: "rpc-addr", DefValue: ":5000", Description: "gRPC listen address"},
		{Name: "packer-addr", DefValue: "", Description: "Packer API address"},
		{Name: "auctioneer-addr", DefValue: "", Description: "Auctioneer address"},
		{Name: "dealer-addr", DefValue: "", Description: "Dealer address"},
		{Name: "mongo-uri", DefValue: "", Description: "MongoDB URI backing go-datastore"},
		{Name: "mongo-dbname", DefValue: "", Description: "MongoDB database name backing go-datastore"},
		{Name: "ipfs-multiaddr", DefValue: "", Description: "IPFS multiaddress"},
		{Name: "metrics-addr", DefValue: ":9090", Description: "Prometheus listen address"},
		{Name: "deal-epochs", DefValue: broker.MaxDealEpochs, Description: "Deal duration in Filecoin epochs"},
		{Name: "debug", DefValue: false, Description: "Enable debug level logs"},
	}

	common.ConfigureCLI(v, "BROKER", flags, rootCmd)
}

var rootCmd = &cobra.Command{
	Use:   daemonName,
	Short: "brokerd is a Broker to store data in Filecoin",
	Long:  `brokerd is a Broker to store data in Filecoin`,
	PersistentPreRun: func(c *cobra.Command, args []string) {
		logging.SetAllLoggers(logging.LevelInfo)
		if v.GetBool("debug") {
			logging.SetAllLoggers(logging.LevelDebug)
		}
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

			PackerAddr:     v.GetString("packer-addr"),
			AuctioneerAddr: v.GetString("auctioneer-addr"),
			DealerAddr:     v.GetString("dealer-addr"),

			MongoURI:    v.GetString("mongo-uri"),
			MongoDBName: v.GetString("mongo-dbname"),

			IpfsMultiaddr: v.GetString("ipfs-multiaddr"),

			DealEpochs: v.GetUint64("deal-epochs"),
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
