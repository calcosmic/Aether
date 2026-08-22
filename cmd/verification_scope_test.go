package cmd

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestScopeIsDerivedFromChangedFiles proves the core D-07 rule: claims naming
// files under two Go packages produce a targeted test command covering
// exactly those two package paths, and a package count of two.
func TestScopeIsDerivedFromChangedFiles(t *testing.T) {
	claims := codexBuildClaims{
		FilesModified: []string{"cmd/foo.go", "pkg/bar/baz.go"},
	}
	commands := codexVerificationCommands{
		Build: "go build ./...",
		Type:  "go vet ./...",
		Test:  "go test ./...",
	}

	scope, scopedCommands := deriveVerificationScope("/repo", colony.Phase{ID: 1}, false, claims, commands)

	if scope.Mode != verificationScopeTargeted {
		t.Fatalf("Mode = %q, want %q (scope=%+v)", scope.Mode, verificationScopeTargeted, scope)
	}
	if scope.PackageCount != 2 {
		t.Fatalf("PackageCount = %d, want 2 (scope=%+v)", scope.PackageCount, scope)
	}
	wantPackages := []string{"./cmd/...", "./pkg/bar/..."}
	if strings.Join(scope.Packages, ",") != strings.Join(wantPackages, ",") {
		t.Fatalf("Packages = %v, want %v", scope.Packages, wantPackages)
	}
	for _, pkg := range wantPackages {
		if !strings.Contains(scopedCommands.Test, pkg) {
			t.Fatalf("scoped test command %q does not name package %q", scopedCommands.Test, pkg)
		}
	}
	if scopedCommands.Test == commands.Test {
		t.Fatalf("scoped test command was not rewritten: %q", scopedCommands.Test)
	}
	// Build/Type must never be scoped (only Test is ever rewritten).
	if scopedCommands.Build != commands.Build {
		t.Fatalf("Build command was rewritten: got %q, want unchanged %q", scopedCommands.Build, commands.Build)
	}
	if scopedCommands.Type != commands.Type {
		t.Fatalf("Type command was rewritten: got %q, want unchanged %q", scopedCommands.Type, commands.Type)
	}
	if strings.TrimSpace(scope.Reason) == "" {
		t.Fatalf("Reason must be non-empty plain English, got empty")
	}
	if strings.Contains(strings.ToLower(scope.Reason), "watcher") || strings.Contains(strings.ToLower(scope.Reason), "caste") {
		t.Fatalf("Reason must be plain English with no repo-invented vocabulary, got %q", scope.Reason)
	}
}

// TestScopeFallsBackToFullWhenUndecidable proves the three "cannot honestly
// derive a scope" fallback rows all produce the unchanged full command and a
// scope marked full: claims naming files with no derivable package, an empty
// claims set, and a repository whose ecosystem has no scoped runner.
func TestScopeFallsBackToFullWhenUndecidable(t *testing.T) {
	commands := codexVerificationCommands{Test: "go test ./..."}

	t.Run("claims name files with no derivable package", func(t *testing.T) {
		claims := codexBuildClaims{FilesModified: []string{"docs/README.md", ".planning/notes.md"}}
		scope, scopedCommands := deriveVerificationScope("/repo", colony.Phase{ID: 1}, false, claims, commands)
		if scope.Mode != verificationScopeFull {
			t.Fatalf("Mode = %q, want %q", scope.Mode, verificationScopeFull)
		}
		if scopedCommands.Test != commands.Test {
			t.Fatalf("full-run command was rewritten: got %q, want unchanged %q", scopedCommands.Test, commands.Test)
		}
	})

	t.Run("empty claims set", func(t *testing.T) {
		scope, scopedCommands := deriveVerificationScope("/repo", colony.Phase{ID: 1}, false, codexBuildClaims{}, commands)
		if scope.Mode != verificationScopeFull {
			t.Fatalf("Mode = %q, want %q", scope.Mode, verificationScopeFull)
		}
		if scopedCommands.Test != commands.Test {
			t.Fatalf("full-run command was rewritten: got %q, want unchanged %q", scopedCommands.Test, commands.Test)
		}
	})

	t.Run("ecosystem has no scoped runner", func(t *testing.T) {
		claims := codexBuildClaims{FilesModified: []string{"cmd/foo.go"}}
		noScopedCommands := codexVerificationCommands{Test: "make test"}
		scope, scopedCommands := deriveVerificationScope("/repo", colony.Phase{ID: 1}, false, claims, noScopedCommands)
		if scope.Mode != verificationScopeFull {
			t.Fatalf("Mode = %q, want %q", scope.Mode, verificationScopeFull)
		}
		if scopedCommands.Test != noScopedCommands.Test {
			t.Fatalf("full-run command was rewritten: got %q, want unchanged %q", scopedCommands.Test, noScopedCommands.Test)
		}
	})
}

