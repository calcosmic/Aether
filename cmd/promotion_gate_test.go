package cmd

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/learn"
	"github.com/calcosmic/Aether/pkg/shadow"
)

func promotionGateTestCandidate(t *testing.T, scope string, expiry time.Time) shadow.Candidate {
	t.Helper()
	return promotionGateTestCandidateWithID(t, "candidate-"+scope, scope, expiry)
}

func promotionGateTestCandidateWithID(t *testing.T, id, scope string, expiry time.Time) shadow.Candidate {
	t.Helper()
	c, err := shadow.NewCandidate(id, scope, "it should help", "it could regress", expiry, "revert the change")
	if err != nil {
		t.Fatalf("construct test candidate %q for scope %q: %v", id, scope, err)
	}
	return c
}

func promotionGateBeneficialComparison(candidateID string) shadow.Comparison {
	return shadow.Comparison{
		CandidateID:      candidateID,
		Verdict:          shadow.VerdictBeneficial,
		VisibleBaseline:  shadow.Score{Numerator: 1, Denominator: 10},
		VisibleCandidate: shadow.Score{Numerator: 9, Denominator: 10},
		HoldoutBaseline:  shadow.Score{Numerator: 1, Denominator: 10},
		HoldoutCandidate: shadow.Score{Numerator: 9, Denominator: 10},
		EvaluatorDigest:  canaryGateResolvesEvaluatorDigest(),
	}
}

func TestOnlyTwoScopesAreCanaryPromotable(t *testing.T) {
	if len(canaryPromotableScopes) != 2 {
		t.Fatalf("expected exactly 2 promotable scopes, got %d: %v", len(canaryPromotableScopes), canaryPromotableScopes)
	}
	if len(canaryPromotableScopeNames()) != len(canaryPromotableScopes) {
		t.Fatalf("canaryPromotableScopeNames() length disagrees with canaryPromotableScopes")
	}
	if !canaryScopePromotableDeclared(canaryScopeProjectKnowledge) {
		t.Fatal("project knowledge should be promotable")
	}
	if !canaryScopePromotableDeclared(canaryScopeRouting) {
		t.Fatal("routing should be promotable")
	}
}

// canaryRetainedScopeIdentifiers names every identifier
// admitCandidateToCanary's own body must never contain -- the retained
// list itself plus each of its nine members' Go identifiers.
var canaryRetainedScopeIdentifiers = []string{
	"canaryRetainedAuthorityScopes",
	"canaryScopePreferences",
	"canaryScopeSkills",
	"canaryScopeWorkflows",
	"canaryScopeSource",
	"canaryScopeSecurity",
	"canaryScopeDeletion",
	"canaryScopePermission",
	"canaryScopeVerification",
	"canaryScopeExternalActions",
}

// funcBodyContainsIdentifiers parses src, finds the function literally
// named funcName, and reports whether its body contains any identifier
// named in forbidden.
func funcBodyContainsIdentifiers(t *testing.T, filename string, src []byte, funcName string, forbidden []string) (hit bool, found string) {
	t.Helper()
	forbiddenSet := make(map[string]bool, len(forbidden))
	for _, f := range forbidden {
		forbiddenSet[f] = true
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", filename, err)
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil || fn.Name == nil || fn.Name.Name != funcName {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			id, ok := n.(*ast.Ident)
			if ok && forbiddenSet[id.Name] {
				hit = true
				found = id.Name
				return false
			}
			return true
		})
	}
	return hit, found
}

