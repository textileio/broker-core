package main

// TODO: Use mongo for auction persistence.

import (
	"encoding/json"
	"net"
	_ "net/http/pprof"
	"time"

	golog "github.com/ipfs/go-log/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/broker-core/cmd/auctioneerd/auctioneer"
	"github.com/textileio/broker-core/cmd/auctioneerd/service"
	"github.com/textileio/broker-core/cmd/brokerd/client"
	"github.com/textileio/broker-core/cmd/common"
	"github.com/textileio/broker-core/finalizer"
	"github.com/textileio/broker-core/logging"
	"github.com/textileio/broker-core/marketpeer"
	"google.golang.org/grpc"
)

var (
	daemonName = "auctioneerd"
	log        = golog.Logger(daemonName)
	v          = viper.New()
)

func init() {
	flags := []common.Flag{
		{Name: "debug", DefValue: false, Description: "Enable debug level logs"},
		{Name: "repo", DefValue: ".auctioneer", Description: "Repo path"},
		{Name: "rpc-addr", DefValue: ":5000", Description: "gRPC listen address"},
		{Name: "host-multiaddr", DefValue: "/ip4/0.0.0.0/tcp/4001", Description: "Libp2p host listen multiaddr"},
		{Name: "broker-addr", DefValue: "", Description: "Broker API address"},
		{Name: "metrics-addr", DefValue: ":9090", Description: "Prometheus listen address"},
		{Name: "auction-duration", DefValue: time.Second * 10, Description: "Auction duration; default is 10s"},
	}

	common.ConfigureCLI(v, "AUCTIONEER", flags, rootCmd)
}

var rootCmd = &cobra.Command{
	Use:   daemonName,
	Short: "auctioneerd handles deal auctions for the Broker",
	Long:  "auctioneerd handles deal auctions for the Broker",
	PersistentPreRun: func(c *cobra.Command, args []string) {
		ll := golog.LevelInfo
		if v.GetBool("debug") {
			ll = golog.LevelDebug
		}
		err := logging.SetLogLevels(map[string]golog.LogLevel{
			"auctioneer":         ll,
			"auctioneer/queue":   ll,
			"auctioneer/service": ll,
			"mpeer":              ll,
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

		listener, err := net.Listen("tcp", v.GetString("rpc-addr"))
		common.CheckErrf("creating listener: %v", err)
		fin.Add(listener)

		broker, err := client.New(v.GetString("broker-addr"), grpc.WithInsecure())
		common.CheckErrf("dialing broker: %v", err)
		fin.Add(broker)

		config := service.Config{
			RepoPath: v.GetString("repo"),
			Listener: listener,
			Peer: marketpeer.Config{
				RepoPath:      v.GetString("repo"),
				HostMultiaddr: v.GetString("host-multiaddr"),
			},
			Auction: auctioneer.AuctionConfig{
				Duration: v.GetDuration("auction-duration"),
			},
		}
		serv, err := service.New(config, broker)
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
