package msgbroker

import "time"

var DefaultRegisterHandlerConfig = RegisterHandlerConfig{
	AckDeadline: time.Second * 10,
}

type RegisterHandlerConfig struct {
	AckDeadline time.Duration
}

type Option func(*RegisterHandlerConfig) error

// WithACKDeadline configures the deadline for the message broker subscription.
func WithACKDeadline(deadline time.Duration) Option {
	return func(c *RegisterHandlerConfig) error {
		c.AckDeadline = deadline
		return nil
	}
}

func ApplyRegisterHandlerOptions(opts ...Option) RegisterHandlerConfig {
	config := DefaultRegisterHandlerConfig
	for _, opt := range opts {
		opt(&config)
	}

	return config
}
