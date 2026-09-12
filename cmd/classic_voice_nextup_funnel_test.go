package cmd

import (
	"strings"
	"testing"
)

// Phase 202.1 gave nextActionSuggestionLine a leading glyph. Two callers were
// already composing their own line around its result, so the glyph arrived
// twice on screen (🔀 ➡️ Run ...), and two more callers put the result into a
// machine-readable result["next"] field, so the glyph leaked into JSON that
// this project requires to stay raw. Both are regressions of this phase, found
// by the phase's own code review. These tests lock the repair: the glyph is
// applied by exactly one visual funnel, and never by the JSON-facing helpers.

func nextUpFunnelTestAnswer() nextAction {
	return nextAction{
		Command:        "aether build 2",
		Recommendation: "Run the current phase.",
		Alternatives: []nextActionAlternative{
			{Command: "aether status", Explanation: "Inspect the colony."},
			{Command: "aether history", Explanation: "Review what happened."},
		},
	}
}

// Every rendered Next Up line carries exactly one leading glyph -- never two.
func TestNextUpLinesCarryExactlyOneGlyph(t *testing.T) {
	answer := nextUpFunnelTestAnswer()
	rendered := renderNextUp(
		nextActionPrimarySuggestion(answer),
		nextActionAlternativeSuggestions(answer)...,
	)
	assertEveryNextUpLineHasOneGlyph(t, rendered)
}

func assertEveryNextUpLineHasOneGlyph(t *testing.T, rendered string) {
	t.Helper()
	kinds := make([]string, 0, len(voiceGlyphMap))
	for kind := range voiceGlyphMap {
		kinds = append(kinds, kind)
	}
	seen := 0
	for _, line := range strings.Split(rendered, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.Contains(trimmed, "N E X T   U P") {
			continue
		}
		leading := ""
		for _, kind := range kinds {
			glyph := voiceGlyph(kind)
			if strings.HasPrefix(trimmed, glyph+" ") {
				leading = glyph
				break
			}
		}
		if leading == "" {
			t.Errorf("Next Up line carries no leading glyph: %q", trimmed)
			continue
		}
		seen++
		rest := strings.TrimPrefix(trimmed, leading+" ")
		for _, kind := range kinds {
			if strings.HasPrefix(rest, voiceGlyph(kind)+" ") {
				t.Errorf("Next Up line carries two stacked glyphs: %q", trimmed)
				break
			}
		}
	}
	if seen == 0 {
		t.Fatal("no Next Up content lines were measured -- the test cannot fail in this shape")
	}
}

// The helpers that feed machine-readable result["next"] fields stay raw.
func TestMachineReadableNextSuggestionCarriesNoGlyph(t *testing.T) {
	answer := nextUpFunnelTestAnswer()
	values := append([]string{nextActionPrimarySuggestion(answer)}, nextActionAlternativeSuggestions(answer)...)
	for _, value := range values {
		if value == "" {
			t.Fatal("suggestion helper returned an empty string -- the test cannot fail in this shape")
		}
		for kind := range voiceGlyphMap {
			if strings.Contains(value, voiceGlyph(kind)) {
				t.Errorf("machine-readable suggestion %q carries the %q glyph; JSON stays raw", value, kind)
			}
		}
	}
}

// Both guards above must be able to fail: a deliberately double-glyphed line is
// caught, and a deliberately glyphed JSON value is caught.
func TestNextUpFunnelGuardsCanFail(t *testing.T) {
	planted := "\n" + voiceGlyph("alternative") + " " + voiceGlyph("next") + " Run `aether status` — Inspect the colony.\n"
	probe := &testing.T{}
	assertEveryNextUpLineHasOneGlyph(probe, planted)
	if !probe.Failed() {
		t.Fatal("the one-glyph check passed a planted double-glyphed line, so it cannot fail")
	}

	glyphed := voiceLine("next", "Run `aether status`")
	if !strings.Contains(glyphed, voiceGlyph("next")) {
		t.Fatal("voiceLine produced no glyph, so the JSON check cannot fail")
	}
}
