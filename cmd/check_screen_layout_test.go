package cmd

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// The check screen (what `aether continue` draws after a phase check) used
// to be one undivided block of text. The owner chose a layout on 2026-09-27:
// a verdict box first, then short sections, each under a heavy header with a
// blank line above it. These tests lock that layout on the real renderers.

// checkScreenFixture builds a realistic successful-check result the way the
// continue path does (in-process typed reports): 7 requirements, 10 gates
// that all passed, 4 helpers that all completed (one of them with a note),
// and the artifact paths the continue path records.
func checkScreenFixture() (colony.ColonyState, colony.Phase, *signalHousekeepingResult, colony.Phase, map[string]interface{}) {
	goal := "Add a French deck to the flashcard app"
	done := colony.Phase{ID: 1, Name: "Import old cards", Status: colony.PhaseCompleted}
	current := colony.Phase{ID: 2, Name: "French Basics deck", Status: colony.PhaseInProgress}
	next := colony.Phase{ID: 3, Name: "Daily reminders", Status: colony.PhaseReady}
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: current.ID,
		Plan:         colony.Plan{Phases: []colony.Phase{done, current, next}},
	}

	criteria := []codexCriterionVerification{}
	for _, label := range []string{
		"Your old cards and history unchanged",
		"French Basics is its own deck, 5/day",
		"Backup made before install",
		"Your go-ahead was recorded",
		"Deck shows in the menu",
		"Progress saves between sessions",
		"Nothing else in the app changed",
	} {
		criteria = append(criteria, codexCriterionVerification{
			Criterion: label,
			Evidence:  []string{"tests check passed: tests passed (exit 0): ok  example.com/app 0.4s"},
			Passed:    true,
		})
	}
	verification := codexContinueVerificationReport{
		Steps: []codexVerificationStep{
			{Name: "build", Passed: true, Duration: 14.2},
			{Name: "types", Skipped: true, Summary: "no command resolved; skipped"},
			{Name: "lint", Skipped: true, Summary: "no command resolved; skipped"},
			{Name: "tests", Passed: true, Duration: 301},
		},
		Criteria: criteria,
		Passed:   true,
	}
	gateNames := []string{
		"manifest_present", "verification_steps_passed", "implementation_evidence",
		"owner_confirmation_pending", "no_critical_flags", "flags", "anti_pattern",
		"anti_pattern_executed", "spawn_gate", "tests_pass",
	}
	gates := codexContinueGateReport{Passed: true}
	for _, name := range gateNames {
		gates.Checks = append(gates.Checks, gateCheck{Name: name, Passed: true})
	}
	workerFlow := []codexContinueWorkerFlowStep{
		{Stage: "verification", Caste: "system", Name: "Deterministic verification", Status: "completed", Summary: "2 passed, 2 skipped"},
		{Stage: "verification", Caste: "watcher", Name: "Keen-13", Status: "completed", Summary: "independent check"},
		{
			Stage: "review", Caste: "gatekeeper", Name: "Ward-4", Status: "completed",
			Findings: []codexReviewFinding{{Severity: "low", Title: "Re-run instructions skip the dry run"}},
		},
		{Stage: "review", Caste: "auditor", Name: "Ledger-9", Status: "completed"},
	}
	housekeeping := &signalHousekeepingResult{TotalSignals: 3, ActiveBefore: 3, ActiveAfter: 1, ExpiredByTime: 1, DeactivatedByStrength: 1, Updated: 2}
	result := map[string]interface{}{
		"verification":   verification,
		"gates":          gates,
		"closed_workers": []string{"Forge-11", "Keen-13", "Ward-4", "Ledger-9"},
		"worker_flow":    workerFlow,
		"review_report":  displayDataPath("build/phase-2/review.json"),
		"consolidation": map[string]interface{}{
			"ran":    false,
			"reason": "nothing new was produced this phase",
		},
	}
	return state, current, housekeeping, next, result
}

func renderCheckScreenFixture(t *testing.T, final bool) string {
	t.Helper()
	setupBuildFlowTest(t)
	state, phase, housekeeping, next, result := checkScreenFixture()
	var nextPhase *colony.Phase
	if !final {
		nextPhase = &next
	}
	return stripANSI(renderContinueVisual(state, phase, housekeeping, final, nextPhase, result, colony.VerificationDepthStandard))
}

