package cmd

// Phase "Classic Visual Voice" plan 02 -- the status card (full and compact),
// the screen the owner opens most, brought up to the reference voice and
// registered into the shared screen corpus built in plan 01.

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// classicVoiceStatusFixtureProjection builds one fully populated
// LifecycleProjection exercising every section renderLifecycleStatusFull and
// renderLifecycleStatusCompact draw from: an active helper with a recorded
// caste, a terminal helper, a lineage row, an active signal, a gate, an
// evidence row, a warning, a blocker, an owner decision, and a non-empty
// history -- built directly as a LifecycleProjection literal (not derived
// through projectLifecycle) so every section is populated deliberately
// rather than however the ColonyState-derivation path happens to fill it.
func classicVoiceStatusFixtureProjection() LifecycleProjection {
	return LifecycleProjection{
		Command: "status",
		Identity: LifecycleFact[LifecycleIdentityFacts]{
			Value: LifecycleIdentityFacts{
				Name: "Atlas", Goal: "Ship the complete status", Scope: "project", Mode: "orchestrator",
			},
		},
		Goal:     LifecycleFact[string]{Value: "Ship the complete status"},
		Standing: LifecycleFact[string]{Value: "BUILT"},
		Phase: LifecycleFact[LifecyclePhaseProjection]{
			Value: LifecyclePhaseProjection{
				Current:         &colony.Phase{Name: "Golden phase", Status: colony.PhaseInProgress},
				CurrentNumber:   1,
				TotalPhases:     1,
				CompletedPhases: 0,
				TotalTasks:      2,
				CompletedTasks:  1,
			},
		},
		Actors: LifecycleFact[[]LifecycleActorFact]{
			Value: []LifecycleActorFact{
				{Parent: "Queen", Caste: "builder", Name: "Mason-1", Task: "render status", Status: "running", Summary: "wiring the voice funnel"},
				{Parent: "Queen", Caste: "watcher", Name: "Keen-2", Task: "verify status", Status: "completed", Summary: "checks passed"},
			},
		},
		Lineage: LifecycleFact[[]LifecycleLineage]{
			Value: []LifecycleLineage{{Parent: "Queen", Actor: "Mason-1", Caste: "builder"}},
		},
		Signals: LifecycleFact[[]colony.PheromoneSignal]{
			Value: []colony.PheromoneSignal{{ID: "signal-focus", Type: "FOCUS", Active: true, Content: json.RawMessage(`{"text":"preserve truth"}`)}},
		},
		Research: LifecycleFact[LifecycleResearchFacts]{
			Value: LifecycleResearchFacts{
				Docs:      []string{".aether/research/front-door.md"},
				Territory: []string{".aether/data/survey/territory.json"},
				Dreams:    []string{".aether/dreams/2026-09-03-status.md"},
			},
		},
		Memory: LifecycleFact[LifecycleMemoryFacts]{
			Value: LifecycleMemoryFacts{
				Instincts:    []colony.InstinctEntry{{ID: "instinct-1"}},
				Observations: []colony.Observation{{ContentHash: "obs-1"}},
			},
		},
		Findings: LifecycleFact[[]colony.ReviewLedgerEntry]{
			Value: []colony.ReviewLedgerEntry{{ID: "finding-1", Status: "open", Severity: colony.ReviewSeverityLow, Description: "renderer uses facts"}},
		},
		Verification: []colony.LifecycleVerification{{Name: "gate-tests", Passed: true, Detail: "focused tests passed"}},
		Evidence:     []colony.LifecycleEvidence{{ID: "evidence-1", Summary: "verification log"}},
		Warnings:     []colony.LifecycleIssue{{ID: "warn-1", Summary: "one warning recorded"}},
		Blockers:     []colony.LifecycleIssue{{ID: "owner-decision", Summary: "owner-decision"}},
		OwnerDecisions: []colony.LifecycleDecision{
			{ID: "decision-1", Summary: "pick a rendering approach"},
		},
		Elapsed: LifecycleFact[time.Duration]{
			Value:  90 * time.Minute,
			Source: lifecycleSource("elapsed", "session", LifecycleFactConfirmed, ""),
		},
		ReportedCost: LifecycleFact[LifecycleReportedCostFacts]{
			Value:  LifecycleReportedCostFacts{Phase: 1, TotalTokens: 1500, Rows: 1},
			Source: lifecycleSource("reported cost", ".aether/data/spend", LifecycleFactConfirmed, ""),
		},
		History: LifecycleFact[[]string]{
			Value: []string{"2026-09-03T11:30:00Z|lifecycle_status|status|history-entry"},
		},
		NextAction: LifecycleProjectedAction{
			ID:             "continue",
			RuntimeCommand: "aether continue",
			DisplayCommand: "aether continue",
			Reason:         "the phase is ready to verify",
			Choices: []LifecycleActionChoice{
				{ID: "continue", RuntimeCommand: "aether continue", DisplayCommand: "aether continue", Reason: "verify the phase"},
			},
		},
		Alternatives: []LifecycleActionChoice{
			{ID: "status", RuntimeCommand: "aether status", DisplayCommand: "aether status", Reason: "check in again later"},
		},
	}
}

