package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Reported from a real colony (M4L-AnalogWave-System, 2026-08-21): a phase
// was passing its tests check, the operator added a "## Verification
// Commands" section naming a build command, and the tests command the same
// file had been supplying all along became unresolved. Narrowing the scan to
// the named section discarded every command outside it, so configuring one
// check silently switched off the others.
func TestAddingAVerificationSectionDoesNotUnresolveOtherCommands(t *testing.T) {
	root := t.TempDir()
	claudeMD := filepath.Join(root, "CLAUDE.md")

	before := "# Project\n\nRun the suite with `go test ./...` to check your work.\n"
	if err := os.WriteFile(claudeMD, []byte(before), 0644); err != nil {
		t.Fatalf("write CLAUDE.md: %v", err)
	}
	discovered := loadVerificationCommandsFromMarkdown(claudeMD, "## Verification Commands")
	if discovered.Test == "" {
		t.Fatalf("control failed: the tests command was not discovered from the whole file, got %+v", discovered)
	}

	after := before + "\n## Verification Commands\n\n- build: go build ./...\n"
	if err := os.WriteFile(claudeMD, []byte(after), 0644); err != nil {
		t.Fatalf("rewrite CLAUDE.md: %v", err)
	}
	withSection := loadVerificationCommandsFromMarkdown(claudeMD, "## Verification Commands")
	if withSection.Build == "" {
		t.Fatalf("the section's own command must be picked up, got %+v", withSection)
	}
	if withSection.Test != discovered.Test {
		t.Fatalf("adding a section naming only a build command unresolved the tests command: was %q, now %q -- "+
			"a section may add precision, never remove a command the file already provided",
			discovered.Test, withSection.Test)
	}
}

// A command named inside the section still outranks one found elsewhere in
// the file: the section is the precise signal, the rest of the file is only
// the fallback.
func TestVerificationSectionOutranksTheRestOfTheFile(t *testing.T) {
	root := t.TempDir()
	claudeMD := filepath.Join(root, "CLAUDE.md")
	content := "# Project\n\nHistorically we ran `go test ./legacy/...`.\n\n" +
		"## Verification Commands\n\n- tests: go test ./...\n"
	if err := os.WriteFile(claudeMD, []byte(content), 0644); err != nil {
		t.Fatalf("write CLAUDE.md: %v", err)
	}
	commands := loadVerificationCommandsFromMarkdown(claudeMD, "## Verification Commands")
	if commands.Test != "go test ./..." {
		t.Fatalf("the section's tests command must win, got %q", commands.Test)
	}
}

// The halt guidance must only name places the reader is actually allowed to
// write. It used to offer .aether/data/codebase.md, which the protected-path
// write guard refuses -- leaving a blocked halt whose only suggested exit is
// itself blocked.
func TestBlockedVerificationGuidanceOnlyNamesWritablePlaces(t *testing.T) {
	guidance := blockedVerificationConfigGuidance()
	if strings.Contains(guidance, ".aether/data") {
		t.Fatalf("guidance points at a protected path the write guard refuses: %q", guidance)
	}
	if !strings.Contains(guidance, "CLAUDE.md") || !strings.Contains(guidance, "AGENTS.md") {
		t.Fatalf("guidance must still name the writable locations, got %q", guidance)
	}
}
