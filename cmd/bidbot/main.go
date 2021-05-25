package main

import (
	"encoding/hex"
	"errors"
	"fmt"
	_ "net/http/pprof"
	"os"
	"path/filepath"

	golog "github.com/ipfs/go-log/v2"
	"github.com/libp2p/go-libp2p-core/crypto"
	"github.com/libp2p/go-libp2p-core/peer"
	mbase "github.com/multiformats/go-multibase"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/broker-core/broker"
	"github.com/textileio/broker-core/cmd/auctioneerd/auctioneer/filclient"
	"github.com/textileio/broker-core/cmd/bidbot/service"
	"github.com/textileio/broker-core/cmd/common"
	"github.com/textileio/broker-core/finalizer"
	"github.com/textileio/broker-core/marketpeer"
)

var (
	cliName           = "bidbot"
	defaultConfigPath = filepath.Join(os.Getenv("HOME"), "."+cliName)
	log               = golog.Logger(cliName)
	v                 = viper.New()
)

func init() {
	rootCmd.AddCommand(initCmd, daemonCmd)

	flags := []common.Flag{
		{
			Name:        "miner-addr",
			DefValue:    "",
			Description: "Miner address (fxxxx); required",
		},
		{
			Name:        "wallet-addr-sig",
			DefValue:    "",
			Description: "Miner wallet address signature; required; see 'bidbot help init' for instructions",
		},
		{
			Name:        "ask-price",
			DefValue:    100000000000,
			Description: "Bid ask price for deals in attoFIL per GiB per epoch; default is 100 nanoFIL",
		},
		{
			Name:        "verified-ask-price",
			DefValue:    100000000000,
			Description: "Bid ask price for verified deals in attoFIL per GiB per epoch; default is 100 nanoFIL",
		},
		{
			Name:        "fast-retrieval",
			DefValue:    false,
			Description: "Offer deals with fast retrieval",
		},
		{
			Name:        "deal-start-window",
			DefValue:    60 * 24 * 2,
			Description: "Number of epochs after which won deals must start on-chain; default is ~one day",
		},
		{
			Name:        "deal-duration-min",
			DefValue:    broker.MinDealEpochs,
			Description: "Minimum deal duration to bid on in epochs; default is ~6 months",
		},
		{
			Name:        "deal-duration-max",
			DefValue:    broker.MaxDealEpochs,
			Description: "Maximum deal duration to bid on in epochs; default is ~1 year",
		},
		{
			Name:        "deal-size-min",
			DefValue:    56 * 1024,
			Description: "Minimum deal size to bid on in bytes",
		},
		{
			Name:        "deal-size-max",
			DefValue:    32 * 1000 * 1000 * 1000,
			Description: "Maximum deal size to bid on in bytes",
		},
		{Name: "lotus-gateway-url", DefValue: "https://api.node.glif.io", Description: "Lotus gateway URL"},
		{Name: "metrics-addr", DefValue: ":9090", Description: "Prometheus listen address"},
		{Name: "log-debug", DefValue: false, Description: "Enable debug level log"},
		{Name: "log-json", DefValue: false, Description: "Enable structured logging"},
	}
	flags = append(flags, marketpeer.Flags...)

	cobra.OnInitialize(func() {
		v.SetConfigType("json")
		v.SetConfigName("config")
		v.AddConfigPath(os.Getenv("BIDBOT_PATH"))
		v.AddConfigPath(defaultConfigPath)
		_ = v.ReadInConfig()
	})

	common.ConfigureCLI(v, "BIDBOT", flags, rootCmd.PersistentFlags())
}

