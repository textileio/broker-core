package packer

import (
	"fmt"
	"net/url"
	"time"
)

type config struct {
	daemonFreq        time.Duration
	exportMetricsFreq time.Duration
	retryDelay        time.Duration

	sectorSize   int64
	batchMinSize int64

	carUploader  CARUploader
	carExportURL *url.URL
}

var defaultConfig = config{
	daemonFreq:        time.Second * 20,
	exportMetricsFreq: time.Minute * 5,
	retryDelay:        time.Second * 30,

	sectorSize:   32 << 30,
	batchMinSize: 10 << 20,
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

// WithCARUploader configures a file uploader for CAR files.
func WithCARUploader(uploader CARUploader) Option {
	return func(c *config) error {
		c.carUploader = uploader
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

// WithCARExportURL configures the frequency of exporting the pin count metric.
func WithCARExportURL(rawURL string) Option {
	return func(c *config) error {
		if rawURL == "" {
			c.carExportURL = nil
			return nil
		}
		u, err := url.Parse(rawURL)
		if err != nil {
			return fmt.Errorf("parsing url: %s", err)
		}
		c.carExportURL = u
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

// WithRetryDelay indicates how many time erorred batch processing should be delayed
// before retrying.
func WithRetryDelay(delay time.Duration) Option {
	return func(c *config) error {
		if delay <= 0 {
			return fmt.Errorf("retry delay isn't negative")
		}
		c.retryDelay = delay
		return nil
	}
}

func (c *config) validate() error {
	if c.batchMinSize <= 0 {
		return fmt.Errorf("batch min size should be positive")
	}
	if c.sectorSize <= 0 {
		return fmt.Errorf("sector size should be positive")
	}
	if c.carUploader == nil && c.carExportURL == nil {
		return fmt.Errorf("at least one car export configuration must be set")
	}
	return nil
}
