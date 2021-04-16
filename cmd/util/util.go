package util

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	logging "github.com/ipfs/go-log/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel/exporters/metric/prometheus"
)

var (
	log = logging.Logger("util")
)

// Flag describes a configuration flag.
type Flag struct {
	Name        string
	DefValue    interface{}
	Description string
}

// ConfigureCLI configures a Viper environment with flags and envs.
func ConfigureCLI(v *viper.Viper, envPrefix string, flags []Flag, rootCmd *cobra.Command) {
	v.SetEnvPrefix(envPrefix)
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	for _, flag := range flags {
		switch defval := flag.DefValue.(type) {
		case string:
			rootCmd.Flags().String(flag.Name, defval, flag.Description)
			v.SetDefault(flag.Name, defval)
		case bool:
			rootCmd.Flags().Bool(flag.Name, defval, flag.Description)
			v.SetDefault(flag.Name, defval)
		case time.Duration:
			rootCmd.Flags().Duration(flag.Name, defval, flag.Description)
		default:
			log.Fatalf("unknown flag type: %T", flag)
		}
		if err := v.BindPFlag(flag.Name, rootCmd.Flags().Lookup(flag.Name)); err != nil {
			log.Fatalf("binding flag %s: %s", flag.Name, err)
		}
	}
}

// CheckErr logs a fatal error and terminates.
func CheckErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// WaitForTerminateSignal blocks until the user trigeers a termination.
func WaitForTerminateSignal() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
}

// SetupInstrumentation sets up a Prometheus server.
func SetupInstrumentation(prometheusAddr string) error {
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
