#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
pg=${PG_BIN_DIR:-$PWD/.local/pg/usr/lib/postgresql/18/bin}
mkdir -p .local
lab=$(mktemp -d "$PWD/.local/p32-db.XXXXXX")
cleanup() { if [[ -f "$lab/db/postmaster.pid" ]]; then "$pg/pg_ctl" -D "$lab/db" -m fast -w stop >/dev/null; fi; }
trap cleanup EXIT
"$pg/initdb" -D "$lab/db" --encoding=UTF8 --no-locale -A trust >"$lab/init.log"
"$pg/pg_ctl" -D "$lab/db" -l "$lab/postgres.log" -o "-k $lab -c listen_addresses='' -p 55467" -w start >/dev/null
"$pg/createdb" -h "$lab" -p 55467 ebash_p32
export EBASH_TEST_DATABASE_URL="host=$lab port=55467 dbname=ebash_p32 sslmode=disable"
go test -count=1 -run '^(TestP32.*|TestGameProfilesPostgres.*)$' -v .
