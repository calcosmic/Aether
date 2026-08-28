package cmd

// Phase 197 plan 03 -- the request that must not come back.
//
// Six shipped documents were checked; four of them ASKED the assistant to look
// at the saved session file on the first message of a conversation and tell the
// owner what it found:
//
//	CLAUDE.md
//	AGENTS.md
//	.claude/rules/aether-colony.md
//	.aether/rules/aether-colony.md
//
// That is a request, not a mechanism. It ran only when it was noticed, and
// CLAUDE.md's own Definition of Done says a documentation claim about runtime
// behaviour must be testable or removed. Plan 03 replaced it with a hook the
// runtime fires (cmd/hook_cmds.go, hookSessionStartCmd), so the greeting now
// happens whether or not anyone remembers.
//
// This guard is what stops the request drifting back in. It matches on the
// instruction's SHAPE -- start-of-conversation, plus the saved session file,
// plus reporting what was found -- held as data below, so a reworded return is
// caught as surely as a copy-paste of the original. A guard pinned to one
// sentence would be silently defeated by the first paraphrase.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// sessionGreetingDocuments are the six documents the plan reviewed. All six are
// scanned, including the two that never carried the instruction (README.md and
// docs/phase3-section-walkthrough.md, which show example output of a command
// the OWNER runs -- not an instruction to the assistant), because the guard's
// job is to stop the request appearing anywhere it plausibly could, not only
// where it once was.
var sessionGreetingDocuments = []string{
	"CLAUDE.md",
	"AGENTS.md",
	"README.md",
	filepath.Join(".claude", "rules", "aether-colony.md"),
	filepath.Join(".aether", "rules", "aether-colony.md"),
	filepath.Join("docs", "phase3-section-walkthrough.md"),
}

// The instruction's three distinguishing ideas. A passage is the forbidden
// instruction only when it carries ALL THREE close together: it is about the
// opening of a conversation, it points at the saved session file, and it asks
// for what was found to be reported. Any one alone is ordinary prose -- the
// state-contract documents name session.json constantly and must stay legal.
var (
	// sessionGreetingNoun is the thing that opens: a conversation, a chat, a
	// session, or any pairing of those words. Written once and shared by every
	// "when" pattern below so a rewording that swaps one noun for another --
	// "chat" for "conversation", "chat session" for "session" -- cannot slip
	// past a pattern that happened to name only one of them.
	sessionGreetingNoun = `(?:new )?(?:chat session|conversation session|conversation|chat|session)`

	sessionGreetingWhenPatterns = []string{
		`first message of (?:a|the|any) ` + sessionGreetingNoun,
		`(?:start|beginning|opening) of (?:a|the|each|every) ` + sessionGreetingNoun,
		`when (?:a|the) ` + sessionGreetingNoun + ` (?:starts|begins|opens|is opened|is started)`,
		`on (?:a|the|each|every) ` + sessionGreetingNoun + ` (?:start|opening)`,
	}
	sessionGreetingWherePatterns = []string{
		`session\.json`,
		`session file`,
		`saved session`,
	}
	sessionGreetingReportPatterns = []string{
		`display`,
		`report`,
		`tell the (user|owner)`,
		`show the (user|owner)`,
		`print`,
		`inform the (user|owner)`,
	}
)

// sessionGreetingWindowLines is how close the three ideas must sit to count as
// one instruction. The original spanned five lines in every document that
// carried it; twelve leaves room for a wordier rewrite without reaching across
// unrelated sections.
const sessionGreetingWindowLines = 12

func compileSessionGreetingPatterns(t *testing.T, patterns []string) []*regexp.Regexp {
	t.Helper()
	compiled := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		re, err := regexp.Compile(`(?i)` + pattern)
		if err != nil {
			t.Fatalf("pattern %q does not compile: %v", pattern, err)
		}
		compiled = append(compiled, re)
	}
	return compiled
}

func matchesAny(res []*regexp.Regexp, text string) bool {
	for _, re := range res {
		if re.MatchString(text) {
			return true
		}
	}
	return false
}

// findDelegatedGreetingInstruction returns the 1-based line number of the first
// passage carrying all three ideas, or 0 when the text is clean.
func findDelegatedGreetingInstruction(t *testing.T, text string) int {
	t.Helper()
	when := compileSessionGreetingPatterns(t, sessionGreetingWhenPatterns)
	where := compileSessionGreetingPatterns(t, sessionGreetingWherePatterns)
	report := compileSessionGreetingPatterns(t, sessionGreetingReportPatterns)

	lines := strings.Split(text, "\n")
	for i := range lines {
		end := i + sessionGreetingWindowLines
		if end > len(lines) {
			end = len(lines)
		}
		window := strings.Join(lines[i:end], "\n")
		if matchesAny(when, window) && matchesAny(where, window) && matchesAny(report, window) {
			return i + 1
		}
	}
	return 0
}

