package cmd

import (
	"path/filepath"
	"testing"
)

// TestEveryOrderedSurveyOutputIsWritable takes the paths the real automatic
// map-refresh manifest orders the survey helpers to write, and asks the real
// write-protection rule about each one. A path the program orders and then
// refuses is a dead end no helper can work around (owner's trial, 2026-09-21:
// three helpers blocked, planning impossible).
func TestEveryOrderedSurveyOutputIsWritable(t *testing.T) {
	root, _ := setupTerritoryLifecycle199(t)
	_, refresh, err := territoryPlanPreflight(root, codexPlanOptions{PlanOnly: true})
	if err != nil {
		t.Fatalf("territoryPlanPreflight: %v", err)
	}
	if refresh == nil || len(refresh.Dispatches) == 0 {
		t.Fatalf("fixture no longer produces an automatic refresh; this test would be vacuous")
	}
	checked := 0
	for _, dispatch := range refresh.Dispatches {
		for _, rel := range dispatch.OutputPaths {
			checked++
			target := filepath.Join(root, filepath.FromSlash(rel))
			if reason := protectedHookWriteReason(target, root); reason != "" {
				t.Errorf("helper %s is ordered to write %s but the write is refused: %s", dispatch.Name, rel, reason)
			}
		}
	}
	if checked == 0 {
		t.Fatalf("no ordered output paths were checked")
	}
	// The carve-out must stay narrow: state files beside it remain protected.
	for _, rel := range []string{".aether/data/COLONY_STATE.json", ".aether/data/territory-candidates.json"} {
		if protectedHookWriteReason(filepath.Join(root, filepath.FromSlash(rel)), root) == "" {
			t.Errorf("%s must stay protected", rel)
		}
	}
}
