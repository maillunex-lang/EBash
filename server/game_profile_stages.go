package main

import (
	"fmt"
	"sort"
)

// planGameProductionStages runs a DAG of recipe stages on private snapshots.
// All lines in one topological layer compete for shared budgets together.
// Cycles are broken by stable line ID, without running a line twice per tick.
func planGameProductionStages(set gameProfileSet, lines []gameLine, stock map[string]int64, budget gameBudget) ([]gameRun, error) {
	if stock == nil || budget.EquipmentCapacity == nil {
		return nil, fmt.Errorf("INVALID_STAGE_SNAPSHOT")
	}
	catalog := map[string]gameProfile{}
	for _, p := range set.Profiles {
		catalog[p.ID] = p
	}
	ordered := append([]gameLine(nil), lines...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })
	seen := map[string]bool{}
	for _, l := range ordered {
		if l.ID == "" || seen[l.ID] {
			return nil, fmt.Errorf("DUPLICATE_OR_EMPTY_LINE")
		}
		seen[l.ID] = true
		if catalog[l.ProfileID].ID == "" {
			return nil, fmt.Errorf("PROFILE_MISSING")
		}
	}
	// D-116: freeze requests and quotas before any intermediate production.
	frozenQuotas := freezeGameProductionQuotas(set, ordered, stock)
	// An edge A->B exists if A can produce an ingredient B requires.
	incoming := make([]int, len(ordered))
	edges := make([][]int, len(ordered))
	for i, a := range ordered {
		for j, b := range ordered {
			if i == j {
				continue
			}
			out := catalog[a.ProfileID].BatchOutputs
			for k := range catalog[b.ProfileID].BatchInputs {
				if _, ok := out[k]; ok {
					edges[i] = append(edges[i], j)
					incoming[j]++
					break
				}
			}
		}
	}
	snapshot := make(map[string]int64, len(stock))
	for k, v := range stock {
		snapshot[k] = v
	}
	capacity := make(map[string]int64, len(budget.EquipmentCapacity))
	for k, v := range budget.EquipmentCapacity {
		capacity[k] = v
	}
	available := gameBudget{EnergyWh: budget.EnergyWh, Workers: budget.Workers, CashKopecks: budget.CashKopecks, EquipmentCapacity: capacity}
	finished := make([]bool, len(ordered))
	result := make([]gameRun, 0, len(ordered))
	for len(result) < len(ordered) {
		stage := make([]gameLine, 0)
		indices := make([]int, 0)
		for i, l := range ordered {
			if !finished[i] && incoming[i] == 0 {
				stage = append(stage, l)
				indices = append(indices, i)
			}
		}
		cyclicStep := false
		if len(stage) == 0 {
			// DES-002: cycles are legal, but a line can run only once.
			// Pick the lowest stable ID among remaining lines; if it cannot
			// execute, do not silently discard a blocked cycle.
			for i, l := range ordered {
				if !finished[i] {
					stage = append(stage, l)
					indices = append(indices, i)
					break
				}
			}
			cyclicStep = true
		}
		// runGameLines handles independent stages, exact optimization handles
		// shared-input same-stage groups. Shared budget scarcity across unrelated
		// lines is intentionally blocked rather than allocated by ID priority.
		var runs []gameRun
		var err error
		if len(stage) > 1 {
			runs, err = optimizeCompetingGameLinesWithQuotas(set, stage, snapshot, available, frozenQuotas)
		} else {
			runs, err = runGameLines(set, stage, snapshot, available)
		}
		if err != nil {
			return nil, err
		}
		if cyclicStep && (len(runs) != 1 || runs[0].Batches == 0) {
			return nil, fmt.Errorf("PRODUCTION_CYCLE_NO_FEASIBLE_START")
		}
		// Replay this stage against its current snapshot with checked
		// arithmetic. No stage may create a negative balance or overflow
		// intermediate stock, even before the final transaction validation.
		nextStock, nextBudget, err := validateGamePlan(set, runs, snapshot, available)
		if err != nil {
			return nil, fmt.Errorf("PRODUCTION_STAGE_INVALID: %w", err)
		}
		snapshot = nextStock
		available = nextBudget
		result = append(result, runs...)
		for _, i := range indices {
			finished[i] = true
			for _, j := range edges[i] {
				incoming[j]--
			}
		}
	}
	return result, nil
}
