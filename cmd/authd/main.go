package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"time"

	_ "net/http/pprof"

	golog "github.com/ipfs/go-log/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/broker-core/cmd/authd/service"
	"github.com/textileio/broker-core/util"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel/exporters/metric/prometheus"
)

var (
	daemonName = "authd"
	log        = golog.Logger(daemonName)
	v          = viper.New()
)

func init() {
	flags := []util.Flag{
		{Name: "grpc.listen.addr", DefValue: ":5000", Description: "gRPC API listen address"},
		{Name: "metrics.addr", DefValue: ":9090", Description: "Prometheus endpoint"},
		{Name: "log.debug", DefValue: false, Description: "Enable debug level logs"},
	}

	util.ConfigureCLI(v, "AUTH", flags, rootCmd)
}

var rootCmd = &cobra.Command{
	Use:   daemonName,
	Short: "authd provides authentication services for the broker",
	Long:  `authd provides authentication services for the broker`,
	PersistentPreRun: func(c *cobra.Command, args []string) {
		golog.SetAllLoggers(golog.LevelInfo)
		if v.GetBool("log.debug") {
			golog.SetAllLoggers(golog.LevelDebug)
		}
	},
	Run: func(c *cobra.Command, args []string) {
		settings, err := json.MarshalIndent(v.AllSettings(), "", "  ")
		checkErr(err)
		log.Infof("loaded config: %s", string(settings))

		if err := setupInstrumentation(v.GetString("metrics.addr")); err != nil {
			log.Fatalf("booting instrumentation: %s", err)
		}

		serv, err := service.New(v.GetString("grpc.listen.addr"))
		checkErr(err)

		quit := make(chan os.Signal, 1)
		signal.Notify(quit, os.Interrupt)
		<-quit
		fmt.Println("Gracefully stopping... (press Ctrl+C again to force)")
		if err := serv.Close(); err != nil {
			log.Errorf("closing service: %s", err)
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
