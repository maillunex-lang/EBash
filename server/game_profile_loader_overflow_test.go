package main

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGameProfileLoaderRejectsArithmeticOverflow(t *testing.T) {
	base, err := readGameProfiles("../docs/catalogs/game-production-profiles.v1.json")
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name   string
		change func(*gameProfile)
	}{
		{"input_sum", func(p *gameProfile) { p.BatchInputs = map[string]int64{"a": math.MaxInt64, "b": 1} }},
		{"output_sum", func(p *gameProfile) { p.BatchOutputs = map[string]int64{"a": math.MaxInt64, "b": 1} }},
		{"loaf_mass", func(p *gameProfile) { p.BatchOutputs = map[string]int64{"bread_loaf": math.MaxInt64}; p.LoafMass = 475 }},
		{"residue", func(p *gameProfile) { p.BatchOutputs = map[string]int64{"a": math.MaxInt64}; p.Residue = 1 }},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			set := base
			set.Profiles = append([]gameProfile(nil), base.Profiles...)
			tc.change(&set.Profiles[0])
			raw, err := json.Marshal(set)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "profiles.json")
			if err = os.WriteFile(path, raw, 0600); err != nil {
				t.Fatal(err)
			}
			_, err = readGameProfiles(path)
			if err == nil || !strings.Contains(err.Error(), "overflow") {
				t.Fatalf("must reject overflow: %v", err)
			}
		})
	}
}
