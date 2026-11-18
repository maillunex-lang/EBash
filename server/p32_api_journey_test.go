package main

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"
)

// P3.2 full HTTP -> PostgreSQL -> server tick journey. Unlike the standalone
// arithmetic model, all actions here use authenticated requests and durable state.
func TestP32PostgresBreadJourney(t *testing.T) {
	ctx, s := openTestStore(t)
	world := fmt.Sprintf("p32-bread-%d", time.Now().UnixNano())
	token := world + "-account-secret"
	must(t, s.seed(ctx, world, "pilot", token))
	srv := httptest.NewServer(s.handler())
	defer srv.Close()
	call := func(method, path, body string, want int) []byte {
		t.Helper()
		status, data, err := request(ctx, srv.Client(), srv.URL, token, method, world+path, body)
		must(t, err)
		if status != want {
			t.Fatalf("%s %s: got %d expected %d: %s", method, path, status, want, data)
		}
		return data
	}
	const order = `{"command_id":"p32-order","type":"order_raw","payload":{"offer_id":"pilot-inputs-v1","quantity":"1","max_price_kopecks":"48000"}}`
	const line = `{"command_id":"p32-line","type":"configure_line","payload":{"enterprise_id":"pilot-bakery","cell_id":"4","profile_id":"test-bread-v1"}}`
	const sale = `{"command_id":"p32-sale","type":"offer_output","payload":{"lot_id":"bread-3","quantity":"24","price_kopecks":"4500"}}`
	call("POST", "/commands", order, 202)
	call("POST", "/commands", line, 202)
	for tick := int64(1); tick <= 3; tick++ {
		must(t, s.tick(ctx, world, tick, nil))
		c, err := s.company(ctx, world, "pilot")
		must(t, err)
		if tick < 3 && c.Bread != 0 {
			t.Fatalf("early bread at tick %d", tick)
		}
		if tick == 3 && (c.Bread != 24 || c.Flour != 200 || c.Cash != 126000) {
			t.Fatalf("incorrect batch: %+v", c)
		}
	}
	call("POST", "/commands", sale, 202)
	// Sale was accepted for tick 4, but stock and cash must remain unchanged now.
	before, err := s.company(ctx, world, "pilot")
	must(t, err)
	if before.Cash != 126000 || before.Bread != 24 {
		t.Fatal("pending sale moved goods early")
	}
	must(t, s.tick(ctx, world, 4, nil))
	must(t, s.tick(ctx, world, 4, nil))
	var c company
	must(t, json.Unmarshal(call("GET", "/company", "", 200), &c))
	if c.Cash != 234000 || c.Bread != 0 || c.Flour != 200 || c.Revenue != 108000 || c.Profit != 35135 {
		t.Fatalf("financial or stock reconciliation failed: %+v", c)
	}
	call("POST", "/commands", sale, 202)
	var ledgers int
	must(t, s.db.QueryRow(ctx, "SELECT count(*) FROM ledger WHERE world=$1", world).Scan(&ledgers))
	if ledgers != 7 {
		t.Fatalf("unexpected ledger count: %d", ledgers)
	}
	var ticks int
	must(t, s.db.QueryRow(ctx, "SELECT count(*) FROM ticks WHERE world=$1", world).Scan(&ticks))
	if ticks != 4 {
		t.Fatalf("duplicate tick rows: %d", ticks)
	}
}
