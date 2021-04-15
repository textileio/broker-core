package util

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
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

type Flag struct {
	Name        string
	DefValue    interface{}
	Description string
}

func BindFlags(flags []Flag, cmd *cobra.Command, v *viper.Viper) {
	for _, flag := range flags {
		switch defval := flag.DefValue.(type) {
		case string:
			cmd.Flags().String(flag.Name, defval, flag.Description)
			if err := v.BindPFlag(flag.Name, cmd.Flags().Lookup(flag.Name)); err != nil {
				log.Fatalf("binding flag %s: %s", flag.Name, err)
			}
			v.SetDefault(flag.Name, defval)
		default:
			log.Fatalf("unknown flag type: %T", flag)
		}
	}
}

func CheckErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func WaitUntilTerminated() {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit
}

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