func TestRetainedAuthorityCannotBecomeCanaryPromotable(t *testing.T) {
	t.Run("the two lists are disjoint", func(t *testing.T) {
		for _, p := range canaryPromotableScopes {
			for _, r := range canaryRetainedAuthorityScopes {
				if p == r {
					t.Fatalf("scope %q appears in both the promotable and retained-authority lists", p)
				}
			}
		}
	})

	t.Run("the retained list has exactly nine members", func(t *testing.T) {
		if len(canaryRetainedAuthorityScopes) != 9 {
			t.Fatalf("expected exactly 9 retained-authority scopes, got %d: %v", len(canaryRetainedAuthorityScopes), canaryRetainedAuthorityScopes)
		}
	})

	t.Run("admitCandidateToCanary's own body names no retained-authority identifier", func(t *testing.T) {
		src, err := os.ReadFile("promotion_gate.go")
		if err != nil {
			t.Fatalf("read promotion_gate.go: %v", err)
		}
		hit, found := funcBodyContainsIdentifiers(t, "promotion_gate.go", src, "admitCandidateToCanary", canaryRetainedScopeIdentifiers)
		if hit {
			t.Fatalf("admitCandidateToCanary's own body references retained-authority identifier %q -- the admissible set must derive only from canaryPromotableScopes, never a computed complement of the retained list", found)
		}
	})

	t.Run("the checker is non-vacuous: a synthetic fixture naming a retained scope constant is caught by symbol name", func(t *testing.T) {
		fixtureSrc := []byte(`package cmd

func admitCandidateToCanaryFixtureViolation() {
	_ = canaryScopeSkills
}
`)
		hit, found := funcBodyContainsIdentifiers(t, "fixture_admission_violation.go", fixtureSrc, "admitCandidateToCanaryFixtureViolation", canaryRetainedScopeIdentifiers)
		if !hit {
			t.Fatal("scanner failed to flag a synthetic function that references a retained scope constant -- the structural check would be vacuous")
		}
		if found != "canaryScopeSkills" {
			t.Fatalf("expected the violation to be reported by symbol name canaryScopeSkills, got %q", found)
		}
		t.Logf("synthetic fixture correctly flagged by symbol name: %s", found)
	})
}

func TestEachRetainedScopeRefusalNamesItsAuthority(t *testing.T) {
	expiry := time.Now().Add(24 * time.Hour)
	for _, scope := range canaryRetainedAuthorityScopes {
		scope := scope
		t.Run(string(scope), func(t *testing.T) {
			candidate := promotionGateTestCandidate(t, string(scope), expiry)
			comparison := promotionGateBeneficialComparison(candidate.ID())
			_, err := admitCandidateToCanary(candidate, comparison)
			if err == nil {
				t.Fatalf("expected scope %q to be refused, got no error", scope)
			}
			authority, retained := canaryRetainedAuthorityFor(scope)
			if !retained {
				t.Fatalf("test setup broken: %q is not in canaryRetainedAuthority", scope)
			}
			if !strings.Contains(err.Error(), authority) {
				t.Fatalf("refusal for scope %q does not name its authority %q: %v", scope, authority, err)
			}
			if !strings.Contains(err.Error(), "aether suggest-approve") {
				t.Fatalf("refusal for scope %q does not name the owner approval surface: %v", scope, err)
			}
		})
	}
}

func TestNeitherCoordinatorNorAutopilotCanWaiveARetainedRefusal(t *testing.T) {
	t.Run("the gate signature carries no actor, caller-identity, or bypass parameter", func(t *testing.T) {
		src, err := os.ReadFile("promotion_gate.go")
		if err != nil {
			t.Fatalf("read promotion_gate.go: %v", err)
		}
		fset := token.NewFileSet()
		file, err := parser.ParseFile(fset, "promotion_gate.go", src, 0)
		if err != nil {
			t.Fatalf("parse promotion_gate.go: %v", err)
		}
		var params int
		found := false
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Name == nil || fn.Name.Name != "admitCandidateToCanary" {
				continue
			}
			found = true
			for _, field := range fn.Type.Params.List {
				if len(field.Names) == 0 {
					params++
					continue
				}
				params += len(field.Names)
			}
		}
		if !found {
			t.Fatal("could not find admitCandidateToCanary in promotion_gate.go")
		}
		if params != 2 {
			t.Fatalf("admitCandidateToCanary takes %d parameters, expected exactly 2 (candidate, comparison) -- a third parameter could become an actor-identity bypass", params)
		}
	})

	t.Run("calling it directly, as any caller (coordinator or autopilot) would, still refuses a retained scope", func(t *testing.T) {
		expiry := time.Now().Add(24 * time.Hour)
		candidate := promotionGateTestCandidate(t, string(canaryScopeSecurity), expiry)
		comparison := promotionGateBeneficialComparison(candidate.ID())
		if _, err := admitCandidateToCanary(candidate, comparison); err == nil {
			t.Fatal("expected a retained-authority scope to be refused with no way to waive it")
		}
	})
}

