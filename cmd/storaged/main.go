package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"

	logging "github.com/ipfs/go-log/v2"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/textileio/broker-core/cmd/storaged/service"
)

var (
	daemonName = "storagerd"
	log        = logging.Logger(daemonName)
	v          = viper.New()
)

var flags = []struct {
	name        string
	defValue    interface{}
	description string
}{
	{name: "http.listen.addr", defValue: ":8888", description: "HTTP API listen address"},
	{name: "uploader.ipfs.multiaddr", defValue: "/ip4/127.0.0.1/tcp/5001", description: "Uploader IPFS API pool"},
}

func init() {
	v.SetEnvPrefix("UPLOADER")
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	for _, flag := range flags {
		switch defval := flag.defValue.(type) {
		case string:
			rootCmd.Flags().String(flag.name, defval, flag.description)
			v.BindPFlag(flag.name, rootCmd.Flags().Lookup(flag.name))
			v.SetDefault(flag.name, defval)
		default:
			log.Fatalf("unkown flag type: %T", flag)
		}
	}
}

var rootCmd = &cobra.Command{
	Use:   daemonName,
	Short: "storaged provides a synchronous data uploader endpoint to store data in a Broker",
	Long:  `storaged provides a synchronous data uploader endpoint to store data in a Broker`,
	PersistentPreRun: func(c *cobra.Command, args []string) {
		logging.SetAllLoggers(logging.LevelInfo)
		if v.GetBool("log.debug") {
			logging.SetAllLoggers(logging.LevelDebug)
		}
	},
	Run: func(c *cobra.Command, args []string) {
		settings, err := json.MarshalIndent(v.AllSettings(), "", "  ")
		checkErr(err)
		log.Infof("loaded config: %s", string(settings))

		serviceConfig := service.Config{
			HttpListenAddr:        v.GetString("http.listen.addr"),
			UploaderIPFSMultiaddr: v.GetString("uploader.ipfs.multiaddr"),
		}
		serv, err := service.New(serviceConfig)
		checkErr(err)

		log.Info("Startup executed successfully, ready to rock...")
		quit := make(chan os.Signal)
		signal.Notify(quit, os.Interrupt)
		<-quit
		fmt.Println("Gracefully stopping... (press Ctrl+C again to force)")
		if err := serv.Close(); err != nil {
			log.Errorf("closing http endpoint: %s", err)
		}
	},
}

func main() {
	checkErr(rootCmd.Execute())
}

func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
