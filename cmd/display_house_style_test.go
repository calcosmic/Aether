package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestPheromoneDisplayGroupsByTypeWithHeadings pins the classic sectioned
// pheromone view the owner asked for by name: emoji headings that explain
// themselves, [NN%] strength, nested age/decay detail, and the decay footer —
// and asserts the flat machine table cannot come back.
func TestPheromoneDisplayGroupsByTypeWithHeadings(t *testing.T) {
	now := time.Date(2026, 8, 16, 12, 0, 0, 0, time.UTC)
	threeDaysAgo := now.Add(-72 * time.Hour).Format(time.RFC3339)
	signals := []colony.PheromoneSignal{
		{ID: "s1", Type: "FOCUS", Content: mustRawContent(t, "the auth module"), Strength: floatPtr(1.0), Active: true, CreatedAt: threeDaysAgo},
		{ID: "s2", Type: "REDIRECT", Content: mustRawContent(t, "no new dependencies"), Strength: floatPtr(0.9), Active: true, CreatedAt: threeDaysAgo},
		{ID: "s3", Type: "FEEDBACK", Content: mustRawContent(t, "shorter commit messages"), Strength: floatPtr(0.8), Active: true, CreatedAt: threeDaysAgo},
	}

	output := renderClassicPheromoneSections(signals, now)
	for _, want := range []string{
		"A C T I V E   P H E R O M O N E S",
		"🎯 FOCUS (Pay attention here)",
		"🚫 REDIRECT (Hard constraints - DO NOT do this)",
		"💬 FEEDBACK (Guidance to consider)",
		`"the auth module"`,
		"└── 3d ago",
		"3 signal(s) active | Decay: FOCUS 30d, REDIRECT 60d, FEEDBACK 90d",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("classic pheromone view missing %q", want)
		}
	}
	if !strings.Contains(output, "[90%]") && !strings.Contains(output, "[85%]") {
		t.Errorf("expected a decayed percentage rendering, got:\n%s", output)
	}
	for _, forbidden := range []string{"TYPE", "PRIORITY", "STRENGTH", "----"} {
		if strings.Contains(output, forbidden) {
			t.Errorf("machine-table artifact %q returned to the pheromone display", forbidden)
		}
	}
}

func mustRawContent(t *testing.T, text string) []byte {
	t.Helper()
	return []byte(`{"text":` + `"` + text + `"}`)
}

// TestHumanDisplaysUseHeadedSectionsNotMachineTables is the invariant behind
// the owner's "it was like that for everything" correction: human-facing
// renders in cmd/ must not emit bordered/fixed-width machine tables. The
// short allowlist names the few displays where a genuine numeric table is the
// right form — shrink-only, one reason each.
func TestHumanDisplaysUseHeadedSectionsNotMachineTables(t *testing.T) {
	allowed := map[string]string{
		"queen_wave_lifecycle.go": "six-column numeric wave dispatch counts; a genuine table of numbers",
		"audit_catalog.go":        "generates the markdown command catalog document, not terminal display",
		"build_print_brief.go":    "budget checklist: aligned label/marker/char-count rows for an inspection command",
		"plan_print_brief.go":     "budget checklist: aligned label/marker/char-count rows for an inspection command",
	}

	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	headerPattern := regexp.MustCompile(`%-\d+s %-\d+s`)
	for _, file := range files {
		if strings.HasSuffix(file, "_test.go") {
			continue
		}
		if _, ok := allowed[filepath.Base(file)]; ok {
			continue
		}
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatalf("read %s: %v", file, err)
		}
		src := string(data)
		if strings.Contains(src, "table.NewWriter()") {
			t.Errorf("%s renders a go-pretty machine table to a human; use the headed house style (emoji heading, one item per line, nested └── detail) or add an allowlist entry with a reason", file)
		}
		if headerPattern.MatchString(src) && strings.Contains(src, `strings.Repeat("-"`) {
			t.Errorf("%s builds a fixed-width column header with a dash rule; use the headed house style or add an allowlist entry with a reason", file)
		}
	}
}

