package main

import (
	_ "net/http/pprof"

	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/broker-core/cmd/storaged/service"
	"github.com/textileio/broker-core/common"
	"github.com/textileio/cli"
	logging "github.com/textileio/go-log/v2"
)

var (
	daemonName = "storagerd"
	log        = logging.Logger(daemonName)
	v          = viper.New()
)

func init() {
	flags := []cli.Flag{
		{Name: "http-addr", DefValue: ":8888", Description: "HTTP API listen address"},
		{Name: "uploader-ipfs-multiaddr", DefValue: "/ip4/127.0.0.1/tcp/5001", Description: "Uploader IPFS API pool"},
		{Name: "broker-addr", DefValue: "", Description: "Broker API address"},
		{Name: "auth-addr", DefValue: "", Description: "Authorizer API address"},
		{Name: "metrics-addr", DefValue: ":9090", Description: "Prometheus listen address"},
		{Name: "skip-auth", DefValue: false, Description: "Disabled authorization check"},
		{Name: "ipfs-multiaddrs", DefValue: []string{}, Description: "IPFS multiaddresses"},
		{Name: "pinata-jwt", DefValue: "",
			Description: "Pinata API JWT to upload files also to Pinata. If empty, the feature will be considered disabled."},
		{Name: "relay-maddr", DefValue: "", Description: "Relay multiaddress for remote wallets"},
		{Name: "max-upload-size", DefValue: "4GB", Description: "Maximum upload size"},
		{Name: "log-debug", DefValue: false, Description: "Enable debug level logging"},
		{Name: "log-json", DefValue: false, Description: "Enable structured logging"},
	}

	cli.ConfigureCLI(v, "STORAGE", flags, rootCmd.Flags())
}

var rootCmd = &cobra.Command{
	Use:   daemonName,
	Short: "storaged provides a synchronous data uploader endpoint to store data in a Broker",
	Long:  `storaged provides a synchronous data uploader endpoint to store data in a Broker`,
	PersistentPreRun: func(c *cobra.Command, args []string) {
		cli.ExpandEnvVars(v, v.AllSettings())
		err := cli.ConfigureLogging(v, nil)
		cli.CheckErrf("setting log levels: %v", err)
	},
	Run: func(c *cobra.Command, args []string) {
		settings, err := cli.MarshalConfig(v, !v.GetBool("log-json"))
		cli.CheckErr(err)
		log.Infof("loaded config: %s", string(settings))

		if err := common.SetupInstrumentation(v.GetString("metrics-addr")); err != nil {
			log.Fatalf("booting instrumentation: %s", err)
		}

		ipfsMultiaddrsStr := cli.ParseStringSlice(v, "ipfs-multiaddrs")
		ipfsMultiaddrs := make([]multiaddr.Multiaddr, len(ipfsMultiaddrsStr))
		for i, maStr := range ipfsMultiaddrsStr {
			ma, err := multiaddr.NewMultiaddr(maStr)
			cli.CheckErrf("parsing multiaddress %s: %s", err)
			ipfsMultiaddrs[i] = ma
		}
		serviceConfig := service.Config{
			HTTPListenAddr:        v.GetString("http-addr"),
			UploaderIPFSMultiaddr: v.GetString("uploader-ipfs-multiaddr"),
			BrokerAPIAddr:         v.GetString("broker-addr"),
			AuthAddr:              v.GetString("auth-addr"),
			SkipAuth:              v.GetBool("skip-auth"),
			IpfsMultiaddrs:        ipfsMultiaddrs,
			PinataJWT:             v.GetString("pinata-jwt"),
			MaxUploadSize:         v.GetSizeInBytes("max-upload-size"),
			RelayMaddr:            v.GetString("relay-maddr"),
		}
		serv, err := service.New(serviceConfig)
		cli.CheckErr(err)

		log.Info("listening to requests...")

		cli.HandleInterrupt(func() {
			if err := serv.Close(); err != nil {
				log.Errorf("closing http endpoint: %s", err)
			}
		})
	},
}

func main() {
	cli.CheckErr(rootCmd.Execute())
}
