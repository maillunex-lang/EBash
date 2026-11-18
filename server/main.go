package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func reply(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}

func fail(w http.ResponseWriter, status int, code string) {
	reply(w, status, map[string]string{"code": code})
}

func result(w http.ResponseWriter, status int, data any, err error) {
	switch {
	case err == nil:
		reply(w, status, data)
	case errors.Is(err, errScope):
		fail(w, 404, "NOT_FOUND")
	case errors.Is(err, errConflict):
		fail(w, 409, errConflict.Error())
	case errors.Is(err, errRecovering):
		fail(w, 503, errRecovering.Error())
	default:
		log.Printf("request failed: %v", err)
		fail(w, 500, "INTERNAL_ERROR")
	}
}

type authenticatedHandler func(http.ResponseWriter, *http.Request, string, string)

func (s store) withAccount(next authenticatedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || len(token) < 24 || len(token) > 512 {
			fail(w, 401, "UNAUTHORIZED")
			return
		}
		world := r.PathValue("world")
		account, err := s.authenticate(r.Context(), world, token)
		if errors.Is(err, pgx.ErrNoRows) {
			fail(w, 401, "UNAUTHORIZED")
			return
		}
		if err != nil {
			result(w, 0, nil, err)
			return
		}
		next(w, r, world, account)
	}
}

func (s store) handler() http.Handler {
	mux := http.NewServeMux()
	demoWorld := os.Getenv("EBASH_DEMO_WORLD")
	demoEnabled := func(world, account string) bool {
		return demoWorld != "" && world == demoWorld && account == "pilot"
	}
	mux.HandleFunc("POST /v1/worlds/{world}/demo-tick/{tick}", s.withAccount(func(w http.ResponseWriter, r *http.Request, world, account string) {
		if !demoEnabled(world, account) {
			fail(w, 404, "NOT_FOUND")
			return
		}
		tick, err := strconv.ParseInt(r.PathValue("tick"), 10, 64)
		if err != nil || tick < 1 {
			fail(w, 400, "INVALID_TICK")
			return
		}
		state, err := s.status(r.Context(), world)
		if err != nil {
			result(w, 0, nil, err)
			return
		}
		if state.Mode == "recovering" {
			result(w, 0, nil, errRecovering)
			return
		}
		if err = s.tick(r.Context(), world, tick, nil); errors.Is(err, errTickOrder) {
			fail(w, 409, "TICK_OUT_OF_ORDER")
			return
		}
		result(w, 200, map[string]string{"tick": strconv.FormatInt(tick, 10)}, err)
	}))

	mux.HandleFunc("GET /v1/worlds/{world}/status", s.withAccount(func(w http.ResponseWriter, r *http.Request, world, account string) {
		data, err := s.status(r.Context(), world)
		result(w, 200, struct {
			worldState
			Demo bool `json:"demo"`
		}{data, demoEnabled(world, account)}, err)
	}))
	mux.HandleFunc("GET /v1/worlds/{world}/pilot-chains", s.withAccount(func(w http.ResponseWriter, r *http.Request, world, account string) {
		result(w, 200, map[string]any{"chains": pilotChains}, nil)
	}))
	mux.HandleFunc("GET /v1/worlds/{world}/mill-offer", s.withAccount(func(w http.ResponseWriter, r *http.Request, world, account string) {
		sellerID, price, delay := millAccount, int64(millPrice), 0
		if r.URL.Query().Get("supplier") == "remote" {
			sellerID, price, delay = remoteMill, remoteMillPrice, remoteDelay
		}
		seller, err := s.company(r.Context(), world, sellerID)
		if errors.Is(err, errScope) {
			err = nil
		}
		result(w, 200, millOffer{
			ID: seller.flourOfferID(), Seller: isMill(account), Delay: delay,
			Available: !isMill(account) && !seller.MillUnlisted && seller.MillFlour >= flourProduced,
			Quantity:  flourProduced, Price: price, Supplies: bakerySuppliesCost,
		}, err)
	}))
	mux.HandleFunc("GET /v1/worlds/{world}/chain-supplies", s.withAccount(func(w http.ResponseWriter, r *http.Request, world, account string) {
		offers := make([]chainSupply, 0, len(chainSupplies))
		if !isMill(account) && accountSupply(account) == nil {
			for _, offer := range chainSupplies {
				seller, err := s.company(r.Context(), world, offer.account())
				if errors.Is(err, errScope) {
					continue
				}
				if err != nil {
					result(w, 0, nil, err)
					return
				}
				offer.Available = seller.SupplyLot
				offer.ID = offer.lotID(seller)
				offer.RestockTick = seller.SupplyRestockTick
				offers = append(offers, offer)
			}
		}
		result(w, 200, map[string]any{"offers": offers}, nil)
	}))
	mux.HandleFunc("GET /v1/worlds/{world}/company", s.withAccount(func(w http.ResponseWriter, r *http.Request, world, account string) {
		data, err := s.company(r.Context(), world, account)
		result(w, 200, data, err)
	}))
	mux.HandleFunc("GET /v1/worlds/{world}/commands/{id}", s.withAccount(func(w http.ResponseWriter, r *http.Request, world, account string) {
		data, err := s.lookup(r.Context(), world, account, r.PathValue("id"))
		result(w, 200, data, err)
	}))
	mux.HandleFunc("POST /v1/worlds/{world}/commands", s.withAccount(func(w http.ResponseWriter, r *http.Request, world, account string) {
		decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8192))
		decoder.DisallowUnknownFields()
		var cmd command
		if decoder.Decode(&cmd) != nil || !errors.Is(decoder.Decode(new(json.RawMessage)), io.EOF) || !cmd.valid() {
			fail(w, 400, "INVALID_COMMAND")
			return
		}
		data, err := s.accept(r.Context(), world, account, cmd)
		result(w, 202, data, err)
	}))
	return mux
}

