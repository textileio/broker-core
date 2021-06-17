package broker

import (
	"errors"
	"fmt"
	"net/url"
	"time"

	"github.com/textileio/broker-core/broker"
)

var defaultConfig = config{
	dealDuration:    broker.MaxDealDuration,
	dealReplication: broker.MinDealReplication,
	verifiedDeals:   true,

	unpinnerFrequency:       time.Minute * 5,
	unpinnerRetryDelay:      time.Minute,
	exportPinCountFrequency: time.Minute * 15,
}

type config struct {
	dealDuration    uint64
	dealReplication uint32
	verifiedDeals   bool

	unpinnerFrequency       time.Duration
	unpinnerRetryDelay      time.Duration
	exportPinCountFrequency time.Duration

	carExportURL *url.URL
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

// WithExportPinCountFrequency configures the frequency of exporting the pin count metric.
func WithExportPinCountFrequency(freq time.Duration) Option {
	return func(c *config) error {
		if freq.Seconds() == 0 {
			return errors.New("export pin count frequency must be positive")
		}
		c.exportPinCountFrequency = freq
		return nil
	}
}

// WithCAR configures the frequency of exporting the pin count metric.
func WithCARExportURL(rawURL string) Option {
	return func(c *config) error {
		u, err := url.Parse(rawURL)
		if err != nil {
			return fmt.Errorf("parsing url: %s", err)
		}
		c.carExportURL = u
		return nil
	}
}

func (c config) validate() error {
	if c.carExportURL == nil {
		return fmt.Errorf("the CAR exporting URL can't be empty")
	}
	return nil
}
