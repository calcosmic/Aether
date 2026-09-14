package cmd

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/events"
	"github.com/calcosmic/Aether/pkg/shadow"
)

// ---------------------------------------------------------------------
// 204-13 (SC3a/SC3b, D-10), Task 3: a closed intervention-kind vocabulary,
// with real production writers.
//
// Every completeness check in this file derives its expected membership
// from the source itself (an AST walk over episode_ledger.go's own const
// block, and over the three declared writer files' own call sites) --
// never a hand-typed mirror list -- so a declared kind with no vocabulary
// entry, or a vocabulary entry with no real writer, is caught structurally,
// the same discipline cmd/episode_ledger_test.go's TestEpisodeLedgerHasOne
// Writer already applies to this file's sibling ledger writer invariant.
// ---------------------------------------------------------------------

// interventionKindConstDecl is one declared episodeInterventionKind const:
// its Go identifier name and its string literal value.
type interventionKindConstDecl struct {
	Name  string
	Value string
}

// interventionKindConstsInFile walks file's top-level const blocks and
// returns every constant explicitly typed episodeInterventionKind, in
// source order.
func interventionKindConstsInFile(file *ast.File) []interventionKindConstDecl {
	var decls []interventionKindConstDecl
	for _, decl := range file.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok || genDecl.Tok != token.CONST {
			continue
		}
		for _, spec := range genDecl.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			typeIdent, ok := valueSpec.Type.(*ast.Ident)
			if !ok || typeIdent.Name != "episodeInterventionKind" {
				continue
			}
			for i, nameIdent := range valueSpec.Names {
				if i >= len(valueSpec.Values) {
					continue
				}
				lit, ok := valueSpec.Values[i].(*ast.BasicLit)
				if !ok || lit.Kind != token.STRING {
					continue
				}
				unquoted, err := strconv.Unquote(lit.Value)
				if err != nil {
					continue
				}
				decls = append(decls, interventionKindConstDecl{Name: nameIdent.Name, Value: unquoted})
			}
		}
	}
	return decls
}

// parseCmdSourceFile parses filename (relative to the cmd package
// directory, exactly like cmd/episode_ledger_test.go's own scanners) with
// t.Fatalf on any parse error.
func parseCmdSourceFile(t *testing.T, filename string) *ast.File {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", filename, err)
	}
	return file
}

// TestInterventionKindVocabularyIsClosed derives the expected vocabulary
// membership from episode_ledger.go's own const block via an AST walk
// (never a hand-typed mirror list, unlike TestGuidanceStateVocabularyIs
// Closed's weaker shape) and asserts episodeInterventionKindVocabulary
// declares exactly those members, in agreement with
// episodeInterventionKindNames().
func TestInterventionKindVocabularyIsClosed(t *testing.T) {
	declared := interventionKindConstsInFile(parseCmdSourceFile(t, "episode_ledger.go"))
	if len(declared) == 0 {
		t.Fatal("fixture is broken: no episodeInterventionKind const declared in episode_ledger.go")
	}

	if len(declared) != len(episodeInterventionKindVocabulary) {
		t.Fatalf("source declares %d episodeInterventionKind const(s) but episodeInterventionKindVocabulary has %d -- keep them in sync",
			len(declared), len(episodeInterventionKindVocabulary))
	}
	vocabSet := map[string]bool{}
	for _, k := range episodeInterventionKindVocabulary {
		vocabSet[string(k)] = true
	}
	for _, d := range declared {
		if !vocabSet[d.Value] {
			t.Errorf("const %s (%q) declared in source is missing from episodeInterventionKindVocabulary", d.Name, d.Value)
		}
	}
	if names := episodeInterventionKindNames(); len(names) != len(episodeInterventionKindVocabulary) {
		t.Fatalf("episodeInterventionKindNames() returned %d names, want %d", len(names), len(episodeInterventionKindVocabulary))
	}

	const undeclared episodeInterventionKind = "not-a-real-intervention-kind"
	if episodeInterventionKindDeclared(undeclared) {
		t.Errorf("episodeInterventionKindDeclared(%q) = true, want false", undeclared)
	}

	t.Run("an extra declared constant missing from the vocabulary slice is reported by name", func(t *testing.T) {
		fixtureSrc := `package cmd

const extraUndeclaredInterventionKind episodeInterventionKind = "a phantom intervention kind"
`
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "fixture_extra_intervention_kind.go", fixtureSrc, 0)
		if err != nil {
			t.Fatalf("parse fixture source: %v", err)
		}
		extraDecls := interventionKindConstsInFile(file)
		if len(extraDecls) != 1 || extraDecls[0].Name != "extraUndeclaredInterventionKind" || extraDecls[0].Value != "a phantom intervention kind" {
			t.Fatalf("scanner failed to detect the synthetic extra constant: %+v", extraDecls)
		}
		if vocabSet[extraDecls[0].Value] {
			t.Fatal("fixture is broken: synthetic value collides with a real declared vocabulary member")
		}
		// The real check (mirrored here against the synthetic decl) reports
		// this exact case by name: a source-declared const absent from the
		// vocabulary slice.
		reported := false
		for _, d := range extraDecls {
			if !vocabSet[d.Value] {
				reported = true
			}
		}
		if !reported {
			t.Fatal("synthetic extra constant was not reported as missing from the vocabulary")
		}
	})
}

