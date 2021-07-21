package packer

import (
	"fmt"
	"time"
)

type config struct {
	daemonFreq        time.Duration
	exportMetricsFreq time.Duration

	sectorSize   int64
	batchMinSize int64
}

var defaultConfig = config{
	daemonFreq:        time.Second * 20,
	exportMetricsFreq: time.Minute * 5,
	sectorSize:        32 << 30,
	batchMinSize:      10 << 20,
}

// Option applies a configuration change.
type Option func(*config) error

// WithDaemonFrequency indicates the frequency in which ready batches are processed.
func WithDaemonFrequency(frequency time.Duration) Option {
	return func(c *config) error {
		if frequency <= 0 {
			return fmt.Errorf("daemon frequency should be positive")
		}
		c.daemonFreq = frequency
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
func WithBatchMinSize(minSize int64) Option {
	return func(c *config) error {
		if minSize <= 0 {
			return fmt.Errorf("batch min size should be positive")
		}
		c.batchMinSize = minSize
		return nil
	}
}

// WithExportMetricsFrequency indicates the exporting metrics frequency of open batches.
func WithExportMetricsFrequency(frequency time.Duration) Option {
	return func(c *config) error {
		if frequency <= 0 {
			return fmt.Errorf("export metrics frequency should be positive")
		}
		c.exportMetricsFreq = frequency
		return nil
	}
}
