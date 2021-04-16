package main

import (
	"encoding/json"
	"fmt"

	_ "net/http/pprof"

	logging "github.com/ipfs/go-log/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/broker-core/cmd/storaged/service"
	"github.com/textileio/broker-core/cmd/util"
)

var (
	daemonName = "storagerd"
	log        = logging.Logger(daemonName)
	v          = viper.New()
)

func init() {
	flags := []util.Flag{
		{Name: "http.listen.addr", DefValue: ":8888", Description: "HTTP API listen address"},
		{Name: "uploader.ipfs.multiaddr", DefValue: "/ip4/127.0.0.1/tcp/5001", Description: "Uploader IPFS API pool"},
		{Name: "metrics.addr", DefValue: ":9090", Description: "Prometheus endpoint"},
		{Name: "log.debug", DefValue: false, Description: "Enable debug level logs"},
	}

	util.ConfigureCLI(v, "STORAGE", flags, rootCmd)
}

var rootCmd = &cobra.Command{
	Use:   daemonName,
	Short: "storaged provides a synchronous data uploader endpoint to store data in a Broker",
	Long:  `storaged provides a synchronous data uploader endpoint to store data in a Broker`,
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
			HTTPListenAddr:        v.GetString("http.listen.addr"),
			UploaderIPFSMultiaddr: v.GetString("uploader.ipfs.multiaddr"),
		}
		serv, err := service.New(serviceConfig)
		util.CheckErr(err)

		log.Info("Listening to requests...")

		util.WaitForTerminateSignal()

		fmt.Println("Gracefully stopping... (press Ctrl+C again to force)")
		if err := serv.Close(); err != nil {
			log.Errorf("closing http endpoint: %s", err)
		}
	},
}

func main() {
	util.CheckErr(rootCmd.Execute())
}