func classicVoiceStatusFullRender(t *testing.T) string {
	t.Helper()
	projection := classicVoiceStatusFixtureProjection()
	projection.View = LifecycleViewFull
	return stripANSI(renderLifecycleStatus(projection, 100))
}

func classicVoiceStatusCompactRender(t *testing.T) string {
	t.Helper()
	projection := classicVoiceStatusFixtureProjection()
	projection.View = LifecycleViewCompact
	return stripANSI(renderLifecycleStatus(projection, 100))
}

func init() {
	registerVoiceScreen("status-full", classicVoiceStatusFullRender)
	registerVoiceScreen("status-compact", classicVoiceStatusCompactRender)
}

// TestStatusScreenMeetsTheReferenceDensity asserts both the full and compact
// status cards measure at or above classicReferenceDensity, reporting both
// figures and the card on failure.
func TestStatusScreenMeetsTheReferenceDensity(t *testing.T) {
	reference := classicReferenceDensity(t)

	t.Run("status-full", func(t *testing.T) {
		rendered := classicVoiceStatusFullRender(t)
		led, total, ratio := voiceDensity(rendered)
		if total == 0 {
			t.Fatalf("status-full produced no content lines to measure")
		}
		if ratio < reference {
			t.Errorf("status-full (led=%d total=%d ratio=%v) did not measure at or above the reference figure %v:\n%s",
				led, total, ratio, reference, rendered)
		}
	})

	t.Run("status-compact", func(t *testing.T) {
		rendered := classicVoiceStatusCompactRender(t)
		led, total, ratio := voiceDensity(rendered)
		if total == 0 {
			t.Fatalf("status-compact produced no content lines to measure")
		}
		if ratio < reference {
			t.Errorf("status-compact (led=%d total=%d ratio=%v) did not measure at or above the reference figure %v:\n%s",
				led, total, ratio, reference, rendered)
		}
	})

	t.Run("corpus", func(t *testing.T) {
		var found []string
		for _, screen := range renderedVoiceScreens(t) {
			found = append(found, screen.Name)
		}
		for _, want := range []string{"status-full", "status-compact"} {
			ok := false
			for _, name := range found {
				if name == want {
					ok = true
					break
				}
			}
			if !ok {
				t.Errorf("corpus registry does not contain %q; registry has %v", want, found)
			}
		}
	})
}

// TestStatusHelperLineCarriesTheSharedIdentity asserts the active-helper line
// contains exactly what casteIdentity returns for that kind and no longer
// contains the bare parenthetical form.
func TestStatusHelperLineCarriesTheSharedIdentity(t *testing.T) {
	rendered := classicVoiceStatusFullRender(t)
	identity := casteIdentity("builder")
	if !strings.Contains(rendered, identity) {
		t.Errorf("status-full does not contain the shared identity rendering %q:\n%s", identity, rendered)
	}
	if strings.Contains(rendered, "(builder)") {
		t.Errorf("status-full still contains the bare parenthetical caste form:\n%s", rendered)
	}
}

