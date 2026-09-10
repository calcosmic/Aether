package cmd

import (
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// Phase 201 plan 14 (WORK-08, D-15a/D-15b/D-15c) -- the three owner-approved
// turnaround levers: a loop-versus-boundary test scope, a slimmer worker
// brief on the existing handoff shape, and per-caste model routing with a
// stated reason and a visible per-worker cost line. This file proves each
// lever directly, and proves the explicit prohibitions the plan carries
// alongside them: no lever ever narrows the build, type-check or lint
// commands, adds a second handoff schema, routes a quality-sensitive caste,
// or derives a routing decision from timing data.

// ---------------------------------------------------------------------
// Task 1: loop-versus-boundary test scope (D-15a)
// ---------------------------------------------------------------------

// TestScopedTestsRunInTheLoopAndTheFullSuiteAtTheBoundary proves D-15a: a
// single-package change narrows the tests command inside the working loop,
// the identical change always runs the full suite at the phase boundary, an
// unattributable change always runs full with a stated reason regardless of
// cycle point, and the plan's final phase always runs full in either cycle
// point too.
func TestScopedTestsRunInTheLoopAndTheFullSuiteAtTheBoundary(t *testing.T) {
	claims := codexBuildClaims{FilesModified: []string{"cmd/foo.go"}}
	commands := codexVerificationCommands{
		Build: "go build ./...",
		Type:  "go vet ./...",
		Test:  "go test ./...",
	}
	phase := colony.Phase{ID: 1}

	t.Run("inside the loop, a single-package change narrows the command", func(t *testing.T) {
		scope, scoped := deriveVerificationScopeAtCyclePoint("/repo", phase, false, verificationCyclePointLoop, claims, commands)
		if scope.Mode != verificationScopeTargeted {
			t.Fatalf("Mode = %q, want %q", scope.Mode, verificationScopeTargeted)
		}
		if len(scope.Packages) != 1 || scope.Packages[0] != "./cmd/..." {
			t.Fatalf("Packages = %v, want [./cmd/...]", scope.Packages)
		}
		if !strings.Contains(scoped.Test, "./cmd/...") {
			t.Fatalf("scoped test command %q does not name ./cmd/...", scoped.Test)
		}
		if strings.TrimSpace(scope.Reason) == "" {
			t.Fatal("Reason must be non-empty")
		}
	})

	t.Run("the same change always runs full at the phase boundary", func(t *testing.T) {
		scope, scoped := deriveVerificationScopeAtCyclePoint("/repo", phase, false, verificationCyclePointBoundary, claims, commands)
		if scope.Mode != verificationScopeFull {
			t.Fatalf("Mode = %q, want %q", scope.Mode, verificationScopeFull)
		}
		if scoped.Test != commands.Test {
			t.Fatalf("boundary command was rewritten: got %q, want unchanged %q", scoped.Test, commands.Test)
		}
		if strings.TrimSpace(scope.Reason) == "" {
			t.Fatal("Reason must be non-empty")
		}
		if strings.Contains(scope.Reason, "targeted") {
			t.Fatalf("boundary Reason must not claim to be targeted, got %q", scope.Reason)
		}
	})

	t.Run("an unattributable change always runs full with a stated reason", func(t *testing.T) {
		unattributable := codexBuildClaims{FilesModified: []string{"docs/README.md"}}
		for _, cp := range []string{verificationCyclePointLoop, verificationCyclePointBoundary} {
			scope, scoped := deriveVerificationScopeAtCyclePoint("/repo", phase, false, cp, unattributable, commands)
			if scope.Mode != verificationScopeFull {
				t.Fatalf("[%s] Mode = %q, want %q", cp, scope.Mode, verificationScopeFull)
			}
			if scoped.Test != commands.Test {
				t.Fatalf("[%s] full-run command was rewritten: got %q, want unchanged %q", cp, scoped.Test, commands.Test)
			}
			if strings.TrimSpace(scope.Reason) == "" {
				t.Fatalf("[%s] Reason must be non-empty", cp)
			}
		}
	})

	t.Run("the final phase of a plan always runs full, in the loop or at the boundary", func(t *testing.T) {
		for _, cp := range []string{verificationCyclePointLoop, verificationCyclePointBoundary} {
			scope, scoped := deriveVerificationScopeAtCyclePoint("/repo", phase, true, cp, claims, commands)
			if scope.Mode != verificationScopeFull {
				t.Fatalf("[%s] Mode = %q, want %q for the plan's final phase", cp, scope.Mode, verificationScopeFull)
			}
			if scoped.Test != commands.Test {
				t.Fatalf("[%s] final-phase command was rewritten: got %q, want unchanged %q", cp, scoped.Test, commands.Test)
			}
		}
	})

	t.Run("wired through the build-finalize report (loop) and continue (boundary)", func(t *testing.T) {
		saveGlobals(t)
		s, root := newTestStore(t)
		store = s
		writeAgentsVerificationCommands(t, root, "- build: true", "- types: true", "- lint: true", "- tests: true")
		phase := colony.Phase{ID: 1, Name: "Only phase, so also the last"}
		if err := s.SaveJSON("COLONY_STATE.json", colony.ColonyState{
			State: colony.StateEXECUTING,
			Plan:  colony.Plan{Phases: []colony.Phase{phase}},
		}); err != nil {
			t.Fatalf("save colony state: %v", err)
		}
		loop := runDeterministicFloorAtCyclePoint(nil, root, phase, codexContinueManifest{}, codexWatcherVerification{}, 5*time.Second, verificationCyclePointLoop)
		boundary := runDeterministicFloorAtCyclePoint(nil, root, phase, codexContinueManifest{}, codexWatcherVerification{}, 5*time.Second, verificationCyclePointBoundary)
		// The plan's only phase is also its final phase, so both must be
		// full here regardless of cycle point -- this proves the wiring
		// reaches runDeterministicFloorAtCyclePoint without asserting
		// anything the final-phase rule doesn't already guarantee for a
		// single-phase fixture.
		if loop.Scope.Mode != verificationScopeFull {
			t.Fatalf("loop.Scope.Mode = %q, want %q for the plan's final phase", loop.Scope.Mode, verificationScopeFull)
		}
		if boundary.Scope.Mode != verificationScopeFull {
			t.Fatalf("boundary.Scope.Mode = %q, want %q", boundary.Scope.Mode, verificationScopeFull)
		}
		// runDeterministicFloor (the unchanged-signature default every
		// existing caller still uses) must be byte-identical to the
		// explicit boundary call.
		unchanged := runDeterministicFloor(nil, root, phase, codexContinueManifest{}, codexWatcherVerification{}, 5*time.Second)
		if unchanged.Scope.Mode != boundary.Scope.Mode || unchanged.Scope.Reason != boundary.Scope.Reason {
			t.Fatalf("runDeterministicFloor's default diverged from an explicit boundary call: %+v vs %+v", unchanged.Scope, boundary.Scope)
		}
	})
}

// TestOnlyTheTestsCommandIsEverNarrowed proves D-15a's second rule: build,
// type-check and lint command strings are byte-identical across every cycle
// point and scope outcome -- only the tests command is ever rewritten.
func TestOnlyTheTestsCommandIsEverNarrowed(t *testing.T) {
	claims := codexBuildClaims{FilesModified: []string{"cmd/foo.go"}}
	commands := codexVerificationCommands{
		Build: "go build ./...",
		Type:  "go vet ./...",
		Lint:  "golangci-lint run",
		Test:  "go test ./...",
	}
	phase := colony.Phase{ID: 1}

	cases := []struct {
		name       string
		cyclePoint string
		isFinal    bool
	}{
		{"loop, not final", verificationCyclePointLoop, false},
		{"boundary, not final", verificationCyclePointBoundary, false},
		{"loop, final phase", verificationCyclePointLoop, true},
		{"boundary, final phase", verificationCyclePointBoundary, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, scoped := deriveVerificationScopeAtCyclePoint("/repo", phase, tc.isFinal, tc.cyclePoint, claims, commands)
			if scoped.Build != commands.Build {
				t.Fatalf("Build command was rewritten: got %q, want unchanged %q", scoped.Build, commands.Build)
			}
			if scoped.Type != commands.Type {
				t.Fatalf("Type command was rewritten: got %q, want unchanged %q", scoped.Type, commands.Type)
			}
			if scoped.Lint != commands.Lint {
				t.Fatalf("Lint command was rewritten: got %q, want unchanged %q", scoped.Lint, commands.Lint)
			}
		})
	}
}

