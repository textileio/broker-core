package main

import (
	"encoding/json"
	_ "net/http/pprof"

	logging "github.com/ipfs/go-log/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
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
		{Name: "rpc.addr", DefValue: ":5000", Description: "gRPC listen address"},
		{Name: "auctioneer.addr", DefValue: ":5001", Description: "Auctioneer address"},
		{Name: "metrics.addr", DefValue: ":9090", Description: "Prometheus listen address"},
		{Name: "log.debug", DefValue: false, Description: "Enable debug level logs"},

		{Name: "mongo.uri", DefValue: "", Description: "MongoDB URI backing go-datastore"},
		{Name: "mongo.dbname", DefValue: "", Description: "MongoDB database name backing go-datastore"},
	}

	common.ConfigureCLI(v, "BROKER", flags, rootCmd)
}

var rootCmd = &cobra.Command{
	Use:   daemonName,
	Short: "brokerd is a Broker to store data in Filecoin",
	Long:  `brokerd is a Broker to store data in Filecoin`,
	PersistentPreRun: func(c *cobra.Command, args []string) {
		logging.SetAllLoggers(logging.LevelInfo)
		if v.GetBool("log.debug") {
			logging.SetAllLoggers(logging.LevelDebug)
		}
	},
	Run: func(c *cobra.Command, args []string) {
		settings, err := json.MarshalIndent(v.AllSettings(), "", "  ")
		common.CheckErr(err)
		log.Infof("loaded config: %s", string(settings))

		if err := common.SetupInstrumentation(v.GetString("metrics.addr")); err != nil {
			log.Fatalf("booting instrumentation: %s", err)
		}

		serviceConfig := service.Config{
			GrpcListenAddress: v.GetString("grpc.listen.addr"),

			AuctioneerAddr: v.GetString("auctioneer.addr"),

			MongoURI:    v.GetString("mongo.uri"),
			MongoDBName: v.GetString("mongo.dbname"),
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
