package common

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	logger "github.com/ipfs/go-log/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/broker-core/logging"
	"go.opentelemetry.io/contrib/instrumentation/runtime"
	"go.opentelemetry.io/otel/exporters/metric/prometheus"
)

// Flag describes a configuration flag.
type Flag struct {
	Name        string
	DefValue    interface{}
	Description string
	Repeatable  bool
}

// ConfigureCLI configures a Viper environment with flags and envs.
func ConfigureCLI(v *viper.Viper, envPrefix string, flags []Flag, rootCmd *cobra.Command) {
	v.SetEnvPrefix(envPrefix)
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_", "-", "_"))

	for _, flag := range flags {
		switch defval := flag.DefValue.(type) {
		case string:
			if flag.Repeatable {
				rootCmd.Flags().StringSlice(flag.Name, []string{defval}, flag.Description)
			} else {
				rootCmd.Flags().String(flag.Name, defval, flag.Description)
			}
			v.SetDefault(flag.Name, defval)
		case bool:
			rootCmd.Flags().Bool(flag.Name, defval, flag.Description)
			v.SetDefault(flag.Name, defval)
		case int:
			rootCmd.Flags().Int(flag.Name, defval, flag.Description)
			v.SetDefault(flag.Name, defval)
		case int64:
			rootCmd.Flags().Int64(flag.Name, defval, flag.Description)
			v.SetDefault(flag.Name, defval)
		case uint64:
			rootCmd.Flags().Uint64(flag.Name, defval, flag.Description)
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

// ExpandEnvVars expands env vars present in the config.
func ExpandEnvVars(v *viper.Viper, settings map[string]interface{}) {
	for name, val := range settings {
		if str, ok := val.(string); ok {
			v.Set(name, os.ExpandEnv(str))
		}
	}
}

// ConfigureLogging configures the default logger with the right setup depending flag/envs.
// If logLevels is not nil, only logLevels values will be configured to Info/Debug depending
// on viper flags. if logLevels is nil, all sub-logs will be configured.
func ConfigureLogging(v *viper.Viper, logLevels []string) error {
	logJSON := v.GetBool("log-json")
	if logJSON {
		logger.SetupLogging(logger.Config{
			Format: logger.JSONOutput,
			Stderr: false,
			Stdout: true,
		})
	}

	logLevel := logger.LevelInfo
	if v.GetBool("log-debug") {
		logLevel = logger.LevelDebug
	}

	if len(logLevels) == 0 {
		logger.SetAllLoggers(logLevel)
		return nil
	}

	mapLevel := make(map[string]logger.LogLevel, len(logLevels))
	for i := range logLevels {
		mapLevel[logLevels[i]] = logLevel
	}

	if err := logging.SetLogLevels(mapLevel); err != nil {
		return fmt.Errorf("set log levels: %s", err)
	}
	return nil
}

// ParseStringSlice returns a single slice of values that may have been set by either repeating
// a flag or using comma seperation in a single flag.
// This is used to enable repeated flags as well as env vars that can't be repeated.
func ParseStringSlice(v *viper.Viper, key string) []string {
	var vals []string
	for _, val := range v.GetStringSlice(key) {
		parts := strings.Split(val, ",")
		vals = append(vals, parts...)
	}
	return vals
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
