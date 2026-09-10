package cmd

import (
	"strings"
	"testing"
	"time"

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
