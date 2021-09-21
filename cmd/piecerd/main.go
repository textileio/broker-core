package main

import (
	_ "net/http/pprof"
	"time"

	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/broker-core/cmd/piecerd/metrics"
	"github.com/textileio/broker-core/cmd/piecerd/service"
	"github.com/textileio/broker-core/common"
	"github.com/textileio/broker-core/msgbroker/gpubsub"
	"github.com/textileio/cli"
	logging "github.com/textileio/go-log/v2"
)

var (
	daemonName = "piecerd"
	log        = logging.Logger(daemonName)
	v          = viper.New()
)

func init() {
	flags := []cli.Flag{
		{Name: "postgres-uri", DefValue: "", Description: "PostgreSQL URI"},
		{Name: "ipfs-multiaddrs", DefValue: []string{}, Description: "IPFS multiaddresses"},
		{Name: "daemon-frequency", DefValue: time.Second * 30, Description: "Daemon frequency to process pending data"},
		{Name: "retry-delay", DefValue: time.Second * 20, Description: "Delay for reprocessing items"},
		{Name: "gpubsub-project-id", DefValue: "", Description: "Google PubSub project id"},
		{Name: "gpubsub-api-key", DefValue: "", Description: "Google PubSub API key"},
		{Name: "msgbroker-topic-prefix", DefValue: "", Description: "Topic prefix to use for msg broker topics"},
		{Name: "pad-to-size", DefValue: uint64(0),
			Description: "If isn't zero, it pads smaller pieces to the defined size. Must be a power of 2 smaller than 32GiB"},
		{Name: "metrics-addr", DefValue: ":9090", Description: "Prometheus listen address"},
		{Name: "log-debug", DefValue: false, Description: "Enable debug level logging"},
		{Name: "log-json", DefValue: false, Description: "Enable structured logging"},
	}

	cli.ConfigureCLI(v, "PIECER", flags, rootCmd.Flags())
}

var rootCmd = &cobra.Command{
	Use:   daemonName,
	Short: "piecerd handles prepares batched data for Filecoin",
	Long:  "piecerd handles prepares batched data for Filecoin",
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

		padToSize := v.GetUint64("pad-to-size")
		daemonFrequency := v.GetDuration("daemon-frequency")
		retryDelay := v.GetDuration("retry-delay")

		ipfsMultiaddrsStr := cli.ParseStringSlice(v, "ipfs-multiaddrs")
		ipfsMultiaddrs := make([]multiaddr.Multiaddr, len(ipfsMultiaddrsStr))
		for i, maStr := range ipfsMultiaddrsStr {
			ma, err := multiaddr.NewMultiaddr(maStr)
			cli.CheckErrf("parsing multiaddress %s: %s", err)
			ipfsMultiaddrs[i] = ma
		}

		projectID := v.GetString("gpubsub-project-id")
		apiKey := v.GetString("gpubsub-api-key")
		topicPrefix := v.GetString("msgbroker-topic-prefix")
		mb, err := gpubsub.NewMetered(projectID, apiKey, topicPrefix, "piecerd", metrics.Meter)
		cli.CheckErrf("creating google pubsub client: %s", err)

		config := service.Config{
			IpfsMultiaddrs:  ipfsMultiaddrs,
			DaemonFrequency: daemonFrequency,
			RetryDelay:      retryDelay,
			PadToSize:       padToSize,
			PostgresURI:     v.GetString("postgres-uri"),
		}
		serv, err := service.New(mb, config)
		cli.CheckErr(err)

		cli.HandleInterrupt(func() {
			if err := serv.Close(); err != nil {
				log.Errorf("closing service: %s", err)
			}
			if err := mb.Close(); err != nil {
				log.Errorf("closing message broker: %s", err)
			}
		})
	},
}

func main() {
	cli.CheckErr(rootCmd.Execute())
}
