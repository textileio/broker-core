package storeutil

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" /*nolint*/
	bindata "github.com/golang-migrate/migrate/v4/source/go_bindata"
	_ "github.com/jackc/pgx/v4/stdlib" /*nolint*/
)

func MigrateAndConnectToDB(postgresURI string, as *bindata.AssetSource) (*sql.DB, error) {
	// To avoid dealing with time zone issues, we just enforce UTC timezone
	if !strings.Contains(postgresURI, "timezone=UTC") {
		return nil, errors.New("timezone=UTC is required in postgres URI")
	}
	d, err := bindata.WithInstance(as)
	if err != nil {
		return nil, err
	}
	m, err := migrate.NewWithSourceInstance("go-bindata", d, postgresURI)
	if err != nil {
		return nil, err
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		return nil, err
	}
	conn, err := sql.Open("pgx", postgresURI)
	if err != nil {
		return nil, err
	}

	return conn, nil
}
