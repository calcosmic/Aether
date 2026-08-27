package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// The original defect, recorded in the framework-polish handoff and observed
// live: a Route-Setter description containing the word "research" caused mode
// inference to classify an implementation phase as discovery, so the build
// dispatched an Oracle instead of a Builder. Rewording prose was the only fix.
// These tests lock in the typed-mode contract that replaces that behavior.

// A phase whose TYPED mode says production must dispatch builders no matter
// what its prose says.
func TestResearchWordedProductionPhaseDispatchesBuilder(t *testing.T) {
	phase := colony.Phase{
		ID:          2,
		Name:        "Implement exporter research findings",
		Description: "Apply the research conclusions: wire the exporter in cmd/exporter.go and add tests",
		Mode:        colony.PhaseModeProduction,
		Tasks: []colony.Task{
			{ID: strPtr("2.1"), Goal: "Wire exporter output into the dashboard", Status: colony.TaskPending},
		},
	}

	dispatches := testPlannedBuildDispatchesForSelection(phase, "standard", nil, colony.VerificationDepthLight)
	if len(dispatches) == 0 {
		t.Fatal("no dispatches planned")
	}

	hasBuilder := false
	for _, d := range dispatches {
		if d.Caste == "oracle" {
			t.Errorf("production-mode phase dispatched an Oracle (%s) because of prose wording", d.Name)
		}
		if d.Caste == "builder" {
			hasBuilder = true
		}
	}
	if !hasBuilder {
		t.Fatalf("production-mode phase dispatched no builder: %+v", dispatches)
	}
}

// effectiveQueenPhaseMode must never re-infer from prose at runtime. A phase
// with no mode gets the neutral default; a phase with a mode keeps it.
func TestEffectiveQueenPhaseModeNeverInfersFromProse(t *testing.T) {
	noMode := colony.Phase{
		Name:        "Deep research and discovery spike",
		Description: "research everything, explore the architecture",
	}
	if got := effectiveQueenPhaseMode(noMode); got != colony.PhaseModePrototype {
		t.Errorf("modeless phase with research prose resolved to %q; runtime keyword inference has returned", got)
	}

	explicit := colony.Phase{
		Name:        "Research-driven implementation",
		Description: "research says do X; implement X",
		Mode:        colony.PhaseModeProduction,
	}
	if got := effectiveQueenPhaseMode(explicit); got != colony.PhaseModeProduction {
		t.Errorf("explicit mode was overridden: %q", got)
	}
}

// The planner's declared mode wins; inference is only the authoring-time
// default for phases that declare nothing.
func TestResolveAuthoredPhaseModePrecedence(t *testing.T) {
	if got := resolveAuthoredPhaseMode("production", "Research the fix", "research research research"); got != colony.PhaseModeProduction {
		t.Errorf("declared mode lost to prose: %q", got)
	}
	if got := resolveAuthoredPhaseMode("not-a-mode", "Explore the codebase", "spike and research"); got != colony.PhaseModeDiscovery {
		t.Errorf("invalid declared mode should fall back to authoring inference, got %q", got)
	}
	if got := resolveAuthoredPhaseMode("", "Ship the release", "deploy to production"); got != colony.PhaseModeProduction {
		t.Errorf("authoring inference changed behavior for undeclared modes: %q", got)
	}
}

// Migration backfills modes durably so old colonies never rely on the neutral
// runtime default.
func TestMigrateStateBackfillsPhaseModes(t *testing.T) {
	saveGlobals(t)

	s, _ := newTestStore(t)
	store = s

	goal := "Backfill test colony"
	if err := s.SaveJSON("COLONY_STATE.json", colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
		Plan: colony.Plan{Phases: []colony.Phase{
			{ID: 1, Name: "Survey and research the territory", Description: "explore the codebase"},
			{ID: 2, Name: "Ship to production", Description: "deploy the release"},
			{ID: 3, Name: "Already typed", Description: "has a mode", Mode: colony.PhaseModeMaintenance},
		}},
	}); err != nil {
		t.Fatalf("seed state: %v", err)
	}

	result, err := runMigrateState(false)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if got := result["modes_backfilled"]; got != 2 {
		t.Fatalf("modes_backfilled = %v, want 2", got)
	}

	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("reload: %v", err)
	}
	for _, phase := range state.Plan.Phases {
		if !phase.Mode.Valid() {
			t.Errorf("phase %q still has no mode after migration", phase.Name)
		}
	}
	if state.Plan.Phases[2].Mode != colony.PhaseModeMaintenance {
		t.Errorf("migration overwrote an existing explicit mode: %q", state.Plan.Phases[2].Mode)
	}
}

// The grounding gate exemption is decided by typed mode, not prose keywords.
func TestGroundingGateUsesTypedModeNotKeywords(t *testing.T) {
	anchors := []string{"cmd/main.go"}
	prosey := colony.Phase{
		ID:   1,
		Name: "Design and architecture work", // keyword-exempt under the old rule
		Mode: colony.PhaseModeProduction,     // but typed as production
		Tasks: []colony.Task{
			{ID: strPtr("1.1"), Goal: "do something vague with no file reference"},
		},
	}
	warnings := checkPlanGrounding([]colony.Phase{prosey}, anchors)
	if len(warnings) == 0 {
		t.Error("production-mode phase escaped grounding via prose keywords in its name")
	}

	discovery := prosey
	discovery.Mode = colony.PhaseModeDiscovery
	if warnings := checkPlanGrounding([]colony.Phase{discovery}, anchors); len(warnings) != 0 {
		t.Errorf("discovery-mode phase should be exempt from grounding: %+v", warnings)
	}
}

// Guard against silent revival: keyword inference may run only at authoring
// time (resolveAuthoredPhaseMode) and at migration (runMigrateState). If it
// reappears in the runtime resolver, prose controls dispatch again.
func TestInferPhaseModeCallSitesAreBounded(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("find repo root: %v", err)
	}
	allowed := map[string]bool{
		"phase_mode_typed.go": true, // resolveAuthoredPhaseMode (authoring default)
		"command_truth.go":    true, // runMigrateState (one-time durable backfill)
	}
	entries, err := os.ReadDir(filepath.Join(repoRoot, "cmd"))
	if err != nil {
		t.Fatalf("read cmd dir: %v", err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(repoRoot, "cmd", name))
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if strings.Contains(string(data), "InferPhaseMode(") && !allowed[name] {
			t.Errorf("cmd/%s calls InferPhaseMode; runtime keyword inference is forbidden — modes are typed at authoring and backfilled at migration only", name)
		}
	}
}
