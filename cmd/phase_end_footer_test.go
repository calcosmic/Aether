package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// WS5 — the classic end-of-phase footer, restored: flags 🚩, steering
// signals with content and strength, progress, an honest clear-context
// verdict, and the next command as the user's choice.

func TestPhaseEndFooterShowsFlagsSignalsProgress(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// Populated colony: one open blocker, one active signal, a 3-phase plan
	// with tasks.
	writeTestFlags(t, colony.FlagEntry{ID: "b1", Type: "blocker", Description: "login is broken on empty password", Source: "verification", CreatedAt: time.Now().UTC().Format(time.RFC3339)})
	expires := time.Now().UTC().Add(time.Hour).Format(time.RFC3339)
	strength := 0.8
	if err := store.SaveJSON("pheromones.json", colony.PheromoneFile{Signals: []colony.PheromoneSignal{{
		ID: "sig1", Type: "FOCUS", Content: []byte(`"keep the auth flow simple"`), Active: true,
		CreatedAt: time.Now().UTC().Format(time.RFC3339), ExpiresAt: &expires, Strength: &strength,
	}}}); err != nil {
		t.Fatalf("write pheromones: %v", err)
	}
	doneID, pendingID := "2.1", "2.2"
	state := colony.ColonyState{}
	state.Plan.Phases = []colony.Phase{
		{ID: 1, Name: "First"},
		{ID: 2, Name: "Second", Tasks: []colony.Task{{ID: &doneID, Status: colony.TaskCompleted}, {ID: &pendingID, Status: colony.TaskPending}}},
		{ID: 3, Name: "Third"},
	}

	out := renderPhaseEndFooter(state, 2)
	// CONTENT, not counts: the blocker's words, the signal's words, and a
	// real progress bar must all be present.
	if !strings.Contains(out, "login is broken on empty password") {
		t.Fatalf("footer does not show the open blocker's content:\n%s", out)
	}
	if !strings.Contains(out, "🚩 Flags: 1 blocker(s)") {
		t.Fatalf("footer does not show flag triage counts:\n%s", out)
	}
	if !strings.Contains(out, "keep the auth flow simple") {
		t.Fatalf("footer degrades signals to a count instead of showing content:\n%s", out)
	}
	if !strings.Contains(out, "Phase 2/3") && !strings.Contains(out, "[Phase 2/3]") {
		t.Fatalf("footer does not show phase progress:\n%s", out)
	}
	if !strings.Contains(out, "Tasks") || !strings.Contains(out, "1/2") {
		t.Fatalf("footer does not show the phase's task progress:\n%s", out)
	}

	// Empty colony: honest "none" lines, never fabricated sections.
	writeTestFlags(t)
	if err := store.SaveJSON("pheromones.json", colony.PheromoneFile{Signals: []colony.PheromoneSignal{}}); err != nil {
		t.Fatalf("clear pheromones: %v", err)
	}
	out = renderPhaseEndFooter(state, 2)
	if !strings.Contains(out, "Flags: none open") {
		t.Fatalf("empty footer is not honest about flags:\n%s", out)
	}
	if !strings.Contains(out, "Steering signals: none") {
		t.Fatalf("empty footer is not honest about signals:\n%s", out)
	}
}