// TestSessionGreetingIsNotDelegatedToTheAssistant is the removal guard. It
// fails, naming the document and the line, if any shipped document asks the
// assistant to do by hand what the runtime now does on its own.
func TestSessionGreetingIsNotDelegatedToTheAssistant(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("findRepoRoot: %v", err)
	}

	scanned := 0
	for _, rel := range sessionGreetingDocuments {
		path := filepath.Join(root, rel)
		data, err := os.ReadFile(path)
		if err != nil {
			// A guard that silently skips a missing document is a guard that
			// stops guarding the day someone renames one.
			t.Errorf("read %s: %v -- this guard cannot protect a document it cannot read", rel, err)
			continue
		}
		scanned++
		if line := findDelegatedGreetingInstruction(t, string(data)); line != 0 {
			t.Errorf("%s:%d asks the assistant to check the saved session file when a conversation opens and report what it finds.\n"+
				"The runtime does this itself now (hook-session-start, registered in .claude/settings.json). "+
				"Delete the request rather than restating it -- an instruction only runs when it is noticed.", rel, line)
		}
	}

	if scanned != len(sessionGreetingDocuments) {
		t.Fatalf("scanned %d of %d documents; a partial sweep proves nothing about the ones it missed",
			scanned, len(sessionGreetingDocuments))
	}
}

// TestSessionGreetingGuardCanFail is the guard on the guard. A check that
// cannot report a violation is a false certificate, and this repository has
// shipped several. Both fixtures are held here, not on disk: the original
// wording, and a paraphrase sharing none of its sentences -- which is what
// proves the guard matches the instruction's shape rather than its words.
func TestSessionGreetingGuardCanFail(t *testing.T) {
	fixtures := map[string]string{
		"the original wording": "## Session Recovery\n\n" +
			"On the first message of a new conversation, check if `.aether/data/session.json` exists. If it does:\n\n" +
			"1. Read the file briefly to check for `colony_goal`\n" +
			"2. If a goal exists, display the previous goal to the user\n",
		"a paraphrase sharing no sentence with it": "## Picking up again\n\n" +
			"When a chat session starts, take a look at the saved session file and, if there is\n" +
			"something in it worth knowing, tell the user what you found before doing anything else.\n",
	}

	for name, fixture := range fixtures {
		t.Run(name, func(t *testing.T) {
			if line := findDelegatedGreetingInstruction(t, fixture); line == 0 {
				t.Errorf("the guard did not report a violation for %s, so it cannot catch the instruction returning:\n%s", name, fixture)
			}
		})
	}

	// The other direction: ordinary prose that names the session file, or
	// describes the runtime doing the greeting, must stay legal. A guard that
	// fires on those would be edited away the first time it blocked something
	// legitimate.
	legal := map[string]string{
		"a state-contract table naming the file": "| `session.json` | Current session metadata: session id, last command, colony goal |\n",
		"a description of the runtime doing it": "When a chat is opened, resumed, or carried on after being cleared, the runtime\n" +
			"prints a short card saying where things stand and what to run next.\n",
	}
	for name, fixture := range legal {
		t.Run("stays legal: "+name, func(t *testing.T) {
			if line := findDelegatedGreetingInstruction(t, fixture); line != 0 {
				t.Errorf("the guard fired on legitimate prose (%s) at line %d:\n%s", name, line, fixture)
			}
		})
	}
}

// TestColonyRulesCopiesStayIdentical keeps the two copies of the colony rules
// file byte-identical. Nothing else in the repository enforced this, so a
// removal applied to one copy and not the other would have shipped silently --
// the hub-published copy is the one downstream repositories actually read.
func TestColonyRulesCopiesStayIdentical(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("findRepoRoot: %v", err)
	}

	claudeCopy, err := os.ReadFile(filepath.Join(root, ".claude", "rules", "aether-colony.md"))
	if err != nil {
		t.Fatalf("read .claude/rules/aether-colony.md: %v", err)
	}
	aetherCopy, err := os.ReadFile(filepath.Join(root, ".aether", "rules", "aether-colony.md"))
	if err != nil {
		t.Fatalf("read .aether/rules/aether-colony.md: %v", err)
	}

	if string(claudeCopy) != string(aetherCopy) {
		t.Error(".claude/rules/aether-colony.md and .aether/rules/aether-colony.md have diverged. " +
			"The second is the copy published to the hub and read by other repositories, so an edit to one must be made to both.")
	}
}
