package main

import (
	"encoding/json"
	"fmt"
	"strings"

	_ "net/http/pprof"

	logging "github.com/ipfs/go-log/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/broker-core/cmd/brokerd/service"
	"github.com/textileio/broker-core/cmd/util"
)

var (
	daemonName = "brokerd"
	log        = logging.Logger(daemonName)
	v          = viper.New()
)

var flags = []util.Flag{
	{Name: "metrics.addr", DefValue: ":9090", Description: "Prometheus endpoint"},
	{Name: "log.debug", DefValue: false, Description: "Enable debug level logs"},

	{Name: "grpc.listen.addr", DefValue: ":5000", Description: "gRPC API listen address"},

	{Name: "mongo.uri", DefValue: "", Description: "MongoDB URI backing go-datastore"},
	{Name: "mongo.dbname", DefValue: "", Description: "MongoDB database name backing go-datastore"},
}

func init() {
	v.SetEnvPrefix("BROKER")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	util.BindFlags(flags, rootCmd, v)
}

var rootCmd = &cobra.Command{
	Use:   daemonName,
	Short: "brokerd is a broker to store data in Filecoin",
	Long:  `brokerd is a broker to store data in Filecoin`,
	PersistentPreRun: func(c *cobra.Command, args []string) {
		logging.SetAllLoggers(logging.LevelInfo)
		if v.GetBool("log.debug") {
			logging.SetAllLoggers(logging.LevelDebug)
		}
	},
	Run: func(c *cobra.Command, args []string) {
		settings, err := json.MarshalIndent(v.AllSettings(), "", "  ")
		util.CheckErr(err)
		log.Infof("loaded config: %s", string(settings))

		if err := util.SetupInstrumentation(v.GetString("metrics.addr")); err != nil {
			log.Fatalf("booting instrumentation: %s", err)
		}

		serviceConfig := service.Config{
			GrpcListenAddress: v.GetString("grpc.listen.addr"),

			MongoURI:    v.GetString("mongo.uri"),
			MongoDBName: v.GetString("mongo.dbname"),
		}
		serv, err := service.New(serviceConfig)
		util.CheckErr(err)

		log.Info("Listening to requests...")

		util.WaitUntilTerminated()

		fmt.Println("Gracefully stopping... (press Ctrl+C again to force)")
		if err := serv.Close(); err != nil {
			log.Errorf("closing http endpoint: %s", err)
		}
	},
}

func main() {
	util.CheckErr(rootCmd.Execute())
}
