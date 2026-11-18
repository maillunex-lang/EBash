package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func openTestStore(t *testing.T) (context.Context, store) {
	t.Helper()
	dsn := os.Getenv("EBASH_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("EBASH_TEST_DATABASE_URL is required for PostgreSQL integration")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	t.Cleanup(cancel)
	pool, err := pgxpool.New(ctx, dsn)
	must(t, err)
	t.Cleanup(pool.Close)
	s := store{pool}
	must(t, s.migrate(ctx))
	return ctx, s
}

func must(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
}

func require(t *testing.T, ok bool, msg string) {
	t.Helper()
	if !ok {
		t.Fatal(msg)
	}
}

func request(ctx context.Context, client *http.Client, url, token, method, path, body string) (int, []byte, error) {
	r, err := http.NewRequestWithContext(ctx, method, url+"/v1/worlds/"+path, strings.NewReader(body))
	if err != nil {
		return 0, nil, err
	}
	r.Header.Set("Authorization", "Bearer "+token)
	r.Header.Set("Content-Type", "application/json")
	resp, err := client.Do(r)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	return resp.StatusCode, data, err
}

func TestPostgresPilot(t *testing.T) {
	ctx, s := openTestStore(t)
	world := fmt.Sprintf("pilot-%d", time.Now().UnixNano())
	token := world + "-first-account-token"
	other := world + "-other-account-token"
	must(t, s.seed(ctx, world, "owner", token))
	must(t, s.seed(ctx, world, "other", other))
	srv := httptest.NewServer(s.handler())
	defer srv.Close()
	call := func(method, path, auth, body string, want int) []byte {
		t.Helper()
		status, data, err := request(ctx, srv.Client(), srv.URL, auth, method, world+path, body)
		must(t, err)
		if status != want {
			t.Fatalf("%s %s status %d, want %d: %s", method, path, status, want, data)
		}
		return data
	}
	const order = `{"command_id":"buy","type":"order_raw","payload":{"offer_id":"pilot-inputs-v1","quantity":"1","max_price_kopecks":"48000"}}`
	const line = `{"command_id":"line","type":"configure_line","payload":{"enterprise_id":"pilot-bakery","cell_id":"4","profile_id":"test-bread-v1"}}`
	accepted := call("POST", "/commands", token, order, 202)
	repeated := call("POST", "/commands", token, order, 202)
	require(t, bytes.Equal(accepted, repeated), "repeat must return the saved acceptance")
	call("POST", "/commands", token, strings.Replace(order, "48000", "49000", 1), 409)
	call("POST", "/commands", token, strings.Replace(line, "test-bread-v1", "draft-catalog", 1), 400)
	call("POST", "/commands", token, line, 202)
	call("GET", "/commands/buy", other, "", 404)
	call("GET", "/company", "wrong-token", "", 401)
	require(t, errors.Is(s.tick(ctx, world, 2, nil), errTickOrder), "must reject skipped tick")
	injected := errors.New("abort after writes")
	require(t, errors.Is(s.tick(ctx, world, 1, func() error { return injected }), injected), "injected failure missing")
	checkState := func(tick, cash, grain, flour, bread int64) {
		t.Helper()
		c, err := s.company(ctx, world, "owner")
		must(t, err)
		if c.LastTick != tick || c.Cash != cash || c.Grain != grain || c.Flour != flour || c.Bread != bread {
			t.Fatalf("unexpected state after tick %d: %+v", tick, c)
		}
	}
	checkState(0, 200_000, 0, 0, 0)
	var count int
	must(t, s.db.QueryRow(ctx, "SELECT count(*) FROM ledger WHERE world=$1", world).Scan(&count))
	require(t, count == 0, "rollback must remove uncommitted ledger entries")
	var commandResult commandState
	must(t, json.Unmarshal(call("GET", "/commands/buy", token, "", 200), &commandResult))
	require(t, commandResult.Status == "accepted", "rollback must keep command pending")

	// An independent read must see the committed snapshot while a tick is in flight.
	entered, release := make(chan struct{}), make(chan struct{})
	finished := make(chan error, 1)
	go func() {
		finished <- s.tick(ctx, world, 1, func() error {
			close(entered)
			<-release
			return injected
		})
	}()
	select {
	case <-entered:
	case err := <-finished:
		t.Fatalf("tick ended before snapshot read: %v", err)
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	func() {
		defer close(release)
		var snapshot company
		must(t, json.Unmarshal(call("GET", "/company", token, "", 200), &snapshot))
		require(t, snapshot.LastTick == 0 && snapshot.Cash == initialCash,
			"reader saw uncommitted state")
	}()
	require(t, errors.Is(<-finished, injected), "in-flight tick did not roll back")

	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- s.tick(ctx, world, 1, nil) }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		must(t, err)
	}
	checkState(1, 152_000, 10_000, 0, 0)
	require(t, bytes.Equal(accepted, call("POST", "/commands", token, order, 202)), "lost response replay after commit")
	const early = `{"command_id":"early","type":"offer_output","payload":{"lot_id":"bread-3","quantity":"24","price_kopecks":"4500"}}`
	must(t, s.tick(ctx, world, 2, nil))
	checkState(2, 140_000, 0, 7_400, 0)
	call("POST", "/commands", token, early, 202)
	must(t, s.tick(ctx, world, 3, nil))
	checkState(3, 126_000, 0, 200, 24)
	var baked company
	must(t, json.Unmarshal(call("GET", "/company", token, "", 200), &baked))
	require(t, baked.BreadLot == "bread-3" && baked.Water == 0 && baked.Salt == 0 && baked.Yeast == 0,
		"API must expose the lot ID and consumed ingredients")
	must(t, json.Unmarshal(call("GET", "/commands/early", token, "", 200), &commandResult))
	require(t, commandResult.Status == "rejected" && commandResult.Reason == "LOT_UNAVAILABLE", "cannot sell on production tick")
	sale := strings.Replace(early, `"early"`, `"sale"`, 1)
	call("POST", "/commands", token, sale, 202)
	must(t, s.tick(ctx, world, 4, nil))
	must(t, s.tick(ctx, world, 4, nil))
	checkState(4, 234_000, 0, 200, 0)
	c, err := s.company(ctx, world, "owner")
	must(t, err)
	require(t, c.Revenue == 108_000 && c.Cost == 72_865 && c.Profit == 35_135 && c.FlourValue == 1_135, "report differs from pilot")
	must(t, s.db.QueryRow(ctx, "SELECT count(*) FROM ledger WHERE world=$1", world).Scan(&count))
	require(t, count == 7, "pilot must have exactly seven economic events")
	must(t, s.setDue(ctx, world, 16))
	call("POST", "/commands", token, strings.Replace(line, `"line"`, `"late"`, 1), 503)
	call("POST", "/commands", token, order, 202)
	call("GET", "/company", token, "", 200)
	must(t, s.recoverWorld(ctx, world, 16))
	checkState(16, 234_000, 0, 200, 0)
	status, err := s.status(ctx, world)
	must(t, err)
	require(t, status.Mode == "running" && status.Last == 16 && status.Due == 16, "recovery did not catch up")
	must(t, s.db.QueryRow(ctx, "SELECT count(*) FROM ticks WHERE world=$1", world).Scan(&count))
	require(t, count == 16, "ticks must be applied sequentially")
	must(t, s.db.QueryRow(ctx, "SELECT count(*) FROM ledger WHERE world=$1", world).Scan(&count))
	require(t, count == 7, "recovery must not duplicate movement")
}

