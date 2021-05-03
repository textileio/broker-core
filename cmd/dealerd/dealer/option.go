package dealer

import (
	"fmt"
	"time"
)

type config struct {
	dealMakingFreq        time.Duration
	dealMakingRateLim     int
	dealMonitoringFreq    time.Duration
	dealMonitoringRateLim int
	dealReportingFreq     time.Duration
	dealReportingRateLim  int
}

var defaultConfig = config{
	dealMakingFreq:        time.Second * 10,
	dealMakingRateLim:     20,
	dealMonitoringFreq:    time.Second * 10,
	dealMonitoringRateLim: 20,
	dealReportingFreq:     time.Second * 10,
	dealReportingRateLim:  20,
}

// Option applies a configuration change.
type Option func(*config) error

func WithDealMakingFreq(f time.Duration) Option {
	return func(c *config) error {
		if f == 0 {
			return fmt.Errorf("frequency is zero")
		}
		c.dealMakingFreq = f
		return nil
	}
}

func WithDealMonitoringFreq(f time.Duration) Option {
	return func(c *config) error {
		if f == 0 {
			return fmt.Errorf("frequency is zero")
		}
		c.dealMonitoringFreq = f
		return nil
	}
}

func WithDealReportingFreq(f time.Duration) Option {
	return func(c *config) error {
		if f == 0 {
			return fmt.Errorf("frequency is zero")
		}
		c.dealReportingFreq = f
		return nil
	}
}

func WithDealMakingRateLim(l int) Option {
	return func(c *config) error {
		if l == 0 {
			return fmt.Errorf("rate limit is zero")
		}
		c.dealMakingRateLim = l
		return nil
	}
}

func WithDealMonitoringRateLim(l int) Option {
	return func(c *config) error {
		if l == 0 {
			return fmt.Errorf("rate limit is zero")
		}
		c.dealMonitoringRateLim = l
		return nil
	}
}

func WithDealReportingRateLim(l int) Option {
	return func(c *config) error {
		if l == 0 {
			return fmt.Errorf("rate limit is zero")
		}
		c.dealReportingRateLim = l
		return nil
	}
}
