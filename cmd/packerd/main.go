package main

import (
	_ "net/http/pprof"

	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/bidbot/lib/common"
	"github.com/textileio/broker-core/cmd/packerd/packer"
	"github.com/textileio/broker-core/cmd/packerd/packer/gcpblob"
	"github.com/textileio/broker-core/cmd/packerd/service"
	"github.com/textileio/broker-core/msgbroker/gpubsub"
	logging "github.com/textileio/go-log/v2"
)

var (
	daemonName = "packerd"
	log        = logging.Logger(daemonName)
	v          = viper.New()
)

func init() {
	flags := []common.Flag{
		{Name: "postgres-uri", DefValue: "", Description: "PostgreSQL URI"},
		{Name: "broker-addr", DefValue: "", Description: "Broker API address"},
		{Name: "pinner-multiaddr", DefValue: "", Description: "IPFS multiaddress"},
		{Name: "ipfs-multiaddrs", DefValue: []string{}, Description: "IPFS multiaddresses"},
		{Name: "daemon-frequency", DefValue: "20s", Description: "Frequency of polling ready batches"},
		{Name: "export-metrics-frequency", DefValue: "5m", Description: "Frequency of metrics exporting"},
		{Name: "batch-min-size", DefValue: "10MB", Description: "Minimum batch size"},
		{Name: "target-sector-size", DefValue: "34359738368", Description: "Target sector-sizes"},
		{Name: "metrics-addr", DefValue: ":9090", Description: "Prometheus listen address"},
		{Name: "gobject-project-id", DefValue: "", Description: "Google Object Storage project id"},
		{Name: "gpubsub-project-id", DefValue: "", Description: "Google PubSub project id"},
		{Name: "gpubsub-api-key", DefValue: "", Description: "Google PubSub API key"},
		{Name: "msgbroker-topic-prefix", DefValue: "", Description: "Topic prefix to use for msg broker topics"},
		{Name: "log-debug", DefValue: false, Description: "Enable debug level logging"},
		{Name: "log-json", DefValue: false, Description: "Enable structured logging"},
	}

	common.ConfigureCLI(v, "PACKER", flags, rootCmd.Flags())
}

var rootCmd = &cobra.Command{
	Use:   daemonName,
	Short: "packerd handles deal auctions for the Broker",
	Long:  "packerd handles deal auctions for the Broker",
	PersistentPreRun: func(c *cobra.Command, args []string) {
		common.ExpandEnvVars(v, v.AllSettings())
		err := common.ConfigureLogging(v, nil)
		common.CheckErrf("setting log levels: %v", err)
	},
	Run: func(c *cobra.Command, args []string) {
		settings, err := common.MarshalConfig(v, !v.GetBool("log-json"), "gpubsub-api-key", "postgres-uri")
		common.CheckErr(err)
		log.Infof("loaded config: %s", string(settings))

		if err := common.SetupInstrumentation(v.GetString("metrics-addr")); err != nil {
			log.Fatalf("booting instrumentation: %s", err)
		}

		ipfsMultiaddrsStr := common.ParseStringSlice(v, "ipfs-multiaddrs")
		ipfsMultiaddrs := make([]multiaddr.Multiaddr, len(ipfsMultiaddrsStr))
		for i, maStr := range ipfsMultiaddrsStr {
			ma, err := multiaddr.NewMultiaddr(maStr)
			common.CheckErrf("parsing multiaddress %s: %s", err)
			ipfsMultiaddrs[i] = ma
		}

		config := service.Config{
			PostgresURI:     v.GetString("postgres-uri"),
			PinnerMultiaddr: v.GetString("pinner-multiaddr"),

			DaemonFrequency:        v.GetDuration("daemon-frequency"),
			ExportMetricsFrequency: v.GetDuration("export-metrics-frequency"),

			TargetSectorSize: v.GetInt64("target-sector-size"),
			BatchMinSize:     int64(v.GetSizeInBytes("batch-min-size")),
		}

		projectID := v.GetString("gpubsub-project-id")
		apiKey := v.GetString("gpubsub-api-key")
		topicPrefix := v.GetString("msgbroker-topic-prefix")
		mb, err := gpubsub.New(projectID, apiKey, topicPrefix, "packerd")
		common.CheckErr(err)

		projectID = v.GetString("gobject-project-id")
		var packerOpts []packer.Option
		if projectID != "" {
			gblob, err := gcpblob.New(projectID)
			common.CheckErr(err)
			packerOpts = append(packerOpts, packer.WithCARUploader(gblob))
		}

		serv, err := service.New(mb, config, packerOpts...)
		common.CheckErr(err)

		common.HandleInterrupt(func() {
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
	common.CheckErr(rootCmd.Execute())
}
