package cmd

// Phase 197 plan 02 -- the one closing card.
//
// These tests are driven over REAL rendered output, never over the source that
// produces it. A check that greps the source for a section name passes the
// moment someone renames the section; a check that reads what the owner would
// actually see does not.

import (
	"os"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// readSourceFile reads a file from the cmd package directory.
func readSourceFile(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(data)
}

// cardCommandRe finds every command a rendered card offers, in either spelling:
// the runtime form on the command line, the slash form on the wrapper
// platforms. Both are matched so the duplicate check works on any platform.
var cardCommandRe = regexp.MustCompile("`(/ant-[a-z][a-z0-9-]*|aether [a-z][^`]*)`")

// commandsOfferedBy lists every command a rendered card puts in front of the
// owner, in the order they appear.
func commandsOfferedBy(rendered string) []string {
	var commands []string
	for _, match := range cardCommandRe.FindAllStringSubmatch(rendered, -1) {
		commands = append(commands, strings.TrimSpace(match[1]))
	}
	return commands
}

// fullNextActionAnswer is an answer with every field populated, so a renderer
// that silently drops one is caught. TestNextActionCardRendersEveryField
// asserts the fixture itself stays complete, so a ninth field added to
// nextAction fails here rather than going unrendered in silence.
func fullNextActionAnswer() nextAction {
	return nextAction{
		Standing: nextActionStanding{
			State:           string(colony.StateEXECUTING),
			Goal:            "Ship the billing rewrite",
			Milestone:       "Open Chambers",
			CurrentPhase:    2,
			PhaseName:       "Billing engine",
			TotalPhases:     4,
			CompletedPhases: 1,
			Explanation:     "Phase 2 of 4: Billing engine. 1 phase(s) finished so far.",
		},
		Changed: []string{
			"The last thing you ran was build.",
			"Phase 1 finished and was checked.",
		},
		Open: nextActionOpenItems{
			Flags: []nextActionFlag{{
				ID:          "flag-1",
				Type:        "blocker",
				Description: "The payment provider has not been chosen yet.",
			}},
			Signals: []string{"Pay close attention to the checkout path."},
		},
		Recommendation: "Phase 2 has produced work that has not been checked yet. The next step runs the checks.",
		Command:        "aether continue",
		Alternatives: []nextActionAlternative{
			{Command: "aether status", Explanation: "Look at the dashboard first, without changing anything."},
			{Command: "aether history", Explanation: "Read back what has happened on this project so far."},
		},
		ContextHealth: nextActionContextVerdict{Health: contextHealthSafe, Reason: contextReasonHandoffSaved},
		Recovery: nextActionRecovery{
			Blocked:     true,
			ReportPath:  ".aether/data/build/phase-2/continue.json",
			Summary:     "Two tasks were never credited.",
			Explanation: "The last check on this phase did not pass, and the report saying why is saved.",
		},
		Notes: []string{"A saved report named a command this version does not have, so a general next step is offered instead."},
	}
}

// resolvedAnswerFromDisk produces an answer the way production does: a saved
// state on disk, read back by the runtime's own loader and passed through the
// one resolver. The card is proved against a genuinely produced answer, not
// only against a constructed one.
func resolvedAnswerFromDisk(t *testing.T) nextAction {
	t.Helper()
	newNextActionFixtureStore(t)
	started := time.Now().UTC().Add(-time.Minute)
	state := normalizedFixtureState(t, colony.ColonyState{
		Version:        "1.0",
		Goal:           fixtureGoal("Ship the billing rewrite"),
		State:          colony.StateEXECUTING,
		CurrentPhase:   2,
		BuildStartedAt: &started,
		Milestone:      "Open Chambers",
		Plan: colony.Plan{Phases: []colony.Phase{
			fixturePhase(1, "Foundations", colony.PhaseCompleted),
			fixturePhase(2, "Billing engine", colony.PhaseInProgress),
		}},
	})
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("write fixture state through the runtime's own store: %v", err)
	}
	return resolveNextAction(loadNextActionInput())
}

// renderedClosingCard is one card as the owner would actually see it.
type renderedClosingCard struct {
	name     string
	rendered string
}

// renderedNextActionCards is the card this plan owns, rendered both from a
// constructed answer with every field filled and from an answer the runtime
// genuinely produced off disk.
func renderedNextActionCards(t *testing.T) []renderedClosingCard {
	t.Helper()
	pinRawCommandNames(t)

	return []renderedClosingCard{
		{
			name:     "the next-action card, fully populated",
			rendered: renderNextActionCard(fullNextActionAnswer()),
		},
		{
			name:     "the next-action card, from a real saved state",
			rendered: renderNextActionCard(resolvedAnswerFromDisk(t)),
		},
	}
}