// TestFinalPhaseAlwaysRunsFull proves the last phase of the plan runs the
// full suite even when a scope could have been derived from its own claims.
func TestFinalPhaseAlwaysRunsFull(t *testing.T) {
	claims := codexBuildClaims{FilesModified: []string{"cmd/foo.go", "pkg/bar/baz.go"}}
	commands := codexVerificationCommands{Test: "go test ./..."}

	scope, scopedCommands := deriveVerificationScope("/repo", colony.Phase{ID: 1}, true, claims, commands)

	if scope.Mode != verificationScopeFull {
		t.Fatalf("Mode = %q, want %q (a derivable scope must still yield full on the final phase)", scope.Mode, verificationScopeFull)
	}
	if scopedCommands.Test != commands.Test {
		t.Fatalf("final-phase command was rewritten: got %q, want unchanged %q", scopedCommands.Test, commands.Test)
	}

	// Integration: runDeterministicFloor must wire isLastPhaseOfActivePlan
	// through to the same result when this phase really is the plan's last.
	t.Run("wired through runDeterministicFloor", func(t *testing.T) {
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
		floor := runDeterministicFloor(context.Background(), root, phase, codexContinueManifest{}, codexWatcherVerification{}, 5*time.Second)
		if floor.Scope.Mode != verificationScopeFull {
			t.Fatalf("floor.Scope.Mode = %q, want %q for the plan's only (and therefore last) phase", floor.Scope.Mode, verificationScopeFull)
		}
	})
}

// TestScopedCommandNeverBroadensTheRun is the invariant assertion: for every
// fixture, the set of files a targeted command could execute over is a
// subset of what the full command would cover. A scoping bug that
// accidentally widens or misses the changed package fails here -- this must
// never be replaced with a string comparison of the command.
func TestScopedCommandNeverBroadensTheRun(t *testing.T) {
	fixtures := []struct {
		name     string
		claims   codexBuildClaims
		wantMode string
	}{
		{"single package", codexBuildClaims{FilesModified: []string{"cmd/foo.go"}}, verificationScopeTargeted},
		{"two packages", codexBuildClaims{FilesCreated: []string{"pkg/a/a.go"}, FilesModified: []string{"pkg/b/b.go"}}, verificationScopeTargeted},
		{"nested package", codexBuildClaims{TestsWritten: []string{"pkg/deep/nested/dir/thing_test.go"}}, verificationScopeTargeted},
		// A changed file at the repository root maps to the same "./..."
		// pattern the full run already uses, so this must be reported as
		// full (not "targeted to 1 package(s)") -- IN-01, 193-REVIEW.md.
		{"root package", codexBuildClaims{FilesModified: []string{"main.go"}}, verificationScopeFull},
	}
	commands := codexVerificationCommands{Test: "go test ./..."}

	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			scope, _ := deriveVerificationScope("/repo", colony.Phase{ID: 1}, false, fixture.claims, commands)
			if scope.Mode != fixture.wantMode {
				t.Fatalf("Mode = %q, want %q", scope.Mode, fixture.wantMode)
			}
			if scope.Mode == verificationScopeFull {
				if strings.Contains(scope.Reason, "targeted") {
					t.Fatalf("full-mode Reason must not claim to be targeted, got %q", scope.Reason)
				}
				if !strings.Contains(scope.Reason, "top level") && !strings.Contains(scope.Reason, "root") {
					t.Fatalf("full-mode Reason for a root-level change should name that, got %q", scope.Reason)
				}
			}
			for _, pkg := range scope.Packages {
				// Every targeted package pattern must be a recursive
				// restriction ("./dir/..." or the root "./...") -- never a
				// pattern that could reach outside what "./..." (the full
				// run's own pattern) already covers. This is what makes the
				// subset property structurally true rather than merely
				// tested by example.
				if !strings.HasPrefix(pkg, "./") || !strings.HasSuffix(pkg, "/...") {
					t.Fatalf("package pattern %q is not a recursive restriction of the full run's ./... pattern", pkg)
				}
			}
		})
	}
}
