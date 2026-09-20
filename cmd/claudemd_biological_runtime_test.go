package cmd

// Phase 203 Plan 15, Task 3: the guard that every claim this phase's own
// CLAUDE.md section makes about runtime behaviour names a test that can
// fail, and that the section itself follows the file's own plain-English
// mandate. Mirrors cmd/claudemd_coherent_jobs_test.go's established
// TestCLAUDEMDCitesLiveTestsForCoherentJobClaims pattern, scoped to the new
// "Biological Runtime (v1.28, Phase 203)" section rather than the whole
// (much larger, partly pre-existing) document.

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

const claudeMDPathForBiologicalRuntime = "../CLAUDE.md"

// biologicalRuntimeSectionRe extracts the "## Biological Runtime (v1.28,
// Phase 203)" section's own body, up to (but not including) the next
// top-level "## " heading. Scoping to this section -- rather than scanning
// the whole CLAUDE.md file -- is deliberate: this test's job is to prove
// THIS phase's new claims are locked and plain-English, not to re-litigate
// every pre-existing section of a much larger document this plan did not
// touch.
var biologicalRuntimeSectionRe = regexp.MustCompile(`(?s)## Biological Runtime \(v1\.28, Phase 203\)\n(.*?)\n## `)

func readBiologicalRuntimeSection(t *testing.T) string {
	t.Helper()
	data, err := os.ReadFile(claudeMDPathForBiologicalRuntime)
	if err != nil {
		t.Fatalf("read %s: %v", claudeMDPathForBiologicalRuntime, err)
	}
	m := biologicalRuntimeSectionRe.FindStringSubmatch(string(data))
	if m == nil {
		t.Fatalf("CLAUDE.md has no \"## Biological Runtime (v1.28, Phase 203)\" section (or it is not followed by another \"## \" heading) -- this phase's own documentation section is missing")
	}
	return m[1]
}

// citedTestNameRe finds every backtick-wrapped `TestXxx` identifier in the
// section -- the citation convention CLAUDE.md's other sections (Team
// Check-In, Coherent Jobs, Live Colony) already use.
var citedTestNameRe = regexp.MustCompile("`(Test[A-Za-z0-9_]+)`")

// TestCLAUDEMDBiologicalRuntimeClaimsCiteLiveTests fails by name for any
// test CLAUDE.md's new Biological Runtime section cites that does not exist
// as a real function in package cmd -- a citation is a promise the claim
// beside it can be broken by a failing test, and a citation that resolves to
// nothing makes that promise false. Also fails if the section cites no test
// at all (a doc-only claim about runtime behaviour, which CLAUDE.md's own
// Definition of Done forbids).
func TestCLAUDEMDBiologicalRuntimeClaimsCiteLiveTests(t *testing.T) {
	section := readBiologicalRuntimeSection(t)

	cited := citedTestNameRe.FindAllStringSubmatch(section, -1)
	if len(cited) == 0 {
		t.Fatal("the Biological Runtime section cites no test at all -- every claim about runtime behaviour must name a test that can fail")
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
	for _, m := range cited {
		name := m[1]
		if seen[name] {
			continue
		}
		seen[name] = true
		if !strings.Contains(pkgSources, "func "+name+"(") {
			t.Errorf("CLAUDE.md's Biological Runtime section cites %s, but no such test function exists in package cmd -- the section names a lock that cannot run", name)
		}
	}
}

// TestCLAUDEMDBiologicalRuntimeSectionIsPlainEnglish reuses the exact
// vocabulary table and per-sentence translation rule
// TestVoicedScreensSpeakPlainEnglish already applies to every rendered
// owner-facing screen (cmd/next_action_card_test.go's repoInventedWords /
// untranslatedRepoWords), applied here to CLAUDE.md's new section text
// instead of a rendered screen -- one shared vocabulary table, never a
// second one this section could quietly disagree with.
func TestCLAUDEMDBiologicalRuntimeSectionIsPlainEnglish(t *testing.T) {
	section := readBiologicalRuntimeSection(t)
	if violations := untranslatedRepoWords(section); len(violations) > 0 {
		t.Errorf("CLAUDE.md's Biological Runtime section uses a word this repository invented without explaining it in the same sentence:\n  %s",
			strings.Join(violations, "\n  "))
	}
}
