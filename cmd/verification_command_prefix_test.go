package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestVerificationCommandPrefixCheckCanFail is the guard on the guard:
// splitVerificationCommandDirectoryPrefix must report ok=false for a line
// with no `cd` prefix at all, and must return the exact directory and
// remainder for a genuine prefix -- proving the check itself can fail
// rather than always agreeing with whatever it's given.
func TestVerificationCommandPrefixCheckCanFail(t *testing.T) {
	if dir, remainder, ok := splitVerificationCommandDirectoryPrefix("go test ./..."); ok || dir != "" || remainder != "" {
		t.Fatalf("expected no prefix to report ok=false with empty dir/remainder, got dir=%q remainder=%q ok=%v", dir, remainder, ok)
	}

	dir, remainder, ok := splitVerificationCommandDirectoryPrefix("cd server && python3 -m pytest")
	if !ok {
		t.Fatalf("expected a genuine cd prefix to be recognised, got ok=false")
	}
	if dir != "server" {
		t.Fatalf("dir = %q, want %q", dir, "server")
	}
	if remainder != "python3 -m pytest" {
		t.Fatalf("remainder = %q, want %q", remainder, "python3 -m pytest")
	}
}

func TestVerificationCommandWithADirectoryPrefixRuns(t *testing.T) {
	t.Run("unquoted directory actually runs there", func(t *testing.T) {
		root := t.TempDir()
		if err := os.MkdirAll(filepath.Join(root, "server"), 0755); err != nil {
			t.Fatalf("mkdir server: %v", err)
		}
		// The command only succeeds if it genuinely runs inside server/ --
		// proving the recorded WorkingDir isn't just a label, it's the real
		// working directory the shell-out used.
		if err := os.WriteFile(filepath.Join(root, "server", "marker.txt"), []byte("x"), 0644); err != nil {
			t.Fatalf("write marker: %v", err)
		}
		if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte(
			"## Verification Commands\n\n- tests: cd server && sh -c \"test -f marker.txt\"\n"), 0644); err != nil {
			t.Fatalf("write CLAUDE.md: %v", err)
		}

		commands := resolveCodexVerificationCommands(root)
		wantTest := `sh -c "test -f marker.txt"`
		if commands.Test != wantTest {
			t.Fatalf("commands.Test = %q, want the cd prefix stripped: %q", commands.Test, wantTest)
		}
		if commands.TestDir != "server" {
			t.Fatalf("commands.TestDir = %q, want %q", commands.TestDir, "server")
		}

		step := runVerificationStepInDir(context.Background(), root, "tests", false, commands.Test, commands.TestDir, 5*time.Second)
		if !step.Passed {
			t.Fatalf("expected the step to pass (proving it ran inside server/), got %+v", step)
		}
		if step.WorkingDir != "server" {
			t.Fatalf("step.WorkingDir = %q, want %q", step.WorkingDir, "server")
		}
	})

	t.Run("quoted directory containing a space", func(t *testing.T) {
		dir, remainder, ok := splitVerificationCommandDirectoryPrefix(`cd "my dir" && npm test`)
		if !ok {
			t.Fatalf("expected a quoted directory to be recognised")
		}
		if dir != "my dir" {
			t.Fatalf("dir = %q, want %q", dir, "my dir")
		}
		if remainder != "npm test" {
			t.Fatalf("remainder = %q, want %q", remainder, "npm test")
		}
	})

	t.Run("absolute path prefix is refused with the path named", func(t *testing.T) {
		dir, _, ok := splitVerificationCommandDirectoryPrefix("cd /absolute/path && go test ./...")
		if ok {
			t.Fatalf("expected an absolute path prefix to be refused")
		}
		if dir != "/absolute/path" {
			t.Fatalf("dir = %q, want the offending path preserved: %q", dir, "/absolute/path")
		}
	})

	t.Run("escaping ../.. prefix is refused with the path named", func(t *testing.T) {
		dir, _, ok := splitVerificationCommandDirectoryPrefix("cd ../.. && make test")
		if ok {
			t.Fatalf("expected an escaping ../.. prefix to be refused")
		}
		if dir != "../.." {
			t.Fatalf("dir = %q, want the offending path preserved: %q", dir, "../..")
		}
	})
}

// TestUnreadableVerificationLineIsNamedNotDropped proves WINDOWS.md row 51's
// real fix: a labelled verification-commands line whose command Aether
// cannot classify is named on screen with a way forward, never silently
// reported as "no command configured".
func TestUnreadableVerificationLineIsNamedNotDropped(t *testing.T) {
	saveGlobals(t)
	s, root := newTestStore(t)
	store = s

	unreadableLine := "- tests: frobnicate the widgets"
	if err := os.WriteFile(filepath.Join(root, "CLAUDE.md"), []byte(
		"## Verification Commands\n\n"+unreadableLine+"\n"), 0644); err != nil {
		t.Fatalf("write CLAUDE.md: %v", err)
	}

	phase := colony.Phase{ID: 1, Name: "Unreadable verification line"}
	manifest := codexContinueManifest{}

	floor := runDeterministicFloor(context.Background(), root, phase, manifest, codexWatcherVerification{}, 5*time.Second)

	found := false
	for _, issue := range floor.BlockingIssues {
		if strings.Contains(issue, unreadableLine) {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a blocking issue naming the exact unreadable line %q, got %+v", unreadableLine, floor.BlockingIssues)
	}
	if floor.ChecksPassed {
		t.Fatalf("expected an unreadable required-kind line to block, not silently pass")
	}

	r := refuse("verification-command-not-understood")
	if strings.TrimSpace(r.NextCommand) == "" {
		t.Fatalf("expected the verification-command-not-understood refusal row to carry a next command")
	}
}

// TestNoVerificationSectionStillWarnsQuietly proves the "no tests to run in
// this project" warning is unaffected for a project with no verification
// section at all -- this change must never turn a quiet, unconfigured
// project into a refusal.
func TestNoVerificationSectionStillWarnsQuietly(t *testing.T) {
	saveGlobals(t)
	s, root := newTestStore(t)
	store = s

	phase := colony.Phase{ID: 1, Name: "No verification section"}
	manifest := codexContinueManifest{}

	floor := runDeterministicFloor(context.Background(), root, phase, manifest, codexWatcherVerification{}, 5*time.Second)

	found := false
	for _, warning := range floor.Warnings {
		if strings.Contains(warning, "no tests to run in this project") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected the existing 'no tests to run in this project' warning for a project with nothing configured, got warnings=%+v blockers=%+v", floor.Warnings, floor.BlockingIssues)
	}
	for _, issue := range floor.BlockingIssues {
		if strings.Contains(issue, "could not be understood") {
			t.Fatalf("expected no unreadable-line refusal for a project with no verification section at all, got %+v", floor.BlockingIssues)
		}
	}
}
