package dealer

import (
	"errors"
	"time"
)

type config struct {
	dealMakingFreq       time.Duration
	dealMakingRateLim    int
	dealMakingMaxRetries int
	dealMakingRetryDelay time.Duration

	dealWatchingFreq                    time.Duration
	dealWatchingRateLim                 int
	dealWatchingResolveDealIDRetryDelay time.Duration
	dealWatchingCheckChainRetryDelay    time.Duration

	dealReportingFreq       time.Duration
	dealReportingRetryDelay time.Duration

	exportStatusesCountFrequency time.Duration
}

var defaultConfig = config{
	dealMakingFreq:       time.Second * 10,
	dealMakingRateLim:    10,
	dealMakingMaxRetries: 50,
	dealMakingRetryDelay: time.Second * 20,

	dealWatchingFreq:                    time.Second * 20,
	dealWatchingRateLim:                 20,
	dealWatchingResolveDealIDRetryDelay: time.Second * 30,
	dealWatchingCheckChainRetryDelay:    time.Minute,

	dealReportingFreq:       time.Second * 10,
	dealReportingRetryDelay: time.Second * 15,

	exportStatusesCountFrequency: time.Minute * 5,
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

// WithDealMakingRateLim configures the max number of parallel execution items.
func WithDealMakingRateLim(l int) Option {
	return func(c *config) error {
		if l == 0 {
			return errors.New("rate limit is zero")
		}
		c.dealMakingRateLim = l
		return nil
	}
}

// WithDealMakingMaxRetries specifies the maximum amount of retries on recoverable
// errros during deal making execution.
func WithDealMakingMaxRetries(max int) Option {
	return func(c *config) error {
		if max < 0 {
			return errors.New("max retries must be positive")
		}
		c.dealMakingMaxRetries = max
		return nil
	}
}

// WithDealWatchingFreq configures the frequency of deal monitoring polling.
func WithDealWatchingFreq(f time.Duration) Option {
	return func(c *config) error {
		if f == 0 {
			return errors.New("frequency is zero")
		}
		c.dealWatchingFreq = f
		return nil
	}
}

// WithDealWatchingRateLim configures the max number of parallel execution items.
func WithDealWatchingRateLim(l int) Option {
	return func(c *config) error {
		if l == 0 {
			return errors.New("rate limit is zero")
		}
		c.dealWatchingRateLim = l
		return nil
	}
}

// WithDealWatchingResolveDealIDRetryDelay specifies how much time to delay retrying
// resolving a new deal-id.
func WithDealWatchingResolveDealIDRetryDelay(delay time.Duration) Option {
	return func(c *config) error {
		c.dealWatchingResolveDealIDRetryDelay = delay
		return nil
	}
}

// WithDealMakingRetryDelay specifies how much time retries are delayed for
// reprocessing.
func WithDealMakingRetryDelay(delay time.Duration) Option {
	return func(c *config) error {
		c.dealMakingRetryDelay = delay
		return nil
	}
}

// WithDealWatchingCheckChainRetryDelay specifies how much time to delay retrying
// checking on-chain for a deal confirmation.
func WithDealWatchingCheckChainRetryDelay(delay time.Duration) Option {
	return func(c *config) error {
		c.dealWatchingCheckChainRetryDelay = delay
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

// WithDealReportingRetryDelay specifies how much time to delay retrying
// reporting results to the broker.
func WithDealReportingRetryDelay(delay time.Duration) Option {
	return func(c *config) error {
		c.dealReportingRetryDelay = delay
		return nil
	}
}
