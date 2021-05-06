package marketpeer

import (
	"time"

	"github.com/textileio/broker-core/cmd/common"
)

// Flags defines daemon flags for a marketpeer.
var Flags = []common.Flag{
	{Name: "repo", DefValue: "${HOME}/.miner", Description: "Repo path"},
	{Name: "listen-multiaddr", DefValue: "/ip4/0.0.0.0/tcp/4001", Description: "Libp2p listen multiaddr"},
	{Name: "bootstrap-multiaddr", DefValue: "", Description: "Libp2p bootstrap peer multiaddr; repeatable"},
	{Name: "announce-multiaddr", DefValue: "", Description: "Libp2p annouce multiaddr; repeatable"},
	{Name: "conn-low", DefValue: 256, Description: "Libp2p connection manager low water mark"},
	{Name: "conn-high", DefValue: 512, Description: "Libp2p connection manager high water mark"},
	{Name: "conn-grace", DefValue: time.Second * 120, Description: "Libp2p connection manager grace period"},
	{Name: "quic", DefValue: false, Description: "Enable the QUIC transport"},
	{Name: "mdns", DefValue: false, Description: "Enable MDNS peer discovery"},
	{Name: "nat", DefValue: false, Description: "Enable NAT port mapping"},
}
