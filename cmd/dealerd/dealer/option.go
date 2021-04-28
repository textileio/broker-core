package dealer

type config struct {
}

var defaultConfig = config{}

// Option applies a configuration change.
type Option func(*config) error
