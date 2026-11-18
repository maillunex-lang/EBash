package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"testing"
	"time"
)

func seedGameProductionState(t *testing.T, ctx context.Context, s store, world, account string, full bool) {
	t.Helper()
	c, err := s.company(ctx, world, account)
	must(t, err)
	grain := int64(10000)
	if !full {
		grain = 9000
	}
	factors := map[string]int64{"raw_material": 70, "technology": 70, "equipment": 70, "personnel": 70, "conditions": 70}
	c.GameProduction = &gameProductionState{
		Stock: map[string]int64{"wheat_grain": grain, "water": 5440, "salt": 144, "yeast": 112},
		Lines: []gameLine{
			{ID: "01-mill", ProfileID: "GAME-WHEAT-FLOUR-V1", RequestedBatches: 1, AvailableEquipment: "roller_mill_sifter", Factors: factors},
			{ID: "02-mixer", ProfileID: "GAME-BREAD-DOUGH-V1", RequestedBatches: 8, AvailableEquipment: "spiral_mixer", Factors: factors},
			{ID: "03-oven", ProfileID: "GAME-BREAD-LOAF-V1", RequestedBatches: 8, AvailableEquipment: "proofer_oven", Factors: factors},
		},
		PerTickBudget: gameBudget{EnergyWh: 4260, Workers: 3, EquipmentCapacity: map[string]int64{"roller_mill_sifter": 1, "spiral_mixer": 8, "proofer_oven": 8}},
	}
	raw, err := json.Marshal(c)
	must(t, err)
	_, err = s.db.Exec(ctx, "UPDATE companies SET state=$3 WHERE world=$1 AND account=$2", world, account, raw)
	must(t, err)
}

func TestGameProfilesPostgresIsolatedTick(t *testing.T) {
	ctx, s := openTestStore(t)
	set, err := readGameProfiles("../docs/catalogs/game-production-profiles.v1.json")
	must(t, err)
	world := fmt.Sprintf("game-profiles-%d", time.Now().UnixNano())
	must(t, s.seed(ctx, world, "baker", world+"-bakery-token"))
	must(t, s.seed(ctx, world, "other", world+"-other-token"))
	seedGameProductionState(t, ctx, s, world, "baker", true)
	seedGameProductionState(t, ctx, s, world, "other", false)
	must(t, s.tickWithApprovedGameProfiles(ctx, world, 1, set))
	must(t, s.tickWithApprovedGameProfiles(ctx, world, 1, set))
	bakery, err := s.company(ctx, world, "baker")
	must(t, err)
	other, err := s.company(ctx, world, "other")
	must(t, err)
	if bakery.GameProduction.Stock["bread_loaf"] != 24 || bakery.GameProduction.Stock["wheat_flour"] != 200 || bakery.GameProduction.ProducedAt["bread_loaf"] != 1 || bakery.Cash != initialCash-26000 {
		t.Fatalf("game profile persistence failed: %+v", bakery)
	}
	if other.GameProduction.Stock["bread_loaf"] != 0 || other.GameProduction.Stock["wheat_grain"] != 9000 || other.Cash != initialCash {
		t.Fatalf("other account mutated: %+v", other)
	}
	var count int
	must(t, s.db.QueryRow(ctx, "SELECT count(*) FROM ticks WHERE world=$1", world).Scan(&count))
	if count != 1 {
		t.Fatalf("duplicate tick: %d", count)
	}
	must(t, s.db.QueryRow(ctx, "SELECT count(*) FROM ledger WHERE world=$1 AND account='baker'", world).Scan(&count))
	if count != 3 {
		t.Fatalf("expected three production events: %d", count)
	}
	rows, err := s.db.Query(ctx, "SELECT movements FROM ledger WHERE world=$1 AND account='baker' AND event='game_profile_production' ORDER BY ordinal", world)
	must(t, err)
	movements := map[string]int64{}
	for rows.Next() {
		var raw []byte
		must(t, rows.Scan(&raw))
		var delta map[string]string
		must(t, json.Unmarshal(raw, &delta))
		for key, value := range delta {
			n, parseErr := strconv.ParseInt(value, 10, 64)
			must(t, parseErr)
			movements[key] += n
		}
	}
	must(t, rows.Err())
	rows.Close()
	for key, expected := range map[string]int64{"wheat_grain": -10000, "wheat_flour": 200, "water": -5440, "bread_loaf": 24, "cash_kopecks": -26000, "energy_wh": -4260, "unstocked_residue_g": 2600, "unstocked_bake_loss_g": 1496} {
		if movements[key] != expected {
			t.Fatalf("ledger mismatch for %s: %d != %d", key, movements[key], expected)
		}
	}
	must(t, s.tickWithApprovedGameProfiles(ctx, world, 2, set))
	again, err := s.company(ctx, world, "baker")
	must(t, err)
	if again.GameProduction.Stock["bread_loaf"] != 24 || again.Cash != bakery.Cash || again.GameProduction.ProducedAt["bread_loaf"] != 1 {
		t.Fatal("repeat tick minted goods or shifted sale eligibility")
	}
}

