package cmd

// 204-16-PLAN.md, Task 2 (WINDOWS.md entry 44): the automatic
// hypothesis-to-validated promoter. Every test below writes its own
// guidance-application chain through the real production writer
// (mustRecordGuidanceState, declared in cmd/application_evidence_test.go,
// same package) rather than a hand-typed credit/records.json literal, so a
// test can never assert against a shape the runtime does not itself
// produce.
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

// TestHelpfulHypothesisIsPromotedAutomatically is the plan's Test 1: a
// learned entry in hypothesis status, with a guidance application record
// for its own identifier in the helpful state, is promoted to validated by
// the automatic pass, and then appears in learningVerifiedEntries.
func TestHelpfulHypothesisIsPromotedAutomatically(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	const id = "lrn_helpful_1"
	seedHypothesisEntry(t, id, "sentinel hypothesis proven helpful", 1)
	mustRecordGuidanceState(t, id, 1, guidanceApplicationStateAvailable)
	mustRecordGuidanceState(t, id, 1, guidanceApplicationStateRendered)
	mustRecordGuidanceState(t, id, 1, guidanceApplicationStateConsulted)
	mustRecordGuidanceState(t, id, 1, guidanceApplicationStateActedOn)
	mustRecordGuidanceState(t, id, 1, guidanceApplicationStateHelpful)

	summary := promoteHelpfulHypotheses(1)
	if !summary.Ran {
		t.Fatalf("expected Ran == true, got %+v", summary)
	}
	if len(summary.Promoted) != 1 || summary.Promoted[0] != id {
		t.Fatalf("expected entry %q promoted, got %+v", id, summary)
	}

	entry := getLearnEntry(t, id)
	if entry.Status != learn.StatusValidated {
		t.Fatalf("entry status = %q, want %q", entry.Status, learn.StatusValidated)
	}

	verified := learningVerifiedEntries([]learn.Entry{entry})
	if len(verified) != 1 {
		t.Fatalf("expected the promoted entry to appear in learningVerifiedEntries, got %+v", verified)
	}
}

// TestActedOnIsNotEnoughToValidate is the plan's Test 2: a hypothesis with
// a guidance record only in the rendered, consulted, or acted-on state is
// NOT promoted.
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
// application is NOT promoted.
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

// TestPromotionIsIdempotent is the plan's Test 6: running the promoter
// twice promotes once and writes nothing the second time.
func TestPromotionIsIdempotent(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	const id = "lrn_idempotent"
	seedHypothesisEntry(t, id, "sentinel hypothesis promoted exactly once", 1)
	mustRecordGuidanceState(t, id, 1, guidanceApplicationStateAvailable)
	mustRecordGuidanceState(t, id, 1, guidanceApplicationStateRendered)
	mustRecordGuidanceState(t, id, 1, guidanceApplicationStateConsulted)
	mustRecordGuidanceState(t, id, 1, guidanceApplicationStateActedOn)
	mustRecordGuidanceState(t, id, 1, guidanceApplicationStateHelpful)

	first := promoteHelpfulHypotheses(1)
	if len(first.Promoted) != 1 {
		t.Fatalf("expected exactly one promotion on the first run, got %+v", first)
	}

	second := promoteHelpfulHypotheses(1)
	if len(second.Promoted) != 0 {
		t.Fatalf("expected zero promotions on the second run, got %+v", second)
	}
	if len(second.Failures) != 0 {
		t.Fatalf("expected no failures on the second run, got %v", second.Failures)
	}
}

// TestDisprovenEntryIsNeverPromoted is the plan's Test 7: a disproven entry
// is never promoted by this path, even carrying a helpful application
// record under the same identifier.
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
