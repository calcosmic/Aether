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

// TestLongFilenameTruncationNoticeStaysInsideTheSanitizerBoundary is a
// boundary-case regression test for WR-02/WR-03: a survey report whose
// filename is long enough that surveyDigestPerReportChars (400) plus the
// truncation notice's own length exceeds colony.SanitizeSignalContent's
// 500-char rejection ceiling. Before the fix, the per-report cut sliced
// content to the full 400-char budget and only then appended the notice,
// so the combined string overshot 500 chars and the sanitizer rejected it
// outright -- a report correctly truncated to fit its own per-report
// budget was instead omitted as "content rejected", not truncated. This
// test fails against that code (the report never appears in the digest
// body, and is named a "content rejected" omission instead) and passes
// once the notice's own length is reserved from the per-report budget
// before slicing.
func TestLongFilenameTruncationNoticeStaysInsideTheSanitizerBoundary(t *testing.T) {
	saveGlobalsCmd(t)
	_, root := setupSurveyStalenessTest(t)
	surveyDir := filepath.Join(root, ".aether", "data", "survey")

	// A realistic-length surveyor filename -- well past today's
	// surveyor-nest/-disciplines templates (BLUEPRINT.md, CHAMBERS.md) --
	// but a survey report's filename is worker-authored, not fixed by
	// this codebase, so the digest must not silently drop a report just
	// because its own name happens to be long.
	longName := "SURVEYOR-DISCIPLINES-DETAILED-NAMING-AND-ERROR-HANDLING-CONVENTIONS-REPORT.md"
	writeSurveyReportFile(t, surveyDir, longName, longSurveyBody("Conventions"))

	digest := resolveSurveyDigestSection()
	if digest == "" {
		t.Fatal("expected a non-empty digest")
	}
	if strings.Contains(digest, "content rejected") {
		t.Fatalf("the long-filename report was rejected by the sanitizer instead of being truncated to fit -- the truncation notice pushed the per-report cut over the 500-char sanitizer ceiling:\n%s", digest)
	}
	if !strings.Contains(digest, "### "+longName) {
		t.Fatalf("expected the long-filename report to be included in the digest (truncated, not omitted), got:\n%s", digest)
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

// --- Task 2: the digest reaches all three briefs, from one call site each ---

// blueprintFixtureContentWithSentinel mirrors blueprintFixtureContent but
// substitutes a caller-supplied sentence for the fixed sentinel bullet, so
// TestSurveyorSentenceReachesAllThreeBriefs can prove each brief carries its
// OWN seeded sentence rather than all three coincidentally matching one
// shared constant.
func blueprintFixtureContentWithSentinel(sentinel string) string {
	return "# Blueprint\n\n" +
		"**Survey Date:** 2026-08-30\n\n" +
		"## Pattern Overview\n\n" +
		"**Overall:** Layered MVC pattern with a thin HTTP layer over domain services.\n\n" +
		"**Key Characteristics:**\n" +
		"- " + sentinel + "\n" +
		"- Handlers stay stateless and delegate all business logic to services under pkg/service.\n\n" +
		"## Layers\n\n" +
		"HTTP Layer -- purpose: parses requests and renders responses.\n"
}

// runBuildBriefCase198_2_06 renders a real build worker brief the same way
// renderCodexBuildWorkerBrief's own proportion test does (TestBuildWorkerBriefIsMostlyTask),
// so this table exercises the exact function composeBuildManifestBrief calls.
func runBuildBriefCase198_2_06(t *testing.T, root string) string {
	t.Helper()
	phase := colony.Phase{
		ID:              1,
		Name:            "Wire the exporter",
		Description:     "Connect the vault exporter to the dashboard command",
		SuccessCriteria: []string{"Dashboard renders exporter output"},
	}
	dispatch := codexBuildDispatch{
		Name:  "Hammer-1",
		Caste: "builder",
		Task:  "Add the exporter call and pass its result to the dashboard view",
	}
	return renderCodexBuildWorkerBrief(root, phase, dispatch, time.Now())
}

// runPlanBriefCase198_2_06 renders a real planning worker brief the same way
// dispatchRealPlanningWorkersWithIterationContext does.
func runPlanBriefCase198_2_06(t *testing.T, root string) string {
	t.Helper()
	survey := codexSurveyContext{}
	spec := planningWorkerSpec{
		Caste:     "route_setter",
		AgentFile: "aether-route-setter.toml",
		Task:      "Draft the phase plan",
		Outputs:   []string{"phase-plan.json"},
	}
	return renderPlanningWorkerBrief(root, survey, spec)
}

// runResearchBriefCase198_2_06 renders a real phase-research Scout brief the
// same way phaseResearchDispatches does.
func runResearchBriefCase198_2_06(t *testing.T, root string) string {
	t.Helper()
	survey := codexSurveyContext{}
	candidate := phaseResearchCandidate{ID: 1, Name: "Wire the exporter"}
	return renderPhaseResearchBrief(root, "Ship the exporter feature", candidate, survey)
}

func briefCases198_2_06() []struct {
	Name string
	Run  func(t *testing.T, root string) string
} {
	return []struct {
		Name string
		Run  func(t *testing.T, root string) string
	}{
		{"build", runBuildBriefCase198_2_06},
		{"plan", runPlanBriefCase198_2_06},
		{"research", runResearchBriefCase198_2_06},
	}
}

// TestSurveyorSentenceReachesAllThreeBriefs is the WIRE-03 proof standard:
// one subtest per brief, each seeding a real survey report with its own
// sentence and asserting that exact sentence arrives in the assembled
// prompt -- not just the report's filename.
func TestSurveyorSentenceReachesAllThreeBriefs(t *testing.T) {
	for _, tc := range briefCases198_2_06() {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			saveGlobalsCmd(t)
			_, root := setupSurveyStalenessTest(t)
			sentinel := "SENTINEL-198-2-06-" + strings.ToUpper(tc.Name) + ": the dispatch layer never touches persistence directly."
			surveyDir := filepath.Join(root, ".aether", "data", "survey")
			writeSurveyReportFile(t, surveyDir, "BLUEPRINT.md", blueprintFixtureContentWithSentinel(sentinel))

			brief := tc.Run(t, root)
			if !strings.Contains(brief, sentinel) {
				t.Fatalf("%s brief does not carry the survey helper's own sentence (len=%d)", tc.Name, len(brief))
			}
		})
	}
}