// checkScreenBody returns the part of a rendered check screen above the
// shared closing card (the "what next" card and the Next Up card come from
// the one shared decision and are not part of this layout).
func checkScreenBody(rendered string) string {
	cut := len(rendered)
	for _, marker := range []string{"━━ 📊 W H A T   N E X T ━━", "━━ 🐜 N E X T   U P ━━"} {
		if i := strings.Index(rendered, marker); i >= 0 && i < cut {
			cut = i
		}
	}
	return rendered[:cut]
}

func isHeavyRuleLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return trimmed != "" && strings.Trim(trimmed, "━") == ""
}

func firstNonBlankLines(text string, n int) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		if strings.TrimSpace(line) == "" {
			if len(out) > 0 {
				break
			}
			continue
		}
		out = append(out, line)
		if len(out) == n {
			break
		}
	}
	return out
}

func TestCheckScreenLeadsWithTheVerdict(t *testing.T) {
	cases := []struct {
		name     string
		final    bool
		sentence string
	}{
		{"next phase waiting", false, "Everything passed for French Basics deck."},
		{"final phase", true, "Everything passed for French Basics deck. The project is complete."},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rendered := renderCheckScreenFixture(t, tc.final)
			box := firstNonBlankLines(rendered, 4)
			if len(box) != 4 {
				t.Fatalf("expected a four-line verdict box at the top, got %q\n%s", box, rendered)
			}
			if !isHeavyRuleLine(box[0]) || !isHeavyRuleLine(box[3]) {
				t.Errorf("verdict box is not framed by full heavy lines: %q\n%s", box, rendered)
			}
			if !strings.Contains(box[1], "PHASE 2 CHECKED AND SIGNED OFF") {
				t.Errorf("verdict line = %q, want it to say PHASE 2 CHECKED AND SIGNED OFF\n%s", box[1], rendered)
			}
			if strings.TrimSpace(box[2]) != tc.sentence {
				t.Errorf("verdict sentence = %q, want %q\n%s", strings.TrimSpace(box[2]), tc.sentence, rendered)
			}
			// Nothing -- no banner, no section -- may come before the box.
			top := strings.Index(rendered, box[0])
			if strings.TrimSpace(rendered[:top]) != "" {
				t.Errorf("something is drawn above the verdict box:\n%s", rendered[:top])
			}
		})
	}
}

func TestPartialCheckVerdictIsAWarning(t *testing.T) {
	setupBuildFlowTest(t)
	state, phase, housekeeping, next, result := checkScreenFixture()
	result["partial_success"] = true
	result["operational_issues"] = []string{"one helper reported a flaky retry"}
	rendered := stripANSI(renderContinueVisual(state, phase, housekeeping, false, &next, result, colony.VerificationDepthStandard))
	box := firstNonBlankLines(rendered, 4)
	if len(box) != 4 || !strings.Contains(box[1], voiceGlyph("warning")) || !strings.Contains(box[1], "WITH WARNINGS") {
		t.Errorf("a partial pass must lead with a warning verdict, got %q\n%s", box, rendered)
	}
	if !strings.Contains(rendered, "one helper reported a flaky retry") {
		t.Errorf("the partial pass hides what went wrong:\n%s", rendered)
	}
}

var thinStageMarkerRe = regexp.MustCompile(`(?m)^── .+ ──$`)