func (s store) recoverWorld(ctx context.Context, world string, due int64) error {
	if err := s.setDue(ctx, world, due); err != nil {
		return err
	}
	batch := 2
	for {
		state, err := s.status(ctx, world)
		if err != nil || state.Last >= state.Due {
			return err
		}
		start := time.Now()
		for next := state.Last + 1; next <= state.Due && next-state.Last <= int64(batch); next++ {
			if err = s.tick(ctx, world, next, nil); err != nil {
				return err
			}
		}
		elapsed := time.Since(start)
		switch {
		case elapsed > 250*time.Millisecond && batch > 1:
			batch--
		case elapsed < 100*time.Millisecond && batch < 8:
			batch++
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(10 * time.Millisecond):
		}
	}
}

func run(ctx context.Context, args []string) error {
	cli := flag.NewFlagSet("ebash-p2", flag.ContinueOnError)
	world := cli.String("world", "pilot", "world ID")
	account := cli.String("account", "pilot", "account ID for seed")
	target := cli.Int64("tick", 0, "target tick")
	listen := cli.String("listen", "127.0.0.1:8080", "HTTP listen address")
	if len(args) == 0 {
		return errors.New("command required: migrate, seed, serve, tick, recover")
	}
	if err := cli.Parse(args[1:]); err != nil {
		return err
	}
	dsn := os.Getenv("EBASH_DATABASE_URL")
	if dsn == "" {
		return errors.New("EBASH_DATABASE_URL is required")
	}
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return err
	}
	defer pool.Close()
	s := store{pool}
	switch args[0] {
	case "migrate":
		return s.migrate(ctx)
	case "seed":
		return s.seed(ctx, *world, *account, os.Getenv("EBASH_PILOT_TOKEN"))
	case "tick":
		return s.tick(ctx, *world, *target, nil)
	case "recover":
		if *target < 1 {
			return errTickOrder
		}
		return s.recoverWorld(ctx, *world, *target)
	case "serve":
		if !strings.HasPrefix(*listen, "127.0.0.1:") {
			return errors.New("P2 server must listen on IPv4 loopback")
		}
		srv := &http.Server{Addr: *listen, Handler: s.handler(), ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout: 10 * time.Second, WriteTimeout: 10 * time.Second, IdleTimeout: 30 * time.Second}
		go func() {
			<-ctx.Done()
			timeout, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = srv.Shutdown(timeout)
		}()
		err = srv.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	default:
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()
	if err := run(ctx, os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}