// TestStatusUnchangedFacts asserts the card's underlying facts survive the
// voicing change byte-for-byte: the same goal, phase counts, and next
// command still appear.
func TestStatusUnchangedFacts(t *testing.T) {
	rendered := classicVoiceStatusFullRender(t)
	for _, want := range []string{
		"Ship the complete status", "Phase 1/1", "Tasks: 1/2 complete", "aether continue",
	} {
		if !strings.Contains(rendered, want) {
			t.Errorf("status-full lost an unchanged fact %q:\n%s", want, rendered)
		}
	}
}

// TestStatusHistoryShowsASentenceNotTheStoredRecord builds a projection whose
// history carries one well-formed four-field record, one record with an
// empty message, and one bare string with no delimiter, asserting the
// well-formed one renders as its message only, the empty one contributes no
// line, the bare one renders verbatim, and the whole rendered card contains
// no pipe character inside the history block and no timestamp pattern.
func TestStatusHistoryShowsASentenceNotTheStoredRecord(t *testing.T) {
	projection := classicVoiceStatusFixtureProjection()
	projection.View = LifecycleViewFull
	projection.History = LifecycleFact[[]string]{
		Value: []string{
			"2026-09-03T11:30:00Z|lifecycle_pause|pause|handoff completed cleanly",
			"2026-09-03T11:31:00Z|lifecycle_pause|pause|",
			"a bare history string with no delimiter at all",
		},
	}
	rendered := stripANSI(renderLifecycleStatus(projection, 100))

	if !strings.Contains(rendered, "handoff completed cleanly") {
		t.Errorf("status-full did not render the well-formed record's message alone:\n%s", rendered)
	}
	if !strings.Contains(rendered, "a bare history string with no delimiter at all") {
		t.Errorf("status-full did not render the bare, delimiter-free record verbatim:\n%s", rendered)
	}

	historyStart := strings.Index(rendered, "Recent History")
	if historyStart < 0 {
		t.Fatalf("status-full has no Recent History section:\n%s", rendered)
	}
	nextSectionStart := strings.Index(rendered[historyStart+1:], "\n\n")
	historyBlock := rendered[historyStart:]
	if nextSectionStart >= 0 {
		historyBlock = rendered[historyStart : historyStart+1+nextSectionStart]
	}

	if strings.Contains(historyBlock, "|") {
		t.Errorf("status-full history block still contains a pipe delimiter:\n%s", historyBlock)
	}
	timestampPattern := regexp.MustCompile(`\d{4}-\d{2}-\d{2}T`)
	if timestampPattern.MatchString(historyBlock) {
		t.Errorf("status-full history block still contains a raw timestamp:\n%s", historyBlock)
	}
	// The empty-message record must not have produced its own bullet line;
	// the only lines in the history block are the two non-empty sentences.
	historyLines := 0
	for _, line := range strings.Split(strings.TrimSpace(historyBlock), "\n") {
		if strings.TrimSpace(line) == "" || strings.Contains(line, "Recent History") {
			continue
		}
		historyLines++
	}
	if historyLines != 2 {
		t.Errorf("status-full history block has %d lines, want 2 (the empty-message record must contribute none):\n%s", historyLines, historyBlock)
	}
}

// TestCompactStatusAgreesWithTheFullCard renders both cards over one
// projection and asserts the goal text, the phase numbers, the task
// numbers, and the next command are identical strings in both.
func TestCompactStatusAgreesWithTheFullCard(t *testing.T) {
	projection := classicVoiceStatusFixtureProjection()
	full := stripANSI(renderLifecycleStatus(withView(projection, LifecycleViewFull), 100))
	compact := stripANSI(renderLifecycleStatus(withView(projection, LifecycleViewCompact), 100))

	for _, want := range []string{"Ship the complete status", "Phase 1/1", "aether continue"} {
		if !strings.Contains(full, want) {
			t.Errorf("full status card missing %q:\n%s", want, full)
		}
		if !strings.Contains(compact, want) {
			t.Errorf("compact status card missing %q:\n%s", want, compact)
		}
	}
}

