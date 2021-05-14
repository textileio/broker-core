package filclient

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/lotus/chain/types"
	"github.com/textileio/broker-core/cmd/dealerd/dealer/sigs/secp"
)

// Config is the config.
type Config struct {
	privKey []byte
	pubKey  address.Address
}

var defaultConfig = Config{}

// Option applies a configuration change.
type Option func(*Config) error

// WithExportedKey configures the wallet address private key used to make deals.
func WithExportedKey(exportedHexKey string) Option {
	return func(c *Config) error {
		if exportedHexKey == "" {
			return fmt.Errorf("exported wallet key is empty")
		}
		buf, err := hex.DecodeString(exportedHexKey)
		if err != nil {
			return fmt.Errorf("hex decoding: %s", err)
		}

		var keyInfo types.KeyInfo
		if err := json.Unmarshal(buf, &keyInfo); err != nil {
			return fmt.Errorf("unmarshaling key info: %s", err)
		}
		c.privKey = keyInfo.PrivateKey
		pubkey, err := secp.ToPublic(c.privKey)
		if err != nil {
			return fmt.Errorf("calculating public key: %s", err)
		}
		c.pubKey, err = address.NewSecp256k1Address(pubkey)
		if err != nil {
			return fmt.Errorf("parsing public key: %s", err)
		}

		return nil
	}
}
