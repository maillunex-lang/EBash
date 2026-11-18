package main

import (
	"math"
	"testing"
)

func TestGameProfileSafetyRejectsOverflowWithoutMutation(t *testing.T) {
	set, err := readGameProfiles("../docs/catalogs/game-production-profiles.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	factors := map[string]int64{"raw_material": 70, "technology": 70, "equipment": 70, "personnel": 70, "conditions": 70}
	line := gameLine{ID: "mill", ProfileID: "GAME-WHEAT-FLOUR-V1", RequestedBatches: math.MaxInt64, AvailableEquipment: "roller_mill_sifter", Factors: factors}
	stock := map[string]int64{"wheat_grain": 10000}
	budget := gameBudget{EnergyWh: 500, Workers: 1, CashKopecks: 12000, EquipmentCapacity: map[string]int64{"roller_mill_sifter": 1}}
	_, err = executeGameLines(set, []gameLine{line}, stock, &budget)
	if err == nil || stock["wheat_grain"] != 10000 || budget.CashKopecks != 12000 {
		t.Fatalf("overflow must fail atomically: %v", err)
	}
	line.RequestedBatches = 1
	stock["wheat_flour"] = math.MaxInt64
	_, err = executeGameLines(set, []gameLine{line}, stock, &budget)
	if err == nil || stock["wheat_flour"] != math.MaxInt64 || stock["wheat_grain"] != 10000 || budget.CashKopecks != 12000 {
		t.Fatalf("output overflow must fail atomically: %v", err)
	}
}
func TestGameProfileSafetyRejectsForgedRuns(t *testing.T) {
	set, err := readGameProfiles("../docs/catalogs/game-production-profiles.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	stock := map[string]int64{"wheat_grain": 10000}
	budget := gameBudget{EnergyWh: 500, Workers: 1, CashKopecks: 12000, EquipmentCapacity: map[string]int64{"roller_mill_sifter": 1}}
	for _, r := range []gameRun{{LineID: "a", ProfileID: "GAME-WHEAT-FLOUR-V1", Batches: 2, EnergyWh: 1000, CostKopecks: 24000}, {LineID: "a", ProfileID: "GAME-WHEAT-FLOUR-V1", Batches: 1, EnergyWh: 0, CostKopecks: 12000}} {
		_, _, err = validateGamePlan(set, []gameRun{r}, stock, budget)
		if err == nil {
			t.Fatalf("forged run accepted: %+v", r)
		}
	}
	if stock["wheat_grain"] != 10000 || budget.CashKopecks != 12000 {
		t.Fatal("validation mutated source")
	}
}

func TestGameProfileSafetyRejectsAggregateClaimsOverflow(t *testing.T) {
	set, err := readGameProfiles("../docs/catalogs/game-production-profiles.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	// Each claim is valid in int64, but their total is not.
	for i := range set.Profiles {
		if set.Profiles[i].ID == "GAME-WHEAT-FLOUR-V1" {
			set.Profiles[i].BatchInputs = map[string]int64{"wheat_grain": math.MaxInt64/2 + 1}
		}
	}
	factors := map[string]int64{"raw_material": 70, "technology": 70, "equipment": 70, "personnel": 70, "conditions": 70}
	lines := []gameLine{{ID: "a", ProfileID: "GAME-WHEAT-FLOUR-V1", RequestedBatches: 1, AvailableEquipment: "roller_mill_sifter", Factors: factors}, {ID: "b", ProfileID: "GAME-WHEAT-FLOUR-V1", RequestedBatches: 1, AvailableEquipment: "roller_mill_sifter", Factors: factors}}
	stock := map[string]int64{"wheat_grain": math.MaxInt64}
	budget := gameBudget{EnergyWh: 1000, Workers: 2, CashKopecks: 24000, EquipmentCapacity: map[string]int64{"roller_mill_sifter": 2}}
	_, err = executeGameLines(set, lines, stock, &budget)
	if err == nil || stock["wheat_grain"] != math.MaxInt64 || budget.CashKopecks != 24000 {
		t.Fatalf("aggregate claims overflow must fail atomically: %v", err)
	}
}

func TestGameProfileDisjointInputsCompeteForSharedEnergy(t *testing.T) {
	set, err := readGameProfiles("../docs/catalogs/game-production-profiles.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	p := profileByID(set, "GAME-WHEAT-FLOUR-V1")
	p.ID = "TEST-OTHER-MILL"
	p.BatchInputs = map[string]int64{"other_grain": 10000}
	p.BatchOutputs = map[string]int64{"other_flour": 7400}
	p.EquipmentType = "other_mill"
	set.Profiles = append(set.Profiles, p)
	f := map[string]int64{"raw_material": 70, "technology": 70, "equipment": 70, "personnel": 70, "conditions": 70}
	lines := []gameLine{{ID: "a", ProfileID: "GAME-WHEAT-FLOUR-V1", RequestedBatches: 1, AvailableEquipment: "roller_mill_sifter", Factors: f}, {ID: "b", ProfileID: "TEST-OTHER-MILL", RequestedBatches: 1, AvailableEquipment: "other_mill", Factors: f}}
	stock := map[string]int64{"wheat_grain": 10000, "other_grain": 10000}
	budget := gameBudget{EnergyWh: 500, Workers: 2, CashKopecks: 24000, EquipmentCapacity: map[string]int64{"roller_mill_sifter": 1, "other_mill": 1}}
	runs, err := executeGameLines(set, lines, stock, &budget)
	if err != nil || len(runs) != 2 || runs[0].Batches+runs[1].Batches != 1 || budget.EnergyWh != 0 || stock["wheat_grain"]+stock["other_grain"] != 10000 {
		t.Fatalf("shared energy not optimized: %v %+v stock=%+v", err, runs, stock)
	}
}

func TestGameOptimizerStableIDTieBreak(t *testing.T) {
	if !gameLexicographicallyPreferred([]int64{2, 1}, []int64{1, 2}) {
		t.Fatal("lower stable ID must win")
	}
	if gameLexicographicallyPreferred([]int64{1, 2}, []int64{2, 1}) {
		t.Fatal("higher stable ID must not win")
	}
	if gameLexicographicallyPreferred([]int64{1, 2}, []int64{1, 2}) {
		t.Fatal("identical vectors are not a strict improvement")
	}
}

func TestGameOptimizerPruningKeepsEqualBatchQuotaAndIDTies(t *testing.T) {
	set, err := readGameProfiles("../docs/catalogs/game-production-profiles.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	factors := map[string]int64{"raw_material": 70, "technology": 70, "equipment": 70, "personnel": 70, "conditions": 70}
	lines := []gameLine{
		{ID: "c", ProfileID: "GAME-WHEAT-FLOUR-V1", RequestedBatches: 2, AvailableEquipment: "roller_mill_sifter", Factors: factors},
		{ID: "a", ProfileID: "GAME-WHEAT-FLOUR-V1", RequestedBatches: 2, AvailableEquipment: "roller_mill_sifter", Factors: factors},
		{ID: "b", ProfileID: "GAME-WHEAT-FLOUR-V1", RequestedBatches: 2, AvailableEquipment: "roller_mill_sifter", Factors: factors},
	}
	stock := map[string]int64{"wheat_grain": 30000}
	budget := gameBudget{EnergyWh: 1500, Workers: 3, CashKopecks: 36000, EquipmentCapacity: map[string]int64{"roller_mill_sifter": 3}}
	runs, err := executeGameLines(set, lines, stock, &budget)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 3 || runs[0].LineID != "a" || runs[0].Batches != 1 || runs[1].Batches != 1 || runs[2].Batches != 1 || stock["wheat_flour"] != 22200 {
		t.Fatalf("pruning must preserve quota and stable ID tie: %+v stock=%+v", runs, stock)
	}
}
