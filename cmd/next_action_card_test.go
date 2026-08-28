package cmd

// Phase 197 plan 02 -- the one closing card.
//
// These tests are driven over REAL rendered output, never over the source that
// produces it. A check that greps the source for a section name passes the
// moment someone renames the section; a check that reads what the owner would
// actually see does not.

import (
	"regexp"
	"strings"
	"testing"
)

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

// renderedClosingCard is one card as the owner would actually see it.
type renderedClosingCard struct {
	name     string
	rendered string
}

// renderedClosingCards produces the closing cards this plan is responsible for,
// each rendered exactly as the terminal would receive it.
func renderedClosingCards(t *testing.T) []renderedClosingCard {
	t.Helper()
	pinRawCommandNames(t)

	return []renderedClosingCard{
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
