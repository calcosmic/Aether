package cmd

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
	"time"

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
