package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// plan-finalize used to os.RemoveAll the phase-research directory and
// regenerate templates from empty, destroying every RESEARCH.md a worker
// wrote during planning iterations. This locks the preservation contract:
// worker research survives finalize; only orphaned files are pruned.
func TestPlanFinalizePreservesWorkerPhaseResearch(t *testing.T) {
	saveGlobals(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/preserve-research\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	goal := "Preserve worker phase research through finalize"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{}},
	})

	researchDir := filepath.Join(root, ".aether", "data", "phase-research")
	if err := os.MkdirAll(researchDir, 0755); err != nil {
		t.Fatalf("create research dir: %v", err)
	}
	sentinel := "SENTINEL-RESEARCH: a worker studied the exporter internals and found the retry path"
	workerResearch := "# Phase 1 Research: Route-setter evidence phase\n\n## Recommended Approach\n" + sentinel + "\n"
	if err := os.WriteFile(filepath.Join(researchDir, "phase-1-research.md"), []byte(workerResearch), 0644); err != nil {
		t.Fatalf("write worker research: %v", err)
	}
	if err := os.WriteFile(filepath.Join(researchDir, "phase-9-research.md"), []byte("orphan research for a phase that no longer exists\n"), 0644); err != nil {
		t.Fatalf("write orphan research: %v", err)
	}

	survey, err := loadCodexSurveyContext(root)
	if err != nil {
		t.Fatalf("load survey context: %v", err)
	}
	dispatches := testPlanningDispatches()
	manifest := testPlanManifest(root, goal, time.Now().UTC(), survey, dispatches)
	manifest.Snapshots = snapshotRelativeFiles(root,
		filepath.ToSlash(filepath.Join(".aether", "data", "planning")),
		filepath.ToSlash(filepath.Join(".aether", "data", "phase-research")),
	)
	results := testCompletedPlanningResults(dispatches)

	if _, err := runCodexPlanFinalize(root, codexExternalPlanCompletion{
		PlanManifest: manifest,
		Dispatches:   results,
	}); err != nil {
		t.Fatalf("finalize failed: %v", err)
	}

	preserved, err := os.ReadFile(filepath.Join(researchDir, "phase-1-research.md"))
	if err != nil {
		t.Fatalf("read preserved research: %v", err)
	}
	if !strings.Contains(string(preserved), sentinel) {
		t.Fatalf("worker research was destroyed by finalize; got:\n%s", string(preserved))
	}
	if _, err := os.Stat(filepath.Join(researchDir, "phase-9-research.md")); !os.IsNotExist(err) {
		t.Errorf("orphan phase-9-research.md should be pruned, stat err = %v", err)
	}
}

// The fallback template (used when no research worker ran for a phase) must
// carry the six-section RESEARCH.md contract, including Recommended Approach —
// the section build briefs surface to workers.
func TestPhaseResearchTemplateCarriesSixSections(t *testing.T) {
	dir := t.TempDir()
	root := t.TempDir() // no pre-existing artifacts, so nothing preserves
	phases := []colony.Phase{{
		ID:          1,
		Name:        "Wire the exporter",
		Description: "Connect exporter output to the dashboard",
		Tasks:       []colony.Task{{Goal: "wire it", Hints: []string{"cmd/exporter.go"}}},
	}}
	report := codexScoutReport{
		Findings:   []codexScoutFinding{{Area: "architecture", Discovery: "exporter lives in cmd/exporter.go", Source: "survey"}},
		StudyFiles: []string{"cmd/exporter.go"},
	}
	written, preserved, err := writePhaseResearchArtifacts(root, dir, codexSurveyContext{}, report, phases, map[string]codexArtifactSnapshot{}, nil)
	if err != nil {
		t.Fatalf("write artifacts: %v", err)
	}
	if len(written) != 1 || preserved != 0 {
		t.Fatalf("written = %v preserved = %d, want 1 written 0 preserved", written, preserved)
	}
	data, err := os.ReadFile(filepath.Join(dir, "phase-1-research.md"))
	if err != nil {
		t.Fatalf("read template: %v", err)
	}
	content := string(data)
	for _, section := range []string{
		"## Hive Wisdom (Pre-existing Knowledge)",
		"## Key Patterns",
		"## External Context",
		"## Gotchas",
		"## Recommended Approach",
		"## Files to Study",
	} {
		if !strings.Contains(content, section) {
			t.Errorf("template missing section %q", section)
		}
	}
}