func TestFlagsCmdUsesClassicRenderer(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	t.Setenv("AETHER_OUTPUT_MODE", "visual")

	writeTestFlags(t, colony.FlagEntry{ID: "b1", Type: "blocker", Description: "hard stop", Source: "chaos-standalone", CreatedAt: time.Now().UTC().Format(time.RFC3339)})

	var buf, errBuf bytes.Buffer
	stdout = &buf
	stderr = &errBuf
	rootCmd.SetArgs([]string{"flag-list"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("flag-list: %v", err)
	}
	out := buf.String()
	// The classic renderer: 🚩 row + nested └── detail + the Iron Law
	// reminder. This was dead code for a full release — flagsCmd rendered a
	// flat machine list while the classic renderer sat unwired.
	if !strings.Contains(out, "🚩 hard stop (open)") {
		t.Fatalf("flag-list does not use the classic 🚩 renderer:\n%s", out)
	}
	if !strings.Contains(out, "└── b1, blocker, from chaos-standalone") {
		t.Fatalf("flag-list lost the nested detail row:\n%s", out)
	}
	if !strings.Contains(out, "Blockers stop advancement") {
		t.Fatalf("flag-list does not state the Iron Law when blockers are open:\n%s", out)
	}
}

func TestSafeToClearOnlyWhenHandoffVerified(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	// No HANDOFF.md on disk: the claim must not appear — the honest line is
	// "don't clear yet".
	out := renderContextClearGuidanceForPlatform("claude")
	if strings.Contains(out, "safe to clear") {
		t.Fatalf("clear-context guidance claims safety without a handoff on disk: %q", out)
	}
	if !strings.Contains(out, "don't clear your context yet") {
		t.Fatalf("missing-handoff guidance is not an explicit hold: %q", out)
	}

	// With the file present, the claim appears and names the file.
	handoffPath := filepath.Join(resolveAetherRootPath(), ".aether", "HANDOFF.md")
	if err := os.MkdirAll(filepath.Dir(handoffPath), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(handoffPath, []byte("# handoff\n"), 0644); err != nil {
		t.Fatalf("write handoff: %v", err)
	}
	out = renderContextClearGuidanceForPlatform("claude")
	if !strings.Contains(out, "Handoff saved (.aether/HANDOFF.md) — safe to clear your context now.") {
		t.Fatalf("verified handoff does not produce the safe-to-clear claim: %q", out)
	}
}

func TestBuildCloseoutHandoffSectionHouseStyle(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s
	// A saved project without a goal is a leftover file, not a project, and the
	// runtime reads it as one -- so the fixture carries the goal the runtime
	// always writes.
	closeoutGoal := "Ship the billing rewrite"
	if err := store.SaveJSON("COLONY_STATE.json", colony.ColonyState{
		Version: "3.0",
		Goal:    &closeoutGoal,
		State:   colony.StateREADY,
		Plan:    colony.Plan{Phases: []colony.Phase{{ID: 1, Name: "First", Status: colony.PhaseReady}}},
	}); err != nil {
		t.Fatalf("write state: %v", err)
	}

	result := map[string]interface{}{
		"workflow":         "build",
		"completion_phase": 1,
		"next":             "Run `aether continue` to verify worker claims and advance.",
	}

	// Phase 197 plan 04: the verdict on walking away is now the shared closing
	// card's sentence rather than renderContextClearGuidance's. The contract is
	// unchanged -- the claim requires the handover note to be seen on disk.
	const safeToClose = "it is safe to close this chat"
	const holdTheChat = "Don't close this chat yet"

	// Without persisted worker handoffs: the section must hold the user
	// back, never claim safety.
	out := renderCeremonyCloseoutVisual(result)
	if !strings.Contains(out, "Handoff") {
		t.Fatalf("build closeout has no Handoff section:\n%s", out)
	}
	if !strings.Contains(out, holdTheChat) {
		t.Fatalf("closeout without handoffs does not hold the user back:\n%s", out)
	}
	if strings.Contains(out, safeToClose) {
		t.Fatalf("closeout says it is safe to close the chat with no handoffs recorded:\n%s", out)
	}
	if !strings.Contains(out, "Colony State") {
		t.Fatalf("build closeout lost the colony-state footer:\n%s", out)
	}

	// With handoffs recorded AND the handoff file on disk, the claim
	// appears.
	if err := store.SaveJSON(workerHandoffsPath, workerHandoffFile{Entries: []workerHandoffRecord{{ID: "h1", Phase: 1, WorkerName: "Mason-1", ChangedFiles: []string{"a.go"}}}}); err != nil {
		t.Fatalf("write handoffs: %v", err)
	}
	handoffPath := filepath.Join(resolveAetherRootPath(), ".aether", "HANDOFF.md")
	if err := os.MkdirAll(filepath.Dir(handoffPath), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(handoffPath, []byte("# handoff\n"), 0644); err != nil {
		t.Fatalf("write handoff file: %v", err)
	}
	out = renderCeremonyCloseoutVisual(result)
	if !strings.Contains(out, "were saved for phase 1") {
		t.Fatalf("closeout does not confirm the notes this build's helpers left:\n%s", out)
	}
	if !strings.Contains(out, safeToClose) {
		t.Fatalf("a verified handover note does not produce the safe-to-close claim:\n%s", out)
	}
}

// TestBuildWrapperEndsWithChoice pins the wrapper contract: the build's
// ending is the USER'S choice via AskUserQuestion, the stop-here option is
// conditional on the verified handoff, and the old advisory-only routing is
// gone.
func TestBuildWrapperEndsWithChoice(t *testing.T) {
	for _, path := range []string{"../.claude/commands/ant/build.md", "../.claude/commands/ant-build.md", "../.opencode/commands/ant/build.md"} {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		text := string(raw)
		for _, anchor := range []string{
			"AskUserQuestion",
			"Verify and advance now",
			"Add steering first",
			"ONLY when the closeout's Handoff section actually said the handoff was saved",
			"Run nothing until the user picks",
		} {
			if !strings.Contains(text, anchor) {
				t.Fatalf("%s lost the end-of-build choice contract anchor %q", path, anchor)
			}
		}
		if strings.Contains(text, "4. Guide the user first to `/ant-continue`.") {
			t.Fatalf("%s still carries the old advisory-only ending", path)
		}
	}
}