// renderedClosingCards is every closing card whose STRUCTURE this plan can
// speak for: the new one, plus the pause card, whose duplicate-command defect
// this plan fixed. The WORDING of the other ten legacy cards is plans 197-04
// and 197-06's work, so they are not held to the plain-English check here --
// but offering one command twice is a structural defect checkable on any card
// today, so the duplicate check covers the pause card as well.
func renderedClosingCards(t *testing.T) []renderedClosingCard {
	t.Helper()
	cards := []renderedClosingCard{
		{
			name: "the pause card",
			rendered: renderPauseVisual(map[string]interface{}{
				"goal":          "Ship the billing rewrite",
				"current_phase": 2,
				"phase_name":    "Billing engine",
				"handoff_path":  ".aether/HANDOFF.md",
			}),
		},
	}
	return append(cards, renderedNextActionCards(t)...)
}

// TestNoCardOffersTheSameCommandTwice.
//
// A closing card that lists the same command as both the recommendation and an
// alternative is telling the owner there is a choice when there is none. The
// pause card shipped exactly that defect: it offered `aether resume` as the way
// to pick the project back up and then offered `aether resume` again, described
// as "the compact dashboard view instead" -- two doors with one room behind
// them, and no way for the owner to reach the fuller restore the second line
// was clearly meant to name.
func TestNoCardOffersTheSameCommandTwice(t *testing.T) {
	for _, card := range renderedClosingCards(t) {
		t.Run(card.name, func(t *testing.T) {
			seen := map[string]bool{}
			for _, command := range commandsOfferedBy(card.rendered) {
				if seen[command] {
					t.Errorf("%s offers %q more than once -- the owner is being shown a choice that is not a choice:\n%s",
						card.name, command, card.rendered)
					continue
				}
				seen[command] = true
			}
		})
	}
}

func TestNextActionCardRendersEveryField(t *testing.T) {
	pinRawCommandNames(t)
	answer := fullNextActionAnswer()

	// A ninth field added to the answer must be populated here, or the
	// coverage below quietly stops being coverage.
	value := reflect.ValueOf(answer)
	for i := 0; i < value.NumField(); i++ {
		if value.Field(i).IsZero() {
			t.Fatalf("the fixture leaves %s empty, so nothing below proves the card renders it",
				value.Type().Field(i).Name)
		}
	}

	rendered := renderNextActionCard(answer)
	for _, want := range []string{
		"Ship the billing rewrite",
		"Phase 2 of 4: Billing engine",
		"The last thing you ran was build.",
		"Phase 1 finished and was checked.",
		"The payment provider has not been chosen yet.",
		"Pay close attention to the checkout path.",
		"Phase 2 has produced work that has not been checked yet.",
		"aether continue",
		"aether status",
		"aether history",
		"safe to close this chat",
		"The last check on this phase did not pass",
		".aether/data/build/phase-2/continue.json",
		"A saved report named a command this version does not have",
	} {
		if !strings.Contains(rendered, want) {
			t.Errorf("the card never shows %q:\n%s", want, rendered)
		}
	}

	t.Run("a section with nothing to say is left out", func(t *testing.T) {
		bare := nextAction{
			Standing:       nextActionStanding{State: "READY", Explanation: "The goal is saved. No phases have been drawn up yet."},
			Recommendation: "The next step is a short conversation to pin down what you want.",
			Command:        "aether discuss",
			ContextHealth:  nextActionContextVerdict{Health: contextHealthKeep, Reason: contextReasonHandoffMissing},
		}
		rendered := renderNextActionCard(bare)
		for _, unwanted := range []string{"What changed", "Waiting on you", "Paused or blocked"} {
			if strings.Contains(rendered, unwanted) {
				t.Errorf("the card prints an empty %q section:\n%s", unwanted, rendered)
			}
		}
		if strings.Contains(rendered, "── \n") || strings.Contains(rendered, "  - \n") {
			t.Errorf("the card prints an empty line where a section used to be:\n%s", rendered)
		}
	})
}

// TestNextActionCardTranslatesOnlyOnTheWayOut is S-01 held at the card.
//
// The owner must be shown the command they can actually type, but the value the
// card was handed must come out the other side unchanged -- wrappers and the TS
// host EXECUTE that value, and `/ant-continue` handed to exec is not a program.
func TestNextActionCardTranslatesOnlyOnTheWayOut(t *testing.T) {
	cases := []struct {
		platform string
		want     string
		notWant  string
	}{
		{platform: "claude", want: "/ant-continue", notWant: "aether continue"},
		{platform: "opencode", want: "/ant-continue", notWant: "aether continue"},
		{platform: "codex", want: "aether continue", notWant: "/ant-continue"},
	}

	for _, tc := range cases {
		t.Run(tc.platform, func(t *testing.T) {
			t.Setenv("AETHER_PLATFORM", tc.platform)
			answer := fullNextActionAnswer()
			rendered := renderNextActionCard(answer)

			if !strings.Contains(rendered, tc.want) {
				t.Errorf("on %s the owner is never shown %q:\n%s", tc.platform, tc.want, rendered)
			}
			if strings.Contains(rendered, tc.notWant) {
				t.Errorf("on %s the card still shows %q, which is not what the owner types:\n%s",
					tc.platform, tc.notWant, rendered)
			}
			if answer.Command != "aether continue" {
				t.Errorf("rendering on %s changed the answer's own command to %q; it must stay the runtime form",
					tc.platform, answer.Command)
			}
			for _, alternative := range answer.Alternatives {
				if strings.HasPrefix(alternative.Command, "/ant-") {
					t.Errorf("rendering on %s rewrote an alternative to %q inside the answer",
						tc.platform, alternative.Command)
				}
			}
		})
	}
}

