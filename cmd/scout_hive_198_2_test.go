package cmd

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// Package note (198.2-08, WIRE-02): renderPhaseResearchBrief is a direct
// function call (not a delegate/plan-only manifest), so every test in this
// file calls it (or, for the fallback-template tests, writePhaseResearchArtifacts)
// directly and inspects the returned/written string. Every hive entry seeded
// here goes through hiveStoreCmd.RunE (the runtime's own writer), never a
// hand-typed wisdom.json literal.

// extractSharedLessonsLines198_2 pulls the bullet lines out of the research
// brief's "## Shared Lessons (Cross-Colony Patterns)" section, stripping the
// leading "- " marker, so a test can compare them directly against the
// entries returned by the one selection rule
// (readHiveWisdomEntriesForDomains + buildHiveWisdomLines) rather than doing
// a substring search that could not catch a reordering or a different
// selection.
func extractSharedLessonsLines198_2(brief string) []string {
	const heading = "## Shared Lessons (Cross-Colony Patterns)\n\n"
	idx := strings.Index(brief, heading)
	if idx == -1 {
		return nil
	}
	rest := brief[idx+len(heading):]
	end := strings.Index(rest, "\n\n")
	if end == -1 {
		end = len(rest)
	}
	var lines []string
	for _, raw := range strings.Split(rest[:end], "\n") {
		raw = strings.TrimPrefix(raw, "- ")
		if strings.TrimSpace(raw) == "" {
			continue
		}
		lines = append(lines, raw)
	}
	return lines
}

// seedHiveEntry198_2 stores one hive-wisdom entry through hiveStoreCmd (the
// runtime's own writer), carrying content that satisfies the admissibility
// gate by naming this test file as its checkable reference.
func seedHiveEntry198_2(t *testing.T, sentinel, sourceRepo string) {
	t.Helper()
	content := sentinel + " -- cmd/scout_hive_198_2_test.go"
	if err := hiveStoreCmd.RunE(hiveStoreCmd, []string{content, "general", sourceRepo}); err != nil {
		t.Fatalf("seed hive entry via hiveStoreCmd: %v", err)
	}
}

// TestResearchHelperIsShownTheSharedLessons is the RED/GREEN test for Task 1:
// given an entry in the shared cross-project store, the research brief
// carries that entry's own sentence.
func TestResearchHelperIsShownTheSharedLessons(t *testing.T) {
	saveGlobalsCmd(t)
	_, tmpDir, hubDir := newSeededColony198_2(t)
	t.Setenv("AETHER_HUB_DIR", hubDir)

	const sentinel = "SENTINEL-198-2-08-SHOWN"
	seedHiveEntry198_2(t, sentinel, "test-repo-198-2-08-shown")

	candidate := phaseResearchCandidate{ID: 1, Name: "Shown phase"}
	brief := renderPhaseResearchBrief(tmpDir, "prove the shared lesson reaches the brief", candidate, codexSurveyContext{})

	if !strings.Contains(brief, sentinel) {
		t.Fatalf("research brief does not carry the seeded shared lesson %q:\n%s", sentinel, brief)
	}
}

// TestResearchHelperSeesTheSameSelectionAsEveryoneElse is the equality
// assertion named in the plan's acceptance criteria: it compares the
// research brief's shared-lessons entries against a direct call to the one
// existing selection rule (readHiveWisdomEntriesForDomains(hubDir, 5,
// domains, &fallbacks) + buildHiveWisdomLines) for the same colony, and
// fails if the research brief is given its own limit or its own domain
// source. Proven by hand during authoring: changing the literal 5 in
// resolveResearchSharedLessonsSection to 3 makes this test fail (5 expected
// entries vs. 3 actual), restored afterward.
func TestResearchHelperSeesTheSameSelectionAsEveryoneElse(t *testing.T) {
	saveGlobalsCmd(t)
	_, tmpDir, hubDir := newSeededColony198_2(t)
	t.Setenv("AETHER_HUB_DIR", hubDir)

	// Seed more entries than the top-5 limit so a research-specific limit
	// (e.g. 3) would visibly diverge from the memory pack's own selection.
	for i := 0; i < 7; i++ {
		sentinel := fmt.Sprintf("SENTINEL-198-2-08-SELECTION-%d", i)
		seedHiveEntry198_2(t, sentinel, fmt.Sprintf("test-repo-198-2-08-selection-%d", i))
	}

	hubPath := resolveHubPath()
	var fallbacks []string
	domains := readRegistryDomainsForRepo(hubPath, tmpDir)
	expectedEntries := readHiveWisdomEntriesForDomains(hubPath, 5, domains, &fallbacks)
	expectedLines := buildHiveWisdomLines(expectedEntries)
	if len(expectedLines) != 5 {
		t.Fatalf("fixture broken: expected the memory pack's own selection to return exactly 5 of the 7 seeded entries, got %d", len(expectedLines))
	}

	candidate := phaseResearchCandidate{ID: 1, Name: "Selection parity phase"}
	brief := renderPhaseResearchBrief(tmpDir, "prove selection parity", candidate, codexSurveyContext{})
	actualLines := extractSharedLessonsLines198_2(brief)

	if !reflect.DeepEqual(actualLines, expectedLines) {
		t.Fatalf("research brief's shared-lessons entries differ from the one existing selection rule's own output:\nexpected (memory-pack selection): %#v\nactual (research brief):         %#v", expectedLines, actualLines)
	}
}

// TestSharedLessonsRespectTheCrossProjectSwitch runs with
// AETHER_HIVE_POLICY=off and asserts nothing from the shared store appears
// in the research brief, while the brief still renders (D-16, and parity
// with every other AETHER_HIVE_POLICY-gated read).
func TestSharedLessonsRespectTheCrossProjectSwitch(t *testing.T) {
	saveGlobalsCmd(t)
	_, tmpDir, hubDir := newSeededColony198_2(t)
	t.Setenv("AETHER_HUB_DIR", hubDir)
	t.Setenv("AETHER_HIVE_POLICY", "off")

	const sentinel = "SENTINEL-198-2-08-SWITCH-OFF"
	seedHiveEntry198_2(t, sentinel, "test-repo-198-2-08-switch")

	candidate := phaseResearchCandidate{ID: 1, Name: "Switch-off phase"}
	brief := renderPhaseResearchBrief(tmpDir, "prove the switch is honoured", candidate, codexSurveyContext{})

	if strings.Contains(brief, sentinel) {
		t.Fatalf("AETHER_HIVE_POLICY=off leaked a shared lesson into the research brief:\n%s", brief)
	}
	if strings.Contains(brief, "Shared Lessons") {
		t.Fatalf("brief still renders the shared-lessons heading with the switch off:\n%s", brief)
	}
	if brief == "" {
		t.Fatal("brief must still render with the cross-project switch off")
	}
}
