package main

import (
	"encoding/json"
	"fmt"

	_ "net/http/pprof"

	logging "github.com/ipfs/go-log/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/broker-core/cmd/authd/service"
	"github.com/textileio/broker-core/cmd/util"
)

var (
	daemonName = "authd"
	log        = logging.Logger(daemonName)
	v          = viper.New()
)

func init() {
	flags := []util.Flag{
		{Name: "grpc.listen.addr", DefValue: ":5000", Description: "gRPC API listen address"},
		{Name: "metrics.addr", DefValue: ":9090", Description: "Prometheus endpoint"},
		{Name: "log.debug", DefValue: false, Description: "Enable debug level logs"},
	}

	util.ConfigureCLI(v, "AUTH", flags, rootCmd)
}

var rootCmd = &cobra.Command{
	Use:   daemonName,
	Short: "authd provides authentication services for the broker",
	Long:  `authd provides authentication services for the broker`,
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

		serv, err := service.New(v.GetString("grpc.listen.addr"))
		util.CheckErr(err)

		util.WaitForTerminateSignal()

		fmt.Println("Gracefully stopping... (press Ctrl+C again to force)")
		if err := serv.Close(); err != nil {
			log.Errorf("closing service: %s", err)
		}
	},
}

func main() {
	util.CheckErr(rootCmd.Execute())
}
