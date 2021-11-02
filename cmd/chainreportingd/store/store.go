package store

type Store struct{}

func New(postgresURI string) (*Store, error) {
	return &Store{}, nil
}