func TestNonBeneficialVerdictsAreEachRefusedInTheirOwnWords(t *testing.T) {
	expiry := time.Now().Add(24 * time.Hour)
	cases := []struct {
		verdict shadow.Verdict
		phrase  string
	}{
		{shadow.VerdictNotBeneficial, "not beneficial"},
		{shadow.VerdictOverfit, "overfit"},
		{shadow.VerdictTied, "tied"},
		{shadow.VerdictInconclusive, "inconclusive"},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(string(tc.verdict), func(t *testing.T) {
			candidate := promotionGateTestCandidateWithID(t, "candidate-verdict-"+string(tc.verdict), string(canaryScopeRouting), expiry)
			comparison := promotionGateBeneficialComparison(candidate.ID())
			comparison.Verdict = tc.verdict
			_, err := admitCandidateToCanary(candidate, comparison)
			if err == nil {
				t.Fatalf("expected verdict %q to be refused", tc.verdict)
			}
			if !strings.Contains(err.Error(), tc.phrase) {
				t.Fatalf("refusal for verdict %q does not use its own words (%q): %v", tc.verdict, tc.phrase, err)
			}
		})
	}

	// Every verdict's refusal wording must differ from every other's -- a
	// generic "refused" message that never names the reason would pass a
	// substring check trivially; this proves the four are distinct.
	seen := map[string]bool{}
	for _, tc := range cases {
		candidate := promotionGateTestCandidateWithID(t, "candidate-verdict-distinct-"+string(tc.verdict), string(canaryScopeRouting), expiry)
		comparison := promotionGateBeneficialComparison(candidate.ID())
		comparison.Verdict = tc.verdict
		_, err := admitCandidateToCanary(candidate, comparison)
		if err == nil {
			t.Fatalf("expected verdict %q to be refused", tc.verdict)
		}
		if seen[err.Error()] {
			t.Fatalf("verdict %q produced a refusal message identical to another verdict's -- not refused in its own words", tc.verdict)
		}
		seen[err.Error()] = true
	}
}

func TestMismatchedGraderDigestIsRefused(t *testing.T) {
	expiry := time.Now().Add(24 * time.Hour)
	candidate := promotionGateTestCandidate(t, string(canaryScopeRouting), expiry)
	comparison := promotionGateBeneficialComparison(candidate.ID())
	comparison.EvaluatorDigest = [32]byte{0xFF} // deliberately not canaryGateResolvesEvaluatorDigest()

	_, err := admitCandidateToCanary(candidate, comparison)
	if err == nil {
		t.Fatal("expected a mismatched grader digest to be refused")
	}
	if !strings.Contains(err.Error(), "evaluator") {
		t.Fatalf("refusal does not name the grader mismatch: %v", err)
	}
}

func TestExpiredCandidateIsRefusedAtTheGate(t *testing.T) {
	expiry := time.Now().Add(30 * time.Millisecond)
	candidate := promotionGateTestCandidate(t, string(canaryScopeProjectKnowledge), expiry)
	comparison := promotionGateBeneficialComparison(candidate.ID())

	time.Sleep(50 * time.Millisecond)

	_, err := admitCandidateToCanary(candidate, comparison)
	if err == nil {
		t.Fatal("expected a candidate whose expiry has passed since its comparison ran to be refused at the gate")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Fatalf("refusal does not name the expiry: %v", err)
	}
}

