#! /bin/sh

quit() {
  echo "$1"
  exit 1
}

[ -z "$POSTGRES_PASSWORD" ] && quit "envvar POSTGRES_PASSWORD is not set"
[ -z "$DB_PASSWORD" ] && quit "envvar DB_PASSWORD is not set. The daemon requires it to connect the database"
[ -z "$DAEMON" ] && quit "envvar DAEMON is not set"

DB_HOST=127.0.0.1:5432
DB_USER=${DAEMON}_user
DB_NAME=${DAEMON}
# just in case if errant newline sneaked into the k8s secret
POSTGRES_PASSWORD=$(echo -n $POSTGRES_PASSWORD | tr -d \\n)
DB_PASSWORD=$(echo -n $DB_PASSWORD | tr -d \\n)

# when starting the pod, connect to postgres with admin privilege, create user
# and database if not exist, and force change password, then starts daemon with
# the correct postgres URI.
psql "postgres://postgres:$POSTGRES_PASSWORD@$DB_HOST" <<EOD
SELECT 'CREATE USER $DB_USER'
WHERE NOT EXISTS (SELECT FROM pg_roles WHERE rolname = '$DB_USER')\gexec
GRANT $DB_USER TO postgres;
SELECT 'CREATE DATABASE $DB_NAME WITH OWNER $DB_USER'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = '$DB_NAME')\gexec
ALTER USER $DB_USER WITH PASSWORD '$DB_PASSWORD'
EOD
unset POSTGRES_PASSWORD
$1 --postgres-uri="postgres://$DB_USER:$DB_PASSWORD@$DB_HOST/$DB_NAME?sslmode=disable&timezone=UTC"
