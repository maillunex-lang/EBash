package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"
)

func TestGameProfilesPostgresHTTPTrade(t *testing.T) {
	t.Run("near", func(t *testing.T) { testGameProfilesTrade(t, millAccount, 42000, 0) })
	t.Run("remote", func(t *testing.T) { testGameProfilesTrade(t, remoteMill, 36000, 2) })
}

func testGameProfilesTrade(t *testing.T, sellerID string, price, delay int64) {
	ctx, s := openTestStore(t)
	set, err := readGameProfiles("../docs/catalogs/game-production-profiles.v1.json")
	must(t, err)
	world := fmt.Sprintf("e104-trade-%d", time.Now().UnixNano())
	token := world + "-baker-token"
	must(t, s.seed(ctx, world, "baker", token))
	must(t, s.seed(ctx, world, millAccount, world+"-mill-token"))
	must(t, s.seed(ctx, world, remoteMill, world+"-remote-token"))
	seedGameProductionState(t, ctx, s, world, "baker", true)
	baker, err := s.company(ctx, world, "baker")
	must(t, err)
	baker.GameProduction.Lines = baker.GameProduction.Lines[1:]
	baker.GameProduction.Stock = map[string]int64{}
	baker.GameProduction.Inventory = &gameInventory{}
	raw, err := json.Marshal(baker)
	must(t, err)
	_, err = s.db.Exec(ctx, "UPDATE companies SET state=$3 WHERE world=$1 AND account=$2", world, "baker", raw)
	must(t, err)
	server := httptest.NewServer(s.handler())
	defer server.Close()
	call := func(method, path, body string, want int) []byte {
		t.Helper()
		status, data, err := request(ctx, server.Client(), server.URL, token, method, world+path, body)
		must(t, err)
		if status != want {
			t.Fatalf("%s %s: %d != %d: %s", method, path, status, want, data)
		}
		return data
	}
	state := func() company {
		var c company
		must(t, json.Unmarshal(call("GET", "/company", "", 200), &c))
		must(t, checkGameInventoryStock(*c.GameProduction.Inventory, c.GameProduction.Stock))
		return c
	}
	var near, remote millOffer
	must(t, json.Unmarshal(call("GET", "/mill-offer", "", 200), &near))
	must(t, json.Unmarshal(call("GET", "/mill-offer?supplier=remote", "", 200), &remote))
	require(t, near.Available && remote.Available && near.ID != remote.ID &&
		near.Price == 42000 && near.Delay == 0 && remote.Price == 36000 && remote.Delay == 2 &&
		near.ProfileSupplies == 18000 && remote.ProfileSupplies == 18000, "both quotes must agree with contract")
	quote := near
	other := remoteMill
	if sellerID == remoteMill {
		quote, other = remote, millAccount
	}
	buy := fmt.Sprintf(`{"command_id":"buy","type":"game_buy_flour","payload":{"offer_id":%q,"quantity":"7400","max_price_kopecks":"%d"}}`, quote.ID, price)
	early := `{"command_id":"early","type":"game_sell_bread","payload":{"lot_id":"game-bread-v1","quantity":"24","price_kopecks":"4500"}}`
	sell := `{"command_id":"sell","type":"game_sell_bread","payload":{"lot_id":"game-bread-v1","quantity":"24","price_kopecks":"4500"}}`
	accepted := call("POST", "/commands", buy, 202)
	arrival := 1 + delay
	paid := int64(initialCash-ingredientsCost) - price
	for tick := int64(1); tick <= arrival; tick++ {
		must(t, s.tickWithApprovedGameProfiles(ctx, world, tick, set))
		must(t, s.tickWithApprovedGameProfiles(ctx, world, tick, set))
		require(t, bytes.Equal(accepted, call("POST", "/commands", buy, 202)), "duplicate purchase acceptance")
		c := state()
		require(t, c.Cash == paid && c.GameProduction.Stock["bread_loaf"] == 0 && c.GameProduction.LastError == "", "early production or repeated charge")
		if tick < arrival {
			pending := c.GameProduction.FlourDelivery
			require(t, pending != nil && pending.Tick == arrival && pending.Price == price && pending.Offer == quote.ID && len(c.GameProduction.Stock) == 0, "goods in transit must be outside stock")
		} else {
			require(t, c.GameProduction.FlourDelivery == nil && c.GameProduction.Stock["wheat_flour"] == 7400, "delivery due now")
			for _, lot := range c.GameProduction.Inventory.Lots {
				require(t, lot.AvailableForProductionTick == arrival+1, "received lot available too early")
			}
		}
	}
	seller, err := s.company(ctx, world, sellerID)
	must(t, err)
	require(t, seller.Cash == initialCash+price && seller.MillFlour == 0, "seller settlement mismatch")
	untouched, err := s.company(ctx, world, other)
	must(t, err)
	require(t, untouched.Cash == initialCash && untouched.MillFlour == flourProduced, "other offer changed")
	call("POST", "/commands", early, 202)
	must(t, s.tickWithApprovedGameProfiles(ctx, world, arrival+1, set))
	rejected, err := s.lookup(ctx, world, "baker", "early")
	must(t, err)
	require(t, rejected.Status == "rejected" && rejected.Reason == "LOT_UNAVAILABLE", "same-tick sale must fail")
	c := state()
	require(t, c.Cash == paid-lineCost && c.GameProduction.Stock["bread_loaf"] == 24 && c.GameProduction.Stock["wheat_flour"] == 200, "incorrect profile execution")
	accepted = call("POST", "/commands", sell, 202)
	must(t, s.tickWithApprovedGameProfiles(ctx, world, arrival+2, set))
	must(t, s.tickWithApprovedGameProfiles(ctx, world, arrival+2, set))
	require(t, bytes.Equal(accepted, call("POST", "/commands", sell, 202)), "duplicate sale acceptance")
	c = state()
	finalCash := paid - lineCost + breadCount*breadPrice
	require(t, c.Cash == finalCash && c.GameProduction.Stock["bread_loaf"] == 0 && c.Revenue == 108000, "incorrect sale balance")
	var deliveryTick int64
	must(t, s.db.QueryRow(ctx, "SELECT tick FROM ledger WHERE world=$1 AND event='game_flour_delivery'", world).Scan(&deliveryTick))
	require(t, deliveryTick == arrival, "promised and actual delivery differ")
	var movements int
	must(t, s.db.QueryRow(ctx, "SELECT count(*) FROM ledger WHERE world=$1 AND event IN ('game_flour_purchase','game_flour_delivery','mill_sale','game_bread_sale')", world).Scan(&movements))
	require(t, movements == 4, "trade ledger was duplicated")
	ordinary := `{"command_id":"ordinary","type":"game_sell_bread","payload":{"lot_id":"game-bread-v1","quantity":"24","price_kopecks":"4500"}}`
	call("POST", "/commands", ordinary, 202)
	must(t, s.tick(ctx, world, arrival+3, nil))
	blocked, err := s.lookup(ctx, world, "baker", "ordinary")
	must(t, err)
	require(t, blocked.Status == "rejected" && blocked.Reason == "GAME_TRADE_DISABLED", "ordinary tick must disable game trade")
	require(t, state().Cash == finalCash, "ordinary tick must not settle game trade")
}