// TestPromotableCandidateIsAdmitted is the positive-path proof that the
// gate above is not vacuously refusing everything: a candidate declaring a
// promotable scope, with a beneficial verdict, an unexpired candidacy, and
// a matching grader digest is admitted.
func TestPromotableCandidateIsAdmitted(t *testing.T) {
	expiry := time.Now().Add(24 * time.Hour)
	for _, scope := range canaryPromotableScopes {
		scope := scope
		t.Run(string(scope), func(t *testing.T) {
			candidate := promotionGateTestCandidate(t, string(scope), expiry)
			comparison := promotionGateBeneficialComparison(candidate.ID())
			admission, err := admitCandidateToCanary(candidate, comparison)
			if err != nil {
				t.Fatalf("expected scope %q to be admitted, got: %v", scope, err)
			}
			if admission.Scope != scope {
				t.Fatalf("admission carries scope %q, expected %q", admission.Scope, scope)
			}
			if admission.CandidateID != candidate.ID() {
				t.Fatalf("admission carries candidate id %q, expected %q", admission.CandidateID, candidate.ID())
			}
			if !admission.Bound.Equal(expiry) {
				t.Fatalf("admission bound %v does not match candidate expiry %v", admission.Bound, expiry)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Task 3 (LEARN-07): auto-derived skills and canary candidates route into
// the one owner approval list, never the active set and never a second
// approval surface.
// ---------------------------------------------------------------------------

func promotionGateTestEntry(id, runID string) learn.Entry {
	return learn.Entry{
		ID:      id,
		Content: "authentication middleware implementation with JWT tokens and refresh rotation",
		Evidence: learn.Evidence{
			RunID: runID,
			Phase: 5,
			Workers: []learn.WorkerEvidence{
				{Name: "Builder-1", Caste: "builder", Status: "failed"},
				{Name: "Builder-2", Caste: "builder", Status: "completed"},
			},
			FilesTouched: []string{"pkg/auth/middleware.go", "pkg/auth/tokens.go"},
			GatesPassed:  2,
			GatesTotal:   3,
			Confidence:   0.8,
		},
		Classification: learn.ClassRepoLocal,
		Phase:          5,
		Confidence:     0.8,
	}
}

func pendingSkillProposalItems(cs colony.ColonyState) []colony.PendingSuggestion {
	if cs.PendingSuggestions == nil {
		return nil
	}
	var items []colony.PendingSuggestion
	for _, item := range *cs.PendingSuggestions {
		if item.Origin != nil && *item.Origin == colony.PendingOriginSkillProposal {
			items = append(items, item)
		}
	}
	return items
}

// TestAutoDerivedSkillEntersTheApprovalListNotTheActiveSet drives all three
// declared auto-skill modes through the real check path
// (learn.AutoCreateSkillIfDifficult with the real colonySkillProposalSink)
// and asserts the skill service holds no new active skill in any of them --
// demonstrated able to fail against the previous behaviour: before this
// plan, "auto" mode called svc.CreateSkill directly, so this exact
// assertion (len(skills) == 0) failed for that mode (recorded in this
// plan's own SUMMARY.md).
func TestAutoDerivedSkillEntersTheApprovalListNotTheActiveSet(t *testing.T) {
	for _, mode := range []string{learn.AutoSkillModeOff, learn.AutoSkillModePropose, learn.AutoSkillModeAuto} {
		mode := mode
		t.Run(mode, func(t *testing.T) {
			saveGlobals(t)
			s, _ := newTestStore(t)
			store = s

			entry := promotionGateTestEntry("entry-"+mode, "run-"+mode)
			if err := learn.AutoCreateSkillIfDifficult(entry, mode, colonySkillProposalSink{}); err != nil {
				t.Fatalf("AutoCreateSkillIfDifficult (%s): %v", mode, err)
			}

			sqliteStore, err := learn.NewSQLiteColonyStore(filepath.Join(store.BasePath(), "colony.db"))
			if err != nil {
				t.Fatalf("open sqlite store: %v", err)
			}
			defer sqliteStore.Close()
			svc := learn.NewSkillService(sqliteStore.DB(), t.TempDir())
			skills, err := svc.ListSkills(learn.SkillStageActive)
			if err != nil {
				t.Fatalf("list skills: %v", err)
			}
			if len(skills) != 0 {
				t.Fatalf("expected no active skill created for mode %q, found %d", mode, len(skills))
			}

			var cs colony.ColonyState
			_ = store.LoadJSON("COLONY_STATE.json", &cs)
			hasProposal := len(pendingSkillProposalItems(cs)) > 0

			if mode == learn.AutoSkillModeOff {
				if hasProposal {
					t.Fatal("expected off mode to raise no proposal")
				}
				return
			}
			if !hasProposal {
				t.Fatalf("expected mode %q to raise a proposal into the approval queue", mode)
			}
		})
	}
}

func TestApprovingASkillProposalCreatesTheSkill(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	entry := promotionGateTestEntry("entry-approve-1", "run-approve-1")
	if err := learn.AutoCreateSkillIfDifficult(entry, learn.AutoSkillModePropose, colonySkillProposalSink{}); err != nil {
		t.Fatalf("AutoCreateSkillIfDifficult: %v", err)
	}

	var cs colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &cs); err != nil {
		t.Fatalf("load colony state: %v", err)
	}
	items := pendingSkillProposalItems(cs)
	if len(items) != 1 {
		t.Fatalf("expected exactly one queued skill proposal, got %d", len(items))
	}
	item := items[0]

	aetherRoot := t.TempDir()
	t.Setenv("AETHER_ROOT", aetherRoot)

	result, err := approvePendingItem(item.ID, "owner", "", false)
	if err != nil {
		t.Fatalf("approve skill proposal: %v", err)
	}
	if !result.Found {
		t.Fatal("expected the queued item to be found")
	}

	sqliteStore, err := learn.NewSQLiteColonyStore(filepath.Join(store.BasePath(), "colony.db"))
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	defer sqliteStore.Close()
	svc := learn.NewSkillService(sqliteStore.DB(), aetherRoot)
	got, err := svc.GetSkill(*item.SkillName)
	if err != nil {
		t.Fatalf("get skill: %v", err)
	}
	if got == nil {
		t.Fatal("expected approving the proposal to create the real skill")
	}
}

func TestDismissingASkillProposalCreatesNothing(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	entry := promotionGateTestEntry("entry-dismiss-1", "run-dismiss-1")
	if err := learn.AutoCreateSkillIfDifficult(entry, learn.AutoSkillModePropose, colonySkillProposalSink{}); err != nil {
		t.Fatalf("AutoCreateSkillIfDifficult: %v", err)
	}
	var cs colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &cs); err != nil {
		t.Fatalf("load colony state: %v", err)
	}
	items := pendingSkillProposalItems(cs)
	if len(items) != 1 {
		t.Fatalf("expected exactly one queued skill proposal, got %d", len(items))
	}
	item := items[0]

	result, err := rejectPendingNote(item.ID, "owner", "", false)
	if err != nil {
		t.Fatalf("reject: %v", err)
	}
	if !result.Found {
		t.Fatal("expected the queued item to be found")
	}

	aetherRoot := t.TempDir()
	sqliteStore, err := learn.NewSQLiteColonyStore(filepath.Join(store.BasePath(), "colony.db"))
	if err != nil {
		t.Fatalf("open sqlite store: %v", err)
	}
	defer sqliteStore.Close()
	svc := learn.NewSkillService(sqliteStore.DB(), aetherRoot)
	got, _ := svc.GetSkill(*item.SkillName)
	if got != nil {
		t.Fatal("expected dismissing a skill proposal to create nothing")
	}
}

func TestQuarantinedCandidateIsReleasableOnlyFromTheApprovalSurface(t *testing.T) {
	root, scopedFile := rollbackTestSetup(t)
	admission := rollbackTestAdmission("candidate-release-via-queue", canaryScopeRouting)
	run, err := startCanary(admission, []string{scopedFile})
	if err != nil {
		t.Fatalf("start canary: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, scopedFile), []byte("changed\n"), 0o644); err != nil {
		t.Fatalf("apply candidate change: %v", err)
	}
	if _, _, err := rollbackCanary(run, "regression for release-via-queue test"); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	var cs colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &cs); err != nil {
		t.Fatalf("load colony state: %v", err)
	}
	var item colony.PendingSuggestion
	found := false
	if cs.PendingSuggestions != nil {
		for _, s := range *cs.PendingSuggestions {
			if s.Origin != nil && *s.Origin == colony.PendingOriginCanaryCandidate &&
				s.CanaryCandidateID != nil && *s.CanaryCandidateID == admission.CandidateID {
				item = s
				found = true
			}
		}
	}
	if !found {
		t.Fatal("expected the quarantined candidate to be queued for owner approval")
	}

	result, err := approvePendingItem(item.ID, "owner", "", false)
	if err != nil {
		t.Fatalf("approve quarantine release: %v", err)
	}
	if !result.Found {
		t.Fatal("expected the queued item to be found")
	}

	stored, found, err := loadCanaryRun(admission.CandidateID)
	if err != nil {
		t.Fatalf("load canary run: %v", err)
	}
	if !found {
		t.Fatal("expected a stored canary run")
	}
	if stored.Quarantined {
		t.Fatal("expected approving the queued item to release the quarantine")
	}
}

func TestSkillProposalNamesItsSourceLearningEntry(t *testing.T) {
	saveGlobals(t)
	s, _ := newTestStore(t)
	store = s

	entry := promotionGateTestEntry("entry-provenance-1", "run-provenance-1")
	if err := learn.AutoCreateSkillIfDifficult(entry, learn.AutoSkillModePropose, colonySkillProposalSink{}); err != nil {
		t.Fatalf("AutoCreateSkillIfDifficult: %v", err)
	}

	var cs colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &cs); err != nil {
		t.Fatalf("load colony state: %v", err)
	}
	items := pendingSkillProposalItems(cs)
	if len(items) != 1 {
		t.Fatalf("expected exactly one queued skill proposal, got %d", len(items))
	}
	item := items[0]
	if item.SkillLearningEntryID == nil || *item.SkillLearningEntryID != entry.ID {
		t.Fatalf("expected the queued proposal to name its source learning entry %q, got %v", entry.ID, item.SkillLearningEntryID)
	}
}

// captureContinueLearningCallsRealSink parses src, finds the function
// literally named captureContinueLearning, and reports whether its body
// calls learn.AutoCreateSkillIfDifficult with a colonySkillProposalSink{}
// composite literal as its final argument -- the real sink, not a nil or
// some other value.
func captureContinueLearningCallsRealSink(t *testing.T, filename string, src interface{}) (wired bool) {
	t.Helper()
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, filename, src, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", filename, err)
	}
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok || fn.Body == nil || fn.Name == nil || fn.Name.Name != "captureContinueLearning" {
			continue
		}
		ast.Inspect(fn.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel == nil || sel.Sel.Name != "AutoCreateSkillIfDifficult" {
				return true
			}
			if len(call.Args) == 0 {
				return true
			}
			last := call.Args[len(call.Args)-1]
			if lit, ok := last.(*ast.CompositeLit); ok {
				if id, ok := lit.Type.(*ast.Ident); ok && id.Name == "colonySkillProposalSink" {
					wired = true
				}
			}
			return true
		})
	}
	return wired
}

