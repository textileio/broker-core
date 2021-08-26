#! /bin/sh
set -x

quit() {
  echo "$1"
  exit 1
}

[ -z "$POSTGRES_PASSWORD" ] && quit "envvar POSTGRES_PASSWORD is not set"
[ -z "$DB_PASSWORD" ] && quit "envvar DB_PASSWORD is not set. The daemon requires it to connect the database"
[ -z "$DAEMON" ] && quit "envvar DAEMON is not set"

POSTGRES_USER="${POSTGRES_USER:-postgres}"
DB_HOST="${DB_HOST:-127.0.0.1:5432}"
DB_NAME="${DB_NAME:-postgres}"
DB_USER=${DAEMON}
# just in case if errant newline sneaked into the k8s secret
POSTGRES_PASSWORD=$(echo -n $POSTGRES_PASSWORD | tr -d \\n)
DB_PASSWORD=$(echo -n $DB_PASSWORD | tr -d \\n)

# when starting the pod, connect to postgres with admin privilege, create user
# and schema if not exist, and force change password, then starts daemon with
# the correct postgres URI. The use is restricted to the schema with the same name.
psql "postgres://$POSTGRES_USER:$POSTGRES_PASSWORD@$DB_HOST" <<EOD
SELECT 'CREATE DATABASE $DB_NAME WITH OWNER $POSTGRES_USER'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = '$DB_NAME')\gexec
\connect $DB_NAME;
SELECT 'CREATE USER $DB_USER'
WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname = '$DB_USER')\gexec
GRANT $DB_USER TO $POSTGRES_USER;
ALTER USER $DB_USER WITH PASSWORD '$DB_PASSWORD';
SELECT 'CREATE SCHEMA AUTHORIZATION $DB_USER'
WHERE NOT EXISTS (SELECT FROM information_schema.schemata WHERE schema_name = '$DB_USER')\gexec
REVOKE CREATE ON SCHEMA public FROM PUBLIC;
EOD
[ $? -eq 0 ] || quit "fail to initialize postgres database"
unset POSTGRES_PASSWORD
unset POSTGRES_USER
$@ --postgres-uri="postgres://$DB_USER:$DB_PASSWORD@$DB_HOST/$DB_NAME?sslmode=disable&search_path=${DB_USER}&timezone=UTC"
