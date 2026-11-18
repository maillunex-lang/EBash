#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"
pg=${PG_BIN_DIR:?Set PG_BIN_DIR to the directory with PostgreSQL server and client tools}
pg=$(cd "$pg" && pwd)
mkdir -p .local
lab=$(mktemp -d "$PWD/.local/restore.XXXXXX")
archive="$lab/archive"
mkdir "$archive"

stop_clusters() {
    for cluster in recovered primary; do
        if [[ -f "$lab/$cluster/postmaster.pid" ]]; then
            "$pg/pg_ctl" -D "$lab/$cluster" -m fast -w stop
        fi
    done
}
trap stop_clusters EXIT

port=55433
restored_port=55434
primary_dsn="host=$lab port=$port dbname=ebash sslmode=disable"
restored_dsn="host=$lab port=$restored_port dbname=ebash sslmode=disable"

go build -o "$lab/ebash-p2" .
"$pg/initdb" -D "$lab/primary" --encoding=UTF8 --no-locale -A trust > "$lab/init.log"
cat >> "$lab/primary/postgresql.conf" <<CONFIG
listen_addresses = ''
unix_socket_directories = '$lab'
port = $port
wal_level = replica
archive_mode = on
archive_timeout = ${EBASH_TEST_ARCHIVE_TIMEOUT:-60}
archive_command = 'bash "$PWD/archive_wal.sh" "$archive" "%p" "%f"'
CONFIG
"$pg/pg_ctl" -D "$lab/primary" -l "$lab/primary.log" -w start
"$pg/createdb" -h "$lab" -p "$port" ebash
export EBASH_DATABASE_URL="$primary_dsn"
export EBASH_PILOT_TOKEN='p2-restore-isolated-account'
"$lab/ebash-p2" migrate
"$lab/ebash-p2" seed
"$lab/ebash-p2" tick -tick 1

backup=$(PGHOST="$lab" PGPORT="$port" PGDATABASE=ebash PG_BIN_DIR="$pg"     EBASH_BACKUP_CHECKPOINT=fast bash base_backup.sh "$lab")
mv -- "$backup" "$lab/recovered"
if [[ ${EBASH_TEST_CORRUPT_BACKUP:-0} == 1 ]]; then
    cp -a --reflink=auto "$lab/recovered" "$lab/corrupt"
    printf 'corruption' >> "$lab/corrupt/PG_VERSION"
    if "$pg/pg_verifybackup" "$lab/corrupt" >"$lab/corrupt-check.log" 2>&1; then
        printf '%s\n' 'FAIL: damaged copy passed manifest verification' >&2
        exit 1
    fi
    grep -F 'PG_VERSION' "$lab/corrupt-check.log"
    printf '%s\n' 'PASS: manifest rejected damaged copy'
fi
# Insert replay inputs after the base backup so they must be recovered from WAL.
# The HTTP admission path is exercised separately by the integration test.
"$pg/psql" "$primary_dsn" -X -v ON_ERROR_STOP=1 <<'SQL'
INSERT INTO commands(world, account, id, body, target) VALUES
('pilot', 'pilot', 'buy',
 '{"command_id":"buy","type":"order_raw","payload":{"offer_id":"pilot-inputs-v1","quantity":"1","max_price_kopecks":"48000"}}', 2),
('pilot', 'pilot', 'line',
 '{"command_id":"line","type":"configure_line","payload":{"enterprise_id":"pilot-bakery","cell_id":"4","profile_id":"test-bread-v1"}}', 2);
SQL
"$lab/ebash-p2" tick -tick 2
archive_started=$SECONDS
segment=$("$pg/psql" "$primary_dsn" -XAt -v ON_ERROR_STOP=1 -c     "SELECT pg_walfile_name(pg_create_restore_point('ebash_tick_2'))")
"$lab/ebash-p2" tick -tick 3
archive_deadline=$((SECONDS + ${EBASH_TEST_ARCHIVE_TIMEOUT:-60} + 30))
until [[ -f "$archive/$segment" ]]; do
    if (( SECONDS >= archive_deadline )); then
        tail -30 "$lab/primary.log" >&2
        exit 1
    fi
    sleep 0.2
done
printf 'PASS: target WAL archived in %s s without a manual switch\n' "$((SECONDS - archive_started))"
if [[ ${EBASH_TEST_WAL_PROBE:-0} == 1 ]]; then
    export PGHOST="$lab" PGPORT="$port" PGDATABASE=ebash PG_BIN_DIR="$pg"
    bash check_wal_lag.sh "$archive" 30
    mkdir "$lab/empty-archive"
    set +e
    failure=$(bash check_wal_lag.sh "$lab/empty-archive" 1 2>&1)
    probe_status=$?
    set -e
    [[ $probe_status == 1 && $failure == *'did not reach the archive'* ]]
    printf '%s\n' 'PASS: delayed or missing WAL exceeds the probe deadline'
fi
"$pg/pg_ctl" -D "$lab/primary" -m fast -w stop

if [[ ${EBASH_RESTORE_EXPORT:-0} == 1 ]]; then
    tar --zstd -cf "$lab/restore-fixture.tar.zst" -C "$lab" recovered archive
    (cd "$lab" && sha256sum restore-fixture.tar.zst > restore-fixture.tar.zst.sha256)
    printf 'Synthetic restore fixture: %s\n' "$lab/restore-fixture.tar.zst"
fi

cat >> "$lab/recovered/postgresql.conf" <<CONFIG
port = $restored_port
archive_mode = off
restore_command = 'cp "$archive/%f" "%p"'
recovery_target_name = 'ebash_tick_2'
recovery_target_action = 'promote'
CONFIG
touch "$lab/recovered/recovery.signal"
"$pg/pg_ctl" -D "$lab/recovered" -l "$lab/recovered.log" -w start

check() {
    local actual
    actual=$("$pg/psql" "$restored_dsn" -XAt -v ON_ERROR_STOP=1 -c "$1")
    if [[ "$actual" != "$2" ]]; then
        printf 'Expected %s, got %s\n' "$2" "$actual" >&2
        exit 1
    fi
}

# pg_ctl can return when the standby is accepting read-only connections.
# Wait for promotion before checking the recovered world or writing to it.
promotion_deadline=$((SECONDS + 30))
until [[ $("$pg/psql" "$restored_dsn" -XAt -c "SELECT NOT pg_is_in_recovery()") == t ]]; do
    if (( SECONDS >= promotion_deadline )); then
        tail -30 "$lab/recovered.log" >&2
        exit 1
    fi
    sleep 0.2
done
check "SELECT last_tick FROM worlds WHERE id='pilot'" 2
check "SELECT count(*) FROM ticks WHERE world='pilot' AND number>2" 0
check "SELECT state->>'cash_kopecks' FROM companies WHERE world='pilot'" 152000
export EBASH_DATABASE_URL="$restored_dsn"
"$lab/ebash-p2" recover -tick 14
"$lab/ebash-p2" recover -tick 14
check "SELECT last_tick || ':' || recovering FROM worlds WHERE id='pilot'" '14:false'
check "SELECT count(*) FROM ticks WHERE world='pilot'" 14
check "SELECT state->>'cash_kopecks' FROM companies WHERE world='pilot'" 126000
check "SELECT state->>'flour_g' FROM companies WHERE world='pilot'" 200
check "SELECT state->>'bread_count' FROM companies WHERE world='pilot'" 24
check "SELECT count(*) FROM ledger WHERE world='pilot'" 6
printf 'PASS: base backup + WAL recovered to tick 2, then caught up 12 ticks exactly once\n'
