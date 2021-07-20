package storeutil

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" /*nolint*/
	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"
	_ "github.com/jackc/pgx/v4/stdlib" /*nolint*/
	logger "github.com/textileio/go-log/v2"
)

var (
	log = logger.Logger("storeutil")
)

// MigrateAndConnectToDB run db migrations and return a ready to use connection to the Postgres database.
func MigrateAndConnectToDB(postgresURI string, as *bindata.AssetSource) (*sql.DB, error) {
	// To avoid dealing with time zone issues, we just enforce UTC timezone
	if !strings.Contains(postgresURI, "timezone=UTC") {
		return nil, errors.New("timezone=UTC is required in postgres URI")
	}
	d, err := bindata.WithInstance(as)
	if err != nil {
		return nil, fmt.Errorf("creating source driver: %s", err)
	}
	m, err := migrate.NewWithSourceInstance("go-bindata", d, postgresURI)
	if err != nil {
		return nil, fmt.Errorf("creating migration: %s", err)
	}
	version, dirty, err := m.Version()
	log.Debugf("current version %d, dirty %v, err: %s", version, dirty, err)
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return nil, fmt.Errorf("running migration up: %s", err)
	}
	conn, err := sql.Open("pgx", postgresURI)
	if err != nil {
		return nil, fmt.Errorf("creating pgx connection: %s", err)
	}

	return conn, nil
}
