package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// Phase 198.2 Plan 06 (WIRE-03): resolveSurveyDigestSection gives the
// planner, the builder and the research helper the survey reports' actual
// content -- not just the filename list resolveSurveySection already
// carries. These tests seed report files the way a real survey helper
// writes them (per the surveyor-nest/surveyor-disciplines templates in
// colony/prompts/), never as one-line fixtures, so a passing test proves
// the digest works on realistic content, not a shape the runtime could
// never produce.

// writeSurveyReportFile writes a survey report under <root>/.aether/data/survey/,
// creating the directory if needed.
func writeSurveyReportFile(t *testing.T, surveyDir, name, content string) {
	t.Helper()
	if err := os.MkdirAll(surveyDir, 0755); err != nil {
		t.Fatalf("mkdir survey dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(surveyDir, name), []byte(content), 0644); err != nil {
		t.Fatalf("write survey report %s: %v", name, err)
	}
}

// blueprintDigestSentinel is a sentence a surveyor actually wrote, seeded
// into the fixture BLUEPRINT.md below and asserted verbatim in the digest.
const blueprintDigestSentinel = "The dispatch layer never touches persistence directly; every write goes through the service layer first."

// blueprintFixtureContent mirrors the real BLUEPRINT.md shape
// colony/prompts/surveyor-nest.md instructs a surveyor to write: a title, a
// survey date, a Pattern Overview with an Overall line and Key
// Characteristics bullets, then a Layers section. No backticks or XML-style
// angle brackets -- this is prose a colony.SanitizeSignalContent call must
// accept, matching what a real surveyor writes for the parts that matter
// most (the opening summary), even though later, file-path-heavy sections
// of a real report may use backticks the digest's per-report cap generally
// truncates away before reaching them.
func blueprintFixtureContent() string {
	return "# Blueprint\n\n" +
		"**Survey Date:** 2026-08-30\n\n" +
		"## Pattern Overview\n\n" +
		"**Overall:** Layered MVC pattern with a thin HTTP layer over domain services.\n\n" +
		"**Key Characteristics:**\n" +
		"- " + blueprintDigestSentinel + "\n" +
		"- Handlers stay stateless and delegate all business logic to services under pkg/service.\n" +
		"- Configuration loads once at startup and is passed down as an explicit dependency, never read from a global mid-request.\n\n" +
		"## Layers\n\n" +
		"HTTP Layer -- purpose: parses requests and renders responses. Location: cmd/handlers. Depends on the service layer. Used by the router.\n"
}

const chambersDigestSentinel = "Config files live under config and are loaded once at process start, never re-read mid-request."

// chambersFixtureContent mirrors the real CHAMBERS.md shape (Directory
// Layout, Directory Purposes) surveyor-nest also writes.
func chambersFixtureContent() string {
	return "# Chambers\n\n" +
		"**Survey Date:** 2026-08-30\n\n" +
		"## Directory Layout\n\n" +
		chambersDigestSentinel + "\n\n" +
		"## Directory Purposes\n\n" +
		"- cmd holds entry points and command wiring.\n" +
		"- pkg holds shared packages consumed by more than one command.\n" +
		"- internal holds package-private helpers not meant for reuse.\n"
}

// longSurveyBody returns a realistic-length prose report (well over
// surveyDigestPerReportChars) so a budget test can exercise truncation and
// omission without depending on brittle exact-byte arithmetic.
func longSurveyBody(label string) string {
	return fmt.Sprintf("# %s Report\n\n**Survey Date:** 2026-08-30\n\n## Overview\n\n"+
		"This report set out to characterise how %s is organised, tracing responsibility "+
		"boundaries across the module and noting where behaviour diverges from the rest of "+
		"the codebase. The investigation covered directory layout, naming conventions, and "+
		"the handful of places where configuration values cross a trust boundary before "+
		"reaching a downstream caller. Every finding below is anchored to a real location in "+
		"the tree rather than a general impression, and each one is written so a later reader "+
		"can verify it independently without repeating the same exploration from scratch.\n\n"+
		"## Key Findings\n\n"+
		"- %s keeps its public surface small and deliberately undocumented internals large.\n"+
		"- Naming follows the rest of the repository's snake_case convention throughout.\n",
		label, label, label)
}

func TestMapDigestCarriesASurveyorsOwnSentence(t *testing.T) {
	saveGlobalsCmd(t)
	_, root := setupSurveyStalenessTest(t)
	surveyDir := filepath.Join(root, ".aether", "data", "survey")
	writeSurveyReportFile(t, surveyDir, "BLUEPRINT.md", blueprintFixtureContent())

	digest := resolveSurveyDigestSection()
	if digest == "" {
		t.Fatal("expected a non-empty digest")
	}
	if !strings.Contains(digest, blueprintDigestSentinel) {
		t.Fatalf("digest does not carry the surveyor's own sentence:\n%s", digest)
	}
}

func TestDeletingASurveyReportChangesTheDigest(t *testing.T) {
	saveGlobalsCmd(t)
	_, root := setupSurveyStalenessTest(t)
	surveyDir := filepath.Join(root, ".aether", "data", "survey")
	writeSurveyReportFile(t, surveyDir, "BLUEPRINT.md", blueprintFixtureContent())
	writeSurveyReportFile(t, surveyDir, "CHAMBERS.md", chambersFixtureContent())

	before := resolveSurveyDigestSection()
	if before == "" {
		t.Fatal("expected a non-empty digest before deletion")
	}
	if !strings.Contains(before, "### CHAMBERS.md") {
		t.Fatalf("fixture broken: CHAMBERS.md never made it into the digest before deletion:\n%s", before)
	}

	if err := os.Remove(filepath.Join(surveyDir, "CHAMBERS.md")); err != nil {
		t.Fatalf("remove report: %v", err)
	}

	after := resolveSurveyDigestSection()
	if after == before {
		t.Fatal("digest did not change after deleting a survey report -- it may be built from a repository scan rather than the report files themselves")
	}
	if strings.Contains(after, "### CHAMBERS.md") {
		t.Errorf("deleted report still appears in the digest:\n%s", after)
	}
	if !strings.Contains(after, blueprintDigestSentinel) {
		t.Errorf("the surviving report's content should still be present:\n%s", after)
	}
}

func TestMapDigestStaysInsideItsOwnBudgetAndNamesOmissions(t *testing.T) {
	saveGlobalsCmd(t)
	_, root := setupSurveyStalenessTest(t)
	surveyDir := filepath.Join(root, ".aether", "data", "survey")

	names := []string{"SURVEY-1.md", "SURVEY-2.md", "SURVEY-3.md", "SURVEY-4.md", "SURVEY-5.md", "SURVEY-6.md", "SURVEY-7.md"}
	for _, name := range names {
		writeSurveyReportFile(t, surveyDir, name, longSurveyBody(name))
	}

	digest := resolveSurveyDigestSection()
	if digest == "" {
		t.Fatal("expected a non-empty digest")
	}

	included := 0
	for _, name := range names {
		if strings.Contains(digest, "### "+name) {
			included++
		}
	}
	if included == 0 {
		t.Fatal("expected at least one report body in the digest")
	}
	if included == len(names) {
		t.Fatalf("expected some reports to be omitted for budget with %d %d-char reports against a %d-char budget, but all %d were included (digest length %d)",
			len(names), surveyDigestPerReportChars, surveyDigestBudgetChars, len(names), len(digest))
	}

	if !strings.Contains(digest, "Not included here") {
		t.Fatalf("digest does not name what it omitted:\n%s", digest)
	}
	omittedCount := 0
	for _, name := range names {
		if !strings.Contains(digest, "### "+name) {
			if !strings.Contains(digest, name) {
				t.Errorf("report %s was excluded but never named anywhere in the digest -- silently dropped", name)
			}
			omittedCount++
		}
	}
	if omittedCount == 0 {
		t.Fatal("expected at least one omitted report")
	}

	// Loose sanity ceiling: catches a genuine "budget does nothing" bug
	// (unbounded growth) without pinning the test to the exact per-report
	// truncation-suffix arithmetic resolveColonyResearchSection's shape
	// already accepts as normal overshoot.
	if len(digest) > surveyDigestBudgetChars*2 {
		t.Errorf("digest grew to %d chars against a %d-char budget -- omission budgeting is not bounding growth", len(digest), surveyDigestBudgetChars)
	}
}

func TestOldMapIsStillSentWithItsAgeLine(t *testing.T) {
	saveGlobalsCmd(t)
	_, root := setupSurveyStalenessTest(t)
	gitInitSurveyTestRepo(t, root)

	surveyedAt := time.Now().UTC().Add(-90 * 24 * time.Hour).Format(time.RFC3339)
	if err := store.SaveJSON("COLONY_STATE.json", colony.ColonyState{TerritorySurveyed: &surveyedAt}); err != nil {
		t.Fatalf("save state: %v", err)
	}
	gitCommitsSurveyTestRepo(t, root, surveyStaleCommitThreshold+5)

	surveyDir := filepath.Join(root, ".aether", "data", "survey")
	writeSurveyReportFile(t, surveyDir, "BLUEPRINT.md", blueprintFixtureContent())

	digest := resolveSurveyDigestSection()
	if digest == "" {
		t.Fatal("expected a non-empty digest even for a very old map -- it must still be sent, never silently dropped")
	}
	if !strings.Contains(digest, "STALE MAP WARNING") {
		t.Errorf("expected the age line to head the digest for a stale map:\n%s", digest)
	}
	if !strings.HasPrefix(digest, "### Codebase Map Digest") {
		t.Errorf("expected the digest to open with its own heading before the age line:\n%s", digest)
	}
	if !strings.Contains(digest, blueprintDigestSentinel) {
		t.Errorf("a stale map must still carry its content, not just the age line:\n%s", digest)
	}
}

func TestMapDigestIsDeterministic(t *testing.T) {
	saveGlobalsCmd(t)
	_, root := setupSurveyStalenessTest(t)
	surveyDir := filepath.Join(root, ".aether", "data", "survey")
	writeSurveyReportFile(t, surveyDir, "BLUEPRINT.md", blueprintFixtureContent())
	writeSurveyReportFile(t, surveyDir, "CHAMBERS.md", chambersFixtureContent())

	first := resolveSurveyDigestSection()
	second := resolveSurveyDigestSection()
	if first == "" {
		t.Fatal("expected a non-empty digest")
	}
	if first != second {
		t.Fatalf("digest is not byte-identical across repeated calls against unchanged files:\nfirst:\n%s\nsecond:\n%s", first, second)
	}
}

func TestNoSurveyMeansNoDigest(t *testing.T) {
	saveGlobalsCmd(t)
	setupSurveyStalenessTest(t)

	if got := resolveSurveyDigestSection(); got != "" {
		t.Fatalf("expected an empty digest with no survey reports on disk, got:\n%s", got)
	}
}