// interventionKindIdentifiersPassedToWriter walks file for every call whose
// function identifier is emitColonyLiveInterventionRecorded and returns the
// set of Go identifier names passed as its third (intervention kind)
// argument.
func interventionKindIdentifiersPassedToWriter(file *ast.File) map[string]bool {
	found := map[string]bool{}
	ast.Inspect(file, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if !ok {
			return true
		}
		ident, ok := call.Fun.(*ast.Ident)
		if !ok || ident.Name != "emitColonyLiveInterventionRecorded" {
			return true
		}
		if len(call.Args) < 3 {
			return true
		}
		argIdent, ok := call.Args[2].(*ast.Ident)
		if !ok {
			return true
		}
		found[argIdent.Name] = true
		return true
	})
	return found
}

// interventionKindWriterSourceFiles are the production files this plan's
// own action text names as writers -- TestEveryInterventionKindHasALive
// Writer scans exactly these for a real call site.
var interventionKindWriterSourceFiles = []string{
	"handoff_decisions_cmd.go",
	"forced_reviewer_waiver.go",
	"rollback.go",
}

// TestEveryInterventionKindHasALiveWriter derives both sides from source:
// the declared vocabulary (episode_ledger.go's own const block) and the
// referenced kinds (every emitColonyLiveInterventionRecorded call site
// across interventionKindWriterSourceFiles) -- never a hand-typed
// "which kinds are covered" map -- and fails by name for a declared kind
// with no real call site anywhere.
func TestEveryInterventionKindHasALiveWriter(t *testing.T) {
	declared := interventionKindConstsInFile(parseCmdSourceFile(t, "episode_ledger.go"))
	if len(declared) == 0 {
		t.Fatal("fixture is broken: no episodeInterventionKind const declared in episode_ledger.go")
	}

	referenced := map[string]bool{}
	for _, f := range interventionKindWriterSourceFiles {
		for name := range interventionKindIdentifiersPassedToWriter(parseCmdSourceFile(t, f)) {
			referenced[name] = true
		}
	}
	if len(referenced) == 0 {
		t.Fatal("fixture is broken: no emitColonyLiveInterventionRecorded call site was found in any declared writer file")
	}

	for _, d := range declared {
		if !referenced[d.Name] {
			t.Errorf("declared intervention kind %s (%q) has no call to emitColonyLiveInterventionRecorded in any of %v", d.Name, d.Value, interventionKindWriterSourceFiles)
		}
	}

	t.Run("a synthetic vocabulary member with no writer is caught", func(t *testing.T) {
		syntheticDecls := append(append([]interventionKindConstDecl{}, declared...),
			interventionKindConstDecl{Name: "episodeInterventionKindPhantomNoWriter", Value: "a phantom kind with no writer"})
		var missing []string
		for _, d := range syntheticDecls {
			if !referenced[d.Name] {
				missing = append(missing, d.Name)
			}
		}
		found := false
		for _, name := range missing {
			if name == "episodeInterventionKindPhantomNoWriter" {
				found = true
			}
		}
		if !found {
			t.Fatalf("scanner failed to detect a synthetic vocabulary member with no writer: %v", missing)
		}
	})
}

