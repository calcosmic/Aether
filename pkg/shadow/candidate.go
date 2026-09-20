package shadow

import (
	"fmt"
	"strings"
	"time"
)

// candidateRequiredFieldNames names the six declarations every Candidate
// must carry, in the order NewCandidate checks them, so a refusal can name
// the missing one exactly. A candidate is a setting or a rule, never a
// piece of code (Planner Assumption P, 204-08-PLAN.md) -- code changes go a
// different route, reviewed by a person, never through this package.
var candidateRequiredFieldNames = []string{
	"identifier",
	"scope",
	"expected benefit",
	"harms",
	"expiry",
	"rollback plan",
}

// candidateForbiddenReferenceKinds is the declared, closed list of things a
// candidate's own scope or rollback-plan text must never name. A candidate
// that even mentions what it is graded by is trying to select it --
// refused before it ever runs, never merely warned about.
var candidateForbiddenReferenceKinds = []string{
	"evaluator",
	"acceptance criteria",
	"acceptance criterion",
	"holdout",
}

// Candidate is a proposed change: immutable once constructed, every field
// unexported and read only through an accessor method. Candidate holds no
// field of type FrozenEvaluator or AcceptanceCriteria and no method takes
// one -- the fourth structural condition isolation_test.go's
// TestCandidateHoldsNoEvaluator checks by reading this file's own source.
type Candidate struct {
	id              string
	scope           string
	expectedBenefit string
	harms           string
	expiry          time.Time
	rollbackPlan    string
}

// NewCandidate is the one way to construct a Candidate. All six required
// declarations must be present up front -- a declaration missing any of
// them is refused, naming the missing field
// (TestCandidateRequiresAllSixDeclarations). An expiry already in the past
// is refused as well (TestExpiredCandidateIsRefused): a Candidate cannot be
// constructed already expired, so an expired candidate can never reach a
// comparison in the first place. A scope or rollback-plan declaration that
// names an evaluator, an acceptance criterion or a holdout is refused,
// naming the offending field and the forbidden term it used
// (TestCandidateNamingItsGraderIsRefused) -- a candidate that even mentions
// what it is graded by is trying to select it.
func NewCandidate(id, scope, expectedBenefit, harms string, expiry time.Time, rollbackPlan string) (Candidate, error) {
	fields := []struct {
		name  string
		value string
	}{
		{candidateRequiredFieldNames[0], id},
		{candidateRequiredFieldNames[1], scope},
		{candidateRequiredFieldNames[2], expectedBenefit},
		{candidateRequiredFieldNames[3], harms},
		{candidateRequiredFieldNames[5], rollbackPlan},
	}
	for _, f := range fields {
		if strings.TrimSpace(f.value) == "" {
			return Candidate{}, fmt.Errorf("candidate declaration is missing its %s -- a candidate cannot be declared without all six: %s", f.name, strings.Join(candidateRequiredFieldNames, ", "))
		}
	}
	if expiry.IsZero() {
		return Candidate{}, fmt.Errorf("candidate declaration is missing its %s -- a candidate cannot be declared without all six: %s", candidateRequiredFieldNames[4], strings.Join(candidateRequiredFieldNames, ", "))
	}
	if expiry.Before(time.Now()) {
		return Candidate{}, fmt.Errorf("candidate %q declares an expiry (%s) that has already passed -- an expired candidate is refused and never graded", id, expiry.Format(time.RFC3339))
	}

	if term, field, ok := candidateForbiddenReference(scope, rollbackPlan); !ok {
		return Candidate{}, fmt.Errorf("candidate %q names %q in its %s -- a candidate that names what it is graded by is refused before it runs", id, term, field)
	}

	return Candidate{
		id:              id,
		scope:           scope,
		expectedBenefit: expectedBenefit,
		harms:           harms,
		expiry:          expiry,
		rollbackPlan:    rollbackPlan,
	}, nil
}

// candidateForbiddenReference reports the first forbidden term found in
// scope or rollbackPlan, and which field it was found in. ok is true when
// neither text names anything forbidden.
func candidateForbiddenReference(scope, rollbackPlan string) (term, field string, ok bool) {
	for _, t := range candidateForbiddenReferenceKinds {
		if strings.Contains(strings.ToLower(scope), t) {
			return t, "scope", false
		}
	}
	for _, t := range candidateForbiddenReferenceKinds {
		if strings.Contains(strings.ToLower(rollbackPlan), t) {
			return t, "rollback plan", false
		}
	}
	return "", "", true
}

// ID returns the candidate's identifier.
func (c Candidate) ID() string { return c.id }

// Scope returns the candidate's declared scope.
func (c Candidate) Scope() string { return c.scope }

// ExpectedBenefit returns the candidate's declared expected benefit.
func (c Candidate) ExpectedBenefit() string { return c.expectedBenefit }

// Harms returns the candidate's declared harms.
func (c Candidate) Harms() string { return c.harms }

// Expiry returns the candidate's declared expiry.
func (c Candidate) Expiry() time.Time { return c.expiry }

// RollbackPlan returns the candidate's declared rollback plan.
func (c Candidate) RollbackPlan() string { return c.rollbackPlan }

// Expired reports whether the candidate's declared expiry is at or before
// now. Compare (comparison.go) checks this again at comparison time -- a
// candidate declared with a future expiry that has since passed is refused
// there too, never graded.
func (c Candidate) Expired(now time.Time) bool {
	return !c.expiry.After(now)
}

// baselineAsCandidate builds an internal, package-private Candidate
// representation of the current behaviour so Baseline can be run through
// the SAME FrozenEvaluator.Run(Candidate, Task) Result signature a real
// candidate is graded through -- there is deliberately no second grading
// path. This bypasses NewCandidate's own validation because it is not a
// declaration: nobody outside this package can construct one, it is never
// stored, and it never reaches the declared-candidate store cmd/shadow_cmds.go
// owns. Comparison.go's Compare is the only caller.
func baselineAsCandidate(baseline Baseline) Candidate {
	return Candidate{
		id:              "baseline",
		scope:           "the current behaviour, kept unchanged as the point of comparison",
		expectedBenefit: "n/a -- this is the reference point, not a proposed change",
		harms:           "n/a -- this is the reference point, not a proposed change",
		expiry:          time.Now().Add(100 * 365 * 24 * time.Hour),
		rollbackPlan:    "n/a -- this is the reference point, not a proposed change",
	}
}
