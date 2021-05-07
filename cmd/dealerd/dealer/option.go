package dealer

import (
	"errors"
	"time"
)

type config struct {
	dealMakingFreq        time.Duration
	dealMakingRateLim     int
	dealMonitoringFreq    time.Duration
	dealMonitoringRateLim int
	dealReportingFreq     time.Duration
}

var defaultConfig = config{
	dealMakingFreq:        time.Second * 10,
	dealMakingRateLim:     20,
	dealMonitoringFreq:    time.Second * 10,
	dealMonitoringRateLim: 20,
	dealReportingFreq:     time.Second * 10,
}

// Option applies a configuration change.
type Option func(*config) error

// WithDealMakingFreq configures the frequency of deal making polling.
func WithDealMakingFreq(f time.Duration) Option {
	return func(c *config) error {
		if f == 0 {
			return errors.New("frequency is zero")
		}
		c.dealMakingFreq = f
		return nil
	}
}

// WithDealMonitoringFreq configures the frequency of deal monitoring polling.
func WithDealMonitoringFreq(f time.Duration) Option {
	return func(c *config) error {
		if f == 0 {
			return errors.New("frequency is zero")
		}
		c.dealMonitoringFreq = f
		return nil
	}
}

// WithDealReportingFreq configures the frequency of deals reporting polling.
func WithDealReportingFreq(f time.Duration) Option {
	return func(c *config) error {
		if f == 0 {
			return errors.New("frequency is zero")
		}
		c.dealReportingFreq = f
		return nil
	}
}

// WithDealMakingRateLim configures the max number of parallel execution items for deal making.
func WithDealMakingRateLim(l int) Option {
	return func(c *config) error {
		if l == 0 {
			return errors.New("rate limit is zero")
		}
		c.dealMakingRateLim = l
		return nil
	}
}

// WithDealMonitoringRateLim configures the max number of parallel execution items for deal monitoring.
func WithDealMonitoringRateLim(l int) Option {
	return func(c *config) error {
		if l == 0 {
			return errors.New("rate limit is zero")
		}
		c.dealMonitoringRateLim = l
		return nil
	}
}
