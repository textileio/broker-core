package broker

import (
	"errors"
	"time"

	"github.com/filecoin-project/go-address"
	"github.com/textileio/bidbot/lib/auction"
	"github.com/textileio/broker-core/broker"
)

var defaultConfig = config{
	dealDuration:    auction.MaxDealDuration,
	dealReplication: broker.MinDealReplication,
	verifiedDeals:   true,

	unpinnerFrequency:       time.Minute * 5,
	unpinnerRetryDelay:      time.Minute,
	exportPinCountFrequency: time.Minute * 30,

	auctionMaxRetries:       5,
	defaultDeadlineDuration: 3 * 24 * time.Hour,
}

type config struct {
	dealDuration      uint64
	dealReplication   uint32
	defaultWalletAddr address.Address
	verifiedDeals     bool

	unpinnerFrequency       time.Duration
	unpinnerRetryDelay      time.Duration
	exportPinCountFrequency time.Duration

	auctionMaxRetries       int
	defaultDeadlineDuration time.Duration
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

// WithDefaultWalletAddress configures the default wallet address of new auctions.
func WithDefaultWalletAddress(walletAddress address.Address) Option {
	return func(c *config) error {
		c.defaultWalletAddr = walletAddress
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

// WithAuctionMaxRetries indicates the maximum number of auctions that can be created
// for a batch.
func WithAuctionMaxRetries(max int) Option {
	return func(c *config) error {
		if max <= 0 {
			return errors.New("auction max number of retries should be positive")
		}
		c.auctionMaxRetries = max
		return nil
	}
}

// WithAuctionDeadlineDuration indicates the auction duration to be used in
// every auction (includes re-auctioning). If a new auction has to be created
// that can't fit into the Batch specified deadline, then it won't be created
// and the Batch would be consider un-auctionable and thus fail.
func WithAuctionDeadlineDuration(duration time.Duration) Option {
	return func(c *config) error {
		if duration == 0 {
			return errors.New("auction deadline duration must be positive")
		}
		c.defaultDeadlineDuration = duration
		return nil
	}
}
