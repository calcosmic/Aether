package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// claudeMDLiveColonySections scopes this guard to the Phase 202 section this
// plan added -- other CLAUDE.md sections cite their own tests under their own
// guard (claudeMDLearningSections and its TestEveryLearningClaimInCLAUDEMDNamesALiveTest),
// authored for a different owner's edit.
var claudeMDLiveColonySections = []string{
	"## Live Colony, Swarm, and Oracle (v1.28)",
}

// claudeMDSentenceSplitRe splits a section's prose into sentence-ish units on
// a period followed by whitespace, or a newline -- coarse, but enough to
// locate which sentence cited a given test name for error reporting. Markdown
// bullets and bold-lead sentences both split cleanly on this.
var claudeMDSentenceSplitRe = regexp.MustCompile(`(?:\.\s+)|\n`)

// claudeMDMissingCitation names one CLAUDE.md sentence that cites a Go test
// symbol which does not exist in the parsed cmd package.
type claudeMDMissingCitation struct {
	Symbol   string
	Sentence string
}

// findClaudeMDMissingTestCitations is the structural claim-finder: it never
// consults a maintained list of sentences to check. It scans every heading's
// section for the Go-test-name shape (testNameShapeRe, shared with the
// learning-claim guard) wherever it occurs -- inside a sentence, a bullet, or
// a parenthetical -- and treats every such citation as a claim that must
// resolve. A sentence naming zero test symbols is not a candidate claim at
// all under this guard (structurally, "asserts runtime behavior" here means
// "names the test that proves it"); a sentence naming a symbol that is not
// live fails, reported with both the symbol and the citing sentence.
func findClaudeMDMissingTestCitations(text string, headings []string, live map[string]bool) []claudeMDMissingCitation {
	var missing []claudeMDMissingCitation
	for _, heading := range headings {
		section := extractMarkdownSection(text, heading)
		if section == "" {
			continue
		}
		for _, sentence := range claudeMDSentenceSplitRe.Split(section, -1) {
			trimmed := strings.TrimSpace(sentence)
			if trimmed == "" {
				continue
			}
			for _, symbol := range testNameShapeRe.FindAllString(trimmed, -1) {
				if !live[symbol] {
					missing = append(missing, claudeMDMissingCitation{Symbol: symbol, Sentence: trimmed})
				}
			}
		}
	}
	return missing
}

// TestEveryLiveColonyClaimInCLAUDEMDNamesALiveTest is the removal-proof guard
// for the "Live Colony, Swarm, and Oracle" section Task 3 of 202-15 added:
// every behavioral claim in that section names a Go test symbol, and every
// named symbol must actually exist as a live func TestXxx(t *testing.T) under
// cmd/ or pkg/. Renaming or deleting a cited test without updating CLAUDE.md
// to match makes this fail, naming both the missing symbol and the sentence
// that cited it -- it cannot find a section it cannot locate, and it cannot
// pass by checking nothing.
func TestEveryLiveColonyClaimInCLAUDEMDNamesALiveTest(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("findRepoRoot: %v", err)
	}
	claudeMD, err := os.ReadFile(filepath.Join(root, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("read CLAUDE.md: %v", err)
	}
	text := string(claudeMD)

	for _, heading := range claudeMDLiveColonySections {
		if extractMarkdownSection(text, heading) == "" {
			t.Fatalf("expected to find section %q in CLAUDE.md -- this guard cannot check a section it cannot locate", heading)
		}
	}

	live := collectLiveGoTestFuncNames(t, root)
	missing := findClaudeMDMissingTestCitations(text, claudeMDLiveColonySections, live)
	if len(missing) == 0 {
		// Sanity: the scan must have actually found citations to clear, or
		// this guard is checking nothing.
		found := false
		for _, heading := range claudeMDLiveColonySections {
			section := extractMarkdownSection(text, heading)
			if len(testNameShapeRe.FindAllString(section, -1)) > 0 {
				found = true
				break
			}
		}
		if !found {
			t.Fatal("extracted zero cited test names from the scanned CLAUDE.md sections -- this guard is checking nothing")
		}
		return
	}

	sort.Slice(missing, func(i, j int) bool { return missing[i].Symbol < missing[j].Symbol })
	var lines []string
	for _, m := range missing {
		lines = append(lines, m.Symbol+" (cited by: "+m.Sentence+")")
	}
	t.Fatalf("CLAUDE.md's Live Colony section names %d test(s) that do not exist as func %s(t *testing.T) under cmd/ or pkg/:\n%s",
		len(missing), missing[0].Symbol, strings.Join(lines, "\n"))
}

// TestClaudeMDLiveColonyGuardCatchesAMissingSymbol is the negative fixture
// the plan requires: it proves findClaudeMDMissingTestCitations can actually
// fail, by feeding it a synthetic section that cites a test symbol
// guaranteed not to exist and asserting the miss is reported by both symbol
// and citing sentence.
func TestClaudeMDLiveColonyGuardCatchesAMissingSymbol(t *testing.T) {
	const bogusSymbol = "TestThisSymbolDoesNotExistAnywhereInTheRepo"
	fixtureHeading := "## Fixture Section"
	fixtureText := fixtureHeading + "\n\n" +
		"This behavior is proven by `" + bogusSymbol + "`.\n\n" +
		"## Next Section\n\nUnrelated content.\n"

	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("findRepoRoot: %v", err)
	}
	live := collectLiveGoTestFuncNames(t, root)
	if live[bogusSymbol] {
		t.Fatalf("fixture symbol %q unexpectedly resolved -- pick a name guaranteed not to exist", bogusSymbol)
	}

	missing := findClaudeMDMissingTestCitations(fixtureText, []string{fixtureHeading}, live)
	if len(missing) != 1 {
		t.Fatalf("missing citations = %d, want exactly 1 for the fixture", len(missing))
	}
	if missing[0].Symbol != bogusSymbol {
		t.Fatalf("missing symbol = %q, want %q", missing[0].Symbol, bogusSymbol)
	}
	if !strings.Contains(missing[0].Sentence, bogusSymbol) {
		t.Fatalf("missing citation's sentence = %q, want it to contain %q", missing[0].Sentence, bogusSymbol)
	}
}
