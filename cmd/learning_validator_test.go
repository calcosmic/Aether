package cmd

// 204-16-PLAN.md, Task 2 / WINDOWS.md entry 44 (REOPENED by 204-REVIEW.md
// CR-01, 2026-09-15). Every test below writes its guidance-application
// records through the real production writer (mustRecordGuidanceState,
// declared in cmd/application_evidence_test.go, same package) rather than a
// hand-typed credit/records.json literal.
//
// IMPORTANT, and the reason this file changed shape at CR-01: the earlier
// version of this file additionally used a learn.Entry's OWN identifier as
// the guidanceID argument to mustRecordGuidanceState, to construct a
// "helpful application for this hypothesis" fixture. No production writer
// ever produces that pairing -- recordGuidanceApplicationState's only
// non-test callers key by an Instinct's ID, never a learn.Entry's ID (see
// cmd/learning_validator.go's package doc comment for the full,
// grep-confirmed account) -- so that construction was a false certificate:
// a green test proving a scenario the real runtime cannot produce. Test 2,
// 3, 4, 5 and 7 below still exercise promoteHelpfulHypotheses' internal
// gating logic (does it correctly refuse a non-helpful, uncorroborated, or
// disproven record) using that same hypothetical same-ID shape, because
// that gating logic must stay correct for the day a real identifier bridge
// is built -- but each is now labelled as exactly that: an internal-logic
// regression test, not a production-reachability proof. The
// production-reachability question itself is answered honestly by
// TestHypothesisPromotionNeverCrossesTheIdentifierGap below, which never
// lets a learn.Entry's ID double as a guidanceID.
import (
	"path/filepath"
	"testing"

	"github.com/calcosmic/Aether/pkg/learn"
)

// seedHypothesisEntry writes one learning entry, in hypothesis status,
// under an explicitly chosen id -- through the real store writer
// (learn.NewColonyStore(store).Add) -- so this test file can look up
// guidance application records against that same id afterward.
// learn.ColonyStore.Add only auto-generates an id when Entry.ID is empty;
// setting it here sidesteps Add's own value-receiver id-return limitation
// (Add's local mutation of its own entry.ID parameter never reaches the
// caller) without touching that pre-existing, out-of-scope behavior.
func seedHypothesisEntry(t *testing.T, id, content string, phase int) {
	t.Helper()
	entry := learn.Entry{
		ID:             id,
		Content:        content,
		Phase:          phase,
		Confidence:     0.6,
		Classification: learn.ClassRepoLocal,
		Status:         learn.StatusHypothesis,
	}
	if err := learn.NewColonyStore(store).Add(entry); err != nil {
		t.Fatalf("seed hypothesis entry %q: %v", id, err)
	}
}

// getLearnEntry re-reads id's own stored entry through the real store
// reader.
func getLearnEntry(t *testing.T, id string) learn.Entry {
	t.Helper()
	entry, err := learn.NewColonyStore(store).Get(id)
	if err != nil {
		t.Fatalf("get entry %q: %v", id, err)
	}
	if entry == nil {
		t.Fatalf("entry %q not found", id)
	}
	return *entry
}

