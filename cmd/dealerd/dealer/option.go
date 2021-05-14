package dealer

import (
	"errors"
	"time"
)

// Config is the config.
type Config struct {
	dealMakingFreq      time.Duration
	dealMakingRateLim   int
	dealWatchingFreq    time.Duration
	dealWatchingRateLim int
	dealReportingFreq   time.Duration
}

var defaultConfig = Config{
	dealMakingFreq:      time.Second * 10,
	dealMakingRateLim:   20,
	dealWatchingFreq:    time.Second * 20,
	dealWatchingRateLim: 20,
	dealReportingFreq:   time.Second * 10,
}

// Option applies a configuration change.
type Option func(*Config) error

// WithDealMakingFreq configures the frequency of deal making polling.
func WithDealMakingFreq(f time.Duration) Option {
	return func(c *Config) error {
		if f == 0 {
			return errors.New("frequency is zero")
		}
		c.dealMakingFreq = f
		return nil
	}
}

// WithDealMonitoringFreq configures the frequency of deal monitoring polling.
func WithDealMonitoringFreq(f time.Duration) Option {
	return func(c *Config) error {
		if f == 0 {
			return errors.New("frequency is zero")
		}
		c.dealWatchingFreq = f
		return nil
	}
}

// WithDealReportingFreq configures the frequency of deals reporting polling.
func WithDealReportingFreq(f time.Duration) Option {
	return func(c *Config) error {
		if f == 0 {
			return errors.New("frequency is zero")
		}
		c.dealReportingFreq = f
		return nil
	}
}

// WithDealMakingRateLim configures the max number of parallel execution items for deal making.
func WithDealMakingRateLim(l int) Option {
	return func(c *Config) error {
		if l == 0 {
			return errors.New("rate limit is zero")
		}
		c.dealMakingRateLim = l
		return nil
	}
}

// WithDealMonitoringRateLim configures the max number of parallel execution items for deal monitoring.
func WithDealMonitoringRateLim(l int) Option {
	return func(c *Config) error {
		if l == 0 {
			return errors.New("rate limit is zero")
		}
		c.dealWatchingRateLim = l
		return nil
	}
}
