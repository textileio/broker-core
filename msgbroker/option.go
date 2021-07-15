package msgbroker

import (
	"fmt"
	"time"
)

// DefaultRegisterHandlerConfig is the default configuration for topic subscriptions.
var DefaultRegisterHandlerConfig = RegisterHandlerConfig{
	AckDeadline: time.Second * 10,
}

// RegisterHandlerConfig contains configuration for topic subscriptions.
type RegisterHandlerConfig struct {
	AckDeadline time.Duration
}

// Option applies a configuration on RegisterHandlerConfig.
type Option func(*RegisterHandlerConfig) error

// WithACKDeadline configures the deadline for the message broker subscription.
// The duration can't be greater than 10min.
func WithACKDeadline(deadline time.Duration) Option {
	return func(c *RegisterHandlerConfig) error {
		c.AckDeadline = deadline
		return nil
	}
}

// ApplyRegisterHandlerOptions applies a list of Option to the default configuration.
func ApplyRegisterHandlerOptions(opts ...Option) (RegisterHandlerConfig, error) {
	config := DefaultRegisterHandlerConfig
	for _, opt := range opts {
		if err := opt(&config); err != nil {
			return RegisterHandlerConfig{}, fmt.Errorf("applying option: %s", err)
		}
	}

	return config, nil
}
