package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestShippedAgentsTemplateNeverReadsAsACheckCommand reproduces the French
// Fluency build-gate failure: with no "## Verification Commands" section the
// reader scans the whole AGENTS.md, and the Aether-generated skills table row
// "| `$ant-build` | Build a phase |" was taken as a build command it could
// not understand -- so Aether's own generated file failed Aether's own gate.
// The fixture is the template Aether actually installs, not a typed copy.
func TestShippedAgentsTemplateNeverReadsAsACheckCommand(t *testing.T) {
	template, err := os.ReadFile(filepath.Join("..", ".aether", "templates", "agents-md-template.md"))
	if err != nil {
		t.Fatalf("read shipped AGENTS.md template: %v", err)
	}
	if !strings.Contains(string(template), "`$ant-build`") {
		t.Fatal("template no longer carries the $ant-build row this test guards; update the fixture reasoning")
	}
	path := filepath.Join(t.TempDir(), "AGENTS.md")
	if err := os.WriteFile(path, template, 0o644); err != nil {
		t.Fatal(err)
	}
	// Invariant, not a named row: Aether's own template declares no check
	// commands, so reading it must yield none and nothing unreadable. The
	// first fix named only the $ant-build row and the "aether build <N>" row
	// still failed the build gate the same day (Phase 210 blocker 4).
	commands := loadVerificationCommandsFromMarkdown(path, "## Verification Commands")
	if len(commands.Unreadable) != 0 {
		t.Fatalf("rows of Aether's own template were read as check commands: %q", commands.Unreadable)
	}
	if commands.Build != "" || commands.Test != "" || commands.Lint != "" || commands.Type != "" {
		t.Fatalf("Aether's own template produced check commands: %+v", commands)
	}

	// A real labelled row in the same table shape still works.
	commands = extractVerificationCommands("| Build | `go build ./...` |\n| `$ant-build` | Build a phase |\n")
	if commands.Build != "go build ./..." || len(commands.Unreadable) != 0 {
		t.Fatalf("real build row lost or menu row misread: build=%q unreadable=%v", commands.Build, commands.Unreadable)
	}
}

// TestNoChangeEvidenceAcceptsTheHandoffContractsOwnPassSpelling: the
// handoff contract (codex.ValidateWorkerHandoff) accepts "passed" and
// normalizes it to "pass", but the no-change evidence rule compared the raw
// spelling and refused an honest no-change result that said "passed".
func TestNoChangeEvidenceAcceptsTheHandoffContractsOwnPassSpelling(t *testing.T) {
	for _, status := range []string{"pass", "passed", "PASSED"} {
		if missing := noChangeEvidenceMissingFrom("already done", status, []string{"pytest"}); len(missing) != 0 {
			t.Fatalf("verification_status %q refused: %v", status, missing)
		}
	}
	for _, status := range []string{"fail", "failed", "not_run", "unknown", ""} {
		if missing := noChangeEvidenceMissingFrom("already done", status, []string{"pytest"}); len(missing) == 0 {
			t.Fatalf("verification_status %q wrongly counted as passing", status)
		}
	}
}

// TestLabelledPythonScriptIsAReadableCheckCommand: a Python project with no
// package manager declared "- build: python3 scripts/validate.py" and it was
// refused as unreadable, while the same script wrapped in sh -c was accepted.
// An unlabelled prose line starting with python3 must still never become a
// check on its own.
func TestLabelledPythonScriptIsAReadableCheckCommand(t *testing.T) {
	commands := extractVerificationCommands("## Verification Commands\n- build: python3 scripts/validate.py\n- tests: python3 -m pytest -q\n")
	if commands.Build != "python3 scripts/validate.py" || len(commands.Unreadable) != 0 {
		t.Fatalf("labelled python build refused: build=%q unreadable=%v", commands.Build, commands.Unreadable)
	}
	commands = extractVerificationCommands("python3 scripts/render_preview.py opens the preview in a browser\n")
	if commands.Build != "" || commands.Test != "" || commands.Lint != "" || commands.Type != "" {
		t.Fatalf("an unlabelled python line became a check: %+v", commands)
	}
}
