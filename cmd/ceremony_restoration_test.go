package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestInitRendersCharterCeremony: `aether init` with an approved charter
// renders the birth ceremony — the charter panel and the classic colony-born
// close — instead of storing the charter silently behind one flat line.
// Fails if renderCharterFields or the colony-born close is unwired from the
// real init path again.
func TestInitRendersCharterCeremony(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	t.Setenv("AETHER_PLATFORM", "claude")
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	store = s
	root := filepath.Dir(filepath.Dir(s.BasePath()))
	withWorkingDir(t, root)

	stdout = &bytes.Buffer{}
	charterJSON := `{"intent":"Ship the widget","vision":"A calm widget","governance":"Tests must pass","goals":"v1 in a week","tech_stack":"Go","key_risks":"scope creep","constraints":"no new deps"}`
	rootCmd.SetArgs([]string{"init", "--charter-json", charterJSON, "Ship the widget"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	for _, want := range []string{
		"── Charter ──",
		"Intent:      Ship the widget",
		"Vision:      A calm widget",
		"Governance:  Tests must pass",
		"👑 Queen has set the colony's intention",
		`"Ship the widget"`,
		"🟢 Colony Status: READY",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("init ceremony output missing %q", want)
		}
	}
}

// TestInitPreservesWisdomClearsResidue is SEE-11 with the boundary stated:
// re-init CLEARS per-colony working residue (session, decisions, assumptions,
// handoffs — the RUNTIME-01 set) and PRESERVES cross-colony wisdom (pheromone
// signals, standalone instincts, learning observations, the midden) plus a
// mandatory backup of the prior colony state. Both directions are asserted;
// the full boundary is recorded in
// .planning/decisions/init-preservation-boundary.md.
func TestInitPreservesWisdomClearsResidue(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	store = s
	root := filepath.Dir(filepath.Dir(s.BasePath()))
	withWorkingDir(t, root)
	dataDir := s.BasePath()

	sealedGoal := "the previous project"
	prior := colony.ColonyState{Version: "3.0", Goal: &sealedGoal, State: colony.StateCOMPLETED}
	if err := s.SaveJSON("COLONY_STATE.json", prior); err != nil {
		t.Fatalf("seed prior state: %v", err)
	}

	residue := map[string]string{
		"session.json":           `{"colony_goal":"the previous project"}`,
		"pending-decisions.json": `{"decisions":[{"id":"d1","description":"old","resolved":true}]}`,
		"assumptions.json":       `{"assumptions":[{"text":"old assumption"}]}`,
	}
	for name, content := range residue {
		if err := os.WriteFile(filepath.Join(dataDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("seed %s: %v", name, err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dataDir, "handoffs"), 0755); err != nil {
		t.Fatalf("seed handoffs dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "handoffs", "worker-handoffs.json"), []byte(`{"handoffs":[]}`), 0644); err != nil {
		t.Fatalf("seed handoffs: %v", err)
	}

	wisdom := map[string]string{
		"pheromones.json":            `{"signals":[{"id":"s1","type":"REDIRECT","content":{"text":"never commit secrets"},"active":true,"created_at":"2026-08-01T00:00:00Z"}]}`,
		"instincts.json":             `{"instincts":[{"id":"i1","domain":"testing","action":"write the failing test first","confidence":0.9}]}`,
		"learning-observations.json": `{"observations":[{"id":"o1","text":"caching helped"}]}`,
	}
	for name, content := range wisdom {
		if err := os.WriteFile(filepath.Join(dataDir, name), []byte(content), 0644); err != nil {
			t.Fatalf("seed %s: %v", name, err)
		}
	}
	if err := os.MkdirAll(filepath.Join(dataDir, "midden"), 0755); err != nil {
		t.Fatalf("seed midden dir: %v", err)
	}
	middenContent := `{"entries":[{"id":"m1","category":"approach","description":"the thing that failed"}]}`
	if err := os.WriteFile(filepath.Join(dataDir, "midden", "midden.json"), []byte(middenContent), 0644); err != nil {
		t.Fatalf("seed midden: %v", err)
	}

	rootCmd.SetArgs([]string{"init", "--confirm-reinit", "a completely new goal"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init: %v", err)
	}

	// Direction 1: residue is gone.
	for leak := range residue {
		if leak == "session.json" {
			continue // recreated fresh for the new colony
		}
		if _, err := os.Stat(filepath.Join(dataDir, leak)); !os.IsNotExist(err) {
			t.Errorf("residue %s survived re-init", leak)
		}
	}
	if _, err := os.Stat(filepath.Join(dataDir, "handoffs", "worker-handoffs.json")); !os.IsNotExist(err) {
		t.Errorf("handoffs residue survived re-init")
	}

	// Direction 2: wisdom is byte-identical.
	for name, want := range wisdom {
		got, err := os.ReadFile(filepath.Join(dataDir, name))
		if err != nil {
			t.Errorf("wisdom %s was deleted by re-init: %v", name, err)
			continue
		}
		if string(got) != want {
			t.Errorf("wisdom %s was modified by re-init", name)
		}
	}
	if got, err := os.ReadFile(filepath.Join(dataDir, "midden", "midden.json")); err != nil || string(got) != middenContent {
		t.Errorf("midden was deleted or modified by re-init (err=%v)", err)
	}

	// The prior colony's state survives as a mandatory backup.
	backups, err := filepath.Glob(filepath.Join(dataDir, "backups", "COLONY_STATE.pre-init.*.bak"))
	if err != nil || len(backups) == 0 {
		t.Errorf("no pre-init backup of the prior colony state found")
	}
}

// TestSealRendersCrownedAnthill: the classic v5.4.0 crowning ceremony — the
// anthill drawing, the letter-spaced title with the colony version, and the
// closing incantation — is back in the runtime seal visual.
func TestSealRendersCrownedAnthill(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	goal := "Ship the widget"
	state := colony.ColonyState{
		Goal:          &goal,
		ColonyVersion: 3,
		Plan:          colony.Plan{Phases: []colony.Phase{{ID: 1}, {ID: 2}}},
	}

	output := renderSealVisual(map[string]interface{}{}, state, ".aether/data/CROWNED-ANTHILL.md")
	for _, want := range []string{
		"|  CROWNED |",
		"| ANTHILL  |",
		"C R O W N E D   A N T H I L L   v3",
		"The colony stands crowned and sealed.",
		"Its wisdom lives on in QUEEN.md.",
		"The anthill has reached its final form.",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("seal ceremony missing %q", want)
		}
	}
}

// TestContinueFinalPhaseCelebratesProjectComplete: the final `aether continue`
// fires the classic project-complete celebration, not a flat sentence.
func TestContinueFinalPhaseCelebratesProjectComplete(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	goal := "Ship the widget"
	state := colony.ColonyState{
		Goal: &goal,
		Plan: colony.Plan{Phases: []colony.Phase{{ID: 1, Name: "Only phase", Status: colony.PhaseCompleted}}},
	}
	phase := state.Plan.Phases[0]

	output := renderContinueVisual(state, phase, nil, true, nil, map[string]interface{}{}, colony.VerificationDepthStandard)
	for _, want := range []string{
		"P R O J E C T   C O M P L E T E",
		"👑 Goal Achieved: Ship the widget",
		"🐜 The project (this colony) rests. Well done!",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("final continue missing %q", want)
		}
	}
}

// TestInitResearchRendersScanPanels: the research scan renders its findings —
// charter panel plus consent-framed signal suggestions — as runtime-owned
// visuals. The suggestions are proposals: the render must say nothing is
// written without approval, and the numbered entries must be present.
func TestInitResearchRendersScanPanels(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	charter := colony.Charter{Intent: "Ship it", Vision: "calm"}
	suggestions := []pheromoneSuggestion{
		{Type: "focus", Content: "the auth module", Reason: "most churn"},
		{Type: "redirect", Content: "no new dependencies", Reason: "governance"},
	}

	output := renderInitResearchVisual(charter, suggestions, ceremonyResearchData{})
	for _, want := range []string{
		"── Charter ──",
		"Intent:      Ship it",
		"── Suggested Signals ──",
		"1. [FOCUS] the auth module",
		"└── most churn",
		"2. [REDIRECT] no new dependencies",
		"nothing is written until you approve it",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("init-research visual missing %q", want)
		}
	}
}

// TestInitResearchWritesNothing: the scan is an inspection — it must not
// write a single byte into colony data, and in particular must never
// auto-approve its own pheromone suggestions (the failure the retired
// interactive ceremony had).
func TestInitResearchWritesNothing(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "json")
	saveGlobals(t)
	resetRootCmd(t)
	s, tmpDir := newTestStore(t)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	store = s
	root := filepath.Dir(filepath.Dir(s.BasePath()))
	withWorkingDir(t, root)

	before := hashDirContents(t, s.BasePath())

	rootCmd.SetArgs([]string{"init-research", "--goal", "Ship the widget", "--target", root})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init-research: %v", err)
	}

	after := hashDirContents(t, s.BasePath())
	if before != after {
		t.Fatalf("init-research mutated colony data — the scan must be read-only")
	}
}
