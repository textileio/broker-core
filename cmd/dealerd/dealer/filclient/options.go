package filclient

import (
	"fmt"

	"github.com/filecoin-project/go-address"
	"github.com/filecoin-project/go-state-types/big"
	"github.com/jsign/go-filsigner/wallet"
)

type config struct {
	exportedHexKey string
	pubKey         address.Address

	allowUnverifiedDeals             bool
	maxVerifiedPricePerGiBPerEpoch   big.Int
	maxUnverifiedPricePerGiBPerEpoch big.Int
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

// WithAllowUnverifiedDeals indicates if unverified deals are allowed.
func WithAllowUnverifiedDeals(allow bool) Option {
	return func(c *config) error {
		c.allowUnverifiedDeals = allow
		return nil
	}
}

// WithMaxPriceLimits indicates maximum attoFIL per GiB per epoch in proosals.
func WithMaxPriceLimits(maxVerifiedPricePerGiBPerEpoch, maxUnverifiedPricePerGiBPerEpoch int64) Option {
	return func(c *config) error {
		c.maxVerifiedPricePerGiBPerEpoch = big.NewInt(maxVerifiedPricePerGiBPerEpoch)
		c.maxUnverifiedPricePerGiBPerEpoch = big.NewInt(maxUnverifiedPricePerGiBPerEpoch)
		return nil
	}
}