func TestGameProfilesPostgresInventoryAtomicAndNextTickSale(t *testing.T) {
	ctx, s := openTestStore(t)
	set, err := readGameProfiles("../docs/catalogs/game-production-profiles.v1.json")
	must(t, err)
	world := fmt.Sprintf("game-inventory-%d", time.Now().UnixNano())
	must(t, s.seed(ctx, world, "baker", world+"-token"))
	seedGameProductionState(t, ctx, s, world, "baker", true)
	c, err := s.company(ctx, world, "baker")
	must(t, err)
	inv := &gameInventory{}
	for good, qty := range c.GameProduction.Stock {
		inv.Lots = append(inv.Lots, gameInventoryLot{Good: good, Quantity: qty, AvailableForProductionTick: 0, AvailableForSaleTick: 0})
	}
	c.GameProduction.Inventory = inv
	raw, err := json.Marshal(c)
	must(t, err)
	_, err = s.db.Exec(ctx, "UPDATE companies SET state=$3 WHERE world=$1 AND account=$2", world, "baker", raw)
	must(t, err)
	must(t, s.tickWithApprovedGameProfiles(ctx, world, 1, set))
	after, err := s.company(ctx, world, "baker")
	must(t, err)
	if after.GameProduction.LastError != "" || after.GameProduction.Stock["bread_loaf"] != 24 || after.GameProduction.Inventory == nil {
		t.Fatalf("lot-backed production failed: %+v", after.GameProduction)
	}
	if err := checkGameInventoryStock(*after.GameProduction.Inventory, after.GameProduction.Stock); err != nil {
		t.Fatal(err)
	}
	if n, err := after.GameProduction.Inventory.available("bread_loaf", 1, true); err != nil || n != 0 {
		t.Fatalf("bread sold on production tick: %d %v", n, err)
	}
	if n, err := after.GameProduction.Inventory.available("bread_loaf", 2, true); err != nil || n != 24 {
		t.Fatalf("bread not saleable next tick: %d %v", n, err)
	}
	must(t, s.tickWithApprovedGameProfiles(ctx, world, 1, set))
	again, err := s.company(ctx, world, "baker")
	must(t, err)
	if again.GameProduction.Stock["bread_loaf"] != 24 || again.Cash != after.Cash {
		t.Fatal("duplicate tick changed lot-backed production")
	}
}

func TestGameProfilesPostgresInventoryRejectsReservedInputs(t *testing.T) {
	ctx, s := openTestStore(t)
	set, err := readGameProfiles("../docs/catalogs/game-production-profiles.v1.json")
	must(t, err)
	world := fmt.Sprintf("game-inventory-reserved-%d", time.Now().UnixNano())
	must(t, s.seed(ctx, world, "baker", world+"-token"))
	seedGameProductionState(t, ctx, s, world, "baker", true)
	c, err := s.company(ctx, world, "baker")
	must(t, err)
	inv := &gameInventory{Reserved: map[string]int64{"wheat_grain": 10000}}
	for good, qty := range c.GameProduction.Stock {
		inv.Lots = append(inv.Lots, gameInventoryLot{Good: good, Quantity: qty, AvailableForProductionTick: 0, AvailableForSaleTick: 0})
	}
	c.GameProduction.Inventory = inv
	raw, err := json.Marshal(c)
	must(t, err)
	_, err = s.db.Exec(ctx, "UPDATE companies SET state=$3 WHERE world=$1 AND account=$2", world, "baker", raw)
	must(t, err)
	must(t, s.tickWithApprovedGameProfiles(ctx, world, 1, set))
	after, err := s.company(ctx, world, "baker")
	must(t, err)
	if after.GameProduction.LastError == "" || after.GameProduction.Stock["wheat_grain"] != 10000 || after.GameProduction.Stock["bread_loaf"] != 0 || after.Cash != initialCash {
		t.Fatalf("reservation bypass or partial mutation: %+v", after)
	}
}