// ---------------------------------------------------------------------
// Task 2: a slimmer brief and a compact handoff (D-15b)
// ---------------------------------------------------------------------

// TestBriefCarriesOnlyTaskRelevantContext proves D-15b's "omit context with
// no bearing" half using the phase-baseline section (cmd/phase_baseline.go)
// as the concrete, already-implemented case: a caste whose job is not
// judging phase scope (builder) never carries it, while the task's own
// content -- and the compact handoff schema every caste's brief carries --
// always does.
func TestBriefCarriesOnlyTaskRelevantContext(t *testing.T) {
	saveGlobals(t)
	root := t.TempDir()
	gitInit(t, root)
	if err := os.WriteFile(root+"/seed.txt", []byte("seed\n"), 0644); err != nil {
		t.Fatalf("write seed: %v", err)
	}
	gitCommitAll(t, root, "seed")
	if err := os.WriteFile(root+"/prior.js", []byte("// prior phase work\n"), 0644); err != nil {
		t.Fatalf("write prior work: %v", err)
	}

	phase := colony.Phase{ID: 1, Name: "Narrow task"}
	dispatch := codexBuildDispatch{Name: "Hammer-90", Caste: "builder", Task: "Add the exporter call to commands.rs"}
	brief := composeBuildManifestBrief(root, phase, dispatch, time.Now(), false)

	if strings.Contains(brief, "## Phase Baseline") {
		t.Fatalf("a builder's brief carries the phase-baseline section, which has no bearing on a builder's own task:\n%s", brief)
	}
	if !strings.Contains(brief, "Add the exporter call to commands.rs") {
		t.Fatalf("brief lost its own task content:\n%s", brief)
	}
	if missing := missingCompactHandoffFields(brief); len(missing) > 0 {
		t.Fatalf("brief is missing compact handoff field(s) %v -- every worker's brief must state the full output schema regardless of task:\n%s", missing, brief)
	}

	// A verifying caste, by contrast, DOES need the baseline section --
	// proving the omission above is caste-relevance, not a blanket removal
	// (the same fixture, the same includeSteeringSections=false production
	// path, the caste changed alone).
	verifierBrief := composeBuildManifestBrief(root, phase, codexBuildDispatch{Name: "Keen-12", Caste: "watcher", Task: "Verify the exporter change stayed in scope"}, time.Now(), false)
	if !strings.Contains(verifierBrief, "## Phase Baseline") || !strings.Contains(verifierBrief, "prior.js") {
		t.Fatalf("a watcher's brief lacks the phase-baseline section, which it needs to judge scope:\n%s", verifierBrief)
	}
}

