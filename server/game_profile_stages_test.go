package main

import (
	"math"
	"testing"
)

func TestGameProfileCompetingMillsToBreadSameTick(t *testing.T) {
	set, lines := competingMillsScenario(t)
	verifyCompetingMillsToBread(t, set, lines)
}

func BenchmarkCompetingMillsToBread(b *testing.B) {
	set, lines := competingMillsScenario(b)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		verifyCompetingMillsToBread(b, set, lines)
	}
}

func competingMillsScenario(t testing.TB) (gameProfileSet, []gameLine) {
	t.Helper()
	set, err := readGameProfiles("../docs/catalogs/game-production-profiles.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	f := map[string]int64{"raw_material": 70, "technology": 70, "equipment": 70, "personnel": 70, "conditions": 70}
	lines := []gameLine{
		{ID: "mill-b", ProfileID: "GAME-WHEAT-FLOUR-V1", RequestedBatches: 1, AvailableEquipment: "roller_mill_sifter", Factors: f},
		{ID: "oven", ProfileID: "GAME-BREAD-LOAF-V1", RequestedBatches: 8, AvailableEquipment: "proofer_oven", Factors: f},
		{ID: "mixer", ProfileID: "GAME-BREAD-DOUGH-V1", RequestedBatches: 8, AvailableEquipment: "spiral_mixer", Factors: f},
		{ID: "mill-a", ProfileID: "GAME-WHEAT-FLOUR-V1", RequestedBatches: 1, AvailableEquipment: "roller_mill_sifter", Factors: f},
	}
	return set, lines
}

func verifyCompetingMillsToBread(t testing.TB, set gameProfileSet, lines []gameLine) {
	t.Helper()
	stock := map[string]int64{"wheat_grain": 20000, "water": 5440, "salt": 144, "yeast": 112}
	budget := gameBudget{EnergyWh: 4760, Workers: 4, CashKopecks: 38000, EquipmentCapacity: map[string]int64{"roller_mill_sifter": 2, "spiral_mixer": 8, "proofer_oven": 8}}
	runs, err := executeGameLines(set, lines, stock, &budget)
	if err != nil {
		t.Fatal(err)
	}
	if len(runs) != 4 || runs[0].LineID != "mill-a" || runs[1].LineID != "mill-b" || runs[2].LineID != "mixer" || runs[3].LineID != "oven" {
		t.Fatalf("wrong stage order: %+v", runs)
	}
	if stock["bread_loaf"] != 24 || stock["wheat_flour"] != 7600 || stock["bread_dough"] != 0 || budget.EnergyWh != 0 || budget.CashKopecks != 0 || budget.Workers != 0 {
		t.Fatalf("wrong accounting: stock=%+v budget=%+v", stock, budget)
	}
}
func TestGameProfileCycleRunsOnceWithSeed(t *testing.T) {
	set, err := readGameProfiles("../docs/catalogs/game-production-profiles.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	// Artificial cycle in test-only copy of approved profiles.
	for i := range set.Profiles {
		if set.Profiles[i].ID == "GAME-WHEAT-FLOUR-V1" {
			set.Profiles[i].BatchInputs = map[string]int64{"bread_loaf": 1}
		}
		if set.Profiles[i].ID == "GAME-BREAD-LOAF-V1" {
			set.Profiles[i].BatchOutputs = map[string]int64{"bread_loaf": 1}
		}
	}
	f := map[string]int64{"raw_material": 70, "technology": 70, "equipment": 70, "personnel": 70, "conditions": 70}
	lines := []gameLine{
		{ID: "mill", ProfileID: "GAME-WHEAT-FLOUR-V1", RequestedBatches: 1, AvailableEquipment: "roller_mill_sifter", Factors: f},
		{ID: "mixer", ProfileID: "GAME-BREAD-DOUGH-V1", RequestedBatches: 1, AvailableEquipment: "spiral_mixer", Factors: f},
		{ID: "oven", ProfileID: "GAME-BREAD-LOAF-V1", RequestedBatches: 1, AvailableEquipment: "proofer_oven", Factors: f},
	}
	stock := map[string]int64{"wheat_grain": 10000, "bread_loaf": 10, "water": 680, "salt": 18, "yeast": 14}
	budget := gameBudget{EnergyWh: 5000, Workers: 3, CashKopecks: 30000, EquipmentCapacity: map[string]int64{"roller_mill_sifter": 1, "spiral_mixer": 1, "proofer_oven": 1}}
	runs, err := executeGameLines(set, lines, stock, &budget)
	if err != nil || len(runs) != 3 || stock["bread_loaf"] != 10 || stock["wheat_flour"] != 6500 || budget.CashKopecks != 16250 {
		t.Fatalf("cycle must run once in stable order: %v %+v stock=%+v budget=%+v", err, runs, stock, budget)
	}
	// No initial stock of the feedback material: the cycle cannot start.
	blockedStock := map[string]int64{"bread_loaf": 0}
	blockedBudget := gameBudget{EnergyWh: 5000, Workers: 3, CashKopecks: 30000, EquipmentCapacity: map[string]int64{"roller_mill_sifter": 1, "spiral_mixer": 1, "proofer_oven": 1}}
	_, err = executeGameLines(set, lines, blockedStock, &blockedBudget)
	if err == nil || blockedStock["bread_loaf"] != 0 || blockedBudget.CashKopecks != 30000 {
		t.Fatalf("unseeded cycle must fail atomically: %v", err)
	}
}

func TestGameProfileStagesIntermediateOverflowAtomic(t *testing.T) {
	set, err := readGameProfiles("../docs/catalogs/game-production-profiles.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	f := map[string]int64{"raw_material": 70, "technology": 70, "equipment": 70, "personnel": 70, "conditions": 70}
	lines := []gameLine{
		{ID: "mill", ProfileID: "GAME-WHEAT-FLOUR-V1", RequestedBatches: 1, AvailableEquipment: "roller_mill_sifter", Factors: f},
		{ID: "mixer", ProfileID: "GAME-BREAD-DOUGH-V1", RequestedBatches: 1, AvailableEquipment: "spiral_mixer", Factors: f},
	}
	stock := map[string]int64{"wheat_grain": 10000, "wheat_flour": math.MaxInt64 - 7000, "water": 680, "salt": 18, "yeast": 14}
	budget := gameBudget{EnergyWh: 1000, Workers: 2, CashKopecks: 20000, EquipmentCapacity: map[string]int64{"roller_mill_sifter": 1, "spiral_mixer": 1}}
	_, err = executeGameLines(set, lines, stock, &budget)
	if err == nil || stock["wheat_grain"] != 10000 || stock["wheat_flour"] != math.MaxInt64-7000 || budget.CashKopecks != 20000 {
		t.Fatalf("intermediate overflow must not mutate: %v stock=%+v budget=%+v", err, stock, budget)
	}
}

func TestGameProfileQuotasFrozenBeforeStages(t *testing.T) {
	set, err := readGameProfiles("../docs/catalogs/game-production-profiles.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	f := map[string]int64{"raw_material": 70, "technology": 70, "equipment": 70, "personnel": 70, "conditions": 70}
	lines := []gameLine{
		{ID: "mill", ProfileID: "GAME-WHEAT-FLOUR-V1", RequestedBatches: 1, AvailableEquipment: "roller_mill_sifter", Factors: f},
		{ID: "mixer-a", ProfileID: "GAME-BREAD-DOUGH-V1", RequestedBatches: 8, AvailableEquipment: "spiral_mixer", Factors: f},
		{ID: "mixer-b", ProfileID: "GAME-BREAD-DOUGH-V1", RequestedBatches: 8, AvailableEquipment: "spiral_mixer", Factors: f},
	}
	stock := map[string]int64{"wheat_grain": 10000, "wheat_flour": 900, "water": 10880, "salt": 288, "yeast": 224}
	frozen := freezeGameProductionQuotas(set, lines, stock)
	if frozen["mixer-a"]["wheat_flour"] != 450 || frozen["mixer-b"]["wheat_flour"] != 450 {
		t.Fatalf("initial flour quotas must be 450 each: %+v", frozen)
	}
	budget := gameBudget{EnergyWh: 1012, Workers: 3, CashKopecks: 23200, EquipmentCapacity: map[string]int64{"roller_mill_sifter": 1, "spiral_mixer": 16}}
	runs, err := executeGameLines(set, lines, stock, &budget)
	if err != nil {
		t.Fatal(err)
	}
	var total int64
	for _, r := range runs {
		if r.ProfileID == "GAME-BREAD-DOUGH-V1" {
			total += r.Batches
		}
	}
	if total != 9 || stock["wheat_flour"] != 200 {
		t.Fatalf("unexpected cross-stage accounting: %+v stock=%+v", runs, stock)
	}
	// The initial quota remains unchanged after new flour is produced.
	if frozen["mixer-a"]["wheat_flour"] != 450 {
		t.Fatal("quota changed during production")
	}
}
