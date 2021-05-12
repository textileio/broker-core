package main

import (
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
	flags := []common.Flag{
		{Name: "rpc-addr", DefValue: ":5000", Description: "gRPC listen address"},
		{Name: "mongo-uri", DefValue: "", Description: "MongoDB URI backing go-datastore"},
		{Name: "mongo-dbname", DefValue: "", Description: "MongoDB database name backing go-datastore"},
		{Name: "broker-addr", DefValue: "", Description: "Broker API address"},
		{Name: "auction-duration", DefValue: time.Second * 10, Description: "Auction duration"},
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
		if err := v.ReadInConfig(); err != nil {
			common.CheckErrf("reading configuration: %s", err)
		}
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
			path, err := marketpeer.WriteConfig(v, "AUCTIONEER_PATH", defaultConfigPath)
			common.CheckErrf("writing config: %v", err)
			fmt.Printf("Initialized configuration file: %s\n", path)
		}

		fin := finalizer.NewFinalizer()

		settings, err := marketpeer.MarshalConfig(v)
		common.CheckErrf("marshaling config: %v", err)
		log.Infof("loaded config: %s", string(settings))

		err = common.SetupInstrumentation(v.GetString("metrics.addr"))
		common.CheckErrf("booting instrumentation: %v", err)

		pconfig, err := marketpeer.GetConfig(v, true)
		common.CheckErrf("getting peer config: %v", err)

		listener, err := net.Listen("tcp", v.GetString("rpc-addr"))
		common.CheckErrf("creating listener: %v", err)

		store, err := dshelper.NewMongoTxnDatastore(v.GetString("mongo-uri"), v.GetString("mongo-dbname"))
		common.CheckErrf("creating datastore: %v", err)

		broker, err := client.New(v.GetString("broker-addr"), grpc.WithInsecure())
		common.CheckErrf("dialing broker: %v", err)
		fin.Add(broker)

		ch, err := filclient.New(v.GetString("lotus-gateway-url"))
		common.CheckErrf("creating chain client: %v", err)
		fin.Add(ch)

		config := service.Config{
			Listener: listener,
			Peer:     pconfig,
			Auction: auctioneer.AuctionConfig{
				Duration: v.GetDuration("auction-duration"),
			},
		}
		serv, err := service.New(config, store, broker, ch)
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
