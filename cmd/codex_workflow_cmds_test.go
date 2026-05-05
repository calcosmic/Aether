package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestSeal_ArchivesReviews(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))

	// Create a sealed colony state with one completed phase
	goal := "Seal archival test"
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:     1,
			Name:   "Complete work",
			Status: colony.PhaseCompleted,
		}}},
	}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	// Create review data under reviews/security/ledger.json
	reviewsDir := filepath.Join(dataDir, "reviews", "security")
	if err := os.MkdirAll(reviewsDir, 0755); err != nil {
		t.Fatalf("mkdir reviews: %v", err)
	}
	ledger := colony.ReviewLedgerFile{
		Entries: []colony.ReviewLedgerEntry{
			{
				ID:          "sec-1-001",
				Phase:       1,
				Agent:       "gatekeeper",
				GeneratedAt: "2026-04-26T00:00:00Z",
				Status:      "open",
				Severity:    colony.ReviewSeverityHigh,
				Description: "Hardcoded secret in config",
				File:        "config.go",
				Line:        42,
			},
		},
		Summary: colony.ReviewLedgerSummary{
			Total:    1,
			Open:     1,
			Resolved: 0,
		},
	}
	ledgerData, _ := json.MarshalIndent(ledger, "", "  ")
	if err := os.WriteFile(filepath.Join(reviewsDir, "ledger.json"), ledgerData, 0644); err != nil {
		t.Fatalf("write ledger: %v", err)
	}

	rootCmd.SetArgs([]string{"seal"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("seal returned error: %v", err)
	}

	// Verify reviews-archive/security/ledger.json exists alongside CROWNED-ANTHILL.md
	archivePath := filepath.Join(root, ".aether", "reviews-archive", "security", "ledger.json")
	data, err := os.ReadFile(archivePath)
	if err != nil {
		t.Fatalf("reviews-archive not created: %v", err)
	}

	var archivedLedger colony.ReviewLedgerFile
	if err := json.Unmarshal(data, &archivedLedger); err != nil {
		t.Fatalf("archived ledger is not valid JSON: %v", err)
	}
	if len(archivedLedger.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(archivedLedger.Entries))
	}
	if archivedLedger.Entries[0].ID != "sec-1-001" {
		t.Errorf("entry ID = %q, want sec-1-001", archivedLedger.Entries[0].ID)
	}

	// Also verify CROWNED-ANTHILL.md exists
	crownedPath := filepath.Join(root, ".aether", "CROWNED-ANTHILL.md")
	if _, err := os.Stat(crownedPath); err != nil {
		t.Fatalf("CROWNED-ANTHILL.md not created: %v", err)
	}
}

func TestSeal_HighSeverityWarning(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))

	goal := "Seal high-severity test"
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:     1,
			Name:   "Complete work",
			Status: colony.PhaseCompleted,
		}}},
	}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	// Create review data with an open HIGH-severity entry
	reviewsDir := filepath.Join(dataDir, "reviews", "security")
	if err := os.MkdirAll(reviewsDir, 0755); err != nil {
		t.Fatalf("mkdir reviews: %v", err)
	}
	ledger := colony.ReviewLedgerFile{
		Entries: []colony.ReviewLedgerEntry{
			{
				ID:          "sec-1-001",
				Phase:       1,
				Agent:       "gatekeeper",
				GeneratedAt: "2026-04-26T00:00:00Z",
				Status:      "open",
				Severity:    colony.ReviewSeverityHigh,
				Description: "Hardcoded secret in config",
				File:        "config.go",
				Line:        42,
			},
			{
				ID:          "sec-1-002",
				Phase:       1,
				Agent:       "gatekeeper",
				GeneratedAt: "2026-04-26T00:00:00Z",
				Status:      "resolved",
				Severity:    colony.ReviewSeverityHigh,
				Description: "Resolved secret issue",
				File:        "config.go",
				Line:        43,
			},
		},
		Summary: colony.ReviewLedgerSummary{
			Total:    2,
			Open:     1,
			Resolved: 1,
		},
	}
	ledgerData, _ := json.MarshalIndent(ledger, "", "  ")
	if err := os.WriteFile(filepath.Join(reviewsDir, "ledger.json"), ledgerData, 0644); err != nil {
		t.Fatalf("write ledger: %v", err)
	}

	rootCmd.SetArgs([]string{"seal"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("seal returned error: %v", err)
	}

	// Verify CROWNED-ANTHILL.md contains "Review Warnings" section
	crownedPath := filepath.Join(root, ".aether", "CROWNED-ANTHILL.md")
	data, err := os.ReadFile(crownedPath)
	if err != nil {
		t.Fatalf("CROWNED-ANTHILL.md not found: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "Review Warnings") {
		t.Errorf("CROWNED-ANTHILL.md missing 'Review Warnings' section.\nContent:\n%s", content)
	}
	if !strings.Contains(content, "Hardcoded secret in config") {
		t.Errorf("CROWNED-ANTHILL.md missing high-severity finding description")
	}
	// Should NOT mention the resolved entry
	if strings.Contains(content, "Resolved secret issue") {
		t.Errorf("CROWNED-ANTHILL.md should not mention resolved findings")
	}
	// Should mention the count
	if !strings.Contains(content, "1 high-severity") {
		t.Errorf("CROWNED-ANTHILL.md should contain '1 high-severity' count")
	}
}

func TestSeal_NoReviewsNoWarnings(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))

	goal := "Seal no reviews test"
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:     1,
			Name:   "Complete work",
			Status: colony.PhaseCompleted,
		}}},
	}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	// No review data created -- reviews directory does not exist

	rootCmd.SetArgs([]string{"seal"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("seal returned error: %v", err)
	}

	// Verify CROWNED-ANTHILL.md does NOT contain "Review Warnings"
	crownedPath := filepath.Join(root, ".aether", "CROWNED-ANTHILL.md")
	data, err := os.ReadFile(crownedPath)
	if err != nil {
		t.Fatalf("CROWNED-ANTHILL.md not found: %v", err)
	}

	content := string(data)
	if strings.Contains(content, "Review Warnings") {
		t.Errorf("CROWNED-ANTHILL.md should NOT contain 'Review Warnings' when no reviews exist.\nContent:\n%s", content)
	}
}

