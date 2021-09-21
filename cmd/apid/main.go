package main

import (
	_ "net/http/pprof"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/broker-core/cmd/apid/service"
	"github.com/textileio/broker-core/common"
	"github.com/textileio/broker-core/msgbroker/gpubsub"
	"github.com/textileio/cli"
	logging "github.com/textileio/go-log/v2"
)

var (
	daemonName = "apid"
	log        = logging.Logger(daemonName)
	v          = viper.New()
)

func init() {
	flags := []cli.Flag{
		{Name: "postgres-uri", DefValue: "", Description: "PostgreSQL URI"},
		{Name: "gpubsub-project-id", DefValue: "", Description: "Google PubSub project id"},
		{Name: "gpubsub-api-key", DefValue: "", Description: "Google PubSub API key"},
		{Name: "msgbroker-topic-prefix", DefValue: "", Description: "Topic prefix to use for msg broker topics"},
		{Name: "http-addr", DefValue: ":8889", Description: "HTTP API listen address"},
		{Name: "metrics-addr", DefValue: ":9090", Description: "Prometheus listen address"},
		{Name: "log-debug", DefValue: false, Description: "Enable debug level logging"},
		{Name: "log-json", DefValue: false, Description: "Enable structured logging"},
	}

	cli.ConfigureCLI(v, "API", flags, rootCmd.Flags())
}

var rootCmd = &cobra.Command{
	Use:   daemonName,
	Short: "apid is a service that aggragates data from other subsystems to provide a common API to the outside world",
	PersistentPreRun: func(c *cobra.Command, args []string) {
		cli.ExpandEnvVars(v, v.AllSettings())
		err := cli.ConfigureLogging(v, nil)
		cli.CheckErrf("setting log levels: %v", err)
	},
	Run: func(c *cobra.Command, args []string) {
		settings, err := cli.MarshalConfig(v, !v.GetBool("log-json"), "gpubsub-api-key", "postgres-uri")
		cli.CheckErr(err)
		log.Infof("loaded config: %s", string(settings))

		if err := common.SetupInstrumentation(v.GetString("metrics-addr")); err != nil {
			log.Fatalf("booting instrumentation: %s", err)
		}

		projectID := v.GetString("gpubsub-project-id")
		apiKey := v.GetString("gpubsub-api-key")
		topicPrefix := v.GetString("msgbroker-topic-prefix")
		mb, err := gpubsub.New(projectID, apiKey, topicPrefix, "apid")
		cli.CheckErr(err)

		_, err = service.New(mb, v.GetString("http-addr"), v.GetString("postgres-uri"))
		cli.CheckErr(err)

		log.Info("listening to message broker messages...")

		cli.HandleInterrupt(func() {
			if err := mb.Close(); err != nil {
				log.Errorf("closing message broker: %s", err)
			}
		})
	},
}

func main() {
	cli.CheckErr(rootCmd.Execute())
}
