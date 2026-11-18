#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "$0")"
pg=${PG_BIN_DIR:?Set PG_BIN_DIR to the directory with PostgreSQL server tools}
godot=${GODOT_BIN:?Set GODOT_BIN to the Godot 4.7.2 executable}
mkdir -p .local
lab=$(mktemp -d "$PWD/.local/godot-pilot.XXXXXX")
api_pid=

cleanup() {
    if [[ -n $api_pid ]]; then
        kill "$api_pid" 2>/dev/null || true
        wait "$api_pid" 2>/dev/null || true
    fi
    if [[ -f "$lab/db/postmaster.pid" ]]; then
        "$pg/pg_ctl" -D "$lab/db" -m fast -w stop
    fi
}
trap cleanup EXIT

"$pg/initdb" -D "$lab/db" --encoding=UTF8 --no-locale -A trust >"$lab/init.log"
"$pg/pg_ctl" -D "$lab/db" -l "$lab/db.log" -o "-k $lab -c listen_addresses='' -p 55436" -w start
"$pg/createdb" -h "$lab" -p 55436 ebash
go build -o "$lab/ebash-p2" .
export EBASH_DATABASE_URL="host=$lab port=55436 dbname=ebash sslmode=disable"
export EBASH_PILOT_TOKEN
EBASH_PILOT_TOKEN=$(openssl rand -hex 24)
export EBASH_PILOT_WORLD=godot-pilot
mode=${EBASH_PILOT_MODE:-smoke}
unset EBASH_DEMO_PROFILES EBASH_DEMO_CONSTRUCTION
if [[ $mode == profile-demo || $mode == construction-smoke ]]; then export EBASH_DEMO_CONSTRUCTION=1; fi
case "$mode" in
    profile-demo|profile-demo-smoke|construction-smoke)
        export EBASH_DEMO_WORLD=bakery-profile-demo
        export EBASH_DEMO_PROFILES="$PWD/../docs/catalogs/game-production-profiles.v1.json"
        EBASH_PILOT_WORLD="$EBASH_DEMO_WORLD"
        ;;
    demo|demo-smoke)
        export EBASH_DEMO_WORLD=bakery-demo
        EBASH_PILOT_WORLD="$EBASH_DEMO_WORLD"
        ;;
    smoke)
        unset EBASH_DEMO_WORLD
        ;;
    *)
        echo "Unknown EBASH_PILOT_MODE: $mode" >&2
        exit 2
        ;;
esac
export EBASH_PILOT_SERVER_BIN="$lab/ebash-p2"
listen="127.0.0.1:${EBASH_PILOT_PORT:-58642}"
export EBASH_PILOT_API="http://$listen"

"$EBASH_PILOT_SERVER_BIN" migrate
"$EBASH_PILOT_SERVER_BIN" seed -world "$EBASH_PILOT_WORLD"
if [[ $mode != smoke ]]; then
    for supplier in pilot-mill pilot-mill-remote; do
        EBASH_PILOT_TOKEN="$(openssl rand -hex 24)" "$EBASH_PILOT_SERVER_BIN" seed -world "$EBASH_PILOT_WORLD" -account "$supplier"
    done
fi
"$EBASH_PILOT_SERVER_BIN" serve -listen "$listen" >"$lab/http.log" 2>&1 &
api_pid=$!
ready=false
for attempt in {1..30}; do
    if curl -fsS --max-time 2 -H "Authorization: Bearer $EBASH_PILOT_TOKEN" \
        "$EBASH_PILOT_API/v1/worlds/$EBASH_PILOT_WORLD/status" >/dev/null 2>&1; then
        ready=true
        break
    fi
    kill -0 "$api_pid" 2>/dev/null || break
    sleep 0.1
done
if [[ $ready != true ]]; then
    cat "$lab/http.log" >&2
    exit 1
fi
if [[ $mode == demo || $mode == profile-demo ]]; then
    "$godot" --path ../prototypes/godot res://Launcher.tscn
    exit
