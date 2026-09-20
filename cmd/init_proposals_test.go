package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// WS6 — post-init proposals: the runtime computes ranked next moves from
// what the repo actually contains; init no longer ends with the same static
// three lines for every repo on earth.

func TestInitProposalsRankColonizeForExistingCode(t *testing.T) {
	// A repo-shaped folder: go.mod plus real source files.
	repo := t.TempDir()
	if err := os.WriteFile(filepath.Join(repo, "go.mod"), []byte("module x\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	for i := 0; i < 12; i++ {
		if err := os.WriteFile(filepath.Join(repo, fmt.Sprintf("f%d.go", i)), []byte("package x\n"), 0644); err != nil {
			t.Fatalf("write source: %v", err)
		}
	}
	proposals := computeInitProposals(repo, "Build feature X with a clear scope", false)
	if len(proposals) != 3 {
		t.Fatalf("want 3 ranked proposals, got %d", len(proposals))
	}
	if proposals[0].Command != "aether colonize" {
		t.Fatalf("existing code did not rank colonize first: %+v", proposals[0])
	}
	if !strings.Contains(proposals[0].Reason, "recommended") {
		t.Fatalf("top proposal does not say it is recommended: %q", proposals[0].Reason)
	}

	// An empty folder with a broad goal: discuss first, colonize NOT first.
	empty := t.TempDir()
	proposals = computeInitProposals(empty, "Make an app", false)
	if proposals[0].Command != "aether discuss" {
		t.Fatalf("broad goal in an empty folder did not rank discuss first: %+v", proposals[0])
	}

	// An empty folder with a specific goal: straight to planning.
	proposals = computeInitProposals(empty, "Add CSV export to the reports page with tests", false)
	if proposals[0].Command != "aether plan" {
		t.Fatalf("clear goal in a fresh folder did not rank plan first: %+v", proposals[0])
	}

	// Prior colony adds a reason line, never a new option.
	proposals = computeInitProposals(empty, "Add CSV export to the reports page with tests", true)
	if len(proposals) != 3 {
		t.Fatalf("prior colony changed the option count: %d", len(proposals))
	}
	if !strings.Contains(proposals[0].Reason, "previous colony") {
		t.Fatalf("prior colony note missing from the top proposal: %q", proposals[0].Reason)
	}
}

func TestInitProposalsHouseStyle(t *testing.T) {
	proposals := computeInitProposals(t.TempDir(), "Make an app", false)
	for _, p := range proposals {
		if strings.TrimSpace(p.Reason) == "" {
			t.Fatalf("proposal %q carries no reason", p.Command)
		}
	}
	out := renderInitProposals(proposals)
	if !strings.Contains(out, "🧭 Next Moves") {
		t.Fatalf("proposals section missing its heading:\n%s", out)
	}
	if !strings.Contains(out, "nothing runs until you choose") {
		t.Fatalf("proposals section missing the consent frame:\n%s", out)
	}
	if !strings.Contains(out, "└──") {
		t.Fatalf("proposals section missing nested reasons:\n%s", out)
	}
	if strings.Contains(out, "│") || strings.Contains(out, "┌─") {
		t.Fatalf("proposals section renders a bordered table (forbidden):\n%s", out)
	}
}

// TestInitSuggestedNextMatchesTopProposal retains its Phase-199 name for the
// focused verification gate. The authoritative top proposal is now the shared
// lifecycle projection: repo-aware exploratory proposals may remain visible,
// but session and recovery artifacts must record discuss before SPEC review.
func TestInitSuggestedNextMatchesTopProposal(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	t.Setenv("AETHER_PLATFORM", "claude")
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	store = s
	root := filepath.Dir(filepath.Dir(s.BasePath()))
	withWorkingDir(t, root)

	// Make the repo look like existing Go code so colonize ranks first.
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module x\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}

	stdout = &bytes.Buffer{}
	rootCmd.SetArgs([]string{"init", "Ship the widget"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}
	output := stdout.(*bytes.Buffer).String()
	if !strings.Contains(output, "🧭 Next Moves") {
		t.Fatalf("init output missing the Next Moves proposals:\n%s", output)
	}
	// The visual layer translates `aether colonize` to the platform's
	// /ant-colonize form; assert the proposal by label + verb.
	if !strings.Contains(output, "Map the codebase first") || !strings.Contains(output, "colonize") {
		t.Fatalf("existing-code init does not propose colonize:\n%s", output)
	}

	var session colony.SessionFile
	if err := store.LoadJSON("session.json", &session); err != nil {
		t.Fatalf("load session: %v", err)
	}
	if session.SuggestedNext != "aether discuss" {
		t.Fatalf("session.SuggestedNext = %q, want the shared init-to-discuss lifecycle projection", session.SuggestedNext)
	}
	if !strings.Contains(output, "Discuss settles intent before specification review; it does not approve a specification or a plan.") ||
		!strings.HasSuffix(strings.TrimSpace(output), "Next Up: /ant-discuss") {
		t.Fatalf("init did not close at the discuss-before-SPEC authority boundary:\n%s", output)
	}
	for _, forbidden := range []string{"Next Up: /ant-plan", "Next Up: /ant-colonize", "Next Up: aether plan"} {
		if strings.Contains(output, forbidden) {
			t.Fatalf("init output contains a direct post-init shortcut %q:\n%s", forbidden, output)
		}
	}

	contextRaw, err := os.ReadFile(filepath.Join(root, ".aether", "CONTEXT.md"))
	if err != nil {
		t.Fatalf("read CONTEXT.md: %v", err)
	}
	if !strings.Contains(string(contextRaw), "aether discuss") {
		t.Fatalf("recovery context did not persist the shared discuss handoff:\n%s", contextRaw)
	}
}

func TestInitWrapperClosesAtDiscussThenDraftSpec(t *testing.T) {
	for _, path := range []string{"../.claude/commands/ant/init.md", "../.claude/commands/ant-init.md", "../.opencode/commands/ant/init.md"} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := string(raw)
		for _, anchor := range []string{
			"Ranked `proposals`",
			"do not replace the runtime-backed",
			"After `/ant-discuss`, Go creates and immediately renders one DRAFT specification revision.",
			"`/ant-spec` owns review, revision, and exact specification approval.",
			"Next Up: /ant-discuss",
		} {
			if !strings.Contains(text, anchor) {
				t.Fatalf("%s lost the post-init discuss-to-draft-spec contract anchor %q", path, anchor)
			}
		}
		if !strings.HasSuffix(strings.TrimSpace(text), "Next Up: /ant-discuss") {
			t.Fatalf("%s does not end at the exact guided discuss command", path)
		}
		for _, forbidden := range []string{"Next Up: /ant-plan", "`next_action` — exact `/ant-plan`", "successful closeout ends with exact `Next Up: /ant-plan`"} {
			if strings.Contains(text, forbidden) {
				t.Fatalf("%s still contains premature planning guidance %q", path, forbidden)
			}
		}
	}
}