// TestCeremonyLevelGatesStreaming proves classifyCommandCeremonyLevel is the
// live streaming gate: a quiet-classified command emits nothing, a
// worker-theatre one streams. Fails if the gate is unwired again.
func TestCeremonyLevelGatesStreaming(t *testing.T) {
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	saveGlobals(t)

	original := currentStreamingCommand
	t.Cleanup(func() { currentStreamingCommand = original })

	buf := &bytes.Buffer{}
	stdout = buf

	currentStreamingCommand = "spawn-log" // classified quiet
	emitVisualLine("this must not appear")
	emitVisualProgress("this block must not appear")
	if buf.Len() != 0 {
		t.Fatalf("quiet-classified command streamed output: %q", buf.String())
	}

	currentStreamingCommand = "build" // worker_theatre
	emitVisualLine("worker theatre line")
	if !strings.Contains(buf.String(), "worker theatre line") {
		t.Fatalf("worker-theatre command did not stream: %q", buf.String())
	}
}

// TestContinueWorkerFlowLineCarriesCasteIdentity closes the one render site
// that still showed a bare "[caste]" tag instead of the caste identity.
func TestContinueWorkerFlowLineCarriesCasteIdentity(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	var b strings.Builder
	renderContinueWorkerFlowLine(&b, "Vigil-12", "watcher", "completed", "verified the build")
	output := b.String()
	if !strings.Contains(output, "👁️🐜 Watcher Vigil-12") {
		t.Errorf("continue worker line missing caste identity, got %q", output)
	}
	if strings.Contains(output, "[watcher]") {
		t.Errorf("continue worker line still uses the bare caste tag: %q", output)
	}
}

// TestStatusRendersHealthBreakdown: the five vital-sign components render
// beneath the health line — the score is explainable, not a bare number.
func TestStatusRendersHealthBreakdown(t *testing.T) {
	vitals := map[string]interface{}{
		"build_velocity":   map[string]interface{}{"phases_per_day": 2.0, "trend": "steady"},
		"error_rate":       map[string]interface{}{"errors_per_day": 0.0, "status": "clean"},
		"signal_health":    map[string]interface{}{"active_count": 3, "status": "active"},
		"memory_pressure":  map[string]interface{}{"instinct_count": 7, "status": "normal"},
		"colony_age_hours": 72.0,
	}
	output := renderColonyHealthBreakdown(vitals)
	for _, want := range []string{
		"Build velocity: 2 phase(s)/day (steady)",
		"Error rate:     0 error(s)/day (clean)",
		"Signal health:  3 active (active)",
		"Memory:         7 instinct(s) (normal)",
		"Colony age:     3d",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("health breakdown missing %q in:\n%s", want, output)
		}
	}
}

// TestHistoryRendersClassicActivityLines: every rendered event line uses the
// classic feed form `[time] icon …` — proportion-style over a seeded log, so
// it catches format drift on any event, not one hardcoded sample.
func TestHistoryRendersClassicActivityLines(t *testing.T) {
	result := map[string]interface{}{
		"events": []interface{}{
			map[string]interface{}{"timestamp": "2026-08-16T10:00:00Z", "type": "worker_spawned", "source": "build", "message": "Hammer-42 dispatched"},
			map[string]interface{}{"timestamp": "2026-08-16T10:05:00Z", "type": "phase_completed", "source": "continue", "message": "Phase 1 done"},
			map[string]interface{}{"timestamp": "2026-08-16T10:06:00Z", "type": "build_failed", "source": "build", "message": "wave halted"},
		},
	}
	output := renderHistoryVisual(result)

	var eventLines []string
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, "[") {
			eventLines = append(eventLines, line)
		}
	}
	if len(eventLines) != 3 {
		t.Fatalf("expected 3 classic feed lines starting with [time], got %d in:\n%s", len(eventLines), output)
	}
	for _, pair := range []struct{ needle, icon string }{
		{"worker_spawned", "⚡"},
		{"phase_completed", "✅"},
		{"build_failed", "❌"},
	} {
		found := false
		for _, line := range eventLines {
			if strings.Contains(line, pair.needle) && strings.Contains(line, pair.icon) {
				found = true
			}
		}
		if !found {
			t.Errorf("no feed line pairs %q with icon %q in:\n%s", pair.needle, pair.icon, output)
		}
	}
}

