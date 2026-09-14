package cmd

// LEARN-07 (204-09-PLAN.md): the promotion gate. Two kinds of thing may be
// changed automatically by a proven-beneficial, reversible, low-risk
// canary -- what the project knows about itself, and which kind of helper
// it sends at which kind of task. Nine kinds of thing may never be, and the
// defence is structural, never a rule in a document: canaryPromotableScopes
// names the only two admissible scopes, admitCandidateToCanary derives its
// admissible set from that list ALONE, and the nine-member retained list
// (canaryRetainedAuthorityScopes) never appears anywhere inside this
// function's own body -- TestRetainedAuthorityCannotBecomeCanaryPromotable
// (cmd/promotion_gate_test.go) parses this file's source to prove it, with
// a synthetic fixture proving the check itself can fail.
//
// Widening canaryPromotableScopes is never computed as "everything not in
// canaryRetainedAuthorityScopes" -- that would make the retained list's own
// membership silently control what becomes promotable. The two lists are
// declared, disjoint, and independent; the promoting function reads only
// the promotable one.
import (
	"fmt"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/shadow"
)

// canaryScope is the closed vocabulary a shadow.Candidate's own declared
// scope is classified into, by an exact (case-insensitive, trimmed) match
// against one of the eleven strings declared below.
type canaryScope string

// ---------------------------------------------------------------------------
// The two promotable scopes -- everything a canary may ever change.
// ---------------------------------------------------------------------------

const (
	canaryScopeProjectKnowledge canaryScope = "project knowledge"
	canaryScopeRouting          canaryScope = "routing"
)

// canaryPromotableScopes names exactly the two scopes a canary may ever
// promote. admitCandidateToCanary derives its admissible set from this
// slice alone.
var canaryPromotableScopes = []canaryScope{
	canaryScopeProjectKnowledge,
	canaryScopeRouting,
}

// canaryPromotableScopeNames returns the string form of every promotable
// scope, for refusal messages.
func canaryPromotableScopeNames() []string {
	names := make([]string, 0, len(canaryPromotableScopes))
	for _, s := range canaryPromotableScopes {
		names = append(names, string(s))
	}
	return names
}

