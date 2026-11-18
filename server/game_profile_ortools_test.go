package main

import (
	"bytes"
	"encoding/json"
	"math/rand"
	"os"
	"os/exec"
	"testing"
)

type oracleCase struct {
	Requests [2]int64 `json:"requests"`
	Grain    int64    `json:"grain"`
	Energy   int64    `json:"energy"`
	Cash     int64    `json:"cash"`
	Capacity int64    `json:"capacity"`
	Workers  int64    `json:"workers"`
}

// Optional external differential test. The standard Go test suite stays dependency-free.
func TestGameOptimizerAgainstORTools(t *testing.T) {
	python := os.Getenv("EBASH_ORTOOLS_PYTHON")
	if python == "" {
		t.Skip("set EBASH_ORTOOLS_PYTHON for independent CP-SAT comparison")
	}
	set, err := readGameProfiles("../docs/catalogs/game-production-profiles.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	rng := rand.New(rand.NewSource(20261118))
	cases := make([]oracleCase, 40)
	actual := make([][2]int64, len(cases))
	factors := map[string]int64{"raw_material": 70, "technology": 70, "equipment": 70, "personnel": 70, "conditions": 70}
	for i := range cases {
		c := oracleCase{Requests: [2]int64{int64(1 + rng.Intn(4)), int64(1 + rng.Intn(4))}, Grain: int64(rng.Intn(9)) * 10000, Energy: int64(rng.Intn(9)) * 500, Cash: int64(rng.Intn(9)) * 12000, Capacity: int64(rng.Intn(9)), Workers: int64(rng.Intn(3))}
		cases[i] = c
		lines := []gameLine{{ID: "a", ProfileID: "GAME-WHEAT-FLOUR-V1", RequestedBatches: c.Requests[0], AvailableEquipment: "roller_mill_sifter", Factors: factors}, {ID: "b", ProfileID: "GAME-WHEAT-FLOUR-V1", RequestedBatches: c.Requests[1], AvailableEquipment: "roller_mill_sifter", Factors: factors}}
		runs, e := optimizeCompetingGameLines(set, lines, map[string]int64{"wheat_grain": c.Grain}, gameBudget{EnergyWh: c.Energy, Workers: c.Workers, CashKopecks: c.Cash, EquipmentCapacity: map[string]int64{"roller_mill_sifter": c.Capacity}})
		if e != nil {
			t.Fatalf("scenario %d: %v", i, e)
		}
		for j, r := range runs {
			actual[i][j] = r.Batches
		}
	}
	payload, _ := json.Marshal(cases)
	cmd := exec.Command(python, "../scripts/compare_optimizer_ortools.py")
	cmd.Stdin = bytes.NewReader(payload)
	out, e := cmd.CombinedOutput()
	if e != nil {
		t.Fatalf("CP-SAT oracle: %v: %s", e, out)
	}
	var expected [][2]int64
	if e = json.Unmarshal(out, &expected); e != nil {
		t.Fatal(e)
	}
	if len(expected) != len(actual) {
		t.Fatal("oracle returned wrong case count")
	}
	for i := range actual {
		if actual[i] != expected[i] {
			t.Errorf("scenario %d %+v: game=%d CP-SAT=%d", i, cases[i], actual[i], expected[i])
		}
	}
}