func assertSectionsSeparated(t *testing.T, body string, wantHeaders []string) {
	t.Helper()
	lines := strings.Split(body, "\n")
	if m := thinStageMarkerRe.FindString(body); m != "" {
		t.Errorf("thin stage marker %q is still on the check screen:\n%s", m, body)
	}
	var seen []string
	for i, line := range lines {
		if !isAetherBannerLine(line) {
			continue
		}
		seen = append(seen, line)
		if i == 0 || strings.TrimSpace(lines[i-1]) != "" {
			t.Errorf("section header %q is not preceded by a blank line:\n%s", line, body)
		}
		if i >= 2 && strings.TrimSpace(lines[i-2]) == "" {
			t.Errorf("section header %q has more than one blank line above it:\n%s", line, body)
		}
		// A section must carry at least one line before the next header.
		if i+1 >= len(lines) || strings.TrimSpace(lines[i+1]) == "" || isAetherBannerLine(lines[i+1]) {
			t.Errorf("section header %q has no content beneath it:\n%s", line, body)
		}
	}
	pos := -1
	for _, want := range wantHeaders {
		spaced := spacedTitle(want)
		idx := -1
		for i, line := range seen {
			if strings.Contains(line, spaced) {
				idx = i
				break
			}
		}
		if idx < 0 {
			t.Errorf("missing heavy section header %q:\n%s", want, body)
			continue
		}
		if idx <= pos {
			t.Errorf("section %q is out of order:\n%s", want, body)
		}
		pos = idx
	}
}

func TestCheckScreenSectionsAreSeparated(t *testing.T) {
	for _, final := range []bool{false, true} {
		t.Run(fmt.Sprintf("final=%v", final), func(t *testing.T) {
			body := checkScreenBody(renderCheckScreenFixture(t, final))
			assertSectionsSeparated(t, body, []string{"Checks", "What Was Proven", "Safety Review", "Behind The Scenes"})
		})
	}
}

func TestCheckScreenStaysShort(t *testing.T) {
	rendered := renderCheckScreenFixture(t, false)
	body := strings.TrimRight(checkScreenBody(rendered), "\n")
	lines := strings.Split(body, "\n")
	if len(lines) > 35 {
		t.Errorf("check screen body is %d lines, want at most 35:\n%s", len(lines), body)
	}
	for _, want := range []string{"… 3 more", "Gates: 10/10 passed", "Build passed", "Tests passed", "(5 min)", "(14s)", "not set up", "1 small note, nothing blocking", "Re-run instructions skip the dry run"} {
		if !strings.Contains(body, want) {
			t.Errorf("check screen is missing %q:\n%s", want, body)
		}
	}
	for _, banned := range []string{".aether/data", "Workers (the helpers", "Review depth:", "the build's own plan file is on disk", "A R T I F A C T S"} {
		if strings.Contains(rendered, banned) {
			t.Errorf("check screen still shows %q:\n%s", banned, rendered)
		}
	}
	// The detail that left the default screen must stay one command away.
	if !strings.Contains(body, "Full detail:") {
		t.Errorf("check screen drops detail without saying where to find it:\n%s", body)
	}
}

func TestCheckScreenKeepsHelpersWhenOneFailed(t *testing.T) {
	setupBuildFlowTest(t)
	state, phase, housekeeping, next, result := checkScreenFixture()
	flow := result["worker_flow"].([]codexContinueWorkerFlowStep)
	flow[3].Status = "failed"
	result["worker_flow"] = flow
	rendered := stripANSI(renderContinueVisual(state, phase, housekeeping, false, &next, result, colony.VerificationDepthStandard))
	if !strings.Contains(rendered, "Ledger-9") || !strings.Contains(rendered, spacedTitle("Helpers")) {
		t.Errorf("a failed helper must keep the helper list on screen:\n%s", rendered)
	}
}

// TestCheckScreenLeavesTheResultUntouched: the layout change is presentation
// only. Drawing either check screen must leave the continue result -- the
// same map the JSON output is written from -- unchanged, keys and values.
func TestCheckScreenLeavesTheResultUntouched(t *testing.T) {
	setupBuildFlowTest(t)
	state, phase, housekeeping, next, result := checkScreenFixture()
	before, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal before: %v", err)
	}
	renderContinueVisual(state, phase, housekeeping, false, &next, result, colony.VerificationDepthStandard)
	renderContinueVisual(state, phase, housekeeping, true, nil, result, colony.VerificationDepthStandard)
	renderContinueBlockedVisual(state, phase, result, colony.VerificationDepthStandard)
	after, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal after: %v", err)
	}
	if string(before) != string(after) {
		t.Errorf("drawing the check screen changed the continue result:\nbefore: %s\nafter:  %s", before, after)
	}
}

