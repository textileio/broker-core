package main

import (
	"encoding/json"
	_ "net/http/pprof"

	logging "github.com/ipfs/go-log/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/broker-core/cmd/common"
	"github.com/textileio/broker-core/cmd/storaged/service"
)

var (
	daemonName = "storagerd"
	log        = logging.Logger(daemonName)
	v          = viper.New()
)

func init() {
	flags := []common.Flag{
		{Name: "http-listen.addr", DefValue: ":8888", Description: "HTTP API listen address"},
		{Name: "uploader-ipfs-multiaddr", DefValue: "/ip4/127.0.0.1/tcp/5001", Description: "Uploader IPFS API pool"},
		{Name: "broker-addr", DefValue: "", Description: "Broker API address"},

		{Name: "metrics-addr", DefValue: ":9090", Description: "Prometheus listen address"},
		{Name: "authd-addr", DefValue: "TODO", Description: "The authd address"},
		{Name: "log-debug", DefValue: false, Description: "Enable debug level logs"},
	}

	common.ConfigureCLI(v, "STORAGE", flags, rootCmd)
}

var rootCmd = &cobra.Command{
	Use:   daemonName,
	Short: "storaged provides a synchronous data uploader endpoint to store data in a Broker",
	Long:  `storaged provides a synchronous data uploader endpoint to store data in a Broker`,
	PersistentPreRun: func(c *cobra.Command, args []string) {
		logging.SetAllLoggers(logging.LevelInfo)
		if v.GetBool("log-debug") {
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
			HTTPListenAddr:        v.GetString("http-listen-addr"),
			UploaderIPFSMultiaddr: v.GetString("uploader-ipfs-multiaddr"),
			BrokerAPIAddr:         v.GetString("broker-addr"),
			AuthAddr:              v.GetString("authd-addr"),
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
