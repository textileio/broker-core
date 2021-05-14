package packer

import (
	"fmt"
	"time"
)

// Config holds options for creating a new packer.
type Config struct {
	frequency  time.Duration
	sectorSize int64
}

var defaultConfig = Config{
	frequency:  time.Second * 20,
	sectorSize: 32 << 30,
}

// Option applies a configuration change.
type Option func(*Config) error

// WithFrequency indicates how much time should pass until a batch is
// created. e.g: every 20 seconds as a maximum waiting time for the next batch
// if data is available.
func WithFrequency(frequency time.Duration) Option {
	return func(c *Config) error {
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
	return func(c *Config) error {
		if sectorSize <= 0 {
			return fmt.Errorf("sector size should be positive")
		}
		c.sectorSize = sectorSize
		return nil
	}
}
