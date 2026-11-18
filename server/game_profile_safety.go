package main

import (
	"fmt"
	"math"
)

func safeGameAdd(a, b int64) (int64, error) {
	if (b > 0 && a > math.MaxInt64-b) || (b < 0 && a < math.MinInt64-b) {
		return 0, fmt.Errorf("GAME_INTEGER_OVERFLOW")
	}
	return a + b, nil
}
func safeGameMul(a, b int64) (int64, error) {
	if a < 0 || b < 0 || (a != 0 && b > math.MaxInt64/a) {
		return 0, fmt.Errorf("GAME_INTEGER_OVERFLOW")
	}
	return a * b, nil
}

// Replay the scheduler result on private snapshots and validate every movement.
// Never trust a plan as a reason to mutate the caller's maps incrementally.
func validateGamePlan(set gameProfileSet, runs []gameRun, stock map[string]int64, budget gameBudget) (map[string]int64, gameBudget, error) {
	if stock == nil || budget.EquipmentCapacity == nil || budget.EnergyWh < 0 || budget.Workers < 0 || budget.CashKopecks < 0 {
		return nil, budget, fmt.Errorf("INVALID_RESOURCE_BUDGET")
	}
	next := make(map[string]int64, len(stock))
	for k, v := range stock {
		if v < 0 {
			return nil, budget, fmt.Errorf("NEGATIVE_STOCK")
		}
		next[k] = v
	}
	caps := make(map[string]int64, len(budget.EquipmentCapacity))
	for k, v := range budget.EquipmentCapacity {
		if v < 0 {
			return nil, budget, fmt.Errorf("NEGATIVE_CAPACITY")
		}
		caps[k] = v
	}
	b := gameBudget{EnergyWh: budget.EnergyWh, Workers: budget.Workers, CashKopecks: budget.CashKopecks, EquipmentCapacity: caps}
	seen := map[string]bool{}
	for _, run := range runs {
		if run.LineID == "" || seen[run.LineID] || run.Batches < 0 {
			return nil, budget, fmt.Errorf("INVALID_GAME_RUN")
		}
		seen[run.LineID] = true
		if run.Batches == 0 {
			if run.EnergyWh != 0 || run.CostKopecks != 0 {
				return nil, budget, fmt.Errorf("INVALID_ZERO_GAME_RUN")
			}
			continue
		}
		p := profileByID(set, run.ProfileID)
		if p.ID == "" || p.MinimumBatches <= 0 || p.EquipmentCapacity <= 0 || p.EnergyWh <= 0 || p.Workers <= 0 || p.OperationCost < 0 || run.Batches < p.MinimumBatches || run.Batches > p.EquipmentCapacity {
			return nil, budget, fmt.Errorf("INVALID_GAME_PROFILE_RUN")
		}
		energy, err := safeGameMul(run.Batches, p.EnergyWh)
		if err != nil {
			return nil, budget, err
		}
		cost, err := safeGameMul(run.Batches, p.OperationCost)
		if err != nil {
			return nil, budget, err
		}
		if run.EnergyWh != energy || run.CostKopecks != cost || energy > b.EnergyWh || cost > b.CashKopecks || p.Workers > b.Workers || run.Batches > b.EquipmentCapacity[p.EquipmentType] {
			return nil, budget, fmt.Errorf("GAME_BUDGET_EXCEEDED")
		}
		for k, n := range p.BatchInputs {
			if n <= 0 {
				return nil, budget, fmt.Errorf("INVALID_GAME_INPUT")
			}
			qty, err := safeGameMul(n, run.Batches)
			if err != nil {
				return nil, budget, err
			}
			if qty > next[k] {
				return nil, budget, fmt.Errorf("GAME_STOCK_EXCEEDED")
			}
			next[k] -= qty
		}
		for k, n := range p.BatchOutputs {
			if n <= 0 {
				return nil, budget, fmt.Errorf("INVALID_GAME_OUTPUT")
			}
			qty, err := safeGameMul(n, run.Batches)
			if err != nil {
				return nil, budget, err
			}
			sum, err := safeGameAdd(next[k], qty)
			if err != nil {
				return nil, budget, err
			}
			next[k] = sum
		}
		b.EnergyWh -= energy
		b.CashKopecks -= cost
		b.Workers -= p.Workers
		b.EquipmentCapacity[p.EquipmentType] -= run.Batches
	}
	return next, b, nil
}

// Preflight protects planner arithmetic, including claims and intermediate stock.
func preflightGameLines(set gameProfileSet, lines []gameLine, stock map[string]int64, budget gameBudget) error {
	if stock == nil || budget.EquipmentCapacity == nil || budget.EnergyWh < 0 || budget.Workers < 0 || budget.CashKopecks < 0 {
		return fmt.Errorf("INVALID_RESOURCE_BUDGET")
	}
	for _, v := range stock {
		if v < 0 {
			return fmt.Errorf("NEGATIVE_STOCK")
		}
	}
	for _, v := range budget.EquipmentCapacity {
		if v < 0 {
			return fmt.Errorf("NEGATIVE_CAPACITY")
		}
	}
	seen := map[string]bool{}
	// Claims for each material are summed before proportional quota math.
	// Reject overflow even when individual line claims are representable.
	claims := map[string]int64{}
	for _, l := range lines {
		if l.ID == "" || seen[l.ID] {
			return fmt.Errorf("DUPLICATE_OR_EMPTY_LINE")
		}
		seen[l.ID] = true
		p := profileByID(set, l.ProfileID)
		if p.ID == "" {
			return fmt.Errorf("PROFILE_MISSING")
		}
		if l.RequestedBatches < 0 || p.MinimumBatches <= 0 || p.EquipmentCapacity <= 0 || p.EnergyWh <= 0 || p.Workers <= 0 || p.OperationCost < 0 || p.Residue < 0 || p.BakeLoss < 0 {
			return fmt.Errorf("INVALID_GAME_PROFILE")
		}
		if _, err := safeGameMul(l.RequestedBatches, p.EnergyWh); err != nil {
			return err
		}
		if _, err := safeGameMul(l.RequestedBatches, p.OperationCost); err != nil {
			return err
		}
		for k, n := range p.BatchInputs {
			qty, err := safeGameMul(l.RequestedBatches, n)
			if err != nil {
				return err
			}
			total, err := safeGameAdd(claims[k], qty)
			if err != nil {
				return err
			}
			claims[k] = total
		}
		for _, items := range []map[string]int64{p.BatchInputs, p.BatchOutputs} {
			for _, n := range items {
				if n <= 0 {
					return fmt.Errorf("INVALID_GAME_QUANTITY")
				}
				if _, err := safeGameMul(l.RequestedBatches, n); err != nil {
					return err
				}
			}
		}
	}
	return nil
}
