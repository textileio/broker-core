package marketpeer

import (
	"time"

	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/spf13/viper"
	"github.com/textileio/broker-core/cmd/common"
)

// Flags defines daemon flags for a marketpeer.
var Flags = []common.Flag{
	{
		Name:        "listen-multiaddr",
		DefValue:    "/ip4/0.0.0.0/tcp/4001",
		Description: "Libp2p listen multiaddr; repeatable",
		Repeatable:  true,
	},
	{
		Name:        "bootstrap-multiaddr",
		DefValue:    "",
		Description: "Libp2p bootstrap peer multiaddr; repeatable",
		Repeatable:  true,
	},
	{
		Name:        "announce-multiaddr",
		DefValue:    "",
		Description: "Libp2p annouce multiaddr; repeatable",
		Repeatable:  true,
	},
	{
		Name:        "conn-low",
		DefValue:    256,
		Description: "Libp2p connection manager low water mark",
	},
	{
		Name:        "conn-high",
		DefValue:    512,
		Description: "Libp2p connection manager high water mark",
	},
	{
		Name:        "conn-grace",
		DefValue:    time.Second * 120,
		Description: "Libp2p connection manager grace period",
	},
	{
		Name:        "quic",
		DefValue:    false,
		Description: "Enable the QUIC transport",
	},
	{
		Name:        "nat",
		DefValue:    false,
		Description: "Enable NAT port mapping",
	},
	{
		Name:        "mdns",
		DefValue:    false,
		Description: "Enable MDNS peer discovery",
	},
	{
		Name:        "mdns-interval",
		DefValue:    1,
		Description: "MDNS peer discovery interval in seconds",
	},
}

// ConfigFromFlags returns a Config from a *viper.Viper instance.
func ConfigFromFlags(v *viper.Viper, isAuctioneer bool) Config {
	return Config{
		RepoPath:           v.GetString("repo"),
		ListenMultiaddrs:   common.ParseStringSlice(v, "listen-multiaddr"),
		AnnounceMultiaddrs: common.ParseStringSlice(v, "announce-multiaddr"),
		BootstrapAddrs:     common.ParseStringSlice(v, "bootstrap-multiaddr"),
		ConnManager: connmgr.NewConnManager(
			v.GetInt("conn-low"),
			v.GetInt("conn-high"),
			v.GetDuration("conn-grace"),
		),
		EnableQUIC:               v.GetBool("quic"),
		EnableNATPortMap:         v.GetBool("nat"),
		EnableMDNS:               v.GetBool("mdns"),
		MDNSIntervalSeconds:      v.GetInt("mdns-interval"),
		EnablePubSubPeerExchange: isAuctioneer,
		EnablePubSubFloodPublish: true,
	}
}
