package packer

import (
	"fmt"
	"time"
)

type config struct {
	frequency    time.Duration
	sectorSize   int64
	batchMinSize uint
}

var defaultConfig = config{
	frequency:    time.Second * 20,
	sectorSize:   32 << 30,
	batchMinSize: 10 << 20,
}

// Option applies a configuration change.
type Option func(*config) error

// WithFrequency indicates how much time should pass until a batch is
// created. e.g: every 20 seconds as a maximum waiting time for the next batch
// if data is available.
func WithFrequency(frequency time.Duration) Option {
	return func(c *config) error {
		if frequency <= 0 {
			return fmt.Errorf("max wait should be positive")
		}
		c.frequency = frequency
		return nil
	}
}

// WithSectorSize configures the sector size that will be considered for the
// maximum size of batches.
func WithSectorSize(sectorSize int64) Option {
	return func(c *config) error {
		if sectorSize <= 0 {
			return fmt.Errorf("sector size should be positive")
		}
		c.sectorSize = sectorSize
		return nil
	}
}

// WithBatchMinSize configures the minimum batch size that can be considered
// for preparation.
func WithBatchMinSize(minSize uint) Option {
	return func(c *config) error {
		if minSize <= 0 {
			return fmt.Errorf("batch min size should be positive")
		}
		c.batchMinSize = minSize
		return nil
	}
}
