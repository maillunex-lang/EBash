package main

import (
	"fmt"
	"sort"
)

// gameLine describes a single independent production line for the isolated
// DES-004 scheduler. It does not automatically enable draft catalog recipes.
type gameLine struct {
	ID                 string
	ProfileID          string
	RequestedBatches   int64
	AvailableEquipment string
	Factors            map[string]int64
}

type gameBudget struct {
	EnergyWh          int64
	Workers           int64
	CashKopecks       int64
	EquipmentCapacity map[string]int64 // shared recipe-batches per equipment type
}

type gameRun struct {
	LineID      string
	ProfileID   string
	Batches     int64
	Reason      string
	EnergyWh    int64
	CostKopecks int64
	Quality     int64
}

// runGameLines computes one tick on private snapshots. Caller must persist
// stock and resource budgets in the same transaction as the tick/ledger.
// Missing/invalid profiles fail closed; outputs can feed later lines once.
func weightedGameQuality(weights map[string]int, factors map[string]int64) (int64, error) {
	if len(weights) != 5 || len(factors) != 5 {
		return 0, fmt.Errorf("QUALITY_FACTORS_MISSING")
	}
	var total int64
	for name, weight := range weights {
		score, ok := factors[name]
		if !ok || score < 1 || score > 100 || weight <= 0 {
			return 0, fmt.Errorf("QUALITY_FACTOR_INVALID: %s", name)
		}
		total += int64(weight) * score
	}
	return (total + 50) / 100, nil
}

func runGameLines(set gameProfileSet, lines []gameLine, stock map[string]int64, budget gameBudget) ([]gameRun, error) {
	if set.Status != "approved_game_balance_v1" {
		return nil, fmt.Errorf("GAME_PROFILE_NOT_APPROVED")
	}
	if stock == nil || budget.EquipmentCapacity == nil || budget.EnergyWh < 0 || budget.Workers < 0 || budget.CashKopecks < 0 {
		return nil, fmt.Errorf("INVALID_RESOURCE_BUDGET")
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
	}
	// For dependent recipes use the stage DAG instead of routing the entire
	// chain into the same-stage optimizer.
	for _, producer := range ordered {
		p := catalog[producer.ProfileID]
		for output := range p.BatchOutputs {
			for _, consumer := range ordered {
				if consumer.ID == producer.ID {
					continue
				}
				if _, ok := catalog[consumer.ProfileID].BatchInputs[output]; ok {
					return planGameProductionStages(set, ordered, stock, budget)
				}
			}
		}
	}
	// Every same-stage group of two or more lines shares at least the
	// company-level energy, cash and workforce pools. Optimize them together
	// even when their material inputs are disjoint.
	if len(ordered) >= 2 {
		return optimizeCompetingGameLines(set, ordered, stock, budget)
	}
	// Conflicting claims need DES-002 fair/exact optimization. Never apply
	// stable-ID greedy priority to a resource-constrained competing group.
	activeInputs := map[string]bool{}
	activeEquipment := map[string]bool{}
	var desiredEnergy, desiredWorkers, desiredCash int64
	for _, l := range ordered {
		p, ok := catalog[l.ProfileID]
		if !ok || l.RequestedBatches <= 0 || l.AvailableEquipment != p.EquipmentType {
			continue
		}
		q, err := weightedGameQuality(set.ProductionRules.QualityWeights, l.Factors)
		if err != nil {
			return nil, err
		}
		if q < p.QualityFloor {
			continue
		}
		planned := l.RequestedBatches
		if planned > p.EquipmentCapacity {
			planned = p.EquipmentCapacity
		}
		if planned > budget.EquipmentCapacity[p.EquipmentType] {
			planned = budget.EquipmentCapacity[p.EquipmentType]
		}
		if planned < p.MinimumBatches {
			continue
		}
		for key := range p.BatchInputs {
			if activeInputs[key] {
				return nil, fmt.Errorf("COMPETING_LINES_REQUIRE_OPTIMIZER: %s", key)
			}
			activeInputs[key] = true
		}
		if activeEquipment[p.EquipmentType] {
			return nil, fmt.Errorf("COMPETING_LINES_REQUIRE_OPTIMIZER: %s", p.EquipmentType)
		}
		activeEquipment[p.EquipmentType] = true
		desiredEnergy += planned * p.EnergyWh
		desiredWorkers += p.Workers
		desiredCash += planned * p.OperationCost
	}
	if len(ordered) > 1 && (desiredEnergy > budget.EnergyWh || desiredWorkers > budget.Workers || desiredCash > budget.CashKopecks) {
		return nil, fmt.Errorf("COMPETING_LINES_REQUIRE_OPTIMIZER: shared budget")
	}
	// Apply to cloned data only. The caller receives neither mutated stock nor
	// lost resources if a later input is invalid.
	remaining := make(map[string]int64, len(stock))
	for k, v := range stock {
		if v < 0 {
			return nil, fmt.Errorf("NEGATIVE_STOCK")
		}
		remaining[k] = v
	}
	capacity := make(map[string]int64, len(budget.EquipmentCapacity))
	for k, v := range budget.EquipmentCapacity {
		if v < 0 {
			return nil, fmt.Errorf("NEGATIVE_CAPACITY")
		}
		capacity[k] = v
	}
	availableEnergy, availableWorkers, availableCash := budget.EnergyWh, budget.Workers, budget.CashKopecks
	results := make([]gameRun, 0, len(ordered))
	for _, l := range ordered {
		p, ok := catalog[l.ProfileID]
		result := gameRun{LineID: l.ID, ProfileID: l.ProfileID}
		switch {
		case !ok:
			result.Reason = "PROFILE_MISSING"
		case l.RequestedBatches <= 0:
			result.Reason = "INVALID_REQUEST"
		case l.AvailableEquipment != p.EquipmentType:
			result.Reason = "INCOMPATIBLE_EQUIPMENT"
		default:
			quality, err := weightedGameQuality(set.ProductionRules.QualityWeights, l.Factors)
			if err != nil {
				return nil, err
			}
			if quality < p.QualityFloor {
				result.Reason = "QUALITY_BELOW_MINIMUM"
				results = append(results, result)
				continue
			}
			max := l.RequestedBatches
			if max > p.EquipmentCapacity {
				max = p.EquipmentCapacity
			}
			if max > capacity[p.EquipmentType] {
				max = capacity[p.EquipmentType]
			}
			if max > availableEnergy/p.EnergyWh {
				max = availableEnergy / p.EnergyWh
			}
			if p.OperationCost > 0 && max > availableCash/p.OperationCost {
				max = availableCash / p.OperationCost
			}
			if availableWorkers < p.Workers {
				max = 0
			}
			for k, n := range p.BatchInputs {
				if remaining[k]/n < max {
					max = remaining[k] / n
				}
			}
			if max < p.MinimumBatches {
				result.Reason = "INSUFFICIENT_INPUTS_OR_CAPACITY"
			} else {
				for k, n := range p.BatchInputs {
					remaining[k] -= n * max
				}
				for k, n := range p.BatchOutputs {
					remaining[k] += n * max
				}
				capacity[p.EquipmentType] -= max
				availableEnergy -= p.EnergyWh * max
				availableCash -= p.OperationCost * max
				availableWorkers -= p.Workers
				result.Batches = max
				result.EnergyWh = p.EnergyWh * max
				result.CostKopecks = p.OperationCost * max
				result.Quality = quality
			}
		}
		results = append(results, result)
	}
	// This currently provides a deterministic isolated scheduler and does NOT
	// implement the DES-002 global lexicographic optimizer for competing lines.
	// The snapshot is returned explicitly for transactional persistence.
	return results, nil
}