// canaryScopePromotableDeclared reports whether scope is one of the two
// promotable scopes.
func canaryScopePromotableDeclared(scope canaryScope) bool {
	for _, s := range canaryPromotableScopes {
		if s == scope {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// The nine retained-authority scopes -- everything a canary may never
// change, ever, by any automatic path.
// ---------------------------------------------------------------------------

const (
	canaryScopePreferences     canaryScope = "preferences"
	canaryScopeSkills          canaryScope = "skills"
	canaryScopeWorkflows       canaryScope = "workflows"
	canaryScopeSource          canaryScope = "source"
	canaryScopeSecurity        canaryScope = "security"
	canaryScopeDeletion        canaryScope = "deletion"
	canaryScopePermission      canaryScope = "permission"
	canaryScopeVerification    canaryScope = "verification"
	canaryScopeExternalActions canaryScope = "external actions"
)

// canaryRetainedAuthorityScopes names exactly the nine scopes a canary may
// never promote -- preferences, skills, workflows, source, security,
// deletion, permission, verification, and external actions. This list is
// never read by admitCandidateToCanary's own body.
var canaryRetainedAuthorityScopes = []canaryScope{
	canaryScopePreferences,
	canaryScopeSkills,
	canaryScopeWorkflows,
	canaryScopeSource,
	canaryScopeSecurity,
	canaryScopeDeletion,
	canaryScopePermission,
	canaryScopeVerification,
	canaryScopeExternalActions,
}

// canaryRetainedAuthorityScopeNames returns the string form of every
// retained-authority scope.
func canaryRetainedAuthorityScopeNames() []string {
	names := make([]string, 0, len(canaryRetainedAuthorityScopes))
	for _, s := range canaryRetainedAuthorityScopes {
		names = append(names, string(s))
	}
	return names
}

// canaryScopeRetainedDeclared reports whether scope is one of the nine
// retained-authority scopes.
func canaryScopeRetainedDeclared(scope canaryScope) bool {
	for _, s := range canaryRetainedAuthorityScopes {
		if s == scope {
			return true
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// Which authority retains each of the nine -- the owner, or an independent
// reviewer -- so a refusal can say who decides instead of the automatic
// path. This mirrors the forced-reviewer pattern's SHAPE (named signal ->
// mandatory gate -> owner-only decision, CLAUDE.md "Team Check-In and Owner
// Decisions") but deliberately does NOT carry over its waiver: a forced
// reviewer is a judgement about risk the owner may decline; these nine are
// a boundary about authority, and a boundary you can waive is not a
// boundary.
// ---------------------------------------------------------------------------

const (
	canaryAuthorityOwner               = "the owner"
	canaryAuthorityIndependentReviewer = "an independent reviewer"
	canaryOwnerApprovalSurface         = "the owner's existing approval queue (`aether suggest-approve`)"
)

// canaryRetainedAuthority maps each retained-authority scope to the
// authority that retains it. security and deletion map to an independent
// reviewer -- the same two of the five named risk signals (CLAUDE.md's
// forced-reviewer table) that route to a security or quality reviewer
// rather than the owner directly; every other retained scope routes to the
// owner. Declared outside admitCandidateToCanary's own body so that
// function never names a single one of these nine constants directly.
var canaryRetainedAuthority = map[canaryScope]string{
	canaryScopePreferences:     canaryAuthorityOwner,
	canaryScopeSkills:          canaryAuthorityOwner,
	canaryScopeWorkflows:       canaryAuthorityOwner,
	canaryScopeSource:          canaryAuthorityIndependentReviewer,
	canaryScopeSecurity:        canaryAuthorityIndependentReviewer,
	canaryScopeDeletion:        canaryAuthorityIndependentReviewer,
	canaryScopePermission:      canaryAuthorityOwner,
	canaryScopeVerification:    canaryAuthorityIndependentReviewer,
	canaryScopeExternalActions: canaryAuthorityOwner,
}

// canaryRetainedAuthorityFor reports the authority that retains scope, and
// whether scope is a retained-authority scope at all. This is the ONLY
// place canaryRetainedAuthority (and therefore any of the nine retained
// scope constants) is read -- admitCandidateToCanary calls this function by
// name but never names a retained scope constant itself.
func canaryRetainedAuthorityFor(scope canaryScope) (authority string, retained bool) {
	authority, retained = canaryRetainedAuthority[scope]
	return authority, retained
}

// ---------------------------------------------------------------------------
// Classifying a candidate's declared scope.
// ---------------------------------------------------------------------------

// canaryDeclaredScope classifies candidate's own declared Scope() text into
// the closed canaryScope vocabulary by an exact, case-insensitive, trimmed
// match. A candidate whose scope text matches none of the eleven declared
// values classifies to that raw text itself -- neither promotable nor
// retained, and refused at the gate for not being recognized as either of
// the two scopes a canary may promote.
func canaryDeclaredScope(candidate shadow.Candidate) canaryScope {
	return canaryScope(strings.ToLower(strings.TrimSpace(candidate.Scope())))
}

// ---------------------------------------------------------------------------
// The gate.
// ---------------------------------------------------------------------------

// canaryAdmission is what a candidate earns by clearing every check in
// admitCandidateToCanary: its identity, its promotable scope, the verdict
// that admitted it, and the bound (Task 2, 204-09-PLAN.md) it must complete
// or roll back within -- taken from the candidate's own declared expiry,
// never a separate dial.
type canaryAdmission struct {
	CandidateID string
	Scope       canaryScope
	Verdict     shadow.Verdict
	Bound       time.Time
}

// canaryBoundExceeded reports whether now is at or past admission's own
// declared bound -- a canary that has not completed or rolled back by then
// must not keep running indefinitely.
func canaryBoundExceeded(admission canaryAdmission, now time.Time) bool {
	return !now.Before(admission.Bound)
}

// canaryGateResolvesEvaluatorDigest returns the digest of the grader THIS
// gate itself resolves (shadowEvaluator, cmd/shadow_cmds.go) -- the value
// admitCandidateToCanary compares comparison.EvaluatorDigest against, so a
// comparison graded by a different evaluator can never be presented here as
// though it had been graded by the real one.
func canaryGateResolvesEvaluatorDigest() [32]byte {
	return shadowEvaluator().Digest()
}

// admitCandidateToCanary is the gate: in order, and each refusal naming its
// own reason, the scope must be in the promotable list; the verdict must be
// beneficial (overfit, not-beneficial, tied, and inconclusive are each
// refused in their own words); the candidate must not have expired since
// its comparison ran; and the comparison's grader digest must match the
// grader this gate itself resolves.
//
// A retained-authority scope is refused by name, naming the authority that
// retains it and the one existing owner approval surface -- never a
// waiver. admitCandidateToCanary takes exactly two parameters (the
// candidate and its comparison) and carries no actor, caller-identity, or
// bypass parameter of any kind -- there is no coordinator- or
// autopilot-shaped input this function could ever branch on
// (TestNeitherCoordinatorNorAutopilotCanWaiveARetainedRefusal,
// cmd/promotion_gate_test.go, asserts this structurally).
func admitCandidateToCanary(candidate shadow.Candidate, comparison shadow.Comparison) (canaryAdmission, error) {
	scope := canaryDeclaredScope(candidate)
	if !canaryScopePromotableDeclared(scope) {
		if authority, retained := canaryRetainedAuthorityFor(scope); retained {
			return canaryAdmission{}, fmt.Errorf(
				"candidate %q declares scope %q -- only %s may change that, and this canary path never offers a waiver; take it to %s instead",
				candidate.ID(), scope, authority, canaryOwnerApprovalSurface,
			)
		}
		return canaryAdmission{}, fmt.Errorf(
			"candidate %q declares scope %q, which is not one of the two scopes a canary may promote (%v)",
			candidate.ID(), scope, canaryPromotableScopeNames(),
		)
	}

	switch comparison.Verdict {
	case shadow.VerdictBeneficial:
		// proceeds to the remaining checks below.
	case shadow.VerdictNotBeneficial:
		return canaryAdmission{}, fmt.Errorf(
			"candidate %q is refused: it did not beat the current behaviour on the work it could see -- not beneficial",
			candidate.ID(),
		)
	case shadow.VerdictOverfit:
		return canaryAdmission{}, fmt.Errorf(
			"candidate %q is refused: it is overfit -- it looked better only on the work it could see and did worse on the checks it never saw",
			candidate.ID(),
		)
	case shadow.VerdictTied:
		return canaryAdmission{}, fmt.Errorf(
			"candidate %q is refused: it tied the current behaviour, and a tie recommends nothing",
			candidate.ID(),
		)
	case shadow.VerdictInconclusive:
		return canaryAdmission{}, fmt.Errorf(
			"candidate %q is refused: there was nothing to compare it against, so no recommendation can be made",
			candidate.ID(),
		)
	default:
		return canaryAdmission{}, fmt.Errorf(
			"candidate %q carries an undeclared verdict %q",
			candidate.ID(), comparison.Verdict,
		)
	}

	if candidate.Expired(time.Now()) {
		return canaryAdmission{}, fmt.Errorf(
			"candidate %q has expired since its comparison ran and is refused at the gate -- an expired candidate is never promoted",
			candidate.ID(),
		)
	}

	if comparison.EvaluatorDigest != canaryGateResolvesEvaluatorDigest() {
		return canaryAdmission{}, fmt.Errorf(
			"candidate %q was graded by a different evaluator than the one this gate itself resolves -- refusing to admit a comparison graded by something else",
			candidate.ID(),
		)
	}

	return canaryAdmission{
		CandidateID: candidate.ID(),
		Scope:       scope,
		Verdict:     comparison.Verdict,
		Bound:       candidate.Expiry(),
	}, nil
}
