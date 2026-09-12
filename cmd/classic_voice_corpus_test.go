package cmd

// Phase "Classic Visual Voice" plan 01, Task 2 -- the screen corpus every
// later plan registers into, and the plain-English scan widened onto it.
//
// Restoring the Classic look must not smuggle in bare repo vocabulary -- a
// symbol is decoration, a word the owner has never seen is jargon. The check
// this corpus applies is CLAUDE.md's rule that a word this repository
// invented is explained in the sentence it appears in. Helper identity
// labels attached to a name by casteIdentity (Builder, Watcher, Queen) are
// names, not category words, and are not what the check catches; the bare
// category words themselves ("caste", "worker", ...) are.
//
// Every later plan wiring a screen into the restored voice registers it here,
// with its own `init` in that screen's own test file (see
// classic_voice_nextaction_test.go's registration of the what-next card) --
// one `init` per screen, so two plans never edit the same file.

import (
	"testing"
)

// voiceScreenCase is one registered screen: a name and a function that
// renders it the way production genuinely does.
type voiceScreenCase struct {
	Name   string
	Render func(t *testing.T) string
}

// voiceScreenRegistry is the package-level list every later plan's screen
// joins via registerVoiceScreen.
var voiceScreenRegistry []voiceScreenCase

// registerVoiceScreen adds a screen to the corpus. Registering the same name
// twice panics naming the duplicate, rather than silently shadowing it --
// two plans quietly registering the same screen name is a bug worth failing
// loudly on, not a shadow to debug later.
func registerVoiceScreen(name string, render func(t *testing.T) string) {
	for _, existing := range voiceScreenRegistry {
		if existing.Name == name {
			panic("registerVoiceScreen: duplicate screen name " + name)
		}
	}
	voiceScreenRegistry = append(voiceScreenRegistry, voiceScreenCase{Name: name, Render: render})
}

// renderedVoiceScreen is one registered screen, rendered.
type renderedVoiceScreen struct {
	Name     string
	Rendered string
}

// renderedVoiceScreens walks the registry and renders every screen. It fails
// the test rather than skipping when the registry is empty -- an empty
// corpus proves nothing about the check that runs over it.
func renderedVoiceScreens(t *testing.T) []renderedVoiceScreen {
	t.Helper()
	if len(voiceScreenRegistry) == 0 {
		t.Fatal("voiceScreenRegistry is empty -- no screen has registered into the corpus")
	}
	rendered := make([]renderedVoiceScreen, 0, len(voiceScreenRegistry))
	for _, screen := range voiceScreenRegistry {
		rendered = append(rendered, renderedVoiceScreen{Name: screen.Name, Rendered: screen.Render(t)})
	}
	return rendered
}

// TestVoicedScreensSpeakPlainEnglish runs the existing untranslatedRepoWords
// check (next_action_card_test.go) over every registered screen's rendered
// output, as a subtest per screen -- the same plain-English rule the
// what-next card already had to satisfy, now applied to every screen a later
// plan wires into the restored voice.
func TestVoicedScreensSpeakPlainEnglish(t *testing.T) {
	for _, screen := range renderedVoiceScreens(t) {
		screen := screen
		t.Run(screen.Name, func(t *testing.T) {
			if violations := untranslatedRepoWords(screen.Rendered); len(violations) > 0 {
				t.Errorf("%s uses words this repository invented without explaining them:\n  %s\n\nfull screen:\n%s",
					screen.Name, joinVoiceViolations(violations), screen.Rendered)
			}
		})
	}
}

func joinVoiceViolations(violations []string) string {
	out := ""
	for i, v := range violations {
		if i > 0 {
			out += "\n  "
		}
		out += v
	}
	return out
}

// TestVoicedScreenPlainEnglishCheckCanFail registers nothing and plants a
// rendered string naming an invented word with no explanation, asserting
// untranslatedRepoWords reports it -- proving the widened scan can fail on
// the same corpus path the real screens take, not only pass.
func TestVoicedScreenPlainEnglishCheckCanFail(t *testing.T) {
	planted := "── Where things stand ──\nThe caste finished its task.\n"
	violations := untranslatedRepoWords(planted)
	if len(violations) == 0 {
		t.Fatalf("the widened plain-English check found nothing wrong with an obviously untranslated screen:\n%s", planted)
	}
}
