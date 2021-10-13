package main

import (
	"encoding/json"
	"fmt"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"time"

	"github.com/joho/godotenv"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/bidbot/lib/filclient"
	"github.com/textileio/bidbot/lib/peerflags"
	"github.com/textileio/broker-core/cmd/auctioneerd/auctioneer"
	"github.com/textileio/broker-core/cmd/auctioneerd/metrics"
	"github.com/textileio/broker-core/cmd/auctioneerd/service"
	"github.com/textileio/broker-core/cmd/brokerd/client"
	"github.com/textileio/broker-core/common"
	"github.com/textileio/broker-core/msgbroker/gpubsub"
	"github.com/textileio/cli"
	"github.com/textileio/go-libp2p-pubsub-rpc/finalizer"
	golog "github.com/textileio/go-log/v2"
	"google.golang.org/grpc"
)

var (
	daemonName        = "auctioneerd"
	defaultConfigPath = filepath.Join(os.Getenv("HOME"), "."+daemonName)
	log               = golog.Logger(daemonName)
	v                 = viper.New()
)

func init() {
	_ = godotenv.Load(".env")
	configPath := os.Getenv("AUCTIONEER_PATH")
	if configPath == "" {
		configPath = defaultConfigPath
	}
	_ = godotenv.Load(filepath.Join(configPath, ".env"))

	rootCmd.AddCommand(initCmd, daemonCmd)

	flags := []cli.Flag{
		{Name: "postgres-uri", DefValue: "", Description: "Postgres database URI"},
		{Name: "broker-addr", DefValue: "", Description: "Broker API address"},
		{Name: "auction-duration", DefValue: time.Second * 30, Description: "Auction duration"},
		{Name: "lotus-gateway-url", DefValue: "https://api.node.glif.io", Description: "Lotus gateway URL"},
		{Name: "gpubsub-project-id", DefValue: "", Description: "Google PubSub project id"},
		{Name: "gpubsub-api-key", DefValue: "", Description: "Google PubSub API key"},
		{Name: "msgbroker-topic-prefix", DefValue: "", Description: "Topic prefix to use for msg broker topics"},
		{Name: "record-bidbot-events", DefValue: false, Description: "Record bidbot events to database"},

		{Name: "metrics-addr", DefValue: ":9090", Description: "Prometheus listen address"},
		{Name: "log-debug", DefValue: false, Description: "Enable debug level logging"},
		{Name: "log-json", DefValue: false, Description: "Enable structured logging"},
	}
	flags = append(flags, peerflags.Flags...)

	cobra.OnInitialize(func() {
		v.SetConfigType("json")
		v.SetConfigName("config")
		v.AddConfigPath(os.Getenv("AUCTIONEER_PATH"))
		v.AddConfigPath(defaultConfigPath)
		_ = v.ReadInConfig()
	})

	cli.ConfigureCLI(v, "AUCTIONEER", flags, rootCmd.PersistentFlags())
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
		path, err := peerflags.WriteConfig(v, "AUCTIONEER_PATH", defaultConfigPath)
		cli.CheckErrf("writing config: %v", err)
		fmt.Printf("Initialized configuration file: %s\n", path)
	},
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run a network-connected storage deal auctioneer for a broker",
	Long:  "Run a network-connected storage deal auctioneer for a broker.",
	PersistentPreRun: func(c *cobra.Command, args []string) {
		cli.ExpandEnvVars(v, v.AllSettings())
		err := cli.ConfigureLogging(v, []string{
			daemonName,
			"auctioneer",
			"auctioneer/queue",
			"auctioneer/service",
			"psrpc",
			"psrpc/mdns",
			"psrpc/peer",
		})
		cli.CheckErrf("setting log levels: %v", err)
	},
	Run: func(c *cobra.Command, args []string) {
		pconfig, err := peerflags.GetConfig(v, "AUCTIONEER_PATH", defaultConfigPath, true)
		cli.CheckErrf("getting peer config: %v", err)

		settings, err := cli.MarshalConfig(v, !v.GetBool("log-json"),
			"private-key", "wallet-addr-sig", "gpubsub-api-key", "postgres-uri")
		cli.CheckErrf("marshaling config: %v", err)
		log.Infof("loaded config: %s", string(settings))

		err = common.SetupInstrumentation(v.GetString("metrics-addr"))
		cli.CheckErrf("booting instrumentation: %v", err)

		fin := finalizer.NewFinalizer()
		broker, err := client.New(v.GetString("broker-addr"), grpc.WithInsecure())
		cli.CheckErrf("dialing broker: %v", err)
		fin.Add(broker)

		fc, err := filclient.New(v.GetString("lotus-gateway-url"), v.GetBool("fake-mode"))
		cli.CheckErrf("creating chain client: %v", err)
		fin.Add(fc)

		config := service.Config{
			Peer: pconfig,
			Auction: auctioneer.AuctionConfig{
				Duration: v.GetDuration("auction-duration"),
			},
			PostgresURI:        v.GetString("postgres-uri"),
			RecordBidbotEvents: v.GetBool("record-bidbot-events"),
		}

		projectID := v.GetString("gpubsub-project-id")
		apiKey := v.GetString("gpubsub-api-key")
		topicPrefix := v.GetString("msgbroker-topic-prefix")
		mb, err := gpubsub.NewMetered(projectID, apiKey, topicPrefix, "auctioneerd", metrics.Meter)
		cli.CheckErrf("creating google pubsub client: %s", err)

		serv, err := service.New(config, mb, fc)
		cli.CheckErrf("starting service: %v", err)
		fin.Add(serv)
		err = serv.Start(true)
		cli.CheckErrf("creating deal auction feed: %v", err)

		info, err := serv.PeerInfo()
		cli.CheckErrf("getting peer information: %v", err)
		b, err := json.MarshalIndent(info, "", "\t")
		cli.CheckErrf("marshaling peer information: %v", err)
		log.Infof("peer information:: %s", string(b))

		fin.Add(mb)

		cli.HandleInterrupt(func() {
			cli.CheckErr(fin.Cleanupf("closing service: %v", nil))
		})
	},
}

func main() {
	cli.CheckErr(rootCmd.Execute())
}