// TestTheAgeLineAppearsOncePerBrief locks the exactly-once rendering
// invariant (D-02 / D-190-05-A style guard): the digest is the age line's
// home, so a brief that also renders resolveSurveySection's own copy (the
// build brief) must not carry the notice twice.
func TestTheAgeLineAppearsOncePerBrief(t *testing.T) {
	const ageMarker = "Territory has never been surveyed"

	for _, tc := range briefCases198_2_06() {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			saveGlobalsCmd(t)
			_, root := setupSurveyStalenessTest(t)
			if err := store.SaveJSON("COLONY_STATE.json", colony.ColonyState{}); err != nil {
				t.Fatalf("save state: %v", err)
			}
			surveyDir := filepath.Join(root, ".aether", "data", "survey")
			writeSurveyReportFile(t, surveyDir, "BLUEPRINT.md", blueprintFixtureContent())

			brief := tc.Run(t, root)
			count := strings.Count(brief, ageMarker)
			if count > 1 {
				t.Fatalf("%s brief renders the age line %d times, expected at most once", tc.Name, count)
			}
			if count == 0 {
				t.Errorf("%s brief does not render the age line at all -- expected exactly once, since the territory has never been surveyed", tc.Name)
			}
		})
	}
}

// TestBriefsAreUnchangedWithoutASurvey proves a colony with no survey
// reports gets a byte-identical brief to before this digest existed: the
// digest injection is guarded by resolveSurveyDigestSection() returning "",
// under the exact same on-disk condition (an empty or absent survey
// directory) resolveSurveySection already no-ops under, so no bytes from
// the new digest block are ever written.
func TestBriefsAreUnchangedWithoutASurvey(t *testing.T) {
	const digestMarker = "Codebase Map Digest"

	for _, tc := range briefCases198_2_06() {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			saveGlobalsCmd(t)
			_, root := setupSurveyStalenessTest(t)
			// No survey directory written -- this is the "never colonized" state.

			if got := resolveSurveyDigestSection(); got != "" {
				t.Fatalf("fixture broken: expected an empty digest with no survey reports, got:\n%s", got)
			}

			first := tc.Run(t, root)
			second := tc.Run(t, root)
			if first != second {
				t.Fatalf("%s brief is not stable across repeated renders with no survey present", tc.Name)
			}
			if strings.Contains(first, digestMarker) {
				t.Errorf("%s brief contains the digest heading with no survey reports on disk -- the digest injection is not byte-identical to before it existed:\n%s", tc.Name, first)
			}
		})
	}
}
