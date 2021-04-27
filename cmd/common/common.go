package common

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel/exporters/metric/prometheus"
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
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

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
			v.SetDefault(flag.Name, defval)
		default:
			log.Fatalf("unknown flag type: %T", flag)
		}
		if err := v.BindPFlag(flag.Name, rootCmd.Flags().Lookup(flag.Name)); err != nil {
			log.Fatalf("binding flag %s: %s", flag.Name, err)
		}
	}
}

// SetupInstrumentation starts a metrics endpoint.
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

// CheckErr ends in a fatal log if err is not nil.
func CheckErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

// CheckErrf ends in a fatal log if err is not nil.
func CheckErrf(format string, err error) {
	if err != nil {
		log.Fatalf(format, err)
	}
}

// HandleInterrupt attempts to cleanup while allowing the user to force stop the process.
func HandleInterrupt(cleanup func()) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	fmt.Println("Gracefully stopping... (press Ctrl+C again to force)")
	cleanup()
	os.Exit(1)
}
