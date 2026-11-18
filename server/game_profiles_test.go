package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGameProfilesApprovedLoader(t *testing.T) {
	path := "../docs/catalogs/game-production-profiles.v1.json"
	set, err := readGameProfiles(path)
	if err != nil {
		t.Fatal(err)
	}
	if set.Status != "approved_game_balance_v1" || len(set.Profiles) != 3 {
		t.Fatalf("unexpected set: %+v", set)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct{ name, from, to string }{
		{"missing-energy", `"energy_wh_per_batch": 500`, `"energy_wh_per_batch": 0`},
		{"negative-workers", `"workers_per_active_line": 1`, `"workers_per_active_line": -1`},
		{"bad-yield", `"wheat_flour": 7400`, `"wheat_flour": 9999`},
		{"missing-equipment", `"equipment_type": "roller_mill_sifter"`, `"equipment_type": ""`},
		{"bad-weights", `"raw_material": 30`, `"raw_material": 29`},
		{"duplicate-id", `"GAME-BREAD-DOUGH-V1"`, `"GAME-WHEAT-FLOUR-V1"`},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			altered := strings.Replace(string(raw), tc.from, tc.to, 1)
			if altered == string(raw) {
				t.Fatalf("test fixture absent: %s", tc.from)
			}
			p := filepath.Join(t.TempDir(), "bad.json")
			if err := os.WriteFile(p, []byte(altered), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := readGameProfiles(p); err == nil {
				t.Fatal("invalid profile accepted")
			}
		})
	}
}
