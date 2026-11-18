package main

import "testing"

func TestMillFlourTradeUsesOneLotAndWaitsForDelivery(t *testing.T) {
	mill := company{Cash: initialCash, MillFlour: flourProduced}
	bakery := company{Cash: initialCash, Line: true}
	_, _, reason := buyMillFlour(&bakery, &mill, "bakery", millOfferID, millPrice, 1)
	if reason != "" || bakery.Cash != initialCash-millPrice-bakerySuppliesCost ||
		mill.Cash != initialCash+millPrice || mill.MillFlour != 0 || bakery.Flour != flourProduced {
		t.Fatalf("unexpected settlement: %s, buyer %+v, seller %+v", reason, bakery, mill)
	}
	second := company{Cash: initialCash}
	before := second
	if _, _, reason = buyMillFlour(&second, &mill, "second", millOfferID, millPrice, 1); reason != "MILL_OFFER_UNAVAILABLE" || second != before {
		t.Fatalf("sold lot was offered again: %s, %+v", reason, second)
	}
	if events := produce(&bakery, 1); len(events) != 0 {
		t.Fatal("production used newly delivered flour")
	}
	if events := produce(&bakery, 2); len(events) != 3 || bakery.PackedBread != breadCount || bakery.Flour != flourProduced-flourUsed {
		t.Fatalf("production after delivery: %+v, %+v", events, bakery)
	}
}
