package main

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestPostgresMarketContention(t *testing.T) {
	ctx, s := openTestStore(t)
	world := fmt.Sprintf("market-%d", time.Now().UnixNano())
	offer := supplyByID("milk-near")
	must(t, s.seed(ctx, world, offer.account(), world+"-supplier-token"))
	buyers := make([]string, 64)
	for i := range buyers {
		buyers[i] = fmt.Sprintf("buyer-%03d", i)
		must(t, s.seed(ctx, world, buyers[i], world+buyers[i]))
	}
	var last int64
	for round := 0; round < 4; round++ {
		seller, err := s.company(ctx, world, offer.account())
		must(t, err)
		require(t, seller.SupplyLot, "lot must be ready")
		cmd := command{ID: fmt.Sprintf("buy-%d", round), Type: "buy_chain_supply",
			Payload: payload{Offer: offer.lotID(seller), Quantity: 1, Limit: offer.Price}}
		started := time.Now()
		errs := make(chan error, len(buyers))
		var wg sync.WaitGroup
		for _, account := range buyers {
			wg.Add(1)
			go func(account string) {
				defer wg.Done()
				_, err := s.accept(ctx, world, account, cmd)
				errs <- err
			}(account)
		}
		wg.Wait()
		close(errs)
		for err := range errs {
			must(t, err)
		}
		acceptedIn := time.Since(started)
		started = time.Now()
		last++
		must(t, s.tick(ctx, world, last, nil))
		tickIn := time.Since(started)
		must(t, s.tick(ctx, world, last, nil))
		winners := 0
		remaining := make([]string, 0, len(buyers))
		for _, account := range buyers {
			status, err := s.lookup(ctx, world, account, cmd.ID)
			must(t, err)
			c, err := s.company(ctx, world, account)
			must(t, err)
			switch status.Status {
			case "applied":
				winners++
				require(t, c.Cash == initialCash-offer.Price && c.Batch.Stock["raw_milk_g"] == 79900, "winning order must debit once")
			case "rejected":
				require(t, status.Reason == "SUPPLY_OFFER_UNAVAILABLE" && c.Cash == initialCash && c.Batch == nil, "loser must be unchanged")
				remaining = append(remaining, account)
			default:
				t.Fatalf("unexpected status %s", status.Status)
			}
		}
		require(t, winners == 1, "expected one buyer per lot")
		t.Logf("round %d: %d concurrent orders, one sale, acceptance %s, commit %s", round+1, len(buyers), acceptedIn, tickIn)
		buyers = remaining
		for i := 0; i < supplyRestockDelay; i++ {
			last++
			must(t, s.tick(ctx, world, last, nil))
		}
	}
	seller, err := s.company(ctx, world, offer.account())
	must(t, err)
	require(t, seller.SupplyLot && seller.Cash == initialCash+4*(offer.Price-offer.Price*3/4), "four sales and four paid restocks")
}