var rootCmd = &cobra.Command{
	Use:   cliName,
	Short: "Bidbot listens for Filecoin storage deal auctions from deal brokers",
	Long: `Bidbot listens for Filecoin storage deal auctions from deal brokers.

bidbot will automatically bid on storage deals that pass configured filters at
the configured prices.

To get started, run 'bidbot init' and follow the instructions. 
`,
	Args: cobra.ExactArgs(0),
}

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initializes bidbot configuration files",
	Long: `Initializes bidbot configuration files and generates a new keypair.

bidbot uses a repository in the local file system. By default, the repo is
located at ~/.bidbot. To change the repo location, set the $BIDBOT_PATH
environment variable:

    export BIDBOT_PATH=/path/to/bidbotrepo
`,
	Args: cobra.ExactArgs(0),
	Run: func(c *cobra.Command, args []string) {
		path, err := marketpeer.WriteConfig(v, "BIDBOT_PATH", defaultConfigPath)
		common.CheckErrf("writing config: %v", err)
		fmt.Printf("Initialized configuration file: %s\n\n", path)

		_, key, err := mbase.Decode(v.GetString("private-key"))
		common.CheckErrf("decoding private key: %v", err)
		priv, err := crypto.UnmarshalPrivateKey(key)
		common.CheckErrf("unmarshaling private key: %v", err)
		id, err := peer.IDFromPrivateKey(priv)
		common.CheckErrf("getting peer id: %v", err)

		signingToken := hex.EncodeToString([]byte(id))

		fmt.Printf(`Bidbot needs a signature from a miner wallet address to authenticate bids.

1. Sign this token with an address from your miner owner Lotus wallet address:

    lotus wallet sign [owner-address] %s

2. Start listening for deal auctions using the wallet address and signature from step 1:

    bidbot daemon --miner-addr [address] --wallet-addr-sig [signature]

Note: In the event you win an auction, you must use this wallet address to make the deal(s).

Good luck!
`, signingToken)
	},
}

var daemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "Run a network-connected bidding bot",
	Long:  "Run a network-connected bidding bot that listens for and bids on storage deal auctions.",
	Args:  cobra.ExactArgs(0),
	PersistentPreRun: func(c *cobra.Command, args []string) {
		common.ExpandEnvVars(v, v.AllSettings())
		err := common.ConfigureLogging(v, []string{
			cliName,
			"bidbot/service",
			"mpeer",
			"mpeer/pubsub",
		})
		common.CheckErrf("setting log levels: %v", err)
	},
	Run: func(c *cobra.Command, args []string) {
		if v.GetString("miner-addr") == "" {
			common.CheckErr(errors.New("--miner-addr is required. See 'bidbot help init' for instructions"))
		}
		if v.GetString("wallet-addr-sig") == "" {
			common.CheckErr(errors.New("--wallet-addr-sig is required. See 'bidbot help init' for instructions"))
		}

		pconfig, err := marketpeer.GetConfig(v, "BIDBOT_PATH", defaultConfigPath, false)
		common.CheckErrf("getting peer config: %v", err)

		settings, err := marketpeer.MarshalConfig(v)
		common.CheckErrf("marshaling config: %v", err)
		log.Infof("loaded config: %s", string(settings))

		err = common.SetupInstrumentation(v.GetString("metrics.addr"))
		common.CheckErrf("booting instrumentation: %v", err)

		walletAddrSig, err := hex.DecodeString(v.GetString("wallet-addr-sig"))
		common.CheckErrf("decoding wallet address signature: %v", err)

		fin := finalizer.NewFinalizer()
		fc, err := filclient.New(v.GetString("lotus-gateway-url"), v.GetBool("fake-mode"))
		common.CheckErrf("creating chain client: %v", err)
		fin.Add(fc)

		config := service.Config{
			RepoPath: pconfig.RepoPath,
			Peer:     pconfig,
			BidParams: service.BidParams{
				MinerAddr:        v.GetString("miner-addr"),
				WalletAddrSig:    walletAddrSig,
				AskPrice:         v.GetInt64("ask-price"),
				VerifiedAskPrice: v.GetInt64("verified-ask-price"),
				FastRetrieval:    v.GetBool("fast-retrieval"),
				DealStartWindow:  v.GetUint64("deal-start-window"),
			},
			AuctionFilters: service.AuctionFilters{
				DealDuration: service.MinMaxFilter{
					Min: v.GetUint64("deal-duration-min"),
					Max: v.GetUint64("deal-duration-max"),
				},
				DealSize: service.MinMaxFilter{
					Min: v.GetUint64("deal-size-min"),
					Max: v.GetUint64("deal-size-max"),
				},
			},
		}
		serv, err := service.New(config, fc)
		common.CheckErrf("starting service: %v", err)
		fin.Add(serv)

		err = serv.Subscribe(true)
		common.CheckErrf("subscribing to deal auction feed: %v", err)

		common.HandleInterrupt(func() {
			common.CheckErr(fin.Cleanupf("closing service: %v", nil))
		})
	},
}

func main() {
	common.CheckErr(rootCmd.Execute())
}