// TestUndeclaredInterventionKindIsRefused proves emitColonyLiveIntervention
// Recorded refuses an undeclared kind by name and writes NOTHING -- no live
// event, no durable record -- keeping its existing non-blocking contract.
func TestUndeclaredInterventionKindIsRefused(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	emitColonyLiveEpisodeStarted("ep-undeclared-intervention", events.EpisodeKindBuild)
	emitColonyLiveInterventionRecorded("ep-undeclared-intervention", events.EpisodeKindBuild, episodeInterventionKind("not-a-real-kind"))

	records, err := episodeLedgerForEpisode("ep-undeclared-intervention")
	if err != nil {
		t.Fatalf("read episode ledger: %v", err)
	}
	for _, r := range records {
		if r.RecordKind == episodeLedgerRecordKindIntervention {
			t.Fatalf("an undeclared intervention kind was written to the durable ledger: %+v", r)
		}
	}

	liveEvents := liveEventsSince(t)
	for _, e := range liveEvents {
		if e.Topic == events.LiveTopicInterventionRecorded {
			t.Fatalf("an undeclared intervention kind reached the live stream: %+v", e)
		}
	}
}

// TestOwnerAnswerWritesOneInterventionRecord drives recordDecisionAnswer
// (its real entry point) with a genuine, non-seal source, and asserts
// exactly one durable intervention record was written, carrying the
// declared answered-worker-question kind.
func TestOwnerAnswerWritesOneInterventionRecord(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	const phase = 204
	if _, err := recordDecisionAnswer("Should the retry use exponential backoff?", "Yes, cap at 30s.", phase, "worker-handoff"); err != nil {
		t.Fatalf("recordDecisionAnswer: %v", err)
	}

	episodeID, _ := currentLiveRecoveryEpisode(phase)
	assertExactlyOneInterventionRecord(t, episodeID, episodeInterventionKindAnsweredWorkerQuestion)
}

// TestOwnerAnswerToSealConfirmationWritesNoInterventionRecord proves the
// seal-ceremony exclusion (recordDecisionAnswer's own doc comment): a
// seal-sourced answer is not a worker's question and must not become an
// intervention record.
func TestOwnerAnswerToSealConfirmationWritesNoInterventionRecord(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	const phase = 204
	if _, err := recordDecisionAnswer("Finish anyway?", "yes", phase, "seal-confirmation"); err != nil {
		t.Fatalf("recordDecisionAnswer: %v", err)
	}

	episodeID, _ := currentLiveRecoveryEpisode(phase)
	records, err := episodeLedgerForEpisode(episodeID)
	if err != nil {
		t.Fatalf("read episode ledger: %v", err)
	}
	for _, r := range records {
		if r.RecordKind == episodeLedgerRecordKindIntervention {
			t.Fatalf("a seal-confirmation answer wrote an intervention record: %+v", r)
		}
	}
}

// TestDeclinedReviewerWritesOneInterventionRecord drives
// resolveForcedReviewerWaiverPendingDecision (its real entry point, reached
// through a genuine ensureForcedReviewerWaiverPendingDecision-created
// pending row and an active build attempt, exactly like
// TestWaiverControlsBothFinalContinueDispatchLists's own fixture) and
// asserts exactly one durable intervention record was written, carrying
// the declared declined-forced-reviewer kind.
func TestDeclinedReviewerWritesOneInterventionRecord(t *testing.T) {
	saveGlobals(t)
	// buildStartPlanOnly is the variant whose reviewer-window effect is
	// "reopen", not "close" (buildStartRuleFor) -- the real
	// runCodexBuildPlanOnlyWithOptions path this plan's own read_first
	// names (TestWaiverControlsBothFinalContinueDispatchLists's fixture)
	// never closes the forced-reviewer decline window at commit time,
	// unlike the default direct-build variant.
	fixture := commitTestBuildStart(t, testBuildStartOptions{Variant: buildStartPlanOnly, GeneratedAt: time.Now().UTC(), ExecutionOwner: "go-runtime"})
	phaseID := fixture.Phase.ID
	attemptID := fixture.Attempt.ID
	if phaseID == 0 || attemptID == "" {
		t.Fatalf("fixture is broken: phaseID=%d attemptID=%q", phaseID, attemptID)
	}

	const signal = "credentials/auth"
	const plainEnglish = "logins and passwords"
	capability, err := ensureForcedReviewerWaiverPendingDecision(phaseID, signal, plainEnglish, attemptID)
	if err != nil {
		t.Fatalf("ensureForcedReviewerWaiverPendingDecision: %v", err)
	}
	if capability == "" {
		t.Fatal("fixture is broken: no waiver pending-decision row was created")
	}

	question := forcedReviewerWaiverQuestionText(phaseID, signal, plainEnglish)
	if _, found, err := resolveForcedReviewerWaiverPendingDecision(question, "owner already reviewed this manually", phaseID, capability); err != nil || !found {
		t.Fatalf("resolveForcedReviewerWaiverPendingDecision: found=%v err=%v", found, err)
	}

	episodeID, _ := currentLiveRecoveryEpisode(phaseID)
	assertExactlyOneInterventionRecord(t, episodeID, episodeInterventionKindDeclinedForcedReviewer)
}

