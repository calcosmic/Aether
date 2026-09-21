package cmd

// Phase "Classic Visual Voice" plan 07, Task 1 -- the coverage ratchet and
// the corpus-wide density gate.
//
// RESEARCH.md's Pitfall 4 names this project's own recurring failure by
// name: capability is rarely deleted, it is simply never called, and a
// per-screen density gate alone cannot catch a screen nobody registered.
// This file is the check that knows which screens are SUPPOSED to carry the
// restored voice -- so a future screen added without joining the corpus is
// caught by name, not discovered later by an owner.
//
// This file deliberately contains no renderer function name and no decimal
// density literal: it walks voiceScreenRegistry (classic_voice_corpus_test.go)
// and a fixed family list, which is what lets it survive a rename of any
// renderer, section, or heading.

import (
	"strings"
	"testing"
)

// voiceScreenFamilies is the eleven required screen families: the original
// eight the roadmap's second criterion lists -- plan, discuss, spec,
// status, build, continue, seal, and the what-next card -- plus the three
// "the owner sees Aether's screens" Part D added: init (project start-up),
// colonize (the code survey), and ceremony (the cards drawn while helpers
// are sent out: spawn-plan, wave-start, worker-complete, closeout). A
// registered screen belongs to a family when its name carries the family
// as a prefix -- allowing a family to register several named variants
// (e.g. "build" and "build-partial", or "ceremony-spawn-plan" and
// "ceremony-closeout") under one family.
var voiceScreenFamilies = []string{
	"plan", "discuss", "spec", "status", "build", "continue", "seal", "what-next",
	"init", "colonize", "ceremony",
}

// missingVoiceFamilies reports every family in families with no registered
// name carrying it as a prefix. Pure over its arguments -- never reads the
// package-level registry directly -- so it can be exercised on a filtered
// copy of registered names without mutating voiceScreenRegistry itself.
func missingVoiceFamilies(names []string, families []string) []string {
	var missing []string
	for _, family := range families {
		found := false
		for _, name := range names {
			if strings.HasPrefix(name, family) {
				found = true
				break
			}
		}
		if !found {
			missing = append(missing, family)
		}
	}
	return missing
}

// registeredVoiceScreenNames returns the name of every screen currently
// registered in the shared corpus.
func registeredVoiceScreenNames(t *testing.T) []string {
	t.Helper()
	screens := renderedVoiceScreens(t) // fails the test outright on an empty registry
	names := make([]string, 0, len(screens))
	for _, screen := range screens {
		names = append(names, screen.Name)
	}
	return names
}

// TestEveryOrdinaryScreenIsMeasuredForVoice fails naming any of the eight
// required screen families with no registration in the corpus, and proves
// (without ever mutating the real registry) that removing all registrations
// for one family is caught by name, and that an empty corpus is missing
// every family.
func TestEveryOrdinaryScreenIsMeasuredForVoice(t *testing.T) {
	names := registeredVoiceScreenNames(t)

	if missing := missingVoiceFamilies(names, voiceScreenFamilies); len(missing) > 0 {
		t.Errorf("the voice corpus has no registration for family(ies) %v; registrations present: %v", missing, names)
	}

	t.Run("fails when a whole family is removed", func(t *testing.T) {
		filtered := make([]string, 0, len(names))
		for _, name := range names {
			if strings.HasPrefix(name, "seal") {
				continue
			}
			filtered = append(filtered, name)
		}
		missing := missingVoiceFamilies(filtered, voiceScreenFamilies)
		if len(missing) != 1 || missing[0] != "seal" {
			t.Errorf("filtering every %q registration out of a copied name list should report exactly [\"seal\"] missing, got %v", "seal", missing)
		}
	})

	t.Run("fails when the corpus is empty", func(t *testing.T) {
		missing := missingVoiceFamilies(nil, voiceScreenFamilies)
		if len(missing) != len(voiceScreenFamilies) {
			t.Errorf("an empty corpus should be missing every one of the %d required families, got %d missing: %v",
				len(voiceScreenFamilies), len(missing), missing)
		}
	})
}

