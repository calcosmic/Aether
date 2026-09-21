package cmd

// Phase 206 (UED-05), Task 3 -- the shipped CLAUDE.md claims about the
// direct screen route must each name a live test, and this guard fails if a
// named test stops existing. Modelled directly on
// TestEveryVoiceClaimInCLAUDEMDNamesALiveTest (cmd/claudemd_classic_voice_test.go):
// the same structural claim-finder (findClaudeMDMissingTestCitations), the
// same live-test collector (collectLiveGoTestFuncNames), the same
// section-extraction and test-name-shape machinery -- scoped here to the
// "The screen reaches the owner directly (v1.29, Phase 206)" section this
// plan added to CLAUDE.md.

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// claudeMDDirectScreenSections scopes this guard to the section this plan
// added -- other CLAUDE.md sections cite their own tests under their own
// guard, authored for a different owner's edit.
var claudeMDDirectScreenSections = []string{
	"### The screen reaches the owner directly (v1.29, Phase 206)",
}

// TestEveryDirectScreenClaimInCLAUDEMDNamesALiveTest is the removal-proof
// guard: every behavioral claim in the section names a Go test symbol, and
// every named symbol must actually exist as a live func TestXxx(t
// *testing.T) under cmd/ or pkg/. Renaming or deleting a cited test without
// updating CLAUDE.md to match makes this fail, naming both the missing
// symbol and the sentence that cited it.
func TestEveryDirectScreenClaimInCLAUDEMDNamesALiveTest(t *testing.T) {
	root, err := findRepoRoot()
	if err != nil {
		t.Fatalf("findRepoRoot: %v", err)
	}
	claudeMD, err := os.ReadFile(filepath.Join(root, "CLAUDE.md"))
	if err != nil {
		t.Fatalf("read CLAUDE.md: %v", err)
	}
	text := string(claudeMD)

	for _, heading := range claudeMDDirectScreenSections {
		if extractMarkdownSection(text, heading) == "" {
			t.Fatalf("expected to find section %q in CLAUDE.md -- this guard cannot check a section it cannot locate", heading)
		}
	}

	live := collectLiveGoTestFuncNames(t, root)
	missing := findClaudeMDMissingTestCitations(text, claudeMDDirectScreenSections, live)
	if len(missing) > 0 {
		sort.Slice(missing, func(i, j int) bool { return missing[i].Symbol < missing[j].Symbol })
		var lines []string
		for _, m := range missing {
			lines = append(lines, m.Symbol+" (cited by: "+m.Sentence+")")
		}
		t.Fatalf("CLAUDE.md's direct-screen section names %d test(s) that do not exist as func %s(t *testing.T) under cmd/ or pkg/:\n%s",
			len(missing), missing[0].Symbol, strings.Join(lines, "\n"))
	}

	// Sanity: the scan must have actually found citations to clear, and it
	// must find at least every test this plan's own artifacts list names, or
	// this guard is checking too little.
	var citedInSection []string
	for _, heading := range claudeMDDirectScreenSections {
		section := extractMarkdownSection(text, heading)
		citedInSection = append(citedInSection, testNameShapeRe.FindAllString(section, -1)...)
	}
	if len(citedInSection) == 0 {
		t.Fatal("extracted zero cited test names from the scanned CLAUDE.md section -- this guard is checking nothing")
	}
	wantAtLeast := []string{
		"TestDirectRouteHandsTheScreenToTheOwner",
		"TestDirectRouteStaysQuietWhenThereIsNoScreen",
		"TestDirectRouteNeverCutsALineInHalf",
		"TestDirectRouteMessageIsValidJSONForEveryScreen",
		"TestDirectRouteIsSkippedForHelpersAndWorkers",
		"TestDirectRouteOffSwitchStopsIt",
		"TestDirectRouteWritesNothing",
		"TestDirectRouteHookIsRegistered",
		"TestDirectRouteInstallsWithoutDisturbingOtherHooks",
		"TestDirectRouteAndTheBackstopUseOneDecision",
		"TestBackstopStaysQuietWhenTheScreenAlreadyArrived",
		"TestBackstopStillFiresWhenOnlyPartOfTheScreenArrived",
	}
	cited := make(map[string]bool, len(citedInSection))
	for _, name := range citedInSection {
		cited[name] = true
	}
	for _, want := range wantAtLeast {
		if !cited[want] {
			t.Errorf("CLAUDE.md's direct-screen section does not cite %s, which this plan's own artifacts require it to name", want)
		}
	}

	// The guard must be provably able to fail, not merely pass by
	// construction: feed it a synthetic section citing a test symbol
	// guaranteed not to exist, and assert the miss is reported by both
	// symbol and citing sentence.
	t.Run("the guard can fail on a fabricated test name", func(t *testing.T) {
		const bogusSymbol = "TestThisDirectScreenSymbolDoesNotExistAnywhereInTheRepo"
		fixtureHeading := "## Fixture Direct Screen Section"
		fixtureText := fixtureHeading + "\n\n" +
			"This behavior is proven by `" + bogusSymbol + "`.\n\n" +
			"## Next Section\n\nUnrelated content.\n"

		if live[bogusSymbol] {
			t.Fatalf("fixture symbol %q unexpectedly resolved -- pick a name guaranteed not to exist", bogusSymbol)
		}
		fixtureMissing := findClaudeMDMissingTestCitations(fixtureText, []string{fixtureHeading}, live)
		if len(fixtureMissing) != 1 {
			t.Fatalf("missing citations = %d, want exactly 1 for the fixture", len(fixtureMissing))
		}
		if fixtureMissing[0].Symbol != bogusSymbol {
			t.Fatalf("missing symbol = %q, want %q", fixtureMissing[0].Symbol, bogusSymbol)
		}
		if !strings.Contains(fixtureMissing[0].Sentence, bogusSymbol) {
			t.Fatalf("missing citation's sentence = %q, want it to contain %q", fixtureMissing[0].Sentence, bogusSymbol)
		}
	})
}
