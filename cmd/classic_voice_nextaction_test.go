package cmd

// Phase "Classic Visual Voice" plan 01 (tracer task) -- the what-next card
// wired through the derived reference figure.

import (
	"strings"
	"testing"
)

// TestWhatNextCardMeetsTheReferenceDensity renders the what-next card through
// the same platform funnel production uses, over the existing fullest
// fixture answer (next_action_card_test.go's fullNextActionAnswer), and
// asserts its measured density meets or exceeds the figure derived from the
// reference commits -- on both platform spellings, since command translation
// happens only on the way out and must not change how glyph-led the card is.
func TestWhatNextCardMeetsTheReferenceDensity(t *testing.T) {
	reference := classicReferenceDensity(t)
	answer := fullNextActionAnswer()

	for _, platform := range []string{"claude", "codex"} {
		t.Run(platform, func(t *testing.T) {
			// renderNextUp (the legacy, non-projection next-up path this
			// fixture exercises) derives its own platform from the process
			// environment rather than the platform parameter threaded
			// through renderNextActionCardForPlatform -- pin it explicitly,
			// exactly as every other cross-platform card test already does
			// (e.g. TestNextActionCardTranslatesOnlyOnTheWayOut), so the
			// assertion is not at the mercy of ambient process detection.
			t.Setenv("AETHER_PLATFORM", platform)
			rendered := renderNextActionCardForPlatform(answer, platform)
			led, total, ratio := voiceDensity(rendered)
			if total == 0 {
				t.Fatalf("the %s card produced no content lines to measure:\n%s", platform, rendered)
			}
			if ratio < reference {
				t.Errorf("the %s card measures %v (led=%d total=%d), which is below the reference figure %v:\n%s",
					platform, ratio, led, total, reference, rendered)
			}
		})
	}
}

// TestNextStepMenuNamesItsEffectBesideItsCommand asserts the rendered
// primary next-step line still names the platform-correct command and its
// real-world effect on one line (unchanged from before this plan), and that
// the line is now glyph-led.
func TestNextStepMenuNamesItsEffectBesideItsCommand(t *testing.T) {
	answer := fullNextActionAnswer()

	cases := []struct {
		platform    string
		wantCommand string
	}{
		{platform: "claude", wantCommand: "/ant-continue"},
		{platform: "codex", wantCommand: "aether continue"},
	}

	for _, tc := range cases {
		t.Run(tc.platform, func(t *testing.T) {
			t.Setenv("AETHER_PLATFORM", tc.platform)
			rendered := renderNextActionCardForPlatform(answer, tc.platform)
			var primaryLine string
			for _, line := range strings.Split(rendered, "\n") {
				if strings.Contains(line, tc.wantCommand) {
					primaryLine = line
					break
				}
			}
			if primaryLine == "" {
				t.Fatalf("no rendered line on %s names the command %q:\n%s", tc.platform, tc.wantCommand, rendered)
			}
			if !strings.Contains(primaryLine, answer.Recommendation) {
				t.Errorf("on %s the line naming %q does not also name the effect %q on the same line: %q",
					tc.platform, tc.wantCommand, answer.Recommendation, primaryLine)
			}
			glyphs := sortedGlyphsLongestFirst(voiceGlyphSet())
			if !voiceLineIsLed(primaryLine, glyphs) {
				t.Errorf("on %s the next-step line is not glyph-led: %q", tc.platform, primaryLine)
			}
		})
	}
}