// TestHypothesisPromotionNeverCrossesTheIdentifierGap is CR-01's honest
// replacement for the plan's original Test 1. It proves the CURRENT
// production behavior (WINDOWS.md entry 44, reopened): a genuinely helpful
// guidance-application record, recorded through the real production writer
// under an Instinct-shaped identifier exactly the way
// cmd/instinct_application.go's real callers do, can never promote an
// unrelated hypothesis -- because no production code path ever gives a
// learn.Entry and a GuidanceApplications record the same identifier.
//
// This test would start FAILING the moment a real identifier bridge is
// built (CR-01's option (a) or (b)) and promotion begins succeeding for a
// case like this -- that is the intended, desired failure. Whoever builds
// that bridge should replace this test with an honest positive one driven
// through the same real production chain, not loosen this one.
func TestHypothesisPromotionNeverCrossesTheIdentifierGap(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	// A genuinely helpful guidance-application record, under an
	// Instinct-shaped id, recorded through the exact same real writer
	// (recordGuidanceApplicationState) instinct_application.go's
	// production callers use -- never a learn.Entry's own id.
	const unrelatedInstinctID = "inst_unrelated_for_gap_test"
	mustRecordGuidanceState(t, unrelatedInstinctID, 1, guidanceApplicationStateAvailable)
	mustRecordGuidanceState(t, unrelatedInstinctID, 1, guidanceApplicationStateRendered)
	mustRecordGuidanceState(t, unrelatedInstinctID, 1, guidanceApplicationStateConsulted)
	mustRecordGuidanceState(t, unrelatedInstinctID, 1, guidanceApplicationStateActedOn)
	mustRecordGuidanceState(t, unrelatedInstinctID, 1, guidanceApplicationStateHelpful)

	if !learningEntryHasHelpfulApplication(unrelatedInstinctID) {
		t.Fatalf("test setup is broken: expected a genuinely helpful guidance record for %q", unrelatedInstinctID)
	}

	// A genuine hypothesis with its OWN, production-assigned identifier
	// (learn.ColonyStore.Add's own generator, never hand-typed to collide
	// with unrelatedInstinctID) -- exactly the shape learning-propose
	// (cmd/learning_cmds.go) actually produces.
	learnStore := learn.NewColonyStore(store)
	if err := learnStore.Add(learn.Entry{
		Content:        "sentinel hypothesis with no identifier connection to any guidance record",
		Phase:          1,
		Confidence:     0.6,
		Classification: learn.ClassRepoLocal,
		Status:         learn.StatusHypothesis,
	}); err != nil {
		t.Fatalf("seed hypothesis entry: %v", err)
	}
	entries, err := learnStore.List(learn.EntryFilter{})
	if err != nil {
		t.Fatalf("list learning entries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly one seeded hypothesis entry, got %d", len(entries))
	}
	hypothesisID := entries[0].ID
	if hypothesisID == "" || hypothesisID == unrelatedInstinctID {
		t.Fatalf("seeded hypothesis id %q is not a distinct, real store-assigned id", hypothesisID)
	}

	summary := promoteHelpfulHypotheses(1)
	if !summary.Ran {
		t.Fatalf("expected Ran == true, got %+v", summary)
	}
	if len(summary.Promoted) != 0 {
		t.Fatalf("expected NO promotion (the identifier spaces never connect in production today), got %+v", summary)
	}

	entry := getLearnEntry(t, hypothesisID)
	if entry.Status != learn.StatusHypothesis {
		t.Fatalf("entry status = %q, want it to remain %q (WINDOWS.md entry 44 is reopened, not fixed)", entry.Status, learn.StatusHypothesis)
	}
	if verified := learningVerifiedEntries([]learn.Entry{entry}); len(verified) != 0 {
		t.Fatalf("expected the hypothesis to stay out of learningVerifiedEntries, got %+v", verified)
	}
}

// TestActedOnIsNotEnoughToValidate is the plan's Test 2: a hypothesis with
// a guidance record only in the rendered, consulted, or acted-on state is
// NOT promoted. INTERNAL-LOGIC TEST ONLY (see this file's header comment):
// it uses the entry's own id as guidanceID, a pairing no production writer
// produces today -- it locks the gating rule's correctness for the day a
// real identifier bridge exists, not a claim this scenario is reachable now.
func TestActedOnIsNotEnoughToValidate(t *testing.T) {
	for _, stopAt := range []guidanceApplicationState{
		guidanceApplicationStateRendered,
		guidanceApplicationStateConsulted,
		guidanceApplicationStateActedOn,
	} {
		stopAt := stopAt
		t.Run(string(stopAt), func(t *testing.T) {
			saveGlobals(t)
			s, _ := newTestStore(t)
			store = s

			id := "lrn_stops_at_" + string(stopAt)
			seedHypothesisEntry(t, id, "sentinel hypothesis not yet helpful", 1)

			chain := []guidanceApplicationState{
				guidanceApplicationStateAvailable,
				guidanceApplicationStateRendered,
				guidanceApplicationStateConsulted,
				guidanceApplicationStateActedOn,
			}
			for _, state := range chain {
				mustRecordGuidanceState(t, id, 1, state)
				if state == stopAt {
					break
				}
			}

			summary := promoteHelpfulHypotheses(1)
			if len(summary.Promoted) != 0 {
				t.Fatalf("expected no promotion stopping at %q, got %+v", stopAt, summary)
			}
			entry := getLearnEntry(t, id)
			if entry.Status != learn.StatusHypothesis {
				t.Fatalf("entry status = %q, want it to remain %q", entry.Status, learn.StatusHypothesis)
			}
		})
	}
}

