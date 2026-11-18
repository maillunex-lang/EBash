package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Reconnects the API through a new connection pool while an actual PostgreSQL
// delivery is in flight. The database is deliberately not reset between boots.
func TestPostgresDeliverySurvivesAPIRestart(t *testing.T) {
	ctx, original := openTestStore(t)
	world := fmt.Sprintf("delivery-restart-%d", time.Now().UnixNano())
	token := world + "-bakery-secret"
	must(t, original.seed(ctx, world, "pilot", token))
	must(t, original.seed(ctx, world, remoteMill, world+"-mill-secret"))
	srv := httptest.NewServer(original.handler())
	call := func(method, path, body string, want int) []byte {
		t.Helper()
		status, data, err := request(ctx, srv.Client(), srv.URL, token, method, world+path, body)
		must(t, err)
		if status != want {
			t.Fatalf("%s %s: %d want %d: %s", method, path, status, want, data)
		}
		return data
	}
	defer func() { srv.Close() }()
	const buy = `{"command_id":"remote-flour-1","type":"buy_mill_flour","payload":{"offer_id":"remote-pilot-mill-flour-v1","quantity":"7400","max_price_kopecks":"36000"}}`
	const line = `{"command_id":"line-1","type":"configure_line","payload":{"enterprise_id":"pilot-bakery","cell_id":"4","profile_id":"test-bread-v1"}}`
	call("POST", "/commands", line, 202)
	accepted := call("POST", "/commands", buy, 202)
	must(t, original.tick(ctx, world, 1, nil))
	before, err := original.company(ctx, world, "pilot")
	must(t, err)
	require(t, before.InTransitUntil == 3 && before.Flour == 0 && before.Cash == 143600, "delivery not pending before restart")
	var ledgerBefore int
	must(t, original.db.QueryRow(ctx, "SELECT count(*) FROM ledger WHERE world=$1", world).Scan(&ledgerBefore))
	srv.Close()
	// The second API uses a fresh pool and reads only persisted state.
	pool, err := pgxpool.New(context.Background(), os.Getenv("EBASH_TEST_DATABASE_URL"))
	must(t, err)
	defer pool.Close()
	rebooted := store{pool}
	srv = httptest.NewServer(rebooted.handler())
	var snapshot company
	must(t, json.Unmarshal(call("GET", "/company", "", 200), &snapshot))
	require(t, snapshot == before, "restarted client/API lost the persisted snapshot")
	require(t, string(call("POST", "/commands", buy, 202)) == string(accepted), "command replay after reboot changed response")
	must(t, rebooted.tick(ctx, world, 1, nil))
	var ledgerRepeated int
	must(t, rebooted.db.QueryRow(ctx, "SELECT count(*) FROM ledger WHERE world=$1", world).Scan(&ledgerRepeated))
	require(t, ledgerRepeated == ledgerBefore, "replayed tick duplicated money or movement")
	must(t, rebooted.tick(ctx, world, 2, nil))
	must(t, rebooted.tick(ctx, world, 3, nil))
	delivered, err := rebooted.company(ctx, world, "pilot")
	must(t, err)
	require(t, delivered.InTransitUntil == 0 && delivered.Flour == flourProduced && delivered.PackedBread == 0 && delivered.Cash == before.Cash, "delivery lost or used too early")
	must(t, rebooted.tick(ctx, world, 3, nil))
	again, err := rebooted.company(ctx, world, "pilot")
	must(t, err)
	require(t, again == delivered, "replayed delivery created goods")
	must(t, rebooted.tick(ctx, world, 4, nil))
	baked, err := rebooted.company(ctx, world, "pilot")
	must(t, err)
	require(t, baked.PackedBread == breadCount && baked.Flour == 200, "bakery failed to resume after restart")
	var ticks int
	must(t, rebooted.db.QueryRow(ctx, "SELECT count(*) FROM ticks WHERE world=$1", world).Scan(&ticks))
	require(t, ticks == 4, "tick ledger contains duplicates")
}
