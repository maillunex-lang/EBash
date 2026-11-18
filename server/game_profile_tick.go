package main

import (
	"context"
	"fmt"
)

// gameProductionState is deliberately opt-in and used only in isolated worlds.
// It is serialized together with the company in the existing transactional tick.
// Budget capacities renew each tick; stock and money do not.
type gameProductionState struct {
	Construction      *bakeryConstruction `json:"construction,omitempty"`
	FlourDelivery     *gameFlourDelivery  `json:"flour_delivery,omitempty"`
	Lines             []gameLine          `json:"lines"`
	Stock             map[string]int64    `json:"stock"`
	Inventory         *gameInventory      `json:"inventory,omitempty"`
	PerTickBudget     gameBudget          `json:"per_tick_budget"`
	LastProcessedTick int64               `json:"last_processed_tick"`
	LastRuns          []gameRun           `json:"last_runs,omitempty"`
	LastError         string              `json:"last_error,omitempty"`
	ProducedAt        map[string]int64    `json:"produced_at,omitempty"`
}

// tickWithApprovedGameProfiles is NOT wired into the production tick entrypoint.
// It is used only by an explicit isolated test-world caller until the
// DES-002 fair resource optimizer and persistent line configuration are ready.
func (s store) tickWithApprovedGameProfiles(ctx context.Context, world string, target int64, set gameProfileSet) error {
	if set.Status != "approved_game_balance_v1" {
		return fmt.Errorf("GAME_PROFILE_NOT_APPROVED")
	}
	return s.tickWithProduction(ctx, world, target, nil, func(c *company, tick int64) []event {
		delivery, err := c.arriveGameFlour(tick)
		if err != nil {
			c.GameProduction.LastError = err.Error()
			c.GameProduction.LastProcessedTick = tick
			return []event{movement("game_delivery_blocked", nil)}
		}
		return append(delivery, runApprovedGameProfiles(c, tick, set)...)
	}, true)
}

func runApprovedGameProfiles(c *company, tick int64, set gameProfileSet) []event {
	p := c.GameProduction
	if p == nil || p.LastProcessedTick >= tick {
		return nil
	}
	p.LastError = ""
	p.LastRuns = nil
	budget := p.PerTickBudget
	budget.EquipmentCapacity = make(map[string]int64, len(p.PerTickBudget.EquipmentCapacity))
	for key, value := range p.PerTickBudget.EquipmentCapacity {
		budget.EquipmentCapacity[key] = value
	}
	budget.CashKopecks = c.Cash
	// Make changes to private data until execution succeeds.
	copyStock := make(map[string]int64, len(p.Stock))
	for key, value := range p.Stock {
		copyStock[key] = value
	}
	// A delivered lot is visible in stock before it is available to the line.
	pending := map[string]int64{}
	if p.Inventory != nil {
		if err := checkGameInventoryStock(*p.Inventory, p.Stock); err != nil {
			p.LastError = err.Error()
			p.LastProcessedTick = tick
			return []event{movement("game_profile_blocked", nil)}
		}
		for _, lot := range p.Inventory.Lots {
			if lot.AvailableForProductionTick > tick {
				pending[lot.Good] += lot.Quantity
				copyStock[lot.Good] -= lot.Quantity
			}
		}
	}
	runs, err := executeGameLines(set, p.Lines, copyStock, &budget)
	if err != nil {
		p.LastError = err.Error()
		p.LastProcessedTick = tick
		return []event{movement("game_profile_blocked", nil)}
	}
	// New isolated worlds can opt into lot accounting. Legacy test-world
	// states without inventory retain their original behavior until migrated.
	var nextInventory *gameInventory
	if p.Inventory != nil {
		for good, qty := range pending {
			total, err := safeGameAdd(copyStock[good], qty)
			if err != nil {
				p.LastError = err.Error()
				p.LastProcessedTick = tick
				return []event{movement("game_profile_blocked", nil)}
			}
			copyStock[good] = total
		}
		computed, invErr := applyGameInventoryRuns(set, runs, *p.Inventory, tick)
		if invErr == nil {
			invErr = checkGameInventoryStock(computed, copyStock)
		}
		if invErr != nil {
			p.LastError = invErr.Error()
			p.LastProcessedTick = tick
			return []event{movement("game_profile_blocked", nil)}
		}
		nextInventory = &computed
	}
	p.Stock = copyStock
	if nextInventory != nil {
		p.Inventory = nextInventory
	}
	c.Cash = budget.CashKopecks
	p.LastProcessedTick = tick
	p.LastRuns = runs
	if p.ProducedAt == nil {
		p.ProducedAt = make(map[string]int64)
	}
	var events []event
	for _, run := range runs {
		if run.Batches == 0 {
			continue
		}
		profile := profileByID(set, run.ProfileID)
		deltas := map[string]int64{"cash_kopecks": -run.CostKopecks, "energy_wh": -run.EnergyWh}
		for good, qty := range profile.BatchInputs {
			deltas[good] -= qty * run.Batches
		}
		for good, qty := range profile.BatchOutputs {
			deltas[good] += qty * run.Batches
			p.ProducedAt[good] = tick
		}
		if profile.Residue > 0 {
			deltas["unstocked_residue_g"] += profile.Residue * run.Batches
		}
		if profile.BakeLoss > 0 {
			deltas["unstocked_bake_loss_g"] += profile.BakeLoss * run.Batches
		}
		events = append(events, movement("game_profile_production", deltas))
	}
	return events
}
func profileByID(set gameProfileSet, id string) gameProfile {
	for _, p := range set.Profiles {
		if p.ID == id {
			return p
		}
	}
	return gameProfile{}
}

func bakeryDemoProduction() *gameProductionState {
	factors := map[string]int64{"raw_material": 70, "technology": 70, "equipment": 70, "personnel": 70, "conditions": 70}
	return &gameProductionState{
		Stock: map[string]int64{}, Inventory: &gameInventory{},
		Lines: []gameLine{
			{ID: "01-mixer", ProfileID: "GAME-BREAD-DOUGH-V1", RequestedBatches: 8, AvailableEquipment: "spiral_mixer", Factors: factors},
			{ID: "02-oven", ProfileID: "GAME-BREAD-LOAF-V1", RequestedBatches: 8, AvailableEquipment: "proofer_oven", Factors: factors},
		},
		PerTickBudget: gameBudget{EnergyWh: 4260, Workers: 3, EquipmentCapacity: map[string]int64{"spiral_mixer": 8, "proofer_oven": 8}},
	}
}
