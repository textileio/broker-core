package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"
	"github.com/textileio/broker-core/cmd/authd/store/internal/db"
	"github.com/textileio/broker-core/cmd/authd/store/migrations"
	"github.com/textileio/broker-core/storeutil"
)

// AuthToken is an authorization token.
type AuthToken db.AuthToken

// Store is a store for authentication information.
type Store struct {
	conn *sql.DB
	db   *db.Queries
}

// New returns a new Store.
func New(postgresURI string) (*Store, error) {
	as := bindata.Resource(migrations.AssetNames(),
		func(name string) ([]byte, error) {
			return migrations.Asset(name)
		})
	conn, err := storeutil.MigrateAndConnectToDB(postgresURI, as)
	if err != nil {
		return nil, fmt.Errorf("initializing db connection: %s", err)
	}

	s := &Store{
		conn: conn,
		db:   db.New(conn),
	}

	return s, nil
}

// GetAuthToken retrieves an authentication token information.
func (s *Store) GetAuthToken(ctx context.Context, token string) (AuthToken, bool, error) {
	if token == "" {
		return AuthToken{}, false, errors.New("raw token is empty")
	}
	rt, err := s.db.GetAuthToken(ctx, token)
	if err == sql.ErrNoRows {
		return AuthToken{}, false, nil
	}
	if err != nil {
		return AuthToken{}, false, fmt.Errorf("db get raw token: %s", err)
	}

	return AuthToken(rt), true, nil
}

// CreateAuthToken creates a new authentication token.
func (s *Store) CreateAuthToken(ctx context.Context, token, identity, origin string) error {
	if identity == "" {
		return errors.New("identity is empty")
	}
	if token == "" {
		return errors.New("token is empty")
	}
	if origin == "" {
		return errors.New("origin is empty")
	}
	params := db.CreateAuthTokenParams{
		Token:    token,
		Identity: identity,
		Origin:   origin,
	}
	err := s.db.CreateAuthToken(ctx, params)
	if err != nil {
		return fmt.Errorf("db create auth token: %s", err)
	}

	return nil
}

// Close closes the store.
func (s *Store) Close() error {
	if err := s.conn.Close(); err != nil {
		return fmt.Errorf("closing sql connection: %s", err)
	}
	return nil
}