// TestSkillProposalSinkIsWiredFromTheCheckPath is a call-graph test proving
// the real sink implementation is reached from the continue path on BOTH
// lanes. There is exactly one call site of AutoCreateSkillIfDifficult
// (inside captureContinueLearning), and exactly one call site of
// captureContinueLearning's own caller, captureContinueMemory
// (cmd/memory_feed_continue.go) -- which both continue lanes
// (runCodexContinue, the direct lane, and runCodexContinueFinalize, the
// external/host lane) call directly. Proving the sink is wired at that one
// shared chokepoint therefore proves it for both lanes without needing two
// separate scans of divergent call chains.
func TestSkillProposalSinkIsWiredFromTheCheckPath(t *testing.T) {
	src, err := os.ReadFile("codex_continue_finalize.go")
	if err != nil {
		t.Fatalf("read codex_continue_finalize.go: %v", err)
	}
	if !captureContinueLearningCallsRealSink(t, "codex_continue_finalize.go", src) {
		t.Fatal("captureContinueLearning does not call AutoCreateSkillIfDifficult with the real colonySkillProposalSink")
	}

	directSrc, err := os.ReadFile("codex_continue.go")
	if err != nil {
		t.Fatalf("read codex_continue.go: %v", err)
	}
	if !strings.Contains(string(directSrc), "captureContinueMemory(") {
		t.Fatal("the direct continue lane does not call captureContinueMemory -- the shared chokepoint to captureContinueLearning")
	}
	if !strings.Contains(string(src), "captureContinueMemory(") {
		t.Fatal("the external/host continue lane does not call captureContinueMemory -- the shared chokepoint to captureContinueLearning")
	}

	t.Run("a synthetic negative fixture (nil sink) is not reported as wired", func(t *testing.T) {
		fixtureSrc := `package cmd

func captureContinueLearning() {
	learn.AutoCreateSkillIfDifficult(entry, mode, nil)
}
`
		if captureContinueLearningCallsRealSink(t, "fixture_unwired_sink.go", fixtureSrc) {
			t.Fatal("scanner incorrectly reported a nil-sink call as wired -- the check would be vacuous")
		}
	})
}
