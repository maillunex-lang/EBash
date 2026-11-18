#!/usr/bin/env bash
set -euo pipefail
umask 077

if (( $# != 3 )); then
    printf '%s\n' 'usage: archive_wal.sh ARCHIVE_DIRECTORY SOURCE FILE_NAME' >&2
    exit 2
fi

archive=$1
source=$2
name=$3
if [[ ! $name =~ ^[0-9A-F]{24}(\.[0-9A-F]{8}\.backup|\.partial)?$ &&
      ! $name =~ ^[0-9A-F]{8}\.history$ ]]; then
    printf 'invalid WAL archive filename: %s\n' "$name" >&2
    exit 2
fi
if [[ ! -d $archive || ! -f $source ]]; then
    printf '%s\n' 'WAL source or archive directory is unavailable' >&2
    exit 1
fi

destination="$archive/$name"
temporary=$(mktemp -- "$archive/.wal-$name.XXXXXXXX")
trap 'rm -f -- "$temporary"' EXIT
cp -- "$source" "$temporary"
sync -- "$temporary"

# A hard link publishes the complete file without replacing an existing one.
if ! ln -T -- "$temporary" "$destination" 2>/dev/null; then
    if [[ ! -f $destination ]] || ! cmp -s -- "$temporary" "$destination"; then
        printf 'cannot publish WAL or archive conflict: %s\n' "$name" >&2
        exit 1
    fi
fi
sync -- "$destination"
sync -- "$archive"
