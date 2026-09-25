package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Phase 210 blocker 4: a user-level session-start hook runs `aether update
// --force` in every new session (every autopilot helper included), and both
// the update and setup paths replaced an Aether-managed AGENTS.md with the
// template wholesale -- deleting the "## Verification Commands" section that
// Aether's own "no verification command" guidance tells the owner to add
// there. The next check then found no commands and autopilot stopped.
const ownerVerificationSection = "\n---\n\n## Verification Commands\n\n- tests: python3 -m pytest -q\n- build: python3 scripts/validate.py\n"

func projectDocCarryFixture(t *testing.T) (hubSystem, repo string) {
	t.Helper()
	template, err := os.ReadFile(filepath.Join("..", ".aether", "templates", "agents-md-template.md"))
	if err != nil {
		t.Fatal(err)
	}
	hubSystem = t.TempDir()
	if err := os.MkdirAll(filepath.Join(hubSystem, "templates"), 0o755); err != nil {
		t.Fatal(err)
	}
	// The hub template differs from what the repo holds, so the doc is due
	// a rewrite -- the exact situation in which the section was lost.
	newer := string(template) + "\n<!-- template revision for test -->\n"
	if err := os.WriteFile(filepath.Join(hubSystem, "templates", "agents-md-template.md"), []byte(newer), 0o644); err != nil {
		t.Fatal(err)
	}
	repo = t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "AGENTS.md"), append(template, []byte(ownerVerificationSection)...), 0o644); err != nil {
		t.Fatal(err)
	}
	return hubSystem, repo
}

func assertOwnerChecksSurvive(t *testing.T, content string) {
	t.Helper()
	if !strings.Contains(content, "<!-- template revision for test -->") {
		t.Fatal("the managed doc was not refreshed from the newer template")
	}
	path := filepath.Join(t.TempDir(), "AGENTS.md")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	commands := loadVerificationCommandsFromMarkdown(path, "## Verification Commands")
	if commands.Test != "python3 -m pytest -q" || commands.Build != "python3 scripts/validate.py" {
		t.Fatalf("owner's check commands lost in the refreshed AGENTS.md: %+v", commands)
	}
}

func TestSetupRefreshKeepsTheOwnersCheckCommands(t *testing.T) {
	hubSystem, repo := projectDocCarryFixture(t)
	if _, _, _, errs := syncProjectDocs(hubSystem, repo); len(errs) != 0 {
		t.Fatalf("sync project docs: %v", errs)
	}
	written, err := os.ReadFile(filepath.Join(repo, "AGENTS.md"))
	if err != nil {
		t.Fatal(err)
	}
	assertOwnerChecksSurvive(t, string(written))
}

func TestUpdateRefreshKeepsTheOwnersCheckCommands(t *testing.T) {
	hubSystem, repo := projectDocCarryFixture(t)
	plan := maintenanceMutationPlan{}
	if _, err := appendMaintenanceProjectDocTargets(&plan, hubSystem, repo); err != nil {
		t.Fatal(err)
	}
	for _, target := range plan.Targets {
		if target.RelativeTarget == "AGENTS.md" {
			assertOwnerChecksSurvive(t, string(target.Content))
			return
		}
	}
	t.Fatal("update planned no AGENTS.md refresh")
}