// TestDecisionBlockFramesOperatorMoments: SEE-06's one distinct frame, shared
// by wave failure, breaker trips, and autopilot pauses.
func TestDecisionBlockFramesOperatorMoments(t *testing.T) {
	block := renderDecisionBlock("⚠", "Wave Failure — Build Halted",
		"Phase 2 dispatch failed: worker timeout",
		"Fix the cause, then rerun the build for this phase.")
	for _, want := range []string{
		"━━━ ⚠ W A V E   F A I L U R E",
		"B U I L D   H A L T E D ━━━",
		"Phase 2 dispatch failed: worker timeout",
	} {
		if !strings.Contains(block, want) {
			t.Errorf("decision block missing %q in:\n%s", want, block)
		}
	}

	pause := renderRunPauseBlock("test_failures", "aether continue")
	if !strings.Contains(pause, "━━━ ⏸ A U T O P I L O T   P A U S E D ━━━") {
		t.Errorf("pause block no longer uses the shared decision frame:\n%s", pause)
	}
}

// TestContinueRendersWorkerFindings: the per-worker detail the runtime always
// carried (findings, recommendations, weak spots) renders as nested
// disclosure, capped with an honest overflow count.
func TestContinueRendersWorkerFindings(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	var b strings.Builder
	steps := []codexContinueWorkerFlowStep{{
		Name:    "Keen-13",
		Caste:   "watcher",
		Status:  "completed",
		Summary: "reviewed the phase",
		Findings: []codexReviewFinding{
			{Severity: "major", Title: "unchecked error in exporter"},
			{Severity: "minor", Title: "naming drift"},
			{Severity: "minor", Title: "third finding"},
		},
		Recommendations: []string{"add a regression test"},
		WeakSpots:       []string{"the retry path"},
	}}
	renderContinueWorkerFlowValue(&b, steps)
	output := b.String()
	for _, want := range []string{
		"└── found: major: unchecked error in exporter",
		"└── found: minor: naming drift",
		"└── found: (+1 more)",
		"└── recommends: add a regression test",
		"└── weak spot: the retry path",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("worker disclosure missing %q in:\n%s", want, output)
		}
	}
}

// TestHandoffConfirmedBeforeClearGuidance: the clear-context advice names the
// saved handoff when it exists and never claims one that does not.
func TestHandoffConfirmedBeforeClearGuidance(t *testing.T) {
	saveGlobals(t)
	tmpDir := t.TempDir()
	t.Setenv("AETHER_ROOT", tmpDir)
	t.Setenv("AETHER_PLATFORM", "claude")
	// Root resolution prefers the live store's workspace over AETHER_ROOT;
	// clear both so the env var decides and the repo's own handoff file
	// cannot leak into the "no handoff" branch.
	store = nil
	t.Setenv("COLONY_DATA_DIR", "")

	without := renderContextClearGuidance()
	if strings.Contains(without, "Handoff saved") {
		t.Errorf("guidance claims a handoff that does not exist: %q", without)
	}

	if err := os.MkdirAll(filepath.Join(tmpDir, ".aether"), 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, ".aether", "HANDOFF.md"), []byte("# handoff"), 0644); err != nil {
		t.Fatalf("write handoff: %v", err)
	}
	with := renderContextClearGuidance()
	if !strings.Contains(with, "Handoff saved (.aether/HANDOFF.md)") {
		t.Errorf("guidance does not confirm the saved handoff: %q", with)
	}
}

// TestPrintBriefMatchesTaskPacketStandard (SEE-13): a generated worker brief
// is a complete task packet for a worker with zero prior context — the task,
// what done means, and how to verify it, inspectable in the actual artifact.
func TestPrintBriefMatchesTaskPacketStandard(t *testing.T) {
	saveGlobals(t)
	tmpDir := t.TempDir()
	dispatch := codexBuildDispatch{
		Name:  "Hammer-25",
		Caste: "builder",
		Task:  "Add the exporter call to commands.rs and pass its result to the dashboard view",
	}
	phase := colony.Phase{
		ID:              1,
		Name:            "Wire the exporter",
		Description:     "Connect the vault exporter to the dashboard command",
		SuccessCriteria: []string{"Dashboard renders exporter output"},
	}

	brief := renderCodexBuildWorkerBrief(tmpDir, phase, dispatch, time.Now())
	for _, want := range []string{
		"Add the exporter call to commands.rs",
		"Dashboard renders exporter output",
	} {
		if !strings.Contains(brief, want) {
			t.Errorf("task packet missing %q — a zero-context worker cannot act on it", want)
		}
	}
	sections := splitBriefSections(brief)
	names := map[string]bool{}
	for _, section := range sections {
		names[section.Name] = true
	}
	if !names["Assignment"] {
		t.Errorf("task packet has no Assignment section; sections: %v", names)
	}
}

