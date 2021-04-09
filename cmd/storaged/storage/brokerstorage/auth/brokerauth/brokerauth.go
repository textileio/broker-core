package brokerauth

type BrokerAuth struct {
}

func New() (*BrokerAuth, error) {
	return &BrokerAuth{}, nil
}

func (bs *BrokerAuth) IsAuthorized(identity string) (bool, string, error) {
	// TODO: Fill this implementation when the Auth API is ready.
	// For now, authorize every request.

	return true, "", nil
}
