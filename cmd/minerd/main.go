package main

import (
	"encoding/json"
	_ "net/http/pprof"

	golog "github.com/ipfs/go-log/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/common"
	"github.com/textileio/broker-core/cmd/minerd/service"
	"github.com/textileio/broker-core/finalizer"
	"github.com/textileio/broker-core/logging"
	"github.com/textileio/broker-core/marketpeer"
)

var (
	daemonName = "minerd"
	log        = golog.Logger(daemonName)
	v          = viper.New()
)

func init() {
	flags := []common.Flag{
		{Name: "debug", DefValue: false, Description: "Enable debug level logs"},
		{Name: "repo", DefValue: "${HOME}/.miner", Description: "Repo path"},
		{Name: "host-multiaddr", DefValue: "/ip4/0.0.0.0/tcp/4001", Description: "Libp2p host listen multiaddr"},
		{Name: "metrics-addr", DefValue: ":9090", Description: "Prometheus listen address"},
		{
			Name:        "ask-price",
			DefValue:    100000000000,
			Description: "Bid ask price in attoFIL per GiB per epoch; default is 100000000000 or 100 nanoFIL",
		},
		{
			Name:        "deal-duration-min",
			DefValue:    broker.MinDealEpochs,
			Description: "Minimum deal duration to bid on; default is 525600 or ~6 months",
		},
		{
			Name:        "deal-duration-max",
			DefValue:    broker.MaxDealEpochs,
			Description: "Maximum deal duration to bid on; default is 1051200 or ~1 year",
		},
		{
			Name:        "deal-size-min",
			DefValue:    56 * 1024,
			Description: "Minimum deal size to bid on; default is 56KiB",
		},
		{
			Name:        "deal-size-max",
			DefValue:    32 * 1000 * 1000 * 1000,
			Description: "Maximum deal size to bid on; default is 32GB",
		},
	}

	common.ConfigureCLI(v, "MINER", flags, rootCmd)
}

var rootCmd = &cobra.Command{
	Use:   daemonName,
	Short: "minerd is used by a miner to listen for deals from the Broker",
	Long:  "minerd is used by a miner to listen for deals from the Broker",
	PersistentPreRun: func(c *cobra.Command, args []string) {
		ll := golog.LevelInfo
		if v.GetBool("debug") {
			ll = golog.LevelDebug
		}
		err := logging.SetLogLevels(map[string]golog.LogLevel{
			"miner/service": ll,
			"mpeer":         ll,
		})
		common.CheckErrf("setting log levels: %v", err)
	},
	Run: func(c *cobra.Command, args []string) {
		fin := finalizer.NewFinalizer()

		settings, err := json.MarshalIndent(v.AllSettings(), "", "  ")
		common.CheckErrf("marshaling config: %v", err)
		log.Infof("loaded config: %s", string(settings))

		err = common.SetupInstrumentation(v.GetString("metrics.addr"))
		common.CheckErrf("booting instrumentation: %v", err)

		config := service.Config{
			RepoPath: v.GetString("repo"),
			Peer: marketpeer.Config{
				RepoPath:      v.GetString("repo"),
				HostMultiaddr: v.GetString("host-multiaddr"),
			},
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

		serv.Bootstrap()
		err = serv.EnableMDNS(1)
		common.CheckErrf("enabling mdns: %v", err)

		common.HandleInterrupt(func() {
			common.CheckErr(fin.Cleanupf("closing service: %v", nil))
		})
	},
}

func main() {
	common.CheckErr(rootCmd.Execute())
}