fi
if [[ $mode == construction-smoke ]]; then
 timeout 45s "$godot" --headless --path ../prototypes/godot --script construction_smoke.gd | tee "$lab/construction.log"
 grep -Fx "CONSTRUCTION OK" "$lab/construction.log"
 exit
fi
if [[ $mode == profile-demo-smoke ]]; then
    timeout 45s "$godot" --headless --path ../prototypes/godot --script pilot_smoke.gd | tee "$lab/profile-demo.log"
    grep -Fx 'E104 DEMO SMOKE OK: two suppliers, delivered stock and sales' "$lab/profile-demo.log"
    exit
fi
if [[ $mode == demo-smoke ]]; then
    timeout 35s "$godot" --headless --path ../prototypes/godot --script pilot_smoke.gd | tee "$lab/demo.log"
    grep -Fx 'BAKERY DEMO SMOKE OK: two batches using only game controls' "$lab/demo.log"
    exit
fi
if [[ ${EBASH_TEST_MARKET_CONTENTION:-0} == 1 ]]; then
    EBASH_TEST_DATABASE_URL="$EBASH_DATABASE_URL" go test -count=1 -run '^TestPostgresMarketContention$' -v .
fi
timeout 35s "$godot" --headless --path ../prototypes/godot --script pilot_smoke.gd | tee "$lab/godot.log"
grep -Fx 'PILOT SMOKE OK: buy, configure, four ticks, sale, repeat tick and reopen' "$lab/godot.log"

index=0
for sector in milk wood glass vegetables construction oil; do
    index=$((index + 1))
    export EBASH_PILOT_CHAIN_INDEX="$index"
    export EBASH_PILOT_WORLD="godot-chain-$index"
    EBASH_PILOT_TOKEN=$(openssl rand -hex 24)
    "$EBASH_PILOT_SERVER_BIN" seed -world "$EBASH_PILOT_WORLD"
    for distance in near remote; do
        EBASH_PILOT_TOKEN="$(openssl rand -hex 24)" "$EBASH_PILOT_SERVER_BIN" seed -world "$EBASH_PILOT_WORLD" -account "supplier-$sector-$distance"
    done
    timeout 120s "$godot" --headless --path ../prototypes/godot --script pilot_smoke.gd | tee "$lab/chain-$index.log"
    grep -q '^CHAIN SMOKE OK:' "$lab/chain-$index.log"
done

export EBASH_PILOT_CHAIN_INDEX=0
export EBASH_PILOT_WORLD=godot-mill-trade
export EBASH_PILOT_MILL_TOKEN
EBASH_PILOT_TOKEN=$(openssl rand -hex 24)
EBASH_PILOT_MILL_TOKEN=$(openssl rand -hex 24)
"$EBASH_PILOT_SERVER_BIN" seed -world "$EBASH_PILOT_WORLD"
EBASH_PILOT_TOKEN="$EBASH_PILOT_MILL_TOKEN" "$EBASH_PILOT_SERVER_BIN" seed -world "$EBASH_PILOT_WORLD" -account pilot-mill
export EBASH_PILOT_COMPETITOR_TOKEN
EBASH_PILOT_COMPETITOR_TOKEN=$(openssl rand -hex 24)
EBASH_PILOT_TOKEN="$EBASH_PILOT_COMPETITOR_TOKEN" "$EBASH_PILOT_SERVER_BIN" seed -world "$EBASH_PILOT_WORLD" -account rival-bakery
EBASH_PILOT_TOKEN="$(openssl rand -hex 24)" "$EBASH_PILOT_SERVER_BIN" seed -world "$EBASH_PILOT_WORLD" -account pilot-mill-remote
timeout 120s "$godot" --headless --path ../prototypes/godot --script pilot_smoke.gd | tee "$lab/mill-trade.log"
grep -Fx 'MILL TRADE SMOKE OK: two deliveries, replenishment and competing purchases' "$lab/mill-trade.log"