// TestEveryVoicedScreenMeetsTheReferenceDensity renders every registration
// in the shared corpus and asserts its measured density is at or above the
// figure derived from the reference commit, as a subtest per screen --
// reporting the screen name, the measured figure, the required figure, and
// the rendered screen on failure.
func TestEveryVoicedScreenMeetsTheReferenceDensity(t *testing.T) {
	reference := classicReferenceDensity(t)
	for _, screen := range renderedVoiceScreens(t) {
		screen := screen
		t.Run(screen.Name, func(t *testing.T) {
			led, total, ratio := voiceDensity(screen.Rendered)
			if total == 0 {
				t.Fatalf("%s produced no content lines to measure", screen.Name)
			}
			if ratio < reference {
				t.Errorf("%s measures %v (led=%d total=%d), below the required figure %v derived from the reference commit:\n%s",
					screen.Name, ratio, led, total, reference, screen.Rendered)
			}
		})
	}
}

// TestCorpusDensityGateCanFail registers a deliberately flat synthetic
// screen through a local copy of the registry-walk shape (voiceScreenCase,
// the same type the real registry uses) -- never into the shared
// voiceScreenRegistry -- and asserts the same density comparison fails for
// it. This proves the gate can fail against the same code path the real
// screens take, without ever mutating the corpus every other test in the
// package shares.
func TestCorpusDensityGateCanFail(t *testing.T) {
	reference := classicReferenceDensity(t)

	local := []voiceScreenCase{
		{
			Name: "planted-flat-screen",
			Render: func(t *testing.T) string {
				return "Goal: Ship the billing rewrite\n" +
					"Phase: 2 of 4\n" +
					"Tasks: 3 of 10 complete\n" +
					"Status: in progress\n"
			},
		},
	}

	for _, screen := range local {
		rendered := screen.Render(t)
		led, total, ratio := voiceDensity(rendered)
		if total == 0 {
			t.Fatalf("%s produced no content lines to measure", screen.Name)
		}
		if ratio >= reference {
			t.Errorf("%s (led=%d total=%d ratio=%v) unexpectedly met or exceeded the reference figure %v -- the gate did not fail on a planted flat screen",
				screen.Name, led, total, ratio, reference)
		}
	}
}

// ---------------------------------------------------------------------------
// Task 2 -- no registered screen shows an internal state token where a
// sentence belongs.
// ---------------------------------------------------------------------------

// TestVoicedScreensCarryNoRawStateToken runs rawStateTokenLeaks
// (classic_voice_corpus_test.go) over every registered screen's rendered
// output, as a subtest per screen -- the corpus-wide sibling of
// TestVoicedScreensSpeakPlainEnglish, closing RESEARCH.md's criterion 4 for
// every screen a plan has wired into the restored voice, not only the one
// place the leak was first noticed.
func TestVoicedScreensCarryNoRawStateToken(t *testing.T) {
	for _, screen := range renderedVoiceScreens(t) {
		screen := screen
		t.Run(screen.Name, func(t *testing.T) {
			if violations := rawStateTokenLeaks(screen.Rendered); len(violations) > 0 {
				t.Errorf("%s shows an internal state token where a sentence belongs:\n  %s\n\nfull screen:\n%s",
					screen.Name, strings.Join(violations, "\n  "), screen.Rendered)
			}
		})
	}
}

// TestRawStateTokenCheckCanFail plants a rendering carrying one of the
// pause-boundary values that used to leak onto the what-next card
// (RESEARCH.md, "Where Owner-Facing Raw Identifiers Leak") and asserts
// rawStateTokenLeaks reports it, proving the check can fail -- and, as its
// own subtest, that a backticked occurrence of the same shape is correctly
// exempted, because an owner types a backticked command.
func TestRawStateTokenCheckCanFail(t *testing.T) {
	planted := "── Where things stand ──\nhandoff=handoff-060d355f95c7b74a744bd164 between_commands_boundary\n"
	violations := rawStateTokenLeaks(planted)
	if len(violations) == 0 {
		t.Fatalf("the raw-state-token check found nothing wrong with an obviously leaked internal token:\n%s", planted)
	}

	t.Run("a backticked occurrence is exempted", func(t *testing.T) {
		exempt := "Run `aether continue --skip between_commands_boundary` to proceed.\n"
		violations := rawStateTokenLeaks(exempt)
		if len(violations) > 0 {
			t.Errorf("a backticked command carrying an underscore was reported as a leak: %v\n%s", violations, exempt)
		}
	})
}
