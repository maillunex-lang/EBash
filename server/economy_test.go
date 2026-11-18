package main

import "testing"

func TestPilotProductionBlockedKeepsInventory(t *testing.T) {
	tests := []struct {
		name  string
		stock company
		tick  int64
	}{
		{"disabled line", company{Cash: lineCost, Flour: flourUsed, Water: waterUsed,
			Salt: saltUsed, Yeast: yeastUsed, FlourReady: 3}, 3},
		{"flour not available yet", company{Cash: lineCost, Flour: flourUsed, Water: waterUsed,
			Salt: saltUsed, Yeast: yeastUsed, Line: true, FlourReady: 3}, 2},
		{"insufficient salt", company{Cash: lineCost, Flour: flourUsed, Water: waterUsed,
			Salt: saltUsed - 1, Yeast: yeastUsed, Line: true, FlourReady: 3}, 3},
		{"insufficient paper packaging", company{Cash: lineCost, Flour: flourUsed, Water: waterUsed,
			Salt: saltUsed, Yeast: yeastUsed, Line: true, FlourReady: 3, PackOrder: true, PaperPack: breadCount - 1}, 3},
		{"insufficient cash", company{Cash: lineCost - 1, Flour: flourUsed, Water: waterUsed,
			Salt: saltUsed, Yeast: yeastUsed, Line: true, FlourReady: 3}, 3},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			stock := tc.stock
			if events := produce(&stock, tc.tick); len(events) != 0 || stock != tc.stock {
				t.Fatalf("blocked recipe changed state: state=%+v events=%+v", stock, events)
			}
		})
	}
}

func TestPilotPurchaseRejectsPriceWithoutDebit(t *testing.T) {
	stock := company{Cash: initialCash}
	before := stock
	events, reason := applyCommand(&stock, command{
		ID: "buy", Type: "order_raw", Payload: payload{
			Offer: "pilot-inputs-v1", Quantity: 1, Limit: grainCost + ingredientsCost - 1,
		},
	}, 1)
	if reason != "PRICE_OR_FUNDS" || len(events) != 0 || stock != before {
		t.Fatalf("rejected purchase changed state: state=%+v reason=%s events=%+v", stock, reason, events)
	}
}
