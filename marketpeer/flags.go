package marketpeer

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	connmgr "github.com/libp2p/go-libp2p-connmgr"
	"github.com/libp2p/go-libp2p-core/crypto"
	mbase "github.com/multiformats/go-multibase"
	"github.com/spf13/viper"
	"github.com/textileio/broker-core/cmd/common"
)

// Flags defines daemon flags for a marketpeer.
var Flags = []common.Flag{
	{
		Name:        "private-key",
		DefValue:    "",
		Description: "Libp2p private key; required",
	},
	{
		Name: "listen-multiaddr",
		DefValue: []string{
			"/ip4/0.0.0.0/tcp/4001",
			"/ip4/0.0.0.0/udp/4001/quic",
		},
		Description: "Libp2p listen multiaddr",
	},
	{
		Name: "bootstrap-multiaddr",
		DefValue: []string{
			"/ip4/34.83.24.156/tcp/4001/p2p/12D3KooWLeNAFPGB1Yc2J52BwVvsZiUaeRdQ4SgnfBk1fDibSyoJ",
			"/ip4/34.83.24.156/udp/4001/quic/p2p/12D3KooWLeNAFPGB1Yc2J52BwVvsZiUaeRdQ4SgnfBk1fDibSyoJ",
		},
		Description: "Libp2p bootstrap peer multiaddr",
	},
	{
		Name:        "announce-multiaddr",
		DefValue:    []string{},
		Description: "Libp2p annouce multiaddr",
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

// GetConfig returns a Config from a *viper.Viper instance.
func GetConfig(v *viper.Viper, repoPathEnv, defaultRepoPath string, isAuctioneer bool) (Config, error) {
	if v.GetString("private-key") == "" {
		return Config{}, fmt.Errorf("--private-key is required. Run 'bidbot init' to generate a new keypair")
	}

	_, key, err := mbase.Decode(v.GetString("private-key"))
	if err != nil {
		return Config{}, fmt.Errorf("decoding private key: %v", err)
	}
	priv, err := crypto.UnmarshalPrivateKey(key)
	if err != nil {
		return Config{}, fmt.Errorf("unmarshaling private key: %v", err)
	}

	repoPath := os.Getenv(repoPathEnv)
	if repoPath == "" {
		repoPath = defaultRepoPath
	}
	return Config{
		RepoPath:           repoPath,
		PrivKey:            priv,
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
	}, nil
}

// WriteConfig writes a *viper.Viper config to file.
// The file is written to a path in pathEnv env var if set, otherwise to defaultPath.
func WriteConfig(v *viper.Viper, repoPathEnv, defaultRepoPath string) (string, error) {
	repoPath := os.Getenv(repoPathEnv)
	if repoPath == "" {
		repoPath = defaultRepoPath
	}
	cf := filepath.Join(repoPath, "config")
	if err := os.MkdirAll(filepath.Dir(cf), os.ModePerm); err != nil {
		return "", fmt.Errorf("making config directory: %v", err)
	}

	// Bail if config already exists
	if _, err := os.Stat(cf); err == nil {
		return "", fmt.Errorf("%s already exists", cf)
	}

	if v.GetString("private-key") == "" {
		priv, _, err := crypto.GenerateEd25519Key(rand.Reader)
		if err != nil {
			return "", fmt.Errorf("generating private key: %v", err)
		}
		key, err := crypto.MarshalPrivateKey(priv)
		if err != nil {
			return "", fmt.Errorf("marshaling private key: %v", err)
		}
		keystr, err := mbase.Encode(mbase.Base64, key)
		if err != nil {
			return "", fmt.Errorf("encoding private key: %v", err)
		}
		v.Set("private-key", keystr)
	}

	if err := v.WriteConfigAs(cf); err != nil {
		return "", fmt.Errorf("error writing config: %v", err)
	}
	v.SetConfigFile(cf)
	if err := v.ReadInConfig(); err != nil {
		common.CheckErrf("reading configuration: %s", err)
	}
	return cf, nil
}

// MarshalConfig marshals a *viper.Viper config to JSON.
func MarshalConfig(v *viper.Viper) ([]byte, error) {
	all := v.AllSettings()
	all["private-key"] = "***"
	return json.MarshalIndent(all, "", "  ")
}
