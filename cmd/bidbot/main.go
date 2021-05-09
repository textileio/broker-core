package main

import (
	"fmt"
	_ "net/http/pprof"
	"os"
	"path/filepath"

	golog "github.com/ipfs/go-log/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/bidbot/service"
	"github.com/textileio/broker-core/cmd/common"
	"github.com/textileio/broker-core/finalizer"
	"github.com/textileio/broker-core/marketpeer"
)

var (
	cliName           = "bidbot"
	defaultConfigPath = filepath.Join(os.Getenv("HOME"), "."+cliName)
	log               = golog.Logger(cliName)
	v                 = viper.New()
)

func init() {
	rootCmd.AddCommand(initCmd, daemonCmd)

	flags := []common.Flag{
		{
			Name:        "ask-price",
			DefValue:    100000000000,
			Description: "Bid ask price in attoFIL per GiB per epoch; default is 100 nanoFIL",
		},
		{
			Name:        "deal-duration-min",
			DefValue:    broker.MinDealEpochs,
			Description: "Minimum deal duration to bid on in epochs; default is ~6 months",
		},
		{
			Name:        "deal-duration-max",
			DefValue:    broker.MaxDealEpochs,
			Description: "Maximum deal duration to bid on in epochs; default is ~1 year",
		},
		{
			Name:        "deal-size-min",
			DefValue:    56 * 1024,
			Description: "Minimum deal size to bid on in bytes",
		},
		{
			Name:        "deal-size-max",
			DefValue:    32 * 1000 * 1000 * 1000,
			Description: "Maximum deal size to bid on in bytes",
		},
		{Name: "metrics-addr", DefValue: ":9090", Description: "Prometheus listen address"},
		{Name: "log-debug", DefValue: false, Description: "Enable debug level log"},
		{Name: "log-json", DefValue: false, Description: "Enable structured logging"},
	}
	flags = append(flags, marketpeer.Flags...)

	cobra.OnInitialize(func() {
		v.SetConfigType("json")
		v.SetConfigName("config")
		v.AddConfigPath(os.Getenv("BIDBOT_PATH"))
		v.AddConfigPath(defaultConfigPath)
		_ = v.ReadInConfig()
	})

	common.ConfigureCLI(v, "BIDBOT", flags, rootCmd.PersistentFlags())
}

var rootCmd = &cobra.Command{
	Use:   cliName,
	Short: "bidbot listens for Filecoin storage deal auctions from deal brokers",
	Long: `bidbot listens for Filecoin storage deal auctions from deal brokers.

bidbot will automatically bid on storage deals that pass configured filters at
the configured prices.

To get started, run 'bidbot init' followed by 'bidbot daemon'. 
`,
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initializes bidbot configuration file",
	Long: `Initializes bidbot configuration file and generates a new keypair.

bidbot uses a repository in the local file system. By default, the repo is
located at ~/.bidbot. To change the repo location, set the $BIDBOT_PATH
environment variable:

    export BIDBOT_PATH=/path/to/bidbotrepo
`,
	Run: func(c *cobra.Command, args []string) {
		path, err := marketpeer.WriteConfig(v, "BIDBOT_PATH", defaultConfigPath)
		common.CheckErrf("writing config: %v", err)

		settings, err := marketpeer.MarshalConfig(v)
		common.CheckErrf("marshaling config: %v", err)
		fmt.Println(string(settings))

		fmt.Printf("Initialized configuration file: %s\n", path)
	},
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run a network-connected bidding bot",
	Long:  "Run a network-connected bidding bot that listens for and bids on storage deal auctions.",
	PersistentPreRun: func(c *cobra.Command, args []string) {
		common.ExpandEnvVars(v, v.AllSettings())
		err := common.ConfigureLogging(v, []string{
			"bidbot",
			"bidbot/service",
			"mpeer",
			"pubsub",
		})
		common.CheckErrf("setting log levels: %v", err)
	},
	Run: func(c *cobra.Command, args []string) {
		if v.ConfigFileUsed() == "" {
			fmt.Printf("Configuration file not found. Run '%s init'.", cliName)
			return
		}

		fin := finalizer.NewFinalizer()

		settings, err := marketpeer.MarshalConfig(v)
		common.CheckErrf("marshaling config: %v", err)
		log.Infof("loaded config: %s", string(settings))

		err = common.SetupInstrumentation(v.GetString("metrics.addr"))
		common.CheckErrf("booting instrumentation: %v", err)

		pconfig, err := marketpeer.GetConfig(v, false)
		common.CheckErrf("getting peer config: %v", err)

		config := service.Config{
			RepoPath: pconfig.RepoPath,
			Peer:     pconfig,
			BidParams: service.BidParams{
				AskPrice: v.GetInt64("ask-price"),
			},
			AuctionFilters: service.AuctionFilters{
				DealDuration: service.MinMaxFilter{
					Min: v.GetUint64("deal-duration-min"),
					Max: v.GetUint64("deal-duration-max"),
				},
				DealSize: service.MinMaxFilter{
					Min: v.GetUint64("deal-size-min"),
					Max: v.GetUint64("deal-size-max"),
				},
			},
		}
		serv, err := service.New(config)
		common.CheckErrf("starting service: %v", err)
		fin.Add(serv)

		err = serv.Subscribe(true)
		common.CheckErrf("subscribing to deal auction feed: %v", err)

		common.HandleInterrupt(func() {
			common.CheckErr(fin.Cleanupf("closing service: %v", nil))
		})
	},
}

func main() {
	common.CheckErr(rootCmd.Execute())
}
