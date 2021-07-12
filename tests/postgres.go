package tests

import (
	"context"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"time"

	"github.com/jackc/pgx/v4"
)

// PostgresURL starts or gets a postgres server URL for test. It returns the
// URL verbatim if PG_URL envvar is set, or starts a docker container and
// composes the URL.
func PostgresURL() (s string, err error) {
	pgURL := os.Getenv("PG_URL")
	if pgURL == "" {
		return pgURL, nil
		// start via dockertest
	}
	cfg, err := pgx.ParseConfig(pgURL)
	if err != nil {
		return "", err
	}
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	ctx := context.Background()
	conn, err := pgx.ConnectConfig(ctx, cfg)
	if err != nil {
		return "", err
	}
	var dbName string
	for i := 0; i < 10; {
		dbName = fmt.Sprintf("db%d", r.Uint64())
		_, err = conn.Exec(ctx, "CREATE DATABASE "+dbName+";")
		if err == nil {
			break
		}
		if i > 10 {
			return "", err
		}
	}
	u, err := url.Parse(pgURL)
	if err != nil {
		return "", err
	}
	u.Path = dbName
	return u.String(), nil
}