// The load fixture uses the same economic functions and transaction writer as
// the pilot. Sales and restocking are synthetic external demand and supply.
func runLoadCells(parent *company, tick int64) []event {
	if parent.LoadCells == nil {
		parent.LoadCells = new([10]company)
		for i := range parent.LoadCells {
			parent.LoadCells[i] = company{Cash: initialCash, Line: true}
		}
	}
	var events []event
	for i := range parent.LoadCells {
		cell := &parent.LoadCells[i]
		var movements []event
		switch tick % 4 {
		case 1:
			movements, _ = applyCommand(cell, command{
				Type: "order_raw", Payload: payload{Limit: grainCost + ingredientsCost},
			}, tick)
		case 2, 3:
			movements = produce(cell, tick)
		case 0:
			movements, _ = applyCommand(cell, command{
				Type: "offer_output", Payload: payload{Lot: cell.BreadLot, Quantity: breadCount, Price: breadPrice},
			}, tick)
			if len(movements) > 0 {
				movements = append(movements, movement("fixture_flour_transfer",
					map[string]int64{"bakery_flour_g": -cell.Flour}))
				cell.Flour, cell.FlourValue, cell.GrainReady = 0, 0, 0
			}
		}
		for _, e := range movements {
			e.Movements["fixture_cell_id"] = fmt.Sprintf("%02d", i+1)
			events = append(events, e)
		}
	}
	return events
}

