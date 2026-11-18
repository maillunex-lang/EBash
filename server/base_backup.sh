#!/usr/bin/env bash
set -euo pipefail
umask 077

if (( $# != 1 )) || [[ ! -d $1 ]]; then
    printf '%s\n' 'usage: base_backup.sh EXISTING_BACKUP_DIRECTORY' >&2
    exit 2
fi

destination=$(cd "$1" && pwd)
pg=${PG_BIN_DIR:?Set PG_BIN_DIR to the directory with PostgreSQL server and client tools}
stamp=$(date -u +%Y%m%dT%H%M%SZ)
staging=$(mktemp -d -- "$destination/.backup-$stamp.XXXXXXXX")
published="$destination/${staging##*/.}"

# Connection settings come from libpq's PG* environment or service config.
# An interrupted run stays hidden and must never be treated as a usable copy.
"$pg/pg_basebackup" --pgdata="$staging" --format=plain --wal-method=stream \
    --checkpoint="${EBASH_BACKUP_CHECKPOINT:-spread}" --manifest-checksums=SHA256
"$pg/pg_verifybackup" "$staging" >&2
sync -f -- "$staging"
mv -T -- "$staging" "$published"
sync -- "$destination"
printf '%s\n' "$published"
