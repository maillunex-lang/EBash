package main

import "testing"

func TestSupplyRestockRejectsStaleOffer(t *testing.T) {
	offer := supplyByID("milk-near")
	seller := company{SupplyLot: true}
	buyer := company{Cash: initialCash}
	_, _, reason := buyChainSupply(&buyer, &seller, offer, offer.ID, offer.Price, 1)
	if reason != "" || seller.SupplyRestockTick != 5 {
		t.Fatalf("first sale and restock date: %s, %+v", reason, seller)
	}
	if len(seller.restockSupply(offer, 4)) != 0 || len(seller.restockSupply(offer, 5)) != 1 {
		t.Fatal("the replacement must appear exactly on tick 5")
	}
	if len(seller.restockSupply(offer, 5)) != 0 || seller.Cash != offer.Price/4 {
		t.Fatal("replaying the restock charged the seller twice")
	}

	nextBuyer := company{Cash: initialCash}
	_, _, reason = buyChainSupply(&nextBuyer, &seller, offer, offer.ID, offer.Price, 6)
	if reason != "SUPPLY_OFFER_UNAVAILABLE" || nextBuyer.Cash != initialCash || !seller.SupplyLot {
		t.Fatalf("stale order changed state: %s, buyer %+v, seller %+v", reason, nextBuyer, seller)
	}
	lot := seller.SupplyLotTick
	_, _, reason = buyChainSupply(&nextBuyer, &seller, offer, offer.lotID(seller), offer.Price, 6)
	if reason != "" || seller.SupplyLot || seller.SupplyLotTick != lot || seller.Cost != offer.Price*3/4 {
		t.Fatalf("replacement lot was not sold correctly: %s, %+v", reason, seller)
	}
}
