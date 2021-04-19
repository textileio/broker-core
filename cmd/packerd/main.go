package main

import (
	"encoding/json"
	_ "net/http/pprof"

	logging "github.com/ipfs/go-log/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/broker-core/cmd/common"
	"github.com/textileio/broker-core/cmd/packerd/service"
)

var (
	daemonName = "packerd"
	log        = logging.Logger(daemonName)
	v          = viper.New()
)

func init() {
	flags := []common.Flag{
		{Name: "repo", DefValue: ".packer", Description: "Repo path"},
		{Name: "rpc.addr", DefValue: ":5000", Description: "gRPC listen address"},

		{Name: "metrics.addr", DefValue: ":9090", Description: "Prometheus listen address"},
		{Name: "log.debug", DefValue: false, Description: "Enable debug level logs"},
	}

	common.ConfigureCLI(v, "packer", flags, rootCmd)
}

var rootCmd = &cobra.Command{
	Use:   daemonName,
	Short: "packerd handles deal auctions for the Broker",
	Long:  "packerd handles deal auctions for the Broker",
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

		config := service.Config{
			RepoPath:   v.GetString("repo"),
			ListenAddr: v.GetString("rpc.addr"),
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
