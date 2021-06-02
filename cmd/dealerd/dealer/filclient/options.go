package filclient

import (
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/jsign/go-filsigner/wallet"
)

type config struct {
	exportedHexKey string
	pubKey         address.Address
}

var defaultConfig = config{}

// Option applies a configuration change.
type Option func(*config) error

// WithExportedKey configures the wallet address private key used to make deals.
func WithExportedKey(exportedHexKey string) Option {
	return func(c *config) error {
		if exportedHexKey == "" {
			return fmt.Errorf("exported wallet key is empty")
		}
		c.exportedHexKey = exportedHexKey
		var err error
		c.pubKey, err = wallet.PublicKey(exportedHexKey)
		if err != nil {
			return fmt.Errorf("parsing public key: %s", err)
		}

		return nil
	}
}