// TestNeutralOrHarmfulIsNeverValidated is the plan's Test 3: a hypothesis
// with a guidance record in the neutral or harmful state is NOT promoted.
// INTERNAL-LOGIC TEST ONLY (see this file's header comment): no production
// writer records Neutral/Harmful at all today, so this locks the gating
// rule's correctness for whenever both that writer and an identifier
// bridge exist, not a claim this scenario is reachable now.
func TestNeutralOrHarmfulIsNeverValidated(t *testing.T) {
	for _, terminal := range []guidanceApplicationState{
		guidanceApplicationStateNeutral,
		guidanceApplicationStateHarmful,
	} {
		terminal := terminal
		t.Run(string(terminal), func(t *testing.T) {
			saveGlobals(t)
			s, _ := newTestStore(t)
			store = s

			id := "lrn_terminal_" + string(terminal)
			seedHypothesisEntry(t, id, "sentinel hypothesis reaching a non-helpful terminal state", 1)
			mustRecordGuidanceState(t, id, 1, guidanceApplicationStateAvailable)
			mustRecordGuidanceState(t, id, 1, guidanceApplicationStateRendered)
			mustRecordGuidanceState(t, id, 1, guidanceApplicationStateConsulted)
			mustRecordGuidanceState(t, id, 1, guidanceApplicationStateActedOn)
			mustRecordGuidanceState(t, id, 1, terminal)

			summary := promoteHelpfulHypotheses(1)
			if len(summary.Promoted) != 0 {
				t.Fatalf("expected no promotion on terminal state %q, got %+v", terminal, summary)
			}
			entry := getLearnEntry(t, id)
			if entry.Status != learn.StatusHypothesis {
				t.Fatalf("entry status = %q, want it to remain %q", entry.Status, learn.StatusHypothesis)
			}
		})
	}
}

// TestUncorroboratedClaimIsNeverValidated is the plan's Test 4: a
// hypothesis with a worker CLAIM record (guidanceClaimRecord, the
// uncorroborated, never-consulted table) but no corroborated helpful
// application is NOT promoted. INTERNAL-LOGIC TEST ONLY (see this file's
// header comment): it uses the entry's own id as the claim record's
// guidanceID, a pairing no production writer produces today.
func TestUncorroboratedClaimIsNeverValidated(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	const id = "lrn_uncorroborated_claim"
	seedHypothesisEntry(t, id, "sentinel hypothesis with only a worker's own unverified claim", 1)
	if _, _, err := recordGuidanceClaimUnverified(id, 1); err != nil {
		t.Fatalf("record unverified claim: %v", err)
	}

	summary := promoteHelpfulHypotheses(1)
	if len(summary.Promoted) != 0 {
		t.Fatalf("expected no promotion from an uncorroborated claim alone, got %+v", summary)
	}
	entry := getLearnEntry(t, id)
	if entry.Status != learn.StatusHypothesis {
		t.Fatalf("entry status = %q, want it to remain %q", entry.Status, learn.StatusHypothesis)
	}
}

