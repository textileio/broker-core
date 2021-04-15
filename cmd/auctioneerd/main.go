package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	_ "net/http/pprof"

	logging "github.com/ipfs/go-log/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/broker-core/cmd/dealsd/service"
	"github.com/textileio/broker-core/peer"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel/exporters/metric/prometheus"
)

var (
	daemonName = "auctioneerd"
	log        = logging.Logger(daemonName)
	v          = viper.New()
)

var flags = []struct {
	name        string
	defValue    interface{}
	description string
}{
	{
		name:        "repo",
		defValue:    ".auctioneer",
		description: "Repo path",
	},
	{
		name:        "host.multiaddr",
		defValue:    "/ip4/127.0.0.1/tcp/4001",
		description: "Libp2p host listen multiaddr",
	},
	{
		name:        "metrics.addr",
		defValue:    ":9090",
		description: "Prometheus listen address",
	},
}

func init() {
	v.SetEnvPrefix("AUCTIONEER")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	for _, flag := range flags {
		switch defval := flag.defValue.(type) {
		case string:
			rootCmd.Flags().String(flag.name, defval, flag.description)
			_ = v.BindPFlag(flag.name, rootCmd.Flags().Lookup(flag.name))
			v.SetDefault(flag.name, defval)
		default:
			log.Fatalf("unkown flag type: %T", flag)
		}
	}
}

var rootCmd = &cobra.Command{
	Use:   daemonName,
	Short: "todo",
	Long:  `todo`,
	PersistentPreRun: func(c *cobra.Command, args []string) {
		logging.SetAllLoggers(logging.LevelInfo)
		if v.GetBool("log.debug") {
			logging.SetAllLoggers(logging.LevelDebug)
		}
	},
	Run: func(c *cobra.Command, args []string) {
		settings, err := json.MarshalIndent(v.AllSettings(), "", "  ")
		checkErr(err)
		log.Infof("loaded config: %s", string(settings))

		if err := setupInstrumentation(v.GetString("metrics.addr")); err != nil {
			log.Fatalf("booting instrumentation: %s", err)
		}

		config := service.Config{
			Peer: peer.Config{
				RepoPath:      v.GetString("repo"),
				HostMultiaddr: v.GetString("host.multiaddr"),
			},
		}
		serv, err := service.New(config)
		checkErr(err)

		log.Info("Listening for auctions...")
		quit := make(chan os.Signal)
		signal.Notify(quit, os.Interrupt)
		<-quit
		fmt.Println("Gracefully stopping... (press Ctrl+C again to force)")
		if err := serv.Close(); err != nil {
			log.Errorf("closing http endpoint: %s", err)
		}
	},
}

func main() {
	checkErr(rootCmd.Execute())
}

func setupInstrumentation(prometheusAddr string) error {
	exporter, err := prometheus.InstallNewPipeline(prometheus.Config{
		DefaultHistogramBoundaries: []float64{1e-3, 1e-2, 1e-1, 1},
	})
	if err != nil {
		return fmt.Errorf("failed to initialize prometheus exporter %v", err)
	}
	http.HandleFunc("/metrics", exporter.ServeHTTP)
	go func() {
		_ = http.ListenAndServe(prometheusAddr, nil)
	}()

	if err := runtime.Start(runtime.WithMinimumReadMemStatsInterval(time.Second)); err != nil {
		return fmt.Errorf("starting Go runtime metrics: %s", err)
	}

	return nil
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
