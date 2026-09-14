package cmd

// Phase 204 Plan 11, Task 1: the guard that every claim this phase's own
// CLAUDE.md section makes about runtime behaviour names a test that can
// fail, and that the section itself follows the file's own plain-English
// mandate. Mirrors cmd/claudemd_biological_runtime_test.go's established
// TestCLAUDEMDBiologicalRuntimeClaimsCiteLiveTests /
// TestCLAUDEMDBiologicalRuntimeSectionIsPlainEnglish pattern exactly, scoped
// to the new "Learning Governor (v1.28, Phase 204)" section rather than the
// whole (much larger, partly pre-existing) document.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const claudeMDPathForLearningGovernor = "../CLAUDE.md"

// learningGovernorSectionRe extracts the "## Learning Governor (v1.28,
// Phase 204)" section's own body, up to (but not including) the next
// top-level "## " heading. Scoping to this section -- rather than scanning
// the whole CLAUDE.md file -- is deliberate: this test's job is to prove
// THIS phase's new claims are locked and plain-English, not to re-litigate
// every pre-existing section of a much larger document this plan did not
// touch.
var learningGovernorSectionRe = regexp.MustCompile(`(?s)## Learning Governor \(v1\.28, Phase 204\)\n(.*?)\n## `)

func readLearningGovernorSection(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(claudeMDPathForLearningGovernor)
	if err != nil {
		t.Fatalf("read %s: %v", claudeMDPathForLearningGovernor, err)
	}
	m := learningGovernorSectionRe.FindStringSubmatch(string(data))
	if m == nil {
		t.Fatalf("CLAUDE.md has no \"## Learning Governor (v1.28, Phase 204)\" section (or it is not followed by another \"## \" heading) -- this phase's own documentation section is missing")
	}
	return m[1]
}

// TestCLAUDEMDLearningGovernorClaimsCiteLiveTests fails by name for any test
// CLAUDE.md's new Learning Governor section cites that does not exist as a
// real function in package cmd -- a citation is a promise the claim beside
// it can be broken by a failing test, and a citation that resolves to
// nothing makes that promise false. Also fails if the section cites no test
// at all (a doc-only claim about runtime behaviour, which CLAUDE.md's own
// Definition of Done forbids).
func TestCLAUDEMDLearningGovernorClaimsCiteLiveTests(t *testing.T) {
	section := readLearningGovernorSection(t)

	cited := citedTestNameRe.FindAllStringSubmatch(section, -1)
	if len(cited) == 0 {
		t.Fatal("the Learning Governor section cites no test at all -- every claim about runtime behaviour must name a test that can fail")
	}

	entries, err := filepath.Glob("*_test.go")
	if err != nil {
		t.Fatalf("glob package test files: %v", err)
	}
	var sources strings.Builder
	for _, entry := range entries {
		data, err := os.ReadFile(entry)
		if err != nil {
			t.Fatalf("read %s: %v", entry, err)
		}
		sources.Write(data)
		sources.WriteString("\n")
	}
	pkgSources := sources.String()

	seen := map[string]bool{}
	distinct := 0
	for _, m := range cited {
		name := m[1]
		if seen[name] {
			continue
		}
		seen[name] = true
		distinct++
		if !strings.Contains(pkgSources, "func "+name+"(") {
			t.Errorf("CLAUDE.md's Learning Governor section cites %s, but no such test function exists in package cmd -- the section names a lock that cannot run", name)
		}
	}
	if distinct < 10 {
		t.Errorf("the Learning Governor section cites %d distinct test(s), want at least 10 (one per claim listed in the phase's own plan)", distinct)
	}
}

// learningGovernorInventedWords extends the shared repoInventedWords table
// (cmd/next_action_card_test.go, unmodified) with the one word this phase
// introduces that CLAUDE.md's vocabulary table did not previously carry:
// "canary". A local superset -- rather than editing the shared table -- is
// deliberate: the shared table drives TestNextActionCardSpeaksPlainEnglish
// against rendered screens that never mention "canary", so widening it
// there would add a check with nothing to catch; this local copy keeps the
// new word's detection scoped to the section that actually introduces it,
// while still failing (not passing vacuously) if a later edit drops its
// inline translation.
func learningGovernorInventedWords() map[string][]string {
	extended := make(map[string][]string, len(repoInventedWords)+1)
	for word, cues := range repoInventedWords {
		extended[word] = cues
	}
	extended["canary"] = []string{"trial", "reversible", "watched", "small"}
	return extended
}

// untranslatedLearningGovernorWords mirrors untranslatedRepoWords's exact
// per-sentence translation rule (cmd/next_action_card_test.go), applied
// against learningGovernorInventedWords's extended vocabulary instead of the
// shared repoInventedWords table directly -- one shared rule, one extended
// table, never a second independently-invented check.
func untranslatedLearningGovernorWords(text string) []string {
	var violations []string
	words := learningGovernorInventedWords()
	for _, sentence := range regexp.MustCompile(`[.!?\n]`).Split(text, -1) {
		stripped := strings.ToLower(codeSpanRe.ReplaceAllString(sentence, " "))
		if strings.TrimSpace(stripped) == "" {
			continue
		}
		for word, cues := range words {
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

// TestCLAUDEMDLearningGovernorSectionIsPlainEnglish reuses the exact
// vocabulary table and per-sentence translation rule
// TestVoicedScreensSpeakPlainEnglish and
// TestCLAUDEMDBiologicalRuntimeSectionIsPlainEnglish already apply,
// extended with the one new word this phase introduces
// (learningGovernorInventedWords), applied here to CLAUDE.md's new section
// text instead of a rendered screen -- one shared base vocabulary, never a
// second one this section could quietly disagree with.
func TestCLAUDEMDLearningGovernorSectionIsPlainEnglish(t *testing.T) {
	section := readLearningGovernorSection(t)
	if violations := untranslatedLearningGovernorWords(section); len(violations) > 0 {
		t.Errorf("CLAUDE.md's Learning Governor section uses a word this repository invented without explaining it in the same sentence:\n  %s",
			strings.Join(violations, "\n  "))
	}
}