// TestHypothesisWithNoRecordStaysAHypothesis is the plan's Test 5: a
// hypothesis with no guidance record at all is NOT promoted, and the
// verified-memory section a worker reads stays honestly empty.
func TestHypothesisWithNoRecordStaysAHypothesis(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	const id = "lrn_no_record"
	seedHypothesisEntry(t, id, "sentinel hypothesis with no guidance record at all", 1)

	summary := promoteHelpfulHypotheses(1)
	if len(summary.Promoted) != 0 {
		t.Fatalf("expected no promotion with no guidance record, got %+v", summary)
	}
	entry := getLearnEntry(t, id)
	if entry.Status != learn.StatusHypothesis {
		t.Fatalf("entry status = %q, want it to remain %q", entry.Status, learn.StatusHypothesis)
	}
	if verified := learningVerifiedEntries([]learn.Entry{entry}); len(verified) != 0 {
		t.Fatalf("expected the verified-memory section to stay empty, got %+v", verified)
	}
}

// TestPromotionPassIsIdempotentAndLeavesAlreadyValidatedEntriesAlone is
// CR-01's honest replacement for the plan's original Test 6. The original
// asserted "promotes once, then nothing the second time" by fabricating a
// helpful record under the hypothesis's own id -- a scenario the real
// runtime cannot produce (see this file's header comment). The property
// worth locking honestly is narrower but real: (1) an entry the hand-run
// `learning-validate` command (or any future writer) already moved to
// StatusValidated is never re-selected or re-touched by the automatic
// pass, and (2) running the pass repeatedly over the same, currently
// unpromotable hypothesis produces stable, error-free, empty results every
// time -- never a growing Failures list, never a spurious write.
func TestPromotionPassIsIdempotentAndLeavesAlreadyValidatedEntriesAlone(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	learnStore := learn.NewColonyStore(store)

	// Already validated by some other path (e.g. the hand-run
	// learning-validate command) -- must never be reconsidered.
	const validatedID = "lrn_already_validated"
	if err := learnStore.Add(learn.Entry{
		ID:             validatedID,
		Content:        "sentinel entry already validated by another path",
		Phase:          1,
		Confidence:     0.9,
		Classification: learn.ClassRepoLocal,
		Status:         learn.StatusValidated,
	}); err != nil {
		t.Fatalf("seed already-validated entry: %v", err)
	}

	// A genuine, currently unpromotable hypothesis (no guidance record at
	// all -- the honest, production-real shape).
	if err := learnStore.Add(learn.Entry{
		Content:        "sentinel hypothesis with no guidance record, run through the pass twice",
		Phase:          1,
		Confidence:     0.5,
		Classification: learn.ClassRepoLocal,
		Status:         learn.StatusHypothesis,
	}); err != nil {
		t.Fatalf("seed hypothesis entry: %v", err)
	}

	for i, label := range []string{"first run", "second run"} {
		summary := promoteHelpfulHypotheses(1)
		if !summary.Ran {
			t.Fatalf("%s: expected Ran == true, got %+v", label, summary)
		}
		if summary.Considered != 1 {
			t.Fatalf("%s: expected exactly 1 hypothesis considered (the already-validated entry must be excluded), got %+v", label, summary)
		}
		if len(summary.Promoted) != 0 {
			t.Fatalf("%s: expected zero promotions, got %+v", label, summary)
		}
		if len(summary.Failures) != 0 {
			t.Fatalf("%s: expected no failures, got %v", label, summary.Failures)
		}
		_ = i

		validated := getLearnEntry(t, validatedID)
		if validated.Status != learn.StatusValidated {
			t.Fatalf("%s: already-validated entry status changed to %q", label, validated.Status)
		}
	}
}