// executeGameLines commits the successful snapshot only after all decisions are
// computed. It is intentionally opt-in, for an isolated test world only.
func executeGameLines(set gameProfileSet, lines []gameLine, stock map[string]int64, budget *gameBudget) ([]gameRun, error) {
	if budget == nil {
		return nil, fmt.Errorf("INVALID_RESOURCE_BUDGET")
	}
	if err := preflightGameLines(set, lines, stock, *budget); err != nil {
		return nil, err
	}
	results, err := runGameLines(set, lines, stock, *budget)
	if err != nil {
		return nil, err
	}
	// The plan must correspond to the submitted requests, not merely to
	// a physically feasible collection of batches.
	requested := make(map[string]gameLine, len(lines))
	for _, l := range lines {
		requested[l.ID] = l
	}
	if len(results) != len(lines) {
		return nil, fmt.Errorf("GAME_PLAN_LINE_COUNT_MISMATCH")
	}
	for _, r := range results {
		l, ok := requested[r.LineID]
		if !ok || l.ProfileID != r.ProfileID || r.Batches > l.RequestedBatches {
			return nil, fmt.Errorf("GAME_PLAN_REQUEST_MISMATCH")
		}
		if r.Batches > 0 && l.AvailableEquipment != profileByID(set, r.ProfileID).EquipmentType {
			return nil, fmt.Errorf("GAME_PLAN_EQUIPMENT_MISMATCH")
		}
	}
	next, remaining, err := validateGamePlan(set, results, stock, *budget)
	if err != nil {
		return nil, err
	}
	for k := range stock {
		delete(stock, k)
	}
	for k, v := range next {
		stock[k] = v
	}
	*budget = remaining
	return results, nil
}