func TestSealPlanOnlyPrintsManifestWithoutMutatingState(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))

	goal := "Seal plan-only test"
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:     1,
			Name:   "Complete work",
			Status: colony.PhaseCompleted,
		}}},
	}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	rootCmd.SetArgs([]string{"seal", "--plan-only"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("seal --plan-only returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got: %v", env)
	}
	result := env["result"].(map[string]interface{})
	if result["dispatch_mode"] != "plan-only" {
		t.Fatalf("dispatch_mode = %v, want plan-only", result["dispatch_mode"])
	}
	if result["requires_finalizer"] != true {
		t.Fatalf("requires_finalizer = %v, want true", result["requires_finalizer"])
	}
	if got := int(result["dispatch_count"].(float64)); got != 3 {
		t.Fatalf("dispatch_count = %d, want 3", got)
	}
	if _, err := os.Stat(filepath.Join(root, ".aether", "CROWNED-ANTHILL.md")); !os.IsNotExist(err) {
		t.Fatalf("plan-only should not write CROWNED-ANTHILL.md, stat err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(dataDir, "seal", "final-review.json")); !os.IsNotExist(err) {
		t.Fatalf("plan-only should not write final review report, stat err=%v", err)
	}
	var after colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &after); err != nil {
		t.Fatalf("load state after plan-only: %v", err)
	}
	if after.State != colony.StateREADY || after.Milestone == "Crowned Anthill" {
		t.Fatalf("plan-only mutated state: %+v", after)
	}
}

func TestSealFinalizeRecordsExternalReviewAndSeals(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))

	goal := "Seal finalize test"
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:     1,
			Name:   "Complete work",
			Status: colony.PhaseCompleted,
		}}},
	}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	planResult, err := runSealPlanOnly(root, false)
	if err != nil {
		t.Fatalf("run seal plan-only: %v", err)
	}
	manifest := planResult["seal_manifest"].(sealPlanManifest)
	results := append([]codexContinueExternalDispatch{}, manifest.Dispatches...)
	for i := range results {
		results[i].Status = "completed"
		results[i].Summary = results[i].Name + " cleared final review"
		results[i].Report = "# Final review\n\nNo blockers."
	}
	completion := externalSealCompletion{SealManifest: &manifest, Dispatches: results}
	payload, err := json.MarshalIndent(completion, "", "  ")
	if err != nil {
		t.Fatalf("marshal completion: %v", err)
	}
	completionPath := filepath.Join(t.TempDir(), "seal-completion.json")
	if err := os.WriteFile(completionPath, payload, 0644); err != nil {
		t.Fatalf("write completion: %v", err)
	}

	stdout.(*bytes.Buffer).Reset()
	rootCmd.SetArgs([]string{"seal-finalize", "--completion-file", completionPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("seal-finalize returned error: %v", err)
	}

	var after colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &after); err != nil {
		t.Fatalf("load sealed state: %v", err)
	}
	if after.State != colony.StateCOMPLETED || after.Milestone != "Crowned Anthill" {
		t.Fatalf("state not sealed: %+v", after)
	}
	var report sealFinalReviewReport
	if err := store.LoadJSON(sealFinalReviewReportRel, &report); err != nil {
		t.Fatalf("load final review report: %v", err)
	}
	if !report.Passed {
		t.Fatalf("final review report did not pass: %+v", report)
	}
	if len(report.Workers) != 3 {
		t.Fatalf("worker count = %d, want 3", len(report.Workers))
	}
	if _, err := os.Stat(filepath.Join(root, ".aether", "CROWNED-ANTHILL.md")); err != nil {
		t.Fatalf("CROWNED-ANTHILL.md not written: %v", err)
	}
}
