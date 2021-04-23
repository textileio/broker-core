package main

import (
	"encoding/json"
	_ "net/http/pprof"

	logging "github.com/ipfs/go-log/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/broker-core/cmd/authd/service"
	"github.com/textileio/broker-core/cmd/common"
	chainapi "github.com/textileio/broker-core/gen/broker/chainapi/v1"
	"google.golang.org/grpc"
)

var (
	daemonName = "authd"
	log        = logging.Logger(daemonName)
	v          = viper.New()
)

func init() {
	flags := []common.Flag{
		{Name: "rpc.addr", DefValue: ":5000", Description: "gRPC listen address"},
		{Name: "near.addr", DefValue: "", Description: "neard service api address"},
		{Name: "metrics.addr", DefValue: ":9090", Description: "Prometheus listen address"},
		{Name: "log.debug", DefValue: false, Description: "Enable debug level logs"},
	}

	common.ConfigureCLI(v, "AUTH", flags, rootCmd)
}

var rootCmd = &cobra.Command{
	Use:   daemonName,
	Short: "authd provides authentication services for the Broker",
	Long:  `authd provides authentication services for the Broker`,
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

		chainAPIClientConn, err := grpc.Dial(v.GetString("near.addr"), grpc.WithInsecure())
		if err != nil {
			log.Fatalf("creating near api connection: %v", err)
		}
		chainAPIClient := chainapi.NewChainApiServiceClient(chainAPIClientConn)

		config := service.Config{ListenAddr: v.GetString("grpc.listen.addr")}
		deps := service.Deps{ChainAPIServiceClient: chainAPIClient}

		serv, err := service.New(config, deps)
		common.CheckErr(err)

		common.HandleInterrupt(func() {
			if err := chainAPIClientConn.Close(); err != nil {
				log.Errorf("closing chain api client conn: %v", err)
			}
			if err := serv.Close(); err != nil {
				log.Errorf("closing service: %s", err)
			}
		})
	},
}

func main() {
	common.CheckErr(rootCmd.Execute())
}
