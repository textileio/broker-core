package main

// TODO: Use mongo for auction persistence.

import (
	"encoding/json"
	"fmt"
	"net"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"time"

	golog "github.com/ipfs/go-log/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/broker-core/cmd/auctioneerd/auctioneer"
	"github.com/textileio/broker-core/cmd/auctioneerd/service"
	"github.com/textileio/broker-core/cmd/brokerd/client"
	"github.com/textileio/broker-core/cmd/common"
	"github.com/textileio/broker-core/finalizer"
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
		{Name: "rpc-addr", DefValue: ":5000", Description: "gRPC listen address"},
		{Name: "broker-addr", DefValue: "", Description: "Broker API address"},
		{Name: "auction-duration", DefValue: time.Second * 10, Description: "Auction duration"},
		{Name: "metrics-addr", DefValue: ":9090", Description: "Prometheus listen address"},
		{Name: "log-debug", DefValue: false, Description: "Enable debug level logging"},
		{Name: "log-json", DefValue: false, Description: "Enable structured logging"},
	}
	flags = append(flags, marketpeer.Flags...)

	cobra.OnInitialize(func() {
		v.SetConfigType("json")
		v.SetConfigName("config")
		v.AddConfigPath(os.Getenv("AUCTIONEER_PATH"))
		v.AddConfigPath(filepath.Join(os.Getenv("HOME"), daemonName))
		_ = v.ReadInConfig()
	})

	common.ConfigureCLI(v, "AUCTIONEER", flags, rootCmd.Flags())
}

var rootCmd = &cobra.Command{
	Use:   daemonName,
	Short: "auctioneerd handles deal auctions for the Broker",
	Long:  "auctioneerd handles deal auctions for the Broker",
	PersistentPreRun: func(c *cobra.Command, args []string) {
		common.ExpandEnvVars(v, v.AllSettings())
		err := common.ConfigureLogging(v, []string{
			"auctioneerd",
			"auctioneer",
			"auctioneer/queue",
			"auctioneer/service",
			"mpeer",
		})
		common.CheckErrf("setting log levels: %v", err)
	},
	Run: func(c *cobra.Command, args []string) {
		if v.ConfigFileUsed() == "" {
			path, err := marketpeer.WriteConfig(v, "AUCTIONEER_PATH", daemonName)
			common.CheckErrf("writing config: %v", err)
			fmt.Printf("Initialized configuration file: %s\n", path)
		}

		fin := finalizer.NewFinalizer()

		settings, err := json.MarshalIndent(v.AllSettings(), "", "  ")
		common.CheckErrf("marshaling config: %v", err)
		log.Infof("loaded config: %s", string(settings))

		err = common.SetupInstrumentation(v.GetString("metrics.addr"))
		common.CheckErrf("booting instrumentation: %v", err)

		pconfig, err := marketpeer.GetConfig(v, true)
		common.CheckErrf("getting peer config: %v", err)

		listener, err := net.Listen("tcp", v.GetString("rpc-addr"))
		common.CheckErrf("creating listener: %v", err)
		fin.Add(listener)

		broker, err := client.New(v.GetString("broker-addr"), grpc.WithInsecure())
		common.CheckErrf("dialing broker: %v", err)
		fin.Add(broker)

		config := service.Config{
			RepoPath: pconfig.RepoPath, // TODO: Remove in favor of mongo
			Listener: listener,
			Peer:     pconfig,
			Auction: auctioneer.AuctionConfig{
				Duration: v.GetDuration("auction-duration"),
			},
		}
		serv, err := service.New(config, broker)
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
