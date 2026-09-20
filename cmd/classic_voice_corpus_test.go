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
	"fmt"
	"regexp"
	"strings"
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

// rawStateTokenKeyEqualsValuePattern matches a bookkeeping key=value pair
// (e.g. "state_effect=rolled_back") -- the shape an owner never types, only
// internal code does.
var rawStateTokenKeyEqualsValuePattern = regexp.MustCompile(`\b[a-zA-Z_][a-zA-Z0-9_]*=\S+`)

// rawStateTokenCommandInvocationRe matches a full CLI invocation an owner is
// told to type verbatim -- a slash command or an "aether <verb>" command --
// through the rest of its line, flags and argument values included (e.g.
// "/ant-discuss --answer D1=local"). Like a bare backticked command or a
// flag, an owner types this whole example; it is not an internal token
// leaking through.
var rawStateTokenCommandInvocationRe = regexp.MustCompile(`(?:/ant-[a-z0-9-]+|\baether [a-z][a-z0-9-]*).*`)

// rawStateTokenLeaks reports every line, outside a backticked span or a
// full CLI invocation an owner types (rawStateTokenCommandInvocationRe,
// which also covers the slash-command shape codeSpanRe matches in
// next_action_card_test.go), that carries a lowercase identifier joined by
// one or more underscores (underscoreTokenPattern, classic_voice_event_test.go
// -- the exact shape of every pauseSafeBoundary/RecoveryProvenance value,
// and of this codebase's other internal enum constants) or a key=value
// bookkeeping pair. This follows untranslatedRepoWords' shape: strip what an
// owner types first, then scan what remains -- but per rendered line rather
// than per sentence, since RESEARCH.md's criterion 4 is about a token
// appearing on an owner-facing line, not about an unexplained word in a
// sentence.
func rawStateTokenLeaks(text string) []string {
	var violations []string
	for _, rawLine := range strings.Split(text, "\n") {
		stripped := rawStateTokenCommandInvocationRe.ReplaceAllString(rawLine, " ")
		stripped = codeSpanRe.ReplaceAllString(stripped, " ")
		if strings.TrimSpace(stripped) == "" {
			continue
		}
		if m := underscoreTokenPattern.FindString(stripped); m != "" {
			violations = append(violations, fmt.Sprintf("line %q carries the raw token %q", strings.TrimSpace(rawLine), m))
		}
		if m := rawStateTokenKeyEqualsValuePattern.FindString(stripped); m != "" {
			violations = append(violations, fmt.Sprintf("line %q carries the bookkeeping pair %q", strings.TrimSpace(rawLine), m))
		}
	}
	return violations
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
