package main

import "testing"

// This compares the pilot's actual economy rules, not a proposed DES-004 production profile.
func TestBakerySupplierTimingAndBalance(t *testing.T) {
	cases := []struct {
		name, offer                   string
		orderTick, bakeTick, saleTick int64
		price, cash, profit           int64
		idleTicks                     int64
	}{
		{"near", millOfferID, 1, 2, 3, millPrice, 231600, 32735, 0},
		{"remote_preorder", "remote-" + millOfferID, 1, 4, 5, remoteMillPrice, 237600, 38573, 2},
		{"remote_late", "remote-" + millOfferID, 2, 5, 6, remoteMillPrice, 237600, 38573, 3},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			buyer := company{Cash: initialCash, Line: true}
			seller := company{Cash: initialCash, MillFlour: flourProduced, MillLot: tc.offer}
			_, _, reason := buyMillFlour(&buyer, &seller, "bakery", tc.offer, tc.price, tc.orderTick)
			if reason != "" {
				t.Fatalf("purchase rejected: %s", reason)
			}
			for tick := int64(1); tick <= tc.saleTick; tick++ {
				buyer.deliverMillFlour(tick)
				produce(&buyer, tick)
				if tick < tc.bakeTick && buyer.PackedBread != 0 {
					t.Fatalf("early production at %d", tick)
				}
				if tick == tc.bakeTick && buyer.PackedBread != breadCount {
					t.Fatalf("missing bread at %d: %+v", tick, buyer)
				}
				if tick == tc.saleTick {
					_, reason = applyCommand(&buyer, command{Type: "offer_output", Payload: payload{Quantity: breadCount, Price: breadPrice, Lot: buyer.BreadLot}}, tick)
					if reason != "" {
						t.Fatalf("sale rejected: %s", reason)
					}
				}
			}
			if buyer.Cash != tc.cash || buyer.Profit != tc.profit || buyer.Flour != 200 || buyer.CompletedBatches != 1 {
				t.Fatalf("unexpected balance: cash=%d profit=%d flour=%d batches=%d", buyer.Cash, buyer.Profit, buyer.Flour, buyer.CompletedBatches)
			}
			if tc.bakeTick-2 != tc.idleTicks {
				t.Fatalf("unexpected idle ticks")
			}
			t.Logf("order=%d bake=%d sale=%d idle=%d cash=%d profit=%d", tc.orderTick, tc.bakeTick, tc.saleTick, tc.idleTicks, buyer.Cash, buyer.Profit)
		})
	}
}
