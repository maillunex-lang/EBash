#!/usr/bin/env bash
set -euo pipefail

if (( $# < 1 || $# > 2 )) || [[ ! -d $1 || ! ${2:-300} =~ ^[1-9][0-9]{0,5}$ ]]; then
    printf '%s\n' 'usage: check_wal_lag.sh ARCHIVE_DIRECTORY [MAX_SECONDS]' >&2
    exit 2
fi
archive=$1
limit=${2:-300}
pg=${PG_BIN_DIR:?Set PG_BIN_DIR to the directory containing psql}
export PGCONNECT_TIMEOUT=${PGCONNECT_TIMEOUT:-5}
marker="ebash_wal_probe_$(date +%s)_$$_$RANDOM"

# A named restore point emits WAL without changing any game records.
started_ms=$(date +%s%3N)
segment=$(timeout "${limit}s" "$pg/psql" -XAt --no-password -v ON_ERROR_STOP=1 \
    -c "SELECT pg_walfile_name(pg_create_restore_point('$marker'))")
if [[ ! $segment =~ ^[0-9A-F]{24}$ ]]; then
    printf '%s\n' 'PostgreSQL did not return a WAL segment name' >&2
    exit 1
fi

deadline_ms=$((started_ms + limit * 1000))
while true; do
    if [[ -s "$archive/$segment" ]]; then
        elapsed_ms=$(( $(date +%s%3N) - started_ms ))
        if (( elapsed_ms <= limit * 1000 )); then
            printf 'PASS: WAL %s visible after %s ms (limit %s s)\n' "$segment" "$elapsed_ms" "$limit"
            exit 0
        fi
    fi
    if (( $(date +%s%3N) >= deadline_ms )); then
        printf 'FAIL: WAL %s did not reach the archive within %s s\n' "$segment" "$limit" >&2
        exit 1
    fi
    sleep 0.2
done