// repoInventedWords is CLAUDE.md's vocabulary table, with the everyday words
// that count as explaining each one in the same sentence.
var repoInventedWords = map[string][]string{
	"colony":    {"project"},
	"queen":     {"coordinator", "decides"},
	"caste":     {"helper", "kind of"},
	"worker":    {"helper"},
	"pheromone": {"note", "instruction", "steer"},
	"instinct":  {"lesson", "learned"},
	"hive":      {"shared", "other projects", "across"},
	"hub":       {"installed copy", "machine"},
	"wrapper":   {"menu command", "shortcut"},
	"seal":      {"sign", "finish", "complete", "mark"},
	"entomb":    {"archive", "file", "away"},
	"midden":    {"log", "went wrong"},
	"orphan":    {"never runs", "nothing calls"},
	"ratchet":   {"check"},
	"allowlist": {"exception"},
}

// codeSpanRe matches a backticked command or a slash command. A command is a
// literal the owner types; the sentence around it is what has to explain it.
var codeSpanRe = regexp.MustCompile("`[^`]*`|/ant-[a-z0-9-]+")

// untranslatedRepoWords reports every sentence that uses a word this repository
// invented without saying, in that same sentence, what it means.
func untranslatedRepoWords(text string) []string {
	var violations []string
	for _, sentence := range regexp.MustCompile(`[.!?\n]`).Split(text, -1) {
		stripped := strings.ToLower(codeSpanRe.ReplaceAllString(sentence, " "))
		if strings.TrimSpace(stripped) == "" {
			continue
		}
		for word, cues := range repoInventedWords {
			if !regexp.MustCompile(`\b` + word + `[a-z]*\b`).MatchString(stripped) {
				continue
			}
			explained := false
			for _, cue := range cues {
				if strings.Contains(stripped, cue) {
					explained = true
					break
				}
			}
			if !explained {
				violations = append(violations, word+": "+strings.TrimSpace(sentence))
			}
		}
	}
	return violations
}

func TestNextActionCardSpeaksPlainEnglish(t *testing.T) {
	for _, card := range renderedNextActionCards(t) {
		t.Run(card.name, func(t *testing.T) {
			if violations := untranslatedRepoWords(card.rendered); len(violations) > 0 {
				t.Errorf("%s uses words this repository invented without explaining them:\n  %s\n\nfull card:\n%s",
					card.name, strings.Join(violations, "\n  "), card.rendered)
			}
		})
	}
}

// TestPlainEnglishCheckCanFail is the guard on the guard. A check that cannot
// report a violation proves nothing about the cards that pass it.
func TestPlainEnglishCheckCanFail(t *testing.T) {
	planted := "── Where things stand ──\nThe colony is ready.\nRun `aether seal` to entomb it.\n"
	violations := untranslatedRepoWords(planted)
	if len(violations) == 0 {
		t.Fatalf("the plain-English check found nothing wrong with obviously untranslated text:\n%s", planted)
	}
}

// TestNextActionCardRefusesUnsafeClear.
//
// "Safe to close the chat" is a CLAIM. It may only be made when the handover
// note that makes it true is verifiably on disk -- the rule that has lived in
// renderContextClearGuidanceForPlatform and moved into the resolver. When the
// note is missing the honest line is "not yet", never a softened yes.
func TestNextActionCardRefusesUnsafeClear(t *testing.T) {
	answer := fullNextActionAnswer()
	answer.ContextHealth = nextActionContextVerdict{Health: contextHealthKeep, Reason: contextReasonHandoffMissing}

	rendered := renderNextActionCard(answer)
	if !strings.Contains(strings.ToLower(rendered), "don't close this chat yet") {
		t.Errorf("with no handover note on disk the card does not tell the owner to stay put:\n%s", rendered)
	}
	if strings.Contains(strings.ToLower(rendered), "safe to close") {
		t.Errorf("with no handover note on disk the card still claims it is safe to close:\n%s", rendered)
	}

	t.Run("the card holds no second copy of the rule", func(t *testing.T) {
		// The card renders the resolved verdict. If it decided for itself it
		// would have to look for the handover file, and two implementations of
		// one rule is exactly how the rule drifts back apart.
		source := readSourceFile(t, "next_action_card.go")
		for _, forbidden := range []string{"HANDOFF.md", "fileExists(", "handoffDocumentPath("} {
			if strings.Contains(source, forbidden) {
				t.Errorf("next_action_card.go contains %q -- the card must render the resolved verdict, not decide it again",
					forbidden)
			}
		}
	})
}
