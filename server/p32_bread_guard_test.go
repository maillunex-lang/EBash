package main

import "testing"

// P3.2 tests the production functions used by the server. DES-004 catalog
// recipes remain draft_not_executable; test-bread-v1 is a separate game fixture.
func TestP32BakeryMissingIngredientDoesNotConsume(t *testing.T) {
	cases := []struct {
		name string
		edit func(*company)
	}{
		{"flour", func(c *company) { c.Flour = flourUsed - 1 }},
		{"water", func(c *company) { c.Water = waterUsed - 1 }},
		{"salt", func(c *company) { c.Salt = saltUsed - 1 }},
		{"yeast", func(c *company) { c.Yeast = yeastUsed - 1 }},
		{"funds", func(c *company) { c.Cash = lineCost - 1 }},
		{"not-ready", func(c *company) { c.FlourReady = 4 }},
		{"disabled", func(c *company) { c.Line = false }},
		{"packaging", func(c *company) { c.PackOrder = true; c.PaperPack = breadCount - 1 }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := company{Cash: initialCash, Line: true, Flour: flourUsed, FlourValue: grainCost + transportCost,
				Water: waterUsed, Salt: saltUsed, Yeast: yeastUsed, FlourReady: 3}
			tc.edit(&c)
			before := c
			if events := produce(&c, 3); len(events) != 0 || c != before {
				t.Fatalf("blocked batch changed state: before=%+v after=%+v events=%+v", before, c, events)
			}
		})
	}
}

func TestP32BakeryOneBatchAndNextTickSale(t *testing.T) {
	c := company{Cash: initialCash, Line: true, Flour: flourProduced, FlourValue: grainCost + transportCost,
		Water: waterUsed, Salt: saltUsed, Yeast: yeastUsed, FlourReady: 3}
	if events := produce(&c, 2); len(events) != 0 {
		t.Fatal("flour was consumed before delivery became usable")
	}
	if events := produce(&c, 3); len(events) != 2 {
		t.Fatalf("expected mixing and baking events, got %d", len(events))
	}
	if c.Flour != 200 || c.Bread != breadCount || c.BreadReady != 4 || c.Cash != initialCash-lineCost {
		t.Fatalf("wrong batch state: %+v", c)
	}
	before := c
	if events := produce(&c, 3); len(events) != 0 || c != before {
		t.Fatal("same-tick second run changed batch")
	}
	cmd := command{Type: "offer_output", Payload: payload{Lot: c.BreadLot, Quantity: breadCount, Price: breadPrice}}
	if _, reason := applyCommand(&c, cmd, 3); reason != "LOT_UNAVAILABLE" || c != before {
		t.Fatalf("early sale changed state: reason=%q", reason)
	}
	if _, reason := applyCommand(&c, cmd, 4); reason != "" {
		t.Fatalf("next-tick sale rejected: %s", reason)
	}
	if c.Bread != 0 || c.Cash != initialCash-lineCost+breadCount*breadPrice || c.CompletedBatches != 1 {
		t.Fatalf("sale accounting mismatch: %+v", c)
	}
	after := c
	if _, reason := applyCommand(&c, cmd, 4); reason == "" || c != after {
		t.Fatal("duplicate sale changed state")
	}
}
