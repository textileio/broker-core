package filclient

import "github.com/filecoin-project/lotus/chain/types"

type config struct {
	keyType    types.KeyType
	keyPrivate []byte
}

var defaultConfig = config{}

// Option applies a configuration change.
type Option func(*config) error

func WithKey(keyType string, keyPrivate []byte) Option {
	return func(c *config) error {
		c.keyType = types.KeyType(keyType)
		c.keyPrivate = keyPrivate
		return nil
	}
}
