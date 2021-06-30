package main

import (
	"encoding/json"
	_ "net/http/pprof"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/broker-core/cmd/packerd/service"
	"github.com/textileio/broker-core/common"
	logging "github.com/textileio/go-log/v2"
)

var (
	daemonName = "packerd"
	log        = logging.Logger(daemonName)
	v          = viper.New()
)

func init() {
	flags := []common.Flag{
		{Name: "rpc-addr", DefValue: ":5000", Description: "gRPC listen address"},
		{Name: "mongo-uri", DefValue: "", Description: "MongoDB URI backing go-datastore"},
		{Name: "mongo-dbname", DefValue: "", Description: "MongoDB database name backing go-datastore"},
		{Name: "broker-addr", DefValue: "", Description: "Broker API address"},
		{Name: "ipfs-multiaddr", DefValue: "", Description: "IPFS multiaddress"},
		{Name: "daemon-frequency", DefValue: "20s", Description: "Frequency of polling ready batches"},
		{Name: "export-metrics-frequency", DefValue: "5m", Description: "Frequency of metrics exporting"},
		{Name: "batch-min-size", DefValue: "10MB", Description: "Minimum batch size"},
		{Name: "target-sector-size", DefValue: "34359738368", Description: "Target sector-sizes"},
		{Name: "metrics-addr", DefValue: ":9090", Description: "Prometheus listen address"},
		{Name: "log-debug", DefValue: false, Description: "Enable debug level logging"},
		{Name: "log-json", DefValue: false, Description: "Enable structured logging"},
	}

	common.ConfigureCLI(v, "PACKER", flags, rootCmd.Flags())
}

var rootCmd = &cobra.Command{
	Use:   daemonName,
	Short: "packerd handles deal auctions for the Broker",
	Long:  "packerd handles deal auctions for the Broker",
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

		config := service.Config{
			ListenAddr:       v.GetString("rpc-addr"),
			IpfsAPIMultiaddr: v.GetString("ipfs-multiaddr"),
			BrokerAPIAddr:    v.GetString("broker-addr"),

			MongoURI:    v.GetString("mongo-uri"),
			MongoDBName: v.GetString("mongo-dbname"),

			DaemonFrequency:        v.GetDuration("daemon-frequency"),
			ExportMetricsFrequency: v.GetDuration("export-metrics-frequency"),

			TargetSectorSize: v.GetInt64("target-sector-size"),
			BatchMinSize:     v.GetSizeInBytes("batch-min-size"),
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
