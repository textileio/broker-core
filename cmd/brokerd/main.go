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
	"github.com/textileio/broker-core/cmd/brokerd/service"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel/exporters/metric/prometheus"
)

var (
	daemonName = "brokerd"
	log        = logging.Logger(daemonName)
	v          = viper.New()
)

var flags = []struct {
	name        string
	defValue    interface{}
	description string
}{
	{name: "metrics.addr", defValue: ":9090", description: "Prometheus endpoint"},
	{name: "log.debug", defValue: false, description: "Enable debug level logs"},

	{name: "grpc.listen.addr", defValue: ":5000", description: "gRPC API listen address"},

	{name: "mongo.uri", defValue: "", description: "MongoDB URI backing go-datastore"},
	{name: "mongo.dbname", defValue: "", description: "MongoDB database name backing go-datastore"},
}

func init() {
	v.SetEnvPrefix("BROKER")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	for _, flag := range flags {
		switch defval := flag.defValue.(type) {
		case string:
			rootCmd.Flags().String(flag.name, defval, flag.description)
			if err := v.BindPFlag(flag.name, rootCmd.Flags().Lookup(flag.name)); err != nil {
				log.Fatalf("binding flag %s: %s", flag.name, err)
			}
			v.SetDefault(flag.name, defval)
		default:
			log.Fatalf("unknown flag type: %T", flag)
		}
	}
}

var rootCmd = &cobra.Command{
	Use:   daemonName,
	Short: "brokerd is a broker to store data in Filecoin",
	Long:  `brokerd is a broker to store data in Filecoin`,
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

		serviceConfig := service.Config{
			GrpcListenAddress: v.GetString("grpc.listen.addr"),

			MongoURI:    v.GetString("mongo.uri"),
			MongoDBName: v.GetString("mongo.dbname"),
		}
		serv, err := service.New(serviceConfig)
		checkErr(err)

		log.Info("Listening to requests...")
		quit := make(chan os.Signal, 1)
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
