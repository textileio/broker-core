package msgbroker

import "time"

type config struct {
	ackDeadline time.Duration
}

type Option func(*config) error

// WithACKDeadline configures the deadline for the message broker subscription.
func WithACKDeadline(deadline time.Duration) Option {
	return func(c *config) error {
		c.ackDeadline = deadline
		return nil
	}
}
