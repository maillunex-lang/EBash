package main

import (
	"fmt"
	"math"
	"sort"
	"time"
)

// optimizeCompetingGameLines implements the exact small-group DES-002 search.
// It accepts only a single-stage snapshot: outputs must not feed any input in
// the same group. Such chains require a separate topological scheduling phase.
// The production snapshot and proportional quotas are frozen before search.
func optimizeCompetingGameLines(set gameProfileSet, lines []gameLine, stock map[string]int64, budget gameBudget) ([]gameRun, error) {
	return optimizeCompetingGameLinesWithQuotas(set, lines, stock, budget, nil)
}

// frozenQuotas are computed once from the tick-opening stock and requests.
// A nil override preserves standalone same-stage behavior.
func optimizeCompetingGameLinesWithQuotas(set gameProfileSet, lines []gameLine, stock map[string]int64, budget gameBudget, frozenQuotas map[string]map[string]float64) ([]gameRun, error) {
	return optimizeCompetingGameLinesWithQuotasClock(set, lines, stock, budget, frozenQuotas, time.Now)
}

// A clock parameter makes the 100ms search cutoff deterministic in tests.
func optimizeCompetingGameLinesWithQuotasClock(set gameProfileSet, lines []gameLine, stock map[string]int64, budget gameBudget, frozenQuotas map[string]map[string]float64, now func() time.Time) ([]gameRun, error) {
	if len(lines) < 2 || len(lines) > 64 {
		return nil, fmt.Errorf("OPTIMIZER_GROUP_SIZE_UNSUPPORTED")
	}
	if set.Status != "approved_game_balance_v1" {
		return nil, fmt.Errorf("GAME_PROFILE_NOT_APPROVED")
	}
	if budget.EnergyWh < 0 || budget.Workers < 0 || budget.CashKopecks < 0 || stock == nil || budget.EquipmentCapacity == nil {
		return nil, fmt.Errorf("INVALID_RESOURCE_BUDGET")
	}
	for _, v := range stock {
		if v < 0 {
			return nil, fmt.Errorf("NEGATIVE_STOCK")
		}
	}
	for _, v := range budget.EquipmentCapacity {
		if v < 0 {
			return nil, fmt.Errorf("NEGATIVE_CAPACITY")
		}
	}
	ordered := append([]gameLine(nil), lines...)
	sort.Slice(ordered, func(i, j int) bool { return ordered[i].ID < ordered[j].ID })
	profiles := make([]gameProfile, len(ordered))
	quality := make([]int64, len(ordered))
	inputs := map[string]bool{}
	outputs := map[string]bool{}
	maxes := make([]int64, len(ordered))
	seen := map[string]bool{}
	for i, l := range ordered {
		if l.ID == "" || seen[l.ID] {
			return nil, fmt.Errorf("DUPLICATE_OR_EMPTY_LINE")
		}
		seen[l.ID] = true
		p := profileByID(set, l.ProfileID)
		if p.ID == "" || l.RequestedBatches <= 0 || l.AvailableEquipment != p.EquipmentType {
			return nil, fmt.Errorf("INVALID_COMPETING_LINE")
		}
		q, err := weightedGameQuality(set.ProductionRules.QualityWeights, l.Factors)
		if err != nil {
			return nil, err
		}
		profiles[i] = p
		quality[i] = q
		if q < p.QualityFloor {
			continue
		}
		max := l.RequestedBatches
		if max > p.EquipmentCapacity {
			max = p.EquipmentCapacity
		}
		if max > budget.EquipmentCapacity[p.EquipmentType] {
			max = budget.EquipmentCapacity[p.EquipmentType]
		}
		if max > budget.EnergyWh/p.EnergyWh {
			max = budget.EnergyWh / p.EnergyWh
		}
		if p.OperationCost > 0 && max > budget.CashKopecks/p.OperationCost {
			max = budget.CashKopecks / p.OperationCost
		}
		if budget.Workers < p.Workers {
			max = 0
		}
		for k, n := range p.BatchInputs {
			if n <= 0 || stock[k] < 0 {
				return nil, fmt.Errorf("INVALID_INPUT_STOCK")
			}
			if max > stock[k]/n {
				max = stock[k] / n
			}
			inputs[k] = true
		}
		for k := range p.BatchOutputs {
			outputs[k] = true
		}
		if max >= p.MinimumBatches {
			maxes[i] = max
		}
	}
	for k := range outputs {
		if inputs[k] {
			return nil, fmt.Errorf("INTERMEDIATE_DEPENDENCY_REQUIRES_STAGE_SCHEDULER")
		}
	}
	// Proportional frozen per-resource claims; compare deviation only among
	// equal-maximal plans, normalized by the requesting line's demand.
	totalClaims := map[string]float64{}
	for i, p := range profiles {
		for k, n := range p.BatchInputs {
			totalClaims[k] += float64(n) * float64(ordered[i].RequestedBatches)
		}
	}
	quotas := make([]map[string]float64, len(ordered))
	for i, p := range profiles {
		quotas[i] = map[string]float64{}
		for k, n := range p.BatchInputs {
			claim := float64(n) * float64(ordered[i].RequestedBatches)
			if totalClaims[k] > 0 {
				quotas[i][k] = math.Min(float64(stock[k]), totalClaims[k]) * claim / totalClaims[k]
			}
		}
	}
	if frozenQuotas != nil {
		for i, l := range ordered {
			original, ok := frozenQuotas[l.ID]
			if !ok {
				return nil, fmt.Errorf("MISSING_FROZEN_QUOTA: %s", l.ID)
			}
			for k := range profiles[i].BatchInputs {
				q, exists := original[k]
				if !exists {
					return nil, fmt.Errorf("MISSING_FROZEN_QUOTA_RESOURCE: %s", k)
				}
				quotas[i][k] = q
			}
		}
	}
	current := make([]int64, len(ordered))
	best := make([]int64, len(ordered))
	used := map[string]int64{}
	equipment := map[string]int64{}
	var energy, cash, workers int64
	bestTotal := int64(-1)
	bestDeviation := math.Inf(1)
	// Large groups have a strict 100ms search budget. Small groups must
	// finish exactly: fail closed if the bounded state space is excessive.
	// Preflight prevents unbounded 8-line loops and integer overflow.
	states := uint64(1)
	for _, max := range maxes {
		if len(ordered) > 8 {
			break
		}
		if max > 1000000 {
			return nil, fmt.Errorf("OPTIMIZER_SEARCH_SPACE_TOO_LARGE")
		}
		next := uint64(max) + 1
		if states > 2000000/next {
			return nil, fmt.Errorf("OPTIMIZER_SEARCH_SPACE_TOO_LARGE")
		}
		states *= next
	}
	// Optimistic suffix bounds safely prune branches that cannot beat the
	// best batch count. Equal-count branches remain for quota and ID ties.
	// Saturation avoids overflow when many lines request huge quantities.
	suffixMax := make([]int64, len(maxes)+1)
	for i := len(maxes) - 1; i >= 0; i-- {
		if maxes[i] > math.MaxInt64-suffixMax[i+1] {
			suffixMax[i] = math.MaxInt64
		} else {
			suffixMax[i] = maxes[i] + suffixMax[i+1]
		}
	}
	deadline := now().Add(100 * time.Millisecond)
	timedOut := false
	var search func(int, int64)
	search = func(i int, total int64) {
		if timedOut {
			return
		}
		if len(ordered) > 8 && !now().Before(deadline) {
			timedOut = true
			return
		}
		if bestTotal >= 0 && total <= bestTotal && suffixMax[i] < bestTotal-total {
			return
		}
		if i == len(ordered) {
			var deviation float64
			for j, p := range profiles {
				for k, n := range p.BatchInputs {
					claim := float64(n) * float64(ordered[j].RequestedBatches)
					if claim > 0 {
						deviation += math.Abs(float64(current[j])*float64(n)-quotas[j][k]) / claim
					}
				}
			}
			if total > bestTotal || (total == bestTotal && (deviation < bestDeviation-1e-10 || (math.Abs(deviation-bestDeviation) <= 1e-10 && gameLexicographicallyPreferred(current, best)))) {
				bestTotal = total
				bestDeviation = deviation
				copy(best, current)
			}
			return
		}
		p := profiles[i]
		for count := maxes[i]; count >= 0; count-- {
			if len(ordered) > 8 && !now().Before(deadline) {
				timedOut = true
				return
			}
			if count > 0 && count < p.MinimumBatches {
				continue
			}
			if count > (budget.EnergyWh-energy)/p.EnergyWh || (p.OperationCost > 0 && count > (budget.CashKopecks-cash)/p.OperationCost) {
				continue
			}
			w := int64(0)
			if count > 0 {
				w = p.Workers
			}
			if w > budget.Workers-workers || count > budget.EquipmentCapacity[p.EquipmentType]-equipment[p.EquipmentType] {
				continue
			}
			feasible := true
			for k, n := range p.BatchInputs {
				if count > (stock[k]-used[k])/n {
					feasible = false
					break
				}
			}
			if !feasible {
				continue
			}
			current[i] = count
			energy += count * p.EnergyWh
			cash += count * p.OperationCost
			workers += w
			equipment[p.EquipmentType] += count
			for k, n := range p.BatchInputs {
				used[k] += n * count
			}
			search(i+1, total+count)
			for k, n := range p.BatchInputs {
				used[k] -= n * count
			}
			equipment[p.EquipmentType] -= count
			workers -= w
			cash -= count * p.OperationCost
			energy -= count * p.EnergyWh
			if count == 0 {
				break
			}
		}
	}
	search(0, 0)
	if timedOut && bestTotal <= 0 {
		return nil, fmt.Errorf("OPTIMIZER_TIME_LIMIT_UNPROVEN")
	}
	results := make([]gameRun, len(ordered))
	for i, l := range ordered {
		results[i] = gameRun{LineID: l.ID, ProfileID: l.ProfileID, Batches: best[i], Quality: quality[i], EnergyWh: best[i] * profiles[i].EnergyWh, CostKopecks: best[i] * profiles[i].OperationCost}
		if best[i] == 0 {
			results[i].Reason = "INSUFFICIENT_INPUTS_OR_CAPACITY"
		}
		if timedOut {
			results[i].Reason = "OPTIMIZER_TIME_LIMIT_UNPROVEN"
		}
	}
	return results, nil
}

// DES-002 D-115: when output and normalized quota deviation are equal,
// prefer more batches on the lowest stable line ID (ordered already sorted).
func gameLexicographicallyPreferred(candidate, incumbent []int64) bool {
	for i := range candidate {
		if candidate[i] != incumbent[i] {
			return candidate[i] > incumbent[i]
		}
	}
	return false
}

// Freeze all proportional material quotas at the start of a production tick.
// This function does not mutate the stock or the submitted lines.
func freezeGameProductionQuotas(set gameProfileSet, lines []gameLine, openingStock map[string]int64) map[string]map[string]float64 {
	total := map[string]float64{}
	for _, l := range lines {
		p := profileByID(set, l.ProfileID)
		for k, n := range p.BatchInputs {
			total[k] += float64(n) * float64(l.RequestedBatches)
		}
	}
	result := map[string]map[string]float64{}
	for _, l := range lines {
		result[l.ID] = map[string]float64{}
		p := profileByID(set, l.ProfileID)
		for k, n := range p.BatchInputs {
			if total[k] > 0 {
				result[l.ID][k] = math.Min(float64(openingStock[k]), total[k]) * float64(n) * float64(l.RequestedBatches) / total[k]
			}
		}
	}
	return result
}
