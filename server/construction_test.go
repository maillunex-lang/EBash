package main

import "testing"

func TestBuildBakeryReservations(t *testing.T) {
	fresh := func(cash int64) company {
		return company{Cash: cash, GameProduction: &gameProductionState{Construction: newBakeryConstruction()}}
	}
	owner := fresh(40000)
	request := payload{Offer: "12", Limit: 40000}
	companies := map[string]company{}
	if _, reason := startBakery(&owner, companies, request, 5); reason != "" || owner.Cash != 0 || owner.GameProduction.Construction.ReadyTick != 7 {
		t.Fatal(reason, owner)
	}
	companies["owner"] = owner
	for _, tc := range []struct {
		cash   int64
		plot   string
		price  int64
		reason string
	}{
		{50000, "12", 40000, "SITE_UNAVAILABLE"}, {29999, "6", 30000, "INSUFFICIENT_FUNDS"}, {50000, "6", 1, "SITE_UNAVAILABLE"},
	} {
		c := fresh(tc.cash)
		_, reason := startBakery(&c, companies, payload{Offer: tc.plot, Limit: tc.price}, 6)
		if reason != tc.reason || c.Cash != tc.cash || c.GameProduction.Construction.Site != "" {
			t.Fatal(reason, c)
		}
	}
	if _, reason := startBakery(&owner, companies, request, 6); reason != "SITE_UNAVAILABLE" || owner.Cash != 0 {
		t.Fatal(reason, owner)
	}
}