func blockedCheckScreenFixture() (colony.ColonyState, colony.Phase, map[string]interface{}) {
	state, phase, _, _, result := checkScreenFixture()
	verification := result["verification"].(codexContinueVerificationReport)
	verification.Steps[3] = codexVerificationStep{Name: "tests", Passed: false, Summary: "2 of 12 tests failed", Command: "npm test", Duration: 40}
	verification.Passed = false
	result["verification"] = verification
	gates := result["gates"].(codexContinueGateReport)
	gates.Checks[1] = gateCheck{Name: "verification_steps_passed", Passed: false, FixHint: "fix the failing build/test check and run aether continue"}
	gates.Passed = false
	result["gates"] = gates
	result["blocking_issues"] = []string{
		"tests failed: 2 of 12",
		"deck import crashes on empty file",
		"progress not saved after restart",
		"menu entry missing",
		"backup step skipped",
		"daily limit ignored",
		"go-ahead not recorded",
	}
	return state, phase, result
}

func TestBlockedCheckScreenSaysWhatToFix(t *testing.T) {
	setupBuildFlowTest(t)
	state, phase, result := blockedCheckScreenFixture()
	rendered := stripANSI(renderContinueBlockedVisual(state, phase, result, colony.VerificationDepthStandard))

	box := firstNonBlankLines(rendered, 4)
	if len(box) != 4 || !isHeavyRuleLine(box[0]) || !isHeavyRuleLine(box[3]) {
		t.Fatalf("blocked screen does not lead with a verdict box: %q\n%s", box, rendered)
	}
	if !strings.Contains(box[1], "PHASE 2 NOT SIGNED OFF") || !strings.Contains(box[1], "tests failed") {
		t.Errorf("blocked verdict = %q, want it to say PHASE 2 NOT SIGNED OFF and that tests failed\n%s", box[1], rendered)
	}

	body := checkScreenBody(rendered)
	assertSectionsSeparated(t, body, []string{"What To Fix", "Checks"})
	fix := body[strings.Index(body, spacedTitle("What To Fix")):]
	for _, want := range []string{"tests failed: 2 of 12", "backup step skipped", "… 2 more"} {
		if !strings.Contains(fix, want) {
			t.Errorf("WHAT TO FIX is missing %q:\n%s", want, fix)
		}
	}
	for _, hidden := range []string{"daily limit ignored", "go-ahead not recorded"} {
		if strings.Contains(fix, hidden) {
			t.Errorf("WHAT TO FIX shows more than five issues (%q):\n%s", hidden, fix)
		}
	}
	if strings.Contains(rendered, ".aether/data") {
		t.Errorf("blocked screen still lists file paths:\n%s", rendered)
	}
	// Its existing next-step card stays.
	if !strings.Contains(rendered, spacedTitle("Next Up")) && !strings.Contains(rendered, spacedTitle("What Next")) {
		t.Errorf("blocked screen lost its next-step card:\n%s", rendered)
	}
}

// TestCheckScreenBehindTheScenesIsPlainEnglish: the owner approved the new
// layout on 2026-09-27 on condition that the "behind the scenes" lines lose
// their jargon ("Librarian phase advanced WITHOUT consolidation",
// "Signals: 3 active -> 1 active after housekeeping", a "run aether focus"
// hint). Each fact stays, in plain words.
func TestCheckScreenBehindTheScenesIsPlainEnglish(t *testing.T) {
	rendered := renderCheckScreenFixture(t, false)
	start := strings.Index(rendered, "B E H I N D")
	if start < 0 {
		t.Fatalf("no behind-the-scenes section:\n%s", rendered)
	}
	section := rendered[start:]
	if end := strings.Index(section, "Next phase ready"); end > 0 {
		section = section[:end]
	}
	for _, jargon := range []string{"Librarian", "consolidation", "WITHOUT", "housekeeping", "Signals:", "Steering signals: none", "aether focus"} {
		if strings.Contains(section, jargon) {
			t.Errorf("behind-the-scenes still says %q:\n%s", jargon, section)
		}
	}
	for _, fact := range []string{"Steering notes", "Lessons"} {
		if !strings.Contains(section, fact) {
			t.Errorf("behind-the-scenes lost the %q fact:\n%s", fact, section)
		}
	}
}