// TestQuarantineReleaseWritesOneInterventionRecord drives
// releaseCanaryQuarantine (its real entry point) against a genuinely
// quarantined candidate -- via the same startCanary/rollbackCanary fixture
// TestReleaseCanaryQuarantineThroughApprovedPath already uses -- and
// asserts exactly one durable intervention record was written, carrying
// the declared released-quarantine kind. A second release call against the
// same, now-unquarantined candidate must write no additional record (the
// no-op branch is not an intervention).
func TestQuarantineReleaseWritesOneInterventionRecord(t *testing.T) {
	root, scopedFile := rollbackTestSetup(t)
	admission := canaryAdmission{
		CandidateID: "candidate-intervention-release",
		Scope:       canaryScopeRouting,
		Verdict:     shadow.VerdictBeneficial,
		Bound:       time.Now().Add(24 * time.Hour),
	}
	run, err := startCanary(admission, []string{scopedFile})
	if err != nil {
		t.Fatalf("start canary: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, scopedFile), []byte("changed\n"), 0o644); err != nil {
		t.Fatalf("apply candidate change: %v", err)
	}
	if _, _, err := rollbackCanary(run, "regression for intervention-writer test"); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	released, err := releaseCanaryQuarantine(admission.CandidateID, "owner reviewed and cleared it")
	if err != nil {
		t.Fatalf("release quarantine: %v", err)
	}
	if released.Quarantined {
		t.Fatal("expected the quarantine to be cleared")
	}

	episodeID, _ := currentLiveRecoveryEpisode(0)
	assertExactlyOneInterventionRecord(t, episodeID, episodeInterventionKindReleasedQuarantine)

	// A second release of the now-unquarantined candidate is a no-op, not a
	// new intervention -- the count must stay exactly 1.
	if _, err := releaseCanaryQuarantine(admission.CandidateID, "owner reviewed and cleared it"); err != nil {
		t.Fatalf("second release: %v", err)
	}
	assertExactlyOneInterventionRecord(t, episodeID, episodeInterventionKindReleasedQuarantine)
}

// assertExactlyOneInterventionRecord reads episodeID's own ledger records
// and fails unless exactly one intervention_recorded record exists,
// carrying exactly one category equal to wantKind's own string form.
func assertExactlyOneInterventionRecord(t *testing.T, episodeID string, wantKind episodeInterventionKind) {
	t.Helper()
	records, err := episodeLedgerForEpisode(episodeID)
	if err != nil {
		t.Fatalf("read episode ledger for %q: %v", episodeID, err)
	}
	count := 0
	for _, r := range records {
		if r.RecordKind != episodeLedgerRecordKindIntervention {
			continue
		}
		count++
		if len(r.Interventions) != 1 || r.Interventions[0] != string(wantKind) {
			t.Fatalf("intervention record = %+v, want exactly one category %q", r, wantKind)
		}
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 durable intervention record for episode %q, got %d", episodeID, count)
	}
}

// TestIdenticalInterventionsCollapseOnlyAtTheSameInstant is LEARN-02's own
// adjacency edge (204-13-PLAN.md must_haves.truths): two intervention
// records for the same episode with the same declared kind recorded at the
// SAME instant collapse to one durable record (episodeLedgerDigestPayload's
// own doc comment: an intervention record's identity deliberately includes
// its timestamp); the identical pair recorded at two DIFFERENT instants
// stay as two.
func TestIdenticalInterventionsCollapseOnlyAtTheSameInstant(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	const episodeID = "ep-intervention-same-instant"
	const sameInstant = "2026-09-01T01:00:00Z"
	category := string(episodeInterventionKindDeclinedForcedReviewer)

	first, credited, err := recordEpisodeOutcome(episodeLedgerRecord{
		RecordKind:    episodeLedgerRecordKindIntervention,
		EpisodeID:     episodeID,
		StartedAt:     sameInstant,
		Interventions: []string{category},
	})
	if err != nil || !credited {
		t.Fatalf("first intervention at %s: credited=%v err=%v", sameInstant, credited, err)
	}
	second, credited, err := recordEpisodeOutcome(episodeLedgerRecord{
		RecordKind:    episodeLedgerRecordKindIntervention,
		EpisodeID:     episodeID,
		StartedAt:     sameInstant,
		Interventions: []string{category},
	})
	if err != nil {
		t.Fatalf("second intervention at the same instant: %v", err)
	}
	if credited {
		t.Fatal("expected the second intervention at the identical instant to collapse as a replay, not be credited again")
	}
	if second.RecordID != first.RecordID {
		t.Fatalf("two interventions at the identical instant should collapse to one record id, got %q vs %q", first.RecordID, second.RecordID)
	}

	records, err := episodeLedgerForEpisode(episodeID)
	if err != nil {
		t.Fatalf("read episode ledger: %v", err)
	}
	if got := countInterventionRecords(records); got != 1 {
		t.Fatalf("expected exactly 1 durable intervention record after the same-instant replay, got %d", got)
	}

	third, credited, err := recordEpisodeOutcome(episodeLedgerRecord{
		RecordKind:    episodeLedgerRecordKindIntervention,
		EpisodeID:     episodeID,
		StartedAt:     "2026-09-01T02:00:00Z",
		Interventions: []string{category},
	})
	if err != nil {
		t.Fatalf("third intervention at a different instant: %v", err)
	}
	if !credited {
		t.Fatal("expected the third intervention at a genuinely different instant to be credited as a distinct record")
	}
	if third.RecordID == first.RecordID {
		t.Fatalf("interventions at different instants collapsed to one id %q", first.RecordID)
	}

	records, err = episodeLedgerForEpisode(episodeID)
	if err != nil {
		t.Fatalf("read episode ledger: %v", err)
	}
	if got := countInterventionRecords(records); got != 2 {
		t.Fatalf("expected exactly 2 durable intervention records after the genuinely-different-instant write, got %d", got)
	}
}

func countInterventionRecords(records []episodeLedgerRecord) int {
	count := 0
	for _, r := range records {
		if r.RecordKind == episodeLedgerRecordKindIntervention {
			count++
		}
	}
	return count
}

// TestUnrecognizedInterventionCategoryIsExcludedAndReportedByName is Test 8
// of this plan's own behaviour list: collectPreventableInterventions
// excludes an undeclared category rather than silently counting it, the
// episode carrying only that undeclared category falls through to
// buildImprovementReport's existing UnclassifiedEpisodes list (never a new
// combined-score-adjacent field -- TestTwoFiguresAreNeverCombined keeps
// improvementReport's field set closed), and
// collectUnrecognizedInterventionCategories names the excluded category by
// episode and text rather than dropping it silently.
func TestUnrecognizedInterventionCategoryIsExcludedAndReportedByName(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	openAndCloseEpisode(t, "ep-unrecognized-category", "2026-09-01T00:00:00Z", "2026-09-01T00:01:00Z", episodeLedgerRecord{
		TerminalResult: "failed",
	})
	recordIntervention(t, "ep-unrecognized-category", "2026-09-01T00:00:30Z", "an owner action nobody declared")

	records, err := readEpisodeLedger()
	if err != nil {
		t.Fatalf("read ledger: %v", err)
	}

	entries := collectPreventableInterventions(records)
	for _, e := range entries {
		if e.EpisodeID == "ep-unrecognized-category" {
			t.Fatalf("an undeclared category was counted into collectPreventableInterventions: %+v", e)
		}
	}

	unrecognized := collectUnrecognizedInterventionCategories(records)
	found := false
	for _, u := range unrecognized {
		if strings.Contains(u, "ep-unrecognized-category") && strings.Contains(u, "an owner action nobody declared") {
			found = true
		}
	}
	if !found {
		t.Fatalf("collectUnrecognizedInterventionCategories did not name the undeclared category by episode: %v", unrecognized)
	}

	report := buildImprovementReport(records, improvementReportTestSentinels(), reportWindow{})
	unclassified := false
	for _, id := range report.UnclassifiedEpisodes {
		if id == "ep-unrecognized-category" {
			unclassified = true
		}
	}
	if !unclassified {
		t.Fatalf("episode with only an undeclared intervention category was not counted as unclassified: %v", report.UnclassifiedEpisodes)
	}
	if report.PreventableInterventions.Count != 0 {
		t.Fatalf("PreventableInterventions.Count = %d, want 0 -- an undeclared category must never inflate this figure", report.PreventableInterventions.Count)
	}
}
