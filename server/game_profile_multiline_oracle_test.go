package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"testing"
)

type multiOracleLine struct {
	Request   int64            `json:"request"`
	Limit     int64            `json:"limit"`
	Inputs    map[string]int64 `json:"inputs"`
	Equipment string           `json:"equipment"`
	Energy    int64            `json:"energy"`
	Cost      int64            `json:"cost"`
	Workers   int64            `json:"workers"`
}
type multiOracleCase struct {
	Lines     []multiOracleLine `json:"lines"`
	Stock     map[string]int64  `json:"stock"`
	Equipment map[string]int64  `json:"equipment"`
	Energy    int64             `json:"energy"`
	Cash      int64             `json:"cash"`
	Workers   int64             `json:"workers"`
}

// Exact independent enumeration covers the full quota objective for 3-8 lines.
func TestGameOptimizerMultiLineOracle(t *testing.T) {
	python := os.Getenv("EBASH_ORTOOLS_PYTHON")
	if python == "" {
		t.Skip("set EBASH_ORTOOLS_PYTHON for extended oracle tests")
	}
	set, err := readGameProfiles("../docs/catalogs/game-production-profiles.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	base := profileByID(set, "GAME-WHEAT-FLOUR-V1")
	for variant, inputs := range []map[string]int64{{"wheat_grain": 10000}, {"other_grain": 10000}, {"wheat_grain": 5000, "other_grain": 5000}} {
		p := base
		p.ID = fmt.Sprintf("TEST-MILL-%d", variant)
		p.EquipmentType = fmt.Sprintf("test_mill_%d", variant)
		p.BatchInputs = inputs
		p.BatchOutputs = map[string]int64{fmt.Sprintf("test_flour_%d", variant): 7400}
		set.Profiles = append(set.Profiles, p)
	}
	rng := rand.New(rand.NewSource(20261118))
	cases := make([]multiOracleCase, 36)
	actual := make([][]int64, len(cases))
	factors := map[string]int64{"raw_material": 70, "technology": 70, "equipment": 70, "personnel": 70, "conditions": 70}
	for i := range cases {
		n := 3 + i%6
		c := multiOracleCase{Stock: map[string]int64{"wheat_grain": int64(rng.Intn(14)) * 5000, "other_grain": int64(rng.Intn(14)) * 5000}, Equipment: map[string]int64{"test_mill_0": int64(rng.Intn(8)), "test_mill_1": int64(rng.Intn(8)), "test_mill_2": int64(rng.Intn(8))}, Energy: int64(rng.Intn(14)) * 500, Cash: int64(rng.Intn(14)) * 12000, Workers: int64(rng.Intn(9))}
		lines := make([]gameLine, n)
		for j := range lines {
			v := rng.Intn(3)
			p := profileByID(set, fmt.Sprintf("TEST-MILL-%d", v))
			request := int64(1 + rng.Intn(3))
			lines[j] = gameLine{ID: fmt.Sprintf("line-%02d", j), ProfileID: p.ID, RequestedBatches: request, AvailableEquipment: p.EquipmentType, Factors: factors}
			c.Lines = append(c.Lines, multiOracleLine{Request: request, Limit: p.EquipmentCapacity, Inputs: p.BatchInputs, Equipment: p.EquipmentType, Energy: p.EnergyWh, Cost: p.OperationCost, Workers: p.Workers})
		}
		cases[i] = c
		runs, e := optimizeCompetingGameLines(set, lines, c.Stock, gameBudget{EnergyWh: c.Energy, CashKopecks: c.Cash, Workers: c.Workers, EquipmentCapacity: c.Equipment})
		if e != nil {
			t.Fatalf("case %d: %v", i, e)
		}
		for _, r := range runs {
			actual[i] = append(actual[i], r.Batches)
		}
	}
	payload, e := json.Marshal(cases)
	if e != nil {
		t.Fatal(e)
	}
	cmd := exec.Command(python, "../scripts/compare_optimizer_multiline.py")
	cmd.Stdin = bytes.NewReader(payload)
	output, e := cmd.CombinedOutput()
	if e != nil {
		t.Fatalf("oracle failed: %v %s", e, output)
	}
	var expected [][]int64
	if e = json.Unmarshal(output, &expected); e != nil {
		t.Fatalf("oracle output: %v %s", e, output)
	}
	if len(expected) != len(actual) {
		t.Fatalf("oracle count %d != %d", len(expected), len(actual))
	}
	for i := range actual {
		if len(expected[i]) != len(actual[i]) {
			t.Fatalf("case %d length mismatch", i)
		}
		for j := range actual[i] {
			if actual[i][j] != expected[i][j] {
				t.Errorf("case %d line %d: Go=%v oracle=%v case=%+v", i, j, actual[i], expected[i], cases[i])
				break
			}
		}
	}
}
