package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
)

// Game profiles are versioned gameplay assumptions, never industrial measurements.
// The catalog's draft recipes remain non-executable until a separate, explicit
// activation step. Loading a profile does not alter a running world.
type gameProfileSet struct {
	SchemaVersion   int    `json:"schema_version"`
	ProfileSet      string `json:"profile_set"`
	Status          string `json:"status"`
	ProductionRules struct {
		QualityWeights map[string]int `json:"quality_weights_percent"`
	} `json:"production_rules"`
	Profiles []gameProfile `json:"profiles"`
}
type gameProfile struct {
	ID                string           `json:"id"`
	RecipeID          string           `json:"recipe_id"`
	RecipeVersion     int              `json:"recipe_version"`
	MinimumBatches    int64            `json:"minimum_batches"`
	BatchInputs       map[string]int64 `json:"batch_inputs"`
	BatchOutputs      map[string]int64 `json:"batch_outputs"`
	EquipmentType     string           `json:"equipment_type"`
	EquipmentCapacity int64            `json:"equipment_capacity_batches_per_tick"`
	EnergyWh          int64            `json:"energy_wh_per_batch"`
	Workers           int64            `json:"workers_per_active_line"`
	QualityFloor      int64            `json:"quality_floor"`
	QualityReference  int64            `json:"quality_reference"`
	OperationCost     int64            `json:"game_operation_cost_kopecks"`
	Residue           int64            `json:"unstocked_residue_g"`
	BakeLoss          int64            `json:"unstocked_bake_loss_g"`
	LoafMass          int64            `json:"assumed_loaf_mass_g"`
}

func readGameProfiles(path string) (gameProfileSet, error) {
	var set gameProfileSet
	b, err := os.ReadFile(path)
	if err != nil {
		return set, err
	}
	if err = json.Unmarshal(b, &set); err != nil {
		return set, err
	}
	if set.SchemaVersion != 1 || set.ProfileSet == "" || (set.Status != "approved_game_balance_v1" && set.Status != "proposed_for_gameplay_review") {
		return set, errors.New("invalid game profile set metadata")
	}
	expected := []string{"raw_material", "technology", "equipment", "personnel", "conditions"}
	total := 0
	if len(set.ProductionRules.QualityWeights) != len(expected) {
		return set, errors.New("quality factors incomplete")
	}
	for _, k := range expected {
		v := set.ProductionRules.QualityWeights[k]
		if v <= 0 {
			return set, fmt.Errorf("missing quality factor %s", k)
		}
		total += v
	}
	if total != 100 {
		return set, errors.New("quality weights must sum to 100")
	}
	if len(set.Profiles) == 0 {
		return set, errors.New("no profiles")
	}
	seen := map[string]bool{}
	for _, p := range set.Profiles {
		if p.ID == "" || p.RecipeID == "" || p.RecipeVersion < 1 || seen[p.ID] || p.MinimumBatches < 1 || p.EquipmentType == "" || p.EquipmentCapacity < p.MinimumBatches || p.EnergyWh < 1 || p.Workers < 1 || p.QualityFloor < 1 || p.QualityReference < p.QualityFloor || p.QualityReference > 100 || p.OperationCost < 0 || len(p.BatchInputs) == 0 || len(p.BatchOutputs) == 0 || p.Residue < 0 || p.BakeLoss < 0 {
			return set, fmt.Errorf("invalid required fields for profile %s", p.ID)
		}
		seen[p.ID] = true
		var input, output int64
		for k, v := range p.BatchInputs {
			if k == "" || v <= 0 {
				return set, fmt.Errorf("invalid input %s", p.ID)
			}
			input, err = safeGameAdd(input, v)
			if err != nil {
				return set, fmt.Errorf("profile %s input overflow: %w", p.ID, err)
			}
		}
		for k, v := range p.BatchOutputs {
			if k == "" || v <= 0 {
				return set, fmt.Errorf("invalid output %s", p.ID)
			}
			if k == "bread_loaf" {
				if p.LoafMass <= 0 {
					return set, fmt.Errorf("missing loaf mass %s", p.ID)
				}
				mass, mulErr := safeGameMul(v, p.LoafMass)
				if mulErr != nil {
					return set, fmt.Errorf("profile %s loaf overflow: %w", p.ID, mulErr)
				}
				output, err = safeGameAdd(output, mass)
				if err != nil {
					return set, fmt.Errorf("profile %s output overflow: %w", p.ID, err)
				}
			} else {
				output, err = safeGameAdd(output, v)
				if err != nil {
					return set, fmt.Errorf("profile %s output overflow: %w", p.ID, err)
				}
			}
		}
		accounted, err := safeGameAdd(output, p.Residue)
		if err != nil {
			return set, fmt.Errorf("profile %s residue overflow: %w", p.ID, err)
		}
		accounted, err = safeGameAdd(accounted, p.BakeLoss)
		if err != nil {
			return set, fmt.Errorf("profile %s loss overflow: %w", p.ID, err)
		}
		if input != accounted {
			return set, fmt.Errorf("mass balance mismatch %s", p.ID)
		}
	}
	sort.Slice(set.Profiles, func(i, j int) bool { return set.Profiles[i].ID < set.Profiles[j].ID })
	return set, nil
}
