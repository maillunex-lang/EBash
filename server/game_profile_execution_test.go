package main

import (
	"fmt"
	"testing"
)

func TestGameProfileBreadTick(t *testing.T) {
	set, err := readGameProfiles("../docs/catalogs/game-production-profiles.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	stock := map[string]int64{"wheat_grain": 10000, "water": 5440, "salt": 144, "yeast": 112}
	budget := gameBudget{EnergyWh: 4260, Workers: 3, CashKopecks: 26000, EquipmentCapacity: map[string]int64{"roller_mill_sifter": 1, "spiral_mixer": 8, "proofer_oven": 8}}
	lines := []gameLine{
		{ID: "01-mill", ProfileID: "GAME-WHEAT-FLOUR-V1", RequestedBatches: 1, AvailableEquipment: "roller_mill_sifter", Factors: map[string]int64{"raw_material": 70, "technology": 70, "equipment": 70, "personnel": 70, "conditions": 70}},
		{ID: "02-mixer", ProfileID: "GAME-BREAD-DOUGH-V1", RequestedBatches: 8, AvailableEquipment: "spiral_mixer", Factors: map[string]int64{"raw_material": 70, "technology": 70, "equipment": 70, "personnel": 70, "conditions": 70}},
		{ID: "03-oven", ProfileID: "GAME-BREAD-LOAF-V1", RequestedBatches: 8, AvailableEquipment: "proofer_oven", Factors: map[string]int64{"raw_material": 70, "technology": 70, "equipment": 70, "personnel": 70, "conditions": 70}},
	}
	runs, err := executeGameLines(set, lines, stock, &budget)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 3 || runs[0].Batches != 1 || runs[1].Batches != 8 || runs[2].Batches != 8 {
		t.Fatalf("unexpected runs: %+v", runs)
	}
	if stock["bread_loaf"] != 24 || stock["wheat_flour"] != 200 || stock["wheat_grain"] != 0 || stock["bread_dough"] != 0 || budget.EnergyWh != 0 || budget.Workers != 0 || budget.CashKopecks != 0 {
		t.Fatalf("incorrect accounting stock=%+v budget=%+v", stock, budget)
	}
	// A second evaluation on the same resource pool cannot create more output.
	twice, err := executeGameLines(set, lines, stock, &budget)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range twice {
		if r.Batches != 0 {
			t.Fatalf("repeated tick produced: %+v", r)
		}
	}
	if stock["bread_loaf"] != 24 {
		t.Fatal("repeated run minted goods")
	}
}
func TestGameProfileInsufficientResourcesFailClosed(t *testing.T) {
	set, err := readGameProfiles("../docs/catalogs/game-production-profiles.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	line := gameLine{ID: "mill", ProfileID: "GAME-WHEAT-FLOUR-V1", RequestedBatches: 1, AvailableEquipment: "roller_mill_sifter", Factors: map[string]int64{"raw_material": 70, "technology": 70, "equipment": 70, "personnel": 70, "conditions": 70}}
	checks := []struct {
		name                                      string
		grain, energy, workers, capacity, quality int64
	}{
		{"grain", 9999, 500, 1, 1, 70}, {"energy", 10000, 499, 1, 1, 70}, {"workers", 10000, 500, 0, 1, 70},
		{"capacity", 10000, 500, 1, 0, 70}, {"quality", 10000, 500, 1, 1, 44},
	}
	for _, tc := range checks {
		t.Run(tc.name, func(t *testing.T) {
			stock := map[string]int64{"wheat_grain": tc.grain}
			budget := gameBudget{EnergyWh: tc.energy, Workers: tc.workers, CashKopecks: 12000, EquipmentCapacity: map[string]int64{"roller_mill_sifter": tc.capacity}}
			l := line
			for k := range l.Factors {
				l.Factors[k] = tc.quality
			}
			result, err := executeGameLines(set, []gameLine{l}, stock, &budget)
			if err != nil || result[0].Batches != 0 || stock["wheat_grain"] != tc.grain || budget.EnergyWh != tc.energy || budget.Workers != tc.workers {
				t.Fatalf("not fail-closed: %v, %+v", err, result)
			}
		})
	}
	stock := map[string]int64{"wheat_grain": 10000}
	budget := gameBudget{EnergyWh: 500, Workers: 1, CashKopecks: 12000, EquipmentCapacity: map[string]int64{"roller_mill_sifter": 1}}
	_, err = executeGameLines(set, []gameLine{line, line}, stock, &budget)
	if err == nil || stock["wheat_grain"] != 10000 || budget.EnergyWh != 500 {
		t.Fatal("duplicate line must fail atomically")
	}
}

func TestGameProfileExactCompetingScheduler(t *testing.T) {
	set, err := readGameProfiles("../docs/catalogs/game-production-profiles.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	factors := map[string]int64{"raw_material": 70, "technology": 70, "equipment": 70, "personnel": 70, "conditions": 70}
	first := gameLine{ID: "a", ProfileID: "GAME-WHEAT-FLOUR-V1", RequestedBatches: 2, AvailableEquipment: "roller_mill_sifter", Factors: factors}
	second := first
	second.ID = "b"
	stock := map[string]int64{"wheat_grain": 30000}
	budget := gameBudget{EnergyWh: 2000, Workers: 2, CashKopecks: 48000, EquipmentCapacity: map[string]int64{"roller_mill_sifter": 4}}
	runs, err := executeGameLines(set, []gameLine{second, first}, stock, &budget)
	if err != nil || len(runs) != 2 || runs[0].Batches+runs[1].Batches != 3 || stock["wheat_grain"] != 0 || stock["wheat_flour"] != 22200 {
		t.Fatalf("incorrect exact result: %v %+v stock=%+v", err, runs, stock)
	}
	// Two requests of two batches each and three available batches: the
	// equal-deviation tie is resolved by stable ID.
	if runs[0].LineID != "a" || runs[0].Batches != 2 || runs[1].Batches != 1 {
		t.Fatalf("unstable exact tie: %+v", runs)
	}
	// Limited cash cannot fund even one batch: do not spend stock.
	stock = map[string]int64{"wheat_grain": 30000}
	budget = gameBudget{EnergyWh: 2000, Workers: 2, CashKopecks: 11999, EquipmentCapacity: map[string]int64{"roller_mill_sifter": 4}}
	runs, err = executeGameLines(set, []gameLine{first, second}, stock, &budget)
	if err != nil || runs[0].Batches+runs[1].Batches != 0 || stock["wheat_grain"] != 30000 {
		t.Fatalf("cash constraint violated: %v %+v", err, runs)
	}
}

func TestGameProfileQualityWeightsAndFunds(t *testing.T) {
	set, err := readGameProfiles("../docs/catalogs/game-production-profiles.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	// 30*80 + 25*60 + 20*80 + 15*60 + 10*60 = 7000 -> 70.
	score, err := weightedGameQuality(set.ProductionRules.QualityWeights, map[string]int64{"raw_material": 80, "technology": 60, "equipment": 80, "personnel": 60, "conditions": 60})
	if err != nil || score != 70 {
		t.Fatalf("weighted quality error: %d %v", score, err)
	}
	rounded, err := weightedGameQuality(set.ProductionRules.QualityWeights, map[string]int64{
		"raw_material": 70, "technology": 70, "equipment": 70, "personnel": 70, "conditions": 75,
	})
	if err != nil || rounded != 71 {
		t.Fatalf("quality .5 must round upward: %d %v", rounded, err)
	}
	stock := map[string]int64{"wheat_grain": 10000}
	budget := gameBudget{EnergyWh: 500, Workers: 1, CashKopecks: 11999, EquipmentCapacity: map[string]int64{"roller_mill_sifter": 1}}
	line := gameLine{ID: "mill", ProfileID: "GAME-WHEAT-FLOUR-V1", RequestedBatches: 1, AvailableEquipment: "roller_mill_sifter", Factors: map[string]int64{"raw_material": 70, "technology": 70, "equipment": 70, "personnel": 70, "conditions": 70}}
	runs, err := executeGameLines(set, []gameLine{line}, stock, &budget)
	if err != nil || runs[0].Batches != 0 || budget.CashKopecks != 11999 || stock["wheat_grain"] != 10000 {
		t.Fatalf("insufficient cash must block: %v %+v", err, runs)
	}
}

func TestGameProfileLargeCompetingGroupBounded(t *testing.T) {
	set, err := readGameProfiles("../docs/catalogs/game-production-profiles.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	factors := map[string]int64{"raw_material": 70, "technology": 70, "equipment": 70, "personnel": 70, "conditions": 70}
	lines := make([]gameLine, 9)
	for i := range lines {
		lines[i] = gameLine{ID: fmt.Sprintf("line-%02d", i), ProfileID: "GAME-WHEAT-FLOUR-V1", RequestedBatches: 1, AvailableEquipment: "roller_mill_sifter", Factors: factors}
	}
	stock := map[string]int64{"wheat_grain": 50000}
	budget := gameBudget{EnergyWh: 5000, Workers: 9, CashKopecks: 120000, EquipmentCapacity: map[string]int64{"roller_mill_sifter": 9}}
	runs, err := executeGameLines(set, lines, stock, &budget)
	if err != nil {
		t.Fatal(err)
	}
	var total int64
	for _, r := range runs {
		total += r.Batches
	}
	if total != 5 || stock["wheat_grain"] != 0 || stock["wheat_flour"] != 37000 {
		t.Fatalf("incorrect 9-line allocation: %+v %+v", runs, stock)
	}
	// Huge requests must be rejected without consuming resources.
	for i := range lines {
		lines[i].RequestedBatches = 8
	}
	before := stock["wheat_flour"]
	for i := 9; i < 14; i++ {
		lines = append(lines, gameLine{ID: fmt.Sprintf("line-%02d", i), ProfileID: "GAME-WHEAT-FLOUR-V1", RequestedBatches: 2, AvailableEquipment: "roller_mill_sifter", Factors: factors})
	}
	stock["wheat_grain"] = 800000
	budget.EnergyWh = 500000
	budget.Workers = 9
	budget.CashKopecks = 5000000
	budget.EquipmentCapacity["roller_mill_sifter"] = 100
	beforeGrain := stock["wheat_grain"]
	largeRuns, err := executeGameLines(set, lines, stock, &budget)
	if err != nil {
		if stock["wheat_flour"] != before || stock["wheat_grain"] != beforeGrain {
			t.Fatalf("failed search mutated stock: %v", err)
		}
	} else {
		var batches int64
		for _, run := range largeRuns {
			batches += run.Batches
		}
		if batches <= 0 || stock["wheat_flour"] != before+batches*7400 || stock["wheat_grain"] != beforeGrain-batches*10000 {
			t.Fatalf("large search accounting: %+v stock=%+v", largeRuns, stock)
		}
	}
}
