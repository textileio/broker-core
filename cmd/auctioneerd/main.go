package main

import (
	"fmt"
	"net"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"time"

	golog "github.com/ipfs/go-log/v2"
	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/broker-core/cmd/auctioneerd/auctioneer"
	"github.com/textileio/broker-core/cmd/auctioneerd/auctioneer/filclient"
	"github.com/textileio/broker-core/cmd/auctioneerd/service"
	"github.com/textileio/broker-core/cmd/brokerd/client"
	"github.com/textileio/broker-core/cmd/common"
	"github.com/textileio/broker-core/dshelper"
	"github.com/textileio/broker-core/finalizer"
	"github.com/textileio/broker-core/marketpeer"
	"google.golang.org/grpc"
)

var (
	daemonName        = "auctioneerd"
	defaultConfigPath = filepath.Join(os.Getenv("HOME"), "."+daemonName)
	log               = golog.Logger(daemonName)
	v                 = viper.New()
)

func init() {
	configPath := os.Getenv("AUCTIONEER_PATH")
	if configPath == "" {
		configPath = defaultConfigPath
	}
	_ = godotenv.Load(filepath.Join(configPath, ".env"), ".env")

	rootCmd.AddCommand(initCmd, daemonCmd)

	flags := []common.Flag{
		{Name: "rpc-addr", DefValue: ":5000", Description: "gRPC listen address"},
		{Name: "mongo-uri", DefValue: "", Description: "MongoDB URI backing go-datastore"},
		{Name: "mongo-dbname", DefValue: "", Description: "MongoDB database name backing go-datastore"},
		{Name: "broker-addr", DefValue: "", Description: "Broker API address"},
		{Name: "auction-duration", DefValue: time.Second * 10, Description: "Auction duration"},
		{Name: "auction-attempts", DefValue: 10, Description: "Number of attempts an auction will run before failing"},
		{Name: "lotus-gateway-url", DefValue: "https://api.node.glif.io", Description: "Lotus gateway URL"},
		{Name: "metrics-addr", DefValue: ":9090", Description: "Prometheus listen address"},
		{Name: "log-debug", DefValue: false, Description: "Enable debug level logging"},
		{Name: "log-json", DefValue: false, Description: "Enable structured logging"},
	}
	flags = append(flags, marketpeer.Flags...)

	cobra.OnInitialize(func() {
		v.SetConfigType("json")
		v.SetConfigName("config")
		v.AddConfigPath(os.Getenv("AUCTIONEER_PATH"))
		v.AddConfigPath(defaultConfigPath)
		_ = v.ReadInConfig()
	})

	common.ConfigureCLI(v, "AUCTIONEER", flags, rootCmd.PersistentFlags())
}

var rootCmd = &cobra.Command{
	Use:   daemonName,
	Short: "Auctioneer runs Filecoin storage deal auctions for a deal broker",
	Long: `Auctioneer runs Filecoin storage deal auctions for a deal broker.

To get started, run 'auctioneerd init'. 
`,
	Args: cobra.ExactArgs(0),
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initializes auctioneer configuration files",
	Long: `Initializes auctioneer configuration files and generates a new keypair.

auctioneer uses a repository in the local file system. By default, the repo is
located at ~/.auctioneerd. To change the repo location, set the $AUCTIONEER_PATH
environment variable:

    export AUCTIONEER_PATH=/path/to/auctioneerdrepo
`,
	Args: cobra.ExactArgs(0),
	Run: func(c *cobra.Command, args []string) {
		path, err := marketpeer.WriteConfig(v, "AUCTIONEER_PATH", defaultConfigPath)
		common.CheckErrf("writing config: %v", err)
		fmt.Printf("Initialized configuration file: %s\n", path)
	},
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run a network-connected storage deal auctioneer for a broker",
	Long:  "Run a network-connected storage deal auctioneer for a broker.",
	PersistentPreRun: func(c *cobra.Command, args []string) {
		common.ExpandEnvVars(v, v.AllSettings())
		err := common.ConfigureLogging(v, []string{
			daemonName,
			"auctioneer",
			"auctioneer/queue",
			"auctioneer/service",
			"mpeer",
			"mpeer/pubsub",
		})
		common.CheckErrf("setting log levels: %v", err)
	},
	Run: func(c *cobra.Command, args []string) {
		pconfig, err := marketpeer.GetConfig(v, "AUCTIONEER_PATH", defaultConfigPath, true)
		common.CheckErrf("getting peer config: %v", err)

		settings, err := marketpeer.MarshalConfig(v, !v.GetBool("log-json"))
		common.CheckErrf("marshaling config: %v", err)
		log.Infof("loaded config: %s", string(settings))

		err = common.SetupInstrumentation(v.GetString("metrics-addr"))
		common.CheckErrf("booting instrumentation: %v", err)

		listener, err := net.Listen("tcp", v.GetString("rpc-addr"))
		common.CheckErrf("creating listener: %v", err)

		store, err := dshelper.NewMongoTxnDatastore(v.GetString("mongo-uri"), v.GetString("mongo-dbname"))
		common.CheckErrf("creating datastore: %v", err)

		fin := finalizer.NewFinalizer()
		broker, err := client.New(v.GetString("broker-addr"), grpc.WithInsecure())
		common.CheckErrf("dialing broker: %v", err)
		fin.Add(broker)

		fc, err := filclient.New(v.GetString("lotus-gateway-url"), v.GetBool("fake-mode"))
		common.CheckErrf("creating chain client: %v", err)
		fin.Add(fc)

		config := service.Config{
			Listener: listener,
			Peer:     pconfig,
			Auction: auctioneer.AuctionConfig{
				Duration: v.GetDuration("auction-duration"),
				Attempts: v.GetUint32("auction-attempts"),
			},
		}
		serv, err := service.New(config, store, broker, fc)
		common.CheckErrf("starting service: %v", err)
		fin.Add(serv)

		err = serv.Start(true)
		common.CheckErrf("creating deal auction feed: %v", err)

		common.HandleInterrupt(func() {
			common.CheckErr(fin.Cleanupf("closing service: %v", nil))
		})
	},
}

func main() {
	common.CheckErr(rootCmd.Execute())
}
