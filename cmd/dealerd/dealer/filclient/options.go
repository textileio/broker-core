package filclient

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	"github.com/filecoin-project/lotus/chain/types"
)

type config struct {
	keyInfo types.KeyInfo
}

var defaultConfig = config{}

// Option applies a configuration change.
type Option func(*config) error

func WithExportedKey(exportedHexKey string) Option {
	return func(c *config) error {
		if exportedHexKey == "" {
			return fmt.Errorf("exported wallet key is empty")
		}
		buf, err := hex.DecodeString(exportedHexKey)
		if err != nil {
			return fmt.Errorf("hex decoding: %s", err)
		}
		if err := json.Unmarshal(buf, &c.keyInfo); err != nil {
			return fmt.Errorf("unmarshaling key info: %s", err)
		}
		return nil
	}
}