// TestDisprovenEntryIsNeverPromoted is the plan's Test 7: a disproven entry
// is never promoted by this path, even carrying a helpful application
// record under the same identifier. INTERNAL-LOGIC TEST ONLY (see this
// file's header comment): the same-identifier pairing is hypothetical, not
// a shape any production writer produces today.
func TestDisprovenEntryIsNeverPromoted(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	const id = "lrn_disproven"
	entry := learn.Entry{
		ID:             id,
		Content:        "sentinel disproven entry",
		Phase:          1,
		Confidence:     0.2,
		Classification: learn.ClassRepoLocal,
		Status:         learn.StatusDisproven,
	}
	if err := learn.NewColonyStore(store).Add(entry); err != nil {
		t.Fatalf("seed disproven entry: %v", err)
	}
	mustRecordGuidanceState(t, id, 1, guidanceApplicationStateAvailable)
	mustRecordGuidanceState(t, id, 1, guidanceApplicationStateRendered)
	mustRecordGuidanceState(t, id, 1, guidanceApplicationStateConsulted)
	mustRecordGuidanceState(t, id, 1, guidanceApplicationStateActedOn)
	mustRecordGuidanceState(t, id, 1, guidanceApplicationStateHelpful)

	summary := promoteHelpfulHypotheses(1)
	if len(summary.Promoted) != 0 {
		t.Fatalf("expected a disproven entry to never be promoted, got %+v", summary)
	}
	got := getLearnEntry(t, id)
	if got.Status != learn.StatusDisproven {
		t.Fatalf("entry status = %q, want it to remain %q", got.Status, learn.StatusDisproven)
	}
}

// TestHypothesisPromoterIsReachedFromBothCheckLanes is the plan's Test 8,
// in the exact shape of
// TestAutomaticImprovementPassIsReachedFromBothCheckLanes
// (cmd/improvement_pass_test.go): an AST-based call-graph guard proving
// promoteHelpfulHypotheses is transitively reachable from BOTH
// runCodexContinue and runCodexContinueFinalize, because it sits on the
// same runPhaseEndConsolidation chain the improvement pass already does.
func TestHypothesisPromoterIsReachedFromBothCheckLanes(t *testing.T) {
	repoRoot, err := repoRootForCommandSourceTest()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	g, err := buildCmdFuncGraph(filepath.Join(repoRoot, "cmd"))
	if err != nil {
		t.Fatalf("build call graph: %v", err)
	}
	if g.funcsIndexed == 0 {
		t.Fatalf("found zero top-level functions while scanning %d files -- a guard that finds no functions to check would pass vacuously forever", g.filesScanned)
	}

	const target = "promoteHelpfulHypotheses"
	lanes := []string{"runCodexContinue", "runCodexContinueFinalize"}

	if _, ok := g.calls[target]; !ok {
		t.Fatalf("expected %s to be an indexed top-level function among %d indexed functions in cmd/, but it was missing -- renamed or moved?", target, g.funcsIndexed)
	}
	for _, lane := range lanes {
		if _, ok := g.calls[lane]; !ok {
			t.Fatalf("expected %s to be an indexed top-level function among %d indexed functions in cmd/, but it was missing -- renamed or moved?", lane, g.funcsIndexed)
		}
	}

	if unreached := applicationCreditUnreachedLanes(g, lanes, target); len(unreached) > 0 {
		t.Fatalf("%s is not transitively reachable from: %v -- both check lanes must reach the automatic hypothesis promoter", target, unreached)
	}

	t.Run("a synthetic unreachable fixture is caught by name on both lanes", func(t *testing.T) {
		synthetic := &cmdFuncGraph{calls: map[string]map[string]bool{
			"runCodexContinue":         {"someOtherHelper": true},
			"runCodexContinueFinalize": {"anotherHelper": true},
			"promoteHelpfulHypotheses": {},
		}}
		unreached := applicationCreditUnreachedLanes(synthetic, lanes, target)
		if len(unreached) != 2 {
			t.Fatalf("scanner failed to detect the synthetic unreachable fixture on both lanes, got unreached=%v", unreached)
		}
		for _, lane := range lanes {
			found := false
			for _, u := range unreached {
				if u == lane {
					found = true
				}
			}
			if !found {
				t.Fatalf("expected %q named in the unreached set %v", lane, unreached)
			}
		}
	})
}
