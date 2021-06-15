package broker

import (
	"errors"
	"time"

	"github.com/textileio/broker-core/broker"
)

var defaultConfig = config{
	dealDuration:    broker.MaxDealDuration,
	dealReplication: broker.MinDealReplication,
	verifiedDeals:   true,

	unpinnerFrequency:  time.Minute * 5,
	unpinnerRetryDelay: time.Minute,
}

type config struct {
	dealDuration    uint64
	dealReplication uint32
	verifiedDeals   bool

	unpinnerFrequency  time.Duration
	unpinnerRetryDelay time.Duration
}

// Option provides configuration for Broker.
type Option func(*config) error

// WithDealDuration configures the default deal duration of new auctions.
func WithDealDuration(duration uint64) Option {
	return func(c *config) error {
		if duration <= 0 {
			return errors.New("deal duration must be positive")
		}
		c.dealDuration = duration
		return nil
	}
}

// WithDealReplication configures the default replication factor of new auctions.
func WithDealReplication(repFactor uint32) Option {
	return func(c *config) error {
		if repFactor <= 0 {
			return errors.New("rep factor must be positive")
		}
		c.dealReplication = repFactor
		return nil
	}
}

// WithVerifiedDeals configures the default deal type in of new auctions.
func WithVerifiedDeals(verifiedDeals bool) Option {
	return func(c *config) error {
		c.verifiedDeals = verifiedDeals
		return nil
	}
}

// WithUnpinnerFrequency configures the frequency of the GC daemon.
func WithUnpinnerFrequency(freq time.Duration) Option {
	return func(c *config) error {
		if freq.Seconds() == 0 {
			return errors.New("unpinner frequency must be positive")
		}
		c.unpinnerFrequency = freq
		return nil
	}
}

// WithUnpinnerRetryDelay configures the default delay for failed unpin jobs to be retried.
func WithUnpinnerRetryDelay(delay time.Duration) Option {
	return func(c *config) error {
		if delay.Seconds() == 0 {
			return errors.New("unpinner retry delay must be positive")
		}
		c.unpinnerRetryDelay = delay
		return nil
	}
}
