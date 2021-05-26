package main

import (
	"encoding/json"
	"net"
	_ "net/http/pprof"
	"time"

	httpapi "github.com/ipfs/go-ipfs-http-client"
	logging "github.com/ipfs/go-log/v2"
	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/broker-core/cmd/brokerd/client"
	"github.com/textileio/broker-core/cmd/common"
	"github.com/textileio/broker-core/cmd/piecerd/service"
	"github.com/textileio/broker-core/dshelper"
)

var (
	daemonName = "piecerd"
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
		{Name: "daemon-frequency", DefValue: time.Second * 30, Description: "Daemon frequency to process pending data"},
		{Name: "retry-delay", DefValue: time.Second * 20, Description: "Delay for reprocessing items"},
		{Name: "metrics-addr", DefValue: ":9090", Description: "Prometheus listen address"},
		{Name: "log-debug", DefValue: false, Description: "Enable debug level logging"},
		{Name: "log-json", DefValue: false, Description: "Enable structured logging"},
	}

	common.ConfigureCLI(v, "PIECER", flags, rootCmd.Flags())
}

var rootCmd = &cobra.Command{
	Use:   daemonName,
	Short: "piecerd handles prepares batched data for Filecoin",
	Long:  "piecerd handles prepares batched data for Filecoin",
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

		listener, err := net.Listen("tcp", v.GetString("rpc-addr"))
		common.CheckErrf("creating listener: %v", err)

		ma, err := multiaddr.NewMultiaddr(v.GetString("ipfs-multiaddr"))
		common.CheckErrf("parsing ipfs multiaddr: %v", err)
		ipfsClient, err := httpapi.NewApi(ma)
		common.CheckErrf("creating ipfs http api client: %v", err)

		broker, err := client.New(v.GetString("broker-addr"))
		common.CheckErrf("creating broker client: %v", err)

		ds, err := dshelper.NewMongoTxnDatastore(v.GetString("mongo-uri"), v.GetString("mongo-dbname"))
		common.CheckErrf("creating mongo datastore: %v", err)

		daemonFrequency := v.GetDuration("daemon-frequency")
		retryDelay := v.GetDuration("retry-delay")

		config := service.Config{
			Listener:        listener,
			IpfsClient:      ipfsClient,
			Broker:          broker,
			Datastore:       ds,
			DaemonFrequency: daemonFrequency,
			RetryDelay:      retryDelay,
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