func TestPostgresLoad(t *testing.T) {
	durationText := os.Getenv("EBASH_LOAD_DURATION")
	if durationText == "" {
		t.Skip("set EBASH_LOAD_DURATION, e.g. 10m")
	}
	duration, err := time.ParseDuration(durationText)
	must(t, err)
	ctx, s := openTestStore(t)
	world := fmt.Sprintf("load-%d", time.Now().UnixNano())
	tokens := make([]string, 100)
	for i := range tokens {
		account := fmt.Sprintf("account-%03d", i)
		tokens[i] = world + "-token-" + account
		must(t, s.seed(ctx, world, account, tokens[i]))
	}
	srv := httptest.NewServer(s.handler())
	defer srv.Close()
	type sample struct {
		kind string
		time time.Duration
		err  error
	}
	results := make(chan sample, 64)
	var senders, requests sync.WaitGroup
	start := time.Now()
	deadline := start.Add(duration)
	for _, kind := range []string{"read", "write", "tick"} {
		senders.Add(1)
		go func(kind string) {
			defer senders.Done()
			interval := 50 * time.Millisecond
			if kind == "write" {
				interval = 200 * time.Millisecond
			}
			if kind == "tick" {
				interval = time.Second
			}
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for i := 0; ; i++ {
				scheduled := <-ticker.C
				if scheduled.After(deadline) {
					return
				}
				requests.Add(1)
				do := func(i int, scheduled time.Time) {
					defer requests.Done()
					var err error
					switch kind {
					case "tick":
						err = s.tickWithProduction(ctx, world, int64(i+1), nil, runLoadCells, false)
					default:
						method, path, body, want := "GET", "/company", "", 200
						if kind == "write" {
							method, path, want = "POST", "/commands", 202
							body = fmt.Sprintf(`{"command_id":"line-%d","type":"configure_line","payload":{"enterprise_id":"pilot-bakery","cell_id":"4","profile_id":"test-bread-v1"}}`, i)
						}
						var got int
						got, _, err = request(ctx, srv.Client(), srv.URL, tokens[i%len(tokens)], method, world+path, body)
						if err == nil && got != want {
							err = fmt.Errorf("%s returned %d", kind, got)
						}
					}
					results <- sample{kind, time.Since(scheduled), err}
				}
				if kind == "tick" {
					do(i, scheduled)
				} else {
					go do(i, scheduled)
				}
			}
		}(kind)
	}
	go func() { senders.Wait(); requests.Wait(); close(results) }()
	times := map[string][]time.Duration{}
	for sample := range results {
		if sample.err != nil {
			t.Error(sample.err)
		}
		times[sample.kind] = append(times[sample.kind], sample.time)
	}
	var ticks, cells, sales, entries int
	must(t, s.db.QueryRow(ctx, `SELECT count(*) FROM ticks WHERE world=$1`, world).Scan(&ticks))
	must(t, s.db.QueryRow(ctx, `SELECT count(DISTINCT (account, movements->>'fixture_cell_id')),
		count(*) FILTER (WHERE event='sale'), count(*)
		FROM ledger WHERE world=$1`, world).Scan(&cells, &sales, &entries))
	require(t, cells == 1000, fmt.Sprintf("expected 1000 cells with postings, got %d", cells))
	require(t, sales == (ticks/4)*cells, fmt.Sprintf("unexpected sales count: %d", sales))
	require(t, entries == 2*ticks*cells, fmt.Sprintf("unexpected ledger count: %d", entries))
	t.Logf("production: %d active cells, %d sales, %d postings", cells, sales, entries)
	for _, kind := range []string{"read", "write", "tick"} {
		samples := times[kind]
		require(t, len(samples) > 0, "no "+kind+" samples")
		sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
		p95 := samples[(len(samples)*95+99)/100-1]
		t.Logf("%s: n=%d p95=%s max=%s", kind, len(samples), p95, samples[len(samples)-1])
		ceiling := 300 * time.Millisecond
		if kind == "write" {
			ceiling = 500 * time.Millisecond
		}
		if kind == "tick" {
			p95 = samples[len(samples)-1]
			ceiling = 30 * time.Second
		}
		if p95 > ceiling {
			t.Errorf("%s latency %s exceeds %s", kind, p95, ceiling)
		}
	}
}