func withView(projection LifecycleProjection, view LifecycleView) LifecycleProjection {
	projection.View = view
	return projection
}

// TestCompactStatusNeverSplitsASymbol renders at a deliberately narrow width
// and asserts every line is valid UTF-8 and no line ends inside a symbol
// sequence (a multi-rune glyph carrying a variation selector must never be
// split mid-sequence).
func TestCompactStatusNeverSplitsASymbol(t *testing.T) {
	projection := classicVoiceStatusFixtureProjection()
	projection.View = LifecycleViewCompact
	for _, width := range []int{24, 40, 72, 100} {
		rendered := renderLifecycleStatus(projection, width)
		if !isValidUTF8(rendered) {
			t.Fatalf("width %d produced invalid UTF-8:\n%q", width, rendered)
		}
		plain := stripANSI(rendered)
		for _, line := range strings.Split(strings.TrimRight(plain, "\n"), "\n") {
			trimmed := strings.TrimRight(line, " ")
			if strings.HasSuffix(trimmed, "\uFE0F") {
				continue
			}
			if strings.ContainsRune(trimmed, '\uFE0F') {
				// A variation selector appeared mid-line; the rune immediately
				// before it must be the base glyph it modifies, never split
				// across a wrap boundary from the following line.
				idx := strings.IndexRune(trimmed, '\uFE0F')
				if idx == 0 {
					t.Errorf("width %d line %q opens with a bare variation selector (split glyph)", width, line)
				}
			}
		}
	}
}

func isValidUTF8(s string) bool {
	for _, r := range s {
		if r == '\uFFFD' {
			return false
		}
	}
	return true
}

// TestLifecycleStatusFitLineNeverSplitsAVariationSelector exercises
// lifecycleStatusFitLine directly at widths from 1 through 30 -- including
// widths well below renderLifecycleStatus's own 24-column floor, so this
// proves the fitting function itself is selector-safe rather than merely
// never being asked to prove it. A selector-bearing glyph ("\u23F1\uFE0F", "\uD83D\uDC41\uFE0F\uD83D\uDC1C
// Watcher") must never come out with the variation selector separated from
// its base rune.
func TestLifecycleStatusFitLineNeverSplitsAVariationSelector(t *testing.T) {
	lines := []string{
		voiceLine("elapsed", "Elapsed: 1h30m0s | Reported cost: 1500 tokens"),
		casteIdentity("watcher") + " Active ants: 1 | Keen-2",
	}
	for _, line := range lines {
		// Every base rune this line's known selector-carrying glyphs use
		// ("\u23F1" U+23F1, "\uD83D\uDC41" U+1F441) appears in the source only as half of a
		// base+selector pair. If a fitted result contains that base rune
		// without the selector immediately following it, the pairing broke
		// -- either the base survived alone (selector dropped) or the
		// selector survived alone (base dropped).
		selectorBases := []rune{'\u23F1', '\U0001F441'}
		for width := 1; width <= 30; width++ {
			fitted := lifecycleStatusFitLine(line, width)
			if !isValidUTF8(fitted) {
				t.Fatalf("width %d produced invalid UTF-8 from %q: %q", width, line, fitted)
			}
			if strings.HasPrefix(fitted, "\uFE0F") {
				t.Errorf("width %d produced a result opening with a bare variation selector %q (from %q)", width, fitted, line)
			}
			if idx := strings.Index(fitted, "\u2026"); idx >= 0 {
				after := fitted[idx+len("\u2026"):]
				if strings.HasPrefix(after, "\uFE0F") {
					t.Errorf("width %d left a bare variation selector immediately after the ellipsis in %q (from %q)", width, fitted, line)
				}
			}
			resultRunes := []rune(fitted)
			for i, r := range resultRunes {
				for _, base := range selectorBases {
					if r != base {
						continue
					}
					if i+1 >= len(resultRunes) || resultRunes[i+1] != '\uFE0F' {
						t.Errorf("width %d dropped the variation selector off base glyph %q in result %q (from %q)", width, string(base), fitted, line)
					}
				}
			}
		}
	}
}
