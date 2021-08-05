package main

import (
	"net"
	_ "net/http/pprof"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/bidbot/lib/common"
	"github.com/textileio/broker-core/cmd/authd/service"
	"github.com/textileio/broker-core/cmd/chainapis/client"
	logging "github.com/textileio/go-log/v2"
	"google.golang.org/grpc"
)

var (
	daemonName = "authd"
	log        = logging.Logger(daemonName)
	v          = viper.New()
)

func init() {
	flags := []common.Flag{
		{Name: "postgres-uri", DefValue: "", Description: "PostgreSQL URI"},
		{Name: "rpc-addr", DefValue: ":5000", Description: "gRPC listen address"},
		{Name: "near-addr", DefValue: "", Description: "NEAR chain API address"},
		{Name: "eth-addr", DefValue: "", Description: "Ethereum chain API address"},
		{Name: "poly-addr", DefValue: "", Description: "Polygon chain API address"},
		{Name: "metrics-addr", DefValue: ":9090", Description: "Prometheus listen address"},
		{Name: "log-debug", DefValue: false, Description: "Enable debug level logging"},
		{Name: "log-json", DefValue: false, Description: "Enable structured logging"},
	}

	common.ConfigureCLI(v, "AUTH", flags, rootCmd.Flags())
}

var rootCmd = &cobra.Command{
	Use:   daemonName,
	Short: "authd provides authentication services for the Broker",
	Long:  `authd provides authentication services for the Broker`,
	PersistentPreRun: func(c *cobra.Command, args []string) {
		common.ExpandEnvVars(v, v.AllSettings())
		err := common.ConfigureLogging(v, nil)
		common.CheckErrf("setting log levels: %v", err)
	},
	Run: func(c *cobra.Command, args []string) {
		settings, err := common.MarshalConfig(v, !v.GetBool("log-json"), "postgres-uri")
		common.CheckErr(err)
		log.Infof("loaded config: %s", string(settings))

		if err := common.SetupInstrumentation(v.GetString("metrics-addr")); err != nil {
			log.Fatalf("booting instrumentation: %s", err)
		}

		nearAPIClientConn, err := grpc.Dial(v.GetString("near-addr"), grpc.WithInsecure())
		if err != nil {
			log.Fatalf("creating near api connection: %v", err)
		}
		nearAPIClient := client.New(nearAPIClientConn)

		ethAPIClientConn, err := grpc.Dial(v.GetString("eth-addr"), grpc.WithInsecure())
		if err != nil {
			log.Fatalf("creating eth api connection: %v", err)
		}
		ethAPIClient := client.New(ethAPIClientConn)

		polyAPIClientConn, err := grpc.Dial(v.GetString("poly-addr"), grpc.WithInsecure())
		if err != nil {
			log.Fatalf("creating poly api connection: %v", err)
		}
		polyAPIClient := client.New(polyAPIClientConn)

		listener, err := net.Listen("tcp", v.GetString("rpc-addr"))
		if err != nil {
			log.Fatalf("creating listener connection: %v", err)
		}
		config := service.Config{
			Listener:    listener,
			PostgresURI: v.GetString("postgres-uri"),
		}
		deps := service.Deps{
			NearAPI: nearAPIClient,
			EthAPI:  ethAPIClient,
			PolyAPI: polyAPIClient,
		}
		serv, err := service.New(config, deps)
		common.CheckErr(err)

		common.HandleInterrupt(func() {
			if err := nearAPIClientConn.Close(); err != nil {
				log.Errorf("closing near chain api client conn: %v", err)
			}
			if err := ethAPIClientConn.Close(); err != nil {
				log.Errorf("closing eth chain api client conn: %v", err)
			}
			if err := polyAPIClientConn.Close(); err != nil {
				log.Errorf("closing poly chain api client conn: %v", err)
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