// TestBuildContextShowsSteeringSignals: the operator's active signals render
// under the build's Context stage — the steering loop is visible at the
// moment it takes effect. Fails if the Context stage goes silent again.
func TestBuildContextShowsSteeringSignals(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	store = s

	pf := colony.PheromoneFile{Signals: []colony.PheromoneSignal{
		{ID: "s1", Type: "REDIRECT", Content: []byte(`{"text":"never touch the billing tables"}`), Active: true, CreatedAt: "2026-08-16T00:00:00Z"},
		{ID: "s2", Type: "FOCUS", Content: []byte(`{"text":"the auth module"}`), Active: true, CreatedAt: "2026-08-16T00:00:00Z"},
		{ID: "s3", Type: "FOCUS", Content: []byte(`{"text":"expired note"}`), Active: false, CreatedAt: "2026-01-01T00:00:00Z"},
	}}
	if err := s.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatalf("seed signals: %v", err)
	}

	goal := "steering fixture"
	state := colony.ColonyState{Goal: &goal, Plan: colony.Plan{Phases: []colony.Phase{{ID: 1, Name: "P1"}}}}
	output := renderBuildVisualWithDispatches(state, state.Plan.Phases[0], nil, colony.VerificationDepthStandard)

	for _, want := range []string{
		"Steering signals: 2 active — injected into every worker prompt",
		`🚫 [`,
		`"never touch the billing tables"`,
		`🎯 [`,
		`"the auth module"`,
	} {
		if !strings.Contains(output, want) {
			t.Errorf("build Context missing %q in:\n%s", want, output)
		}
	}
	if strings.Contains(output, "expired note") {
		t.Errorf("inactive signal leaked into the build Context")
	}

	// With no signals, the stage says so and teaches the steering commands.
	if err := s.SaveJSON("pheromones.json", colony.PheromoneFile{Signals: []colony.PheromoneSignal{}}); err != nil {
		t.Fatalf("clear signals: %v", err)
	}
	empty := renderBuildVisualWithDispatches(state, state.Plan.Phases[0], nil, colony.VerificationDepthStandard)
	if !strings.Contains(empty, "Steering signals: none") {
		t.Errorf("empty-signal build Context missing the none line:\n%s", empty)
	}
}

// TestContinueSurfacesSuggestedSteering: pending recommendations render as a
// consent-framed numbered section at the end-of-phase checkpoint, and each
// carries the exact approve command. Fails if suggestions go invisible again
// — the analysis engine stored them for years while nothing showed them.
func TestContinueSurfacesSuggestedSteering(t *testing.T) {
	t.Setenv("NO_COLOR", "1")
	goal := "steering suggestions fixture"
	pending := []colony.PendingSuggestion{
		{ID: "sg_1", Type: "REDIRECT", Content: "never edit generated files by hand", Reason: "3 generated files were hand-edited this phase"},
		{ID: "sg_2", Type: "FOCUS", Content: "the exporter module", Reason: "most churn this phase", Dismissed: false},
		{ID: "sg_3", Type: "FOCUS", Content: "already rejected", Dismissed: true},
	}
	state := colony.ColonyState{
		Goal:               &goal,
		PendingSuggestions: &pending,
		Plan:               colony.Plan{Phases: []colony.Phase{{ID: 1, Name: "P1", Status: colony.PhaseCompleted}}},
	}

	output := renderContinueVisual(state, state.Plan.Phases[0], nil, false, nil, map[string]interface{}{}, colony.VerificationDepthStandard)
	for _, want := range []string{
		"── Suggested Steering ──",
		"nothing is written until you approve it",
		"1. 🚫 [REDIRECT] never edit generated files by hand",
		"└── 3 generated files were hand-edited this phase",
		"aether suggest-approve --approve sg_1",
		"2. 🎯 [FOCUS] the exporter module",
	} {
		if !strings.Contains(output, want) {
			t.Errorf("continue missing %q in:\n%s", want, output)
		}
	}
	if strings.Contains(output, "already rejected") {
		t.Errorf("dismissed suggestion re-surfaced")
	}

	// No pending suggestions → no section, no nagging.
	state.PendingSuggestions = nil
	quiet := renderContinueVisual(state, state.Plan.Phases[0], nil, false, nil, map[string]interface{}{}, colony.VerificationDepthStandard)
	if strings.Contains(quiet, "Suggested Steering") {
		t.Errorf("empty suggestion list still rendered the section")
	}
}