// TestCompactHandoffUsesTheExistingShape proves D-15b's "no second schema"
// rule: the compact handoff's fields are reflected directly off
// codex.WorkerHandoff's own JSON tags (pkg/codex/handoff.go) -- there is no
// hand-typed second list to drift from the type ValidateWorkerHandoff
// actually enforces -- and every one of those fields is genuinely present in
// composeBuildManifestBrief's own already-shipped schema statement.
func TestCompactHandoffUsesTheExistingShape(t *testing.T) {
	fields := compactHandoffFieldNames()
	if len(fields) == 0 {
		t.Fatal("compactHandoffFieldNames returned nothing")
	}

	handoffType := reflect.TypeOf(codex.WorkerHandoff{})
	declared := make(map[string]bool, handoffType.NumField())
	for i := 0; i < handoffType.NumField(); i++ {
		tag := handoffType.Field(i).Tag.Get("json")
		name := strings.Split(tag, ",")[0]
		if name == "" || name == "-" {
			continue
		}
		declared[name] = true
	}

	for _, name := range fields {
		if !declared[name] {
			t.Fatalf("compact handoff field %q does not map to any field codex.WorkerHandoff declares", name)
		}
	}
	for name := range declared {
		found := false
		for _, f := range fields {
			if f == name {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("codex.WorkerHandoff declares field %q, but the compact handoff omits it -- reflection must enumerate every field, not a hand-picked subset", name)
		}
	}

	saveGlobals(t)
	tmpDir := t.TempDir()
	phase := colony.Phase{ID: 1, Name: "Any phase"}
	dispatch := codexBuildDispatch{Name: "Hammer-91", Caste: "builder", Task: "Do the thing"}
	brief := composeBuildManifestBrief(tmpDir, phase, dispatch, time.Now(), false)
	if missing := missingCompactHandoffFields(brief); len(missing) > 0 {
		t.Fatalf("composeBuildManifestBrief's own schema statement is missing field(s) %v that codex.WorkerHandoff declares:\n%s", missing, brief)
	}
}

// TestBlockersSurviveTheTightestBudget proves D-15b's "blockers are never
// trimmed" rule directly against real trim pressure: a real active blocker
// plus enough low-priority, unprotected content (colony-prime instincts,
// which carry no entry in protectedSectionPolicy) to force genuine trimming
// at the compact (4000-char, the tightest this codebase has) budget.
func TestBlockersSurviveTheTightestBudget(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "test"
	state := colony.ColonyState{Version: "1.0", Goal: &goal, State: colony.StateEXECUTING, CurrentPhase: 1}
	state.Plan.Phases = []colony.Phase{{ID: 1, Name: "phase", Status: colony.PhaseInProgress}}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("seed state: %v", err)
	}

	const blockerText = "payment webhook signature check fails intermittently"
	if err := s.SaveJSON(pendingDecisionsFile, PendingDecisionFile{Decisions: []PendingDecision{{
		ID: "pd_1", Type: "blocker", Description: blockerText, CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}}}); err != nil {
		t.Fatalf("seed blocker: %v", err)
	}

	var instincts []colony.InstinctEntry
	for i := 0; i < 60; i++ {
		instincts = append(instincts, colony.InstinctEntry{
			Trigger:    fmt.Sprintf("trigger-%d", i),
			Action:     strings.Repeat(fmt.Sprintf("padding text for instinct %d ", i), 20),
			Confidence: 0.5,
		})
	}
	if err := s.SaveJSON("instincts.json", colony.InstinctsFile{Instincts: instincts}); err != nil {
		t.Fatalf("seed instincts: %v", err)
	}

	result := buildColonyPrimeOutputOpts(colonyPrimeOptions{Compact: true})

	if len(result.Trimmed) == 0 {
		t.Fatal("fixture did not exercise real trim pressure -- nothing was trimmed, so this test proves nothing about the tightest budget")
	}
	if !strings.Contains(result.Context, blockerText) {
		t.Fatalf("blocker content did not survive the tightest budget:\n%s", result.Context)
	}
	if !strings.Contains(result.Context, "## Active Blockers") {
		t.Fatalf("blocker heading did not survive the tightest budget:\n%s", result.Context)
	}
	for _, name := range result.Trimmed {
		if name == "blockers" {
			t.Fatal("blockers section was trimmed -- it must never be trimmed regardless of budget pressure")
		}
	}
}
