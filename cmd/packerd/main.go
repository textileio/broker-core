package main

import (
	_ "net/http/pprof"

	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/broker-core/cmd/packerd/metrics"
	"github.com/textileio/broker-core/cmd/packerd/packer"
	"github.com/textileio/broker-core/cmd/packerd/packer/gcpblob"
	"github.com/textileio/broker-core/cmd/packerd/service"
	"github.com/textileio/broker-core/common"
	"github.com/textileio/broker-core/msgbroker/gpubsub"
	"github.com/textileio/cli"
	logging "github.com/textileio/go-log/v2"
)

var (
	daemonName = "packerd"
	log        = logging.Logger(daemonName)
	v          = viper.New()
)

func init() {
	flags := []cli.Flag{
		{Name: "postgres-uri", DefValue: "", Description: "PostgreSQL URI"},
		{Name: "broker-addr", DefValue: "", Description: "Broker API address"},
		{Name: "pinner-multiaddr", DefValue: "", Description: "IPFS cluster pinner multiaddr"},
		{Name: "ipfs-multiaddrs", DefValue: []string{}, Description: "IPFS multiaddresses"},
		{Name: "daemon-frequency", DefValue: "20s", Description: "Frequency of polling ready batches"},
		{Name: "export-metrics-frequency", DefValue: "5m", Description: "Frequency of metrics exporting"},
		{Name: "batch-min-size", DefValue: "10MB", Description: "Minimum batch size"},
		{Name: "batch-min-waiting", DefValue: "15m", Description: "Minimum batch waiting time before closing"},
		{Name: "target-sector-size", DefValue: "34359738368", Description: "Target sector-sizes"},
		{Name: "metrics-addr", DefValue: ":9090", Description: "Prometheus listen address"},
		{Name: "gobject-project-id", DefValue: "", Description: "Google Object Storage project id"},
		{Name: "gpubsub-project-id", DefValue: "", Description: "Google PubSub project id"},
		{Name: "gpubsub-api-key", DefValue: "", Description: "Google PubSub API key"},
		{Name: "msgbroker-topic-prefix", DefValue: "", Description: "Topic prefix to use for msg broker topics"},
		{Name: "car-export-url", DefValue: "", Description: "URL that generates CAR files for stored cids"},
		{Name: "log-debug", DefValue: false, Description: "Enable debug level logging"},
		{Name: "log-json", DefValue: false, Description: "Enable structured logging"},
	}

	cli.ConfigureCLI(v, "PACKER", flags, rootCmd.Flags())
}

var rootCmd = &cobra.Command{
	Use:   daemonName,
	Short: "packerd handles deal auctions for the Broker",
	Long:  "packerd handles deal auctions for the Broker",
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

		ipfsMaddrsStr := cli.ParseStringSlice(v, "ipfs-multiaddrs")
		ipfsMaddrs := make([]multiaddr.Multiaddr, len(ipfsMaddrsStr))
		for i, maStr := range ipfsMaddrsStr {
			ma, err := multiaddr.NewMultiaddr(maStr)
			cli.CheckErrf("parsing multiaddress %s: %s", err)
			ipfsMaddrs[i] = ma
		}

		projectID := v.GetString("gobject-project-id")
		var carUploader packer.CARUploader
		if projectID != "" {
			carUploader, err = gcpblob.New(projectID)
			cli.CheckErr(err)
		}

		config := service.Config{
			PostgresURI:     v.GetString("postgres-uri"),
			PinnerMultiaddr: v.GetString("pinner-multiaddr"),
			IpfsMaddrs:      ipfsMaddrs,

			DaemonFrequency:        v.GetDuration("daemon-frequency"),
			ExportMetricsFrequency: v.GetDuration("export-metrics-frequency"),

			TargetSectorSize: v.GetInt64("target-sector-size"),
			BatchMinSize:     int64(v.GetSizeInBytes("batch-min-size")),
			BatchMinWaiting:  v.GetDuration("batch-min-waiting"),
			CARExportURL:     v.GetString("car-export-url"),
			CARUploader:      carUploader,
		}

		projectID = v.GetString("gpubsub-project-id")
		apiKey := v.GetString("gpubsub-api-key")
		topicPrefix := v.GetString("msgbroker-topic-prefix")
		mb, err := gpubsub.NewMetered(projectID, apiKey, topicPrefix, "packerd", metrics.Meter)
		cli.CheckErr(err)

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
