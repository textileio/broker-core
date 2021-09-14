package main

import (
	_ "net/http/pprof"
	"time"

	"github.com/libp2p/go-libp2p"
	relay "github.com/libp2p/go-libp2p-circuit"
	connmgr "github.com/libp2p/go-libp2p-connmgr"
	crypto "github.com/libp2p/go-libp2p-crypto"
	"github.com/multiformats/go-multiaddr"
	mbase "github.com/multiformats/go-multibase"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/bidbot/lib/common"
	"github.com/textileio/broker-core/cmd/relayd/metrics"
	"github.com/textileio/cli"
	"github.com/textileio/go-libp2p-pubsub-rpc/finalizer"
	logging "github.com/textileio/go-log/v2"
)

var (
	daemonName = "relayd"
	log        = logging.Logger(daemonName)
	v          = viper.New()
)

func init() {
	flags := []cli.Flag{
		{Name: "private-key", DefValue: "", Description: "Multibase-encoded libp2p marshaled peer-id private key"},
		{Name: "listen-multiaddr", DefValue: "/ip4/0.0.0.0/tcp/4001", Description: "libp2p identity base64-encoded RSA private key"},

		{Name: "metrics-addr", DefValue: ":9090", Description: "Prometheus listen address"},
		{Name: "log-debug", DefValue: false, Description: "Enable debug level logging"},
		{Name: "log-json", DefValue: false, Description: "Enable structured logging"},
	}

	cli.ConfigureCLI(v, "RELAY", flags, rootCmd.Flags())
}

var rootCmd = &cobra.Command{
	Use:   daemonName,
	Short: "relayd is a service that provides a libp2p-relay for remote wallets",
	PersistentPreRun: func(c *cobra.Command, args []string) {
		cli.ExpandEnvVars(v, v.AllSettings())
		err := cli.ConfigureLogging(v, nil)
		cli.CheckErrf("setting log levels: %v", err)
	},
	Run: func(c *cobra.Command, args []string) {
		settings, err := cli.MarshalConfig(v, !v.GetBool("log-json"), "private-key")
		cli.CheckErr(err)
		log.Infof("loaded config: %s", string(settings))

		if err := common.SetupInstrumentation(v.GetString("metrics-addr")); err != nil {
			log.Fatalf("booting instrumentation: %s", err)
		}

		fin := finalizer.NewFinalizer()

		listenAddr, err := multiaddr.NewMultiaddr(v.GetString("listen-multiaddr"))
		cli.CheckErr(err)

		libp2pPrivateKey := v.GetString("private-key")
		if libp2pPrivateKey == "" {
			log.Fatal("--private-key can't be empty")
		}
		_, key, err := mbase.Decode(v.GetString("private-key"))
		cli.CheckErrf("decoding private key: %s", err)
		pk, err := crypto.UnmarshalPrivateKey(key)
		cli.CheckErrf("unmarshaling private key: %s", err)

		opts := []libp2p.Option{
			libp2p.ConnectionManager(connmgr.NewConnManager(500, 800, time.Minute)),
			libp2p.ListenAddrs(listenAddr),
			libp2p.Identity(pk),
			libp2p.EnableRelay(relay.OptHop),
		}
		h, err := libp2p.New(c.Context(), opts...)
		cli.CheckErrf("bootstraping libp2p host: %s", err)
		fin.Add(h)

		metrics.Register(h)

		log.Infof("relay peer-id: %s", h.ID())
		log.Info("relay open to accept connections...")

		cli.HandleInterrupt(func() {
			common.CheckErr(fin.Cleanup(nil))
		})
	},
}

func main() {
	cli.CheckErr(rootCmd.Execute())
}
