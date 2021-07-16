package main

import (
	_ "net/http/pprof"

	"github.com/multiformats/go-multiaddr"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/bidbot/lib/common"
	"github.com/textileio/broker-core/cmd/storaged/service"
	logging "github.com/textileio/go-log/v2"
)

var (
	daemonName = "storagerd"
	log        = logging.Logger(daemonName)
	v          = viper.New()
)

func init() {
	flags := []common.Flag{
		{Name: "http-addr", DefValue: ":8888", Description: "HTTP API listen address"},
		{Name: "uploader-ipfs-multiaddr", DefValue: "/ip4/127.0.0.1/tcp/5001", Description: "Uploader IPFS API pool"},
		{Name: "broker-addr", DefValue: "", Description: "Broker API address"},
		{Name: "auth-addr", DefValue: "", Description: "Authorizer API address"},
		{Name: "metrics-addr", DefValue: ":9090", Description: "Prometheus listen address"},
		{Name: "skip-auth", DefValue: false, Description: "Disabled authorization check"},
		{Name: "ipfs-multiaddrs", DefValue: []string{}, Description: "IPFS multiaddresses"},
		{Name: "pinata-jwt", DefValue: "",
			Description: "Pinata API JWT to upload files also to Pinata. If empty, the feature will be considered disabled."},
		{Name: "bearer-tokens", DefValue: []string{}, Description: "Raw accepted bearer tokens"},
		{Name: "max-upload-size", DefValue: "4GB", Description: "Maximum upload size"},
		{Name: "log-debug", DefValue: false, Description: "Enable debug level logging"},
		{Name: "log-json", DefValue: false, Description: "Enable structured logging"},
	}

	common.ConfigureCLI(v, "STORAGE", flags, rootCmd.Flags())
}

var rootCmd = &cobra.Command{
	Use:   daemonName,
	Short: "storaged provides a synchronous data uploader endpoint to store data in a Broker",
	Long:  `storaged provides a synchronous data uploader endpoint to store data in a Broker`,
	PersistentPreRun: func(c *cobra.Command, args []string) {
		common.ExpandEnvVars(v, v.AllSettings())
		err := common.ConfigureLogging(v, nil)
		common.CheckErrf("setting log levels: %v", err)
	},
	Run: func(c *cobra.Command, args []string) {
		settings, err := common.MarshalConfig(v, !v.GetBool("log-json"), "bearer-tokens")
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
		serviceConfig := service.Config{
			HTTPListenAddr:        v.GetString("http-addr"),
			UploaderIPFSMultiaddr: v.GetString("uploader-ipfs-multiaddr"),
			BrokerAPIAddr:         v.GetString("broker-addr"),
			AuthAddr:              v.GetString("auth-addr"),
			SkipAuth:              v.GetBool("skip-auth"),
			IpfsMultiaddrs:        ipfsMultiaddrs,
			PinataJWT:             v.GetString("pinata-jwt"),
			BearerTokens:          common.ParseStringSlice(v, "bearer-tokens"),
			MaxUploadSize:         v.GetSizeInBytes("max-upload-size"),
		}
		serv, err := service.New(serviceConfig)
		common.CheckErr(err)

		log.Info("listening to requests...")

		common.HandleInterrupt(func() {
			if err := serv.Close(); err != nil {
				log.Errorf("closing http endpoint: %s", err)
			}
		})
	},
}

func main() {
	common.CheckErr(rootCmd.Execute())
}
