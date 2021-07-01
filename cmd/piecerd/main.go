package main

import (
	"encoding/json"
	"net"
	_ "net/http/pprof"
	"time"

	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/bidbot/lib/common"
	"github.com/textileio/bidbot/lib/dshelper"
	"github.com/textileio/broker-core/cmd/brokerd/client"
	"github.com/textileio/broker-core/cmd/piecerd/service"
	logging "github.com/textileio/go-log/v2"
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
		{Name: "ipfs-multiaddrs", DefValue: []string{}, Description: "IPFS multiaddresses"},
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

		broker, err := client.New(v.GetString("broker-addr"))
		common.CheckErrf("creating broker client: %v", err)

		ds, err := dshelper.NewMongoTxnDatastore(v.GetString("mongo-uri"), v.GetString("mongo-dbname"))
		common.CheckErrf("creating mongo datastore: %v", err)

		daemonFrequency := v.GetDuration("daemon-frequency")
		retryDelay := v.GetDuration("retry-delay")

		ipfsMultiaddrsStr := common.ParseStringSlice(v, "ipfs-multiaddrs")
		ipfsMultiaddrs := make([]multiaddr.Multiaddr, len(ipfsMultiaddrsStr))
		for i, maStr := range ipfsMultiaddrsStr {
			ma, err := multiaddr.NewMultiaddr(maStr)
			common.CheckErrf("parsing multiaddress %s: %s", err)
			ipfsMultiaddrs[i] = ma
		}

		config := service.Config{
			Listener:        listener,
			IpfsMultiaddrs:  ipfsMultiaddrs,
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
