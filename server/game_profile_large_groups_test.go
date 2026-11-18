package main

import (
	"fmt"
	"strings"
	"testing"
	"time"
)

// Large-group safety: bounded search, deterministic accounting, no resource mutation.
func TestGameOptimizerLargeGroupsBoundedAndConservative(t *testing.T) {
	set, err := readGameProfiles("../docs/catalogs/game-production-profiles.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	factors := map[string]int64{"raw_material": 70, "technology": 70, "equipment": 70, "personnel": 70, "conditions": 70}
	for _, n := range []int{9, 16, 32, 64} {
		t.Run(fmt.Sprintf("lines-%d", n), func(t *testing.T) {
			lines := make([]gameLine, n)
			for i := range lines {
				lines[i] = gameLine{ID: fmt.Sprintf("line-%02d", i), ProfileID: "GAME-WHEAT-FLOUR-V1", RequestedBatches: 2, AvailableEquipment: "roller_mill_sifter", Factors: factors}
			}
			stock := map[string]int64{"wheat_grain": int64(n) * 10000}
			budget := gameBudget{EnergyWh: int64(n) * 500, Workers: int64(n), CashKopecks: int64(n) * 12000, EquipmentCapacity: map[string]int64{"roller_mill_sifter": int64(n)}}
			start := time.Now()
			runs, e := optimizeCompetingGameLines(set, lines, stock, budget)
			elapsed := time.Since(start)
			// Deadline applies to search only. Allow generous wall-clock margin for preparation and VM load.
			if elapsed > 3*time.Second {
				t.Errorf("large group took %s", elapsed)
			}
			if e != nil {
				if !strings.Contains(e.Error(), "OPTIMIZER_TIME_LIMIT_UNPROVEN") {
					t.Fatalf("unexpected error: %v", e)
				}
			} else {
				var batches, energy, cash int64
				var workers int64
				for _, r := range runs {
					if r.Batches < 0 || r.Batches > 2 {
						t.Fatalf("invalid batches: %+v", r)
					}
					batches += r.Batches
					energy += r.EnergyWh
					cash += r.CostKopecks
					if r.Batches > 0 {
						workers++
					}
				}
				if batches > int64(n) || batches*10000 > stock["wheat_grain"] || energy > budget.EnergyWh || cash > budget.CashKopecks || workers > budget.Workers {
					t.Fatalf("overcommit: batches=%d energy=%d cash=%d workers=%d", batches, energy, cash, workers)
				}
				if len(runs) != n {
					t.Fatalf("wrong run count: %d", len(runs))
				}
			}
			if stock["wheat_grain"] != int64(n)*10000 || budget.EnergyWh != int64(n)*500 {
				t.Fatal("optimizer mutated input snapshots")
			}
		})
	}
}

// An infeasible large group must never consume stock or fabricate production.
func TestGameOptimizerLargeGroupNoFeasibleBatch(t *testing.T) {
	set, err := readGameProfiles("../docs/catalogs/game-production-profiles.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	factors := map[string]int64{"raw_material": 70, "technology": 70, "equipment": 70, "personnel": 70, "conditions": 70}
	lines := make([]gameLine, 64)
	for i := range lines {
		lines[i] = gameLine{ID: fmt.Sprintf("line-%02d", i), ProfileID: "GAME-WHEAT-FLOUR-V1", RequestedBatches: 2, AvailableEquipment: "roller_mill_sifter", Factors: factors}
	}
	stock := map[string]int64{"wheat_grain": 0}
	budget := gameBudget{EnergyWh: 100000, Workers: 64, CashKopecks: 1000000, EquipmentCapacity: map[string]int64{"roller_mill_sifter": 128}}
	runs, e := optimizeCompetingGameLines(set, lines, stock, budget)
	if e != nil && !strings.Contains(e.Error(), "OPTIMIZER_TIME_LIMIT_UNPROVEN") {
		t.Fatalf("unexpected error: %v", e)
	}
	for _, r := range runs {
		if r.Batches != 0 || r.EnergyWh != 0 || r.CostKopecks != 0 {
			t.Fatalf("production without grain: %+v", r)
		}
	}
	if stock["wheat_grain"] != 0 {
		t.Fatal("mutated stock")
	}
}

// Force deadline expiration before the first search state. The optimizer must
// fail closed, rather than inventing a zero-batch proof or spending resources.
func TestGameOptimizerLargeGroupDeterministicDeadline(t *testing.T) {
	set, err := readGameProfiles("../docs/catalogs/game-production-profiles.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	factors := map[string]int64{"raw_material": 70, "technology": 70, "equipment": 70, "personnel": 70, "conditions": 70}
	lines := make([]gameLine, 9)
	for i := range lines {
		lines[i] = gameLine{ID: fmt.Sprintf("line-%02d", i), ProfileID: "GAME-WHEAT-FLOUR-V1", RequestedBatches: 2, AvailableEquipment: "roller_mill_sifter", Factors: factors}
	}
	stock := map[string]int64{"wheat_grain": 180000}
	budget := gameBudget{EnergyWh: 9000, Workers: 9, CashKopecks: 216000, EquipmentCapacity: map[string]int64{"roller_mill_sifter": 18}}
	base := time.Unix(0, 0)
	calls := 0
	now := func() time.Time {
		calls++
		if calls == 1 {
			return base
		}
		return base.Add(101 * time.Millisecond)
	}
	runs, e := optimizeCompetingGameLinesWithQuotasClock(set, lines, stock, budget, nil, now)
	if e == nil || !strings.Contains(e.Error(), "OPTIMIZER_TIME_LIMIT_UNPROVEN") || len(runs) != 0 {
		t.Fatalf("expired search must fail closed: runs=%+v err=%v", runs, e)
	}
	if calls < 2 || stock["wheat_grain"] != 180000 || budget.EnergyWh != 9000 {
		t.Fatal("deadline or immutable snapshot violated")
	}
}
