package shadow

import (
	"fmt"
	"time"
)

// Score carries a count and a total as integers, never a percentage. Percent
// derives the display figure only -- nothing else in this package compares
// two Score values by their Percent() output. Every verdict this package
// reaches compares numerators cross-multiplied by denominators
// (scoreCompare), so two scores built over different denominators compare
// exactly, and rounding a displayed percentage can never decide a verdict.
// TestScoresCompareExactlyNotByPercentage proves this with a pair of scores
// whose percentages round identically and whose exact comparison differs.
type Score struct {
	Numerator   int
	Denominator int
}

// Percent derives the display figure for Score -- for display only. Never
// compare two Percent() results against each other; compare the Score
// values themselves with scoreCompare.
func (s Score) Percent() float64 {
	if s.Denominator == 0 {
		return 0
	}
	return float64(s.Numerator) / float64(s.Denominator) * 100
}

// scoreCompare compares a and b exactly, by cross-multiplying numerator and
// denominator, and returns -1, 0 or 1 the way a normal comparator does.
// Two scores sharing a zero denominator compare equal -- an empty set
// compared against another empty set is a tie by this rule alone, which is
// why determineVerdict checks the fully-empty case (both visible and
// holdout carrying nothing) before consulting this function at all.
func scoreCompare(a, b Score) int {
	left := int64(a.Numerator) * int64(b.Denominator)
	right := int64(b.Numerator) * int64(a.Denominator)
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

// Verdict is the closed vocabulary a Comparison's outcome may hold.
type Verdict string

const (
	// VerdictBeneficial: the candidate beat the baseline on the work it
	// could see AND did not lose on the checks it never saw.
	VerdictBeneficial Verdict = "beneficial"
	// VerdictNotBeneficial: the candidate lost or tied on the visible set.
	// A candidate that loses on the visible set is not beneficial
	// regardless of the holdout result.
	VerdictNotBeneficial Verdict = "not_beneficial"
	// VerdictOverfit: the candidate beat the baseline on the visible set
	// and lost on the holdout set -- it fitted itself to the work it could
	// see rather than solving the problem. Never reported as beneficial.
	VerdictOverfit Verdict = "overfit"
	// VerdictTied: the candidate scored exactly equal to the baseline on
	// the visible set. A tie recommends nothing; it is never broken in the
	// candidate's favour.
	VerdictTied Verdict = "tied"
	// VerdictInconclusive: both the visible set and the holdout set were
	// empty -- there was nothing to compare. Inconclusive never recommends
	// promotion; a comparison with nothing to compare is not a licence.
	VerdictInconclusive Verdict = "inconclusive"
)

// verdictVocabulary is the declared, closed set of every verdict a
// Comparison may report -- same completeness convention this phase already
// uses (recruitmentCreditOutcomeVocabulary, episodeLedgerRecordKindVocabulary):
// a verdict added to the const block above must also be added here.
var verdictVocabulary = []Verdict{
	VerdictBeneficial,
	VerdictNotBeneficial,
	VerdictOverfit,
	VerdictTied,
	VerdictInconclusive,
}

// VerdictNames returns the string form of every declared verdict.
func VerdictNames() []string {
	names := make([]string, 0, len(verdictVocabulary))
	for _, v := range verdictVocabulary {
		names = append(names, string(v))
	}
	return names
}

// VerdictDeclared reports whether v is one of the declared verdicts.
func VerdictDeclared(v Verdict) bool {
	for _, declared := range verdictVocabulary {
		if declared == v {
			return true
		}
	}
	return false
}

// Comparison is the full, four-score result of running a candidate beside a
// baseline over a visible task set and a holdout task set.
type Comparison struct {
	CandidateID      string
	Verdict          Verdict
	VisibleBaseline  Score
	VisibleCandidate Score
	HoldoutBaseline  Score
	HoldoutCandidate Score
	EvaluatorDigest  [32]byte
	BaselineDigest   [32]byte
}

// Recommended reports whether this Comparison recommends promoting the
// candidate. Only VerdictBeneficial recommends promotion -- overfit, tied,
// not-beneficial and inconclusive verdicts never do, whatever their scores
// look like.
func (c Comparison) Recommended() bool {
	return c.Verdict == VerdictBeneficial
}

// runScore runs evaluator over every task in tasks for subject, and returns
// the count that passed over the total.
func runScore(evaluator FrozenEvaluator, subject Candidate, tasks []Task) Score {
	passed := 0
	for _, task := range tasks {
		if evaluator.Run(subject, task).Passed() {
			passed++
		}
	}
	return Score{Numerator: passed, Denominator: len(tasks)}
}

// determineVerdict applies the verdict rules, each named as its own small
// predicate rather than one nested condition, in this order:
//  1. Both the visible set and the holdout set are empty: inconclusive,
//     regardless of anything else -- there was nothing to compare.
//  2. The candidate loses on the visible set (exact comparison): not
//     beneficial, regardless of the holdout result.
//  3. The candidate ties the baseline on the visible set (exact
//     comparison): tied. A tie recommends nothing.
//  4. The candidate beats the baseline on the visible set (exact
//     comparison) and loses on the holdout set: overfit. Never beneficial.
//  5. The candidate beats the baseline on the visible set and does not
//     lose on the holdout set: beneficial.
func determineVerdict(visibleTaskCount, holdoutTaskCount int, visibleBaseline, visibleCandidate, holdoutBaseline, holdoutCandidate Score) Verdict {
	if comparisonHasNothingToCompare(visibleTaskCount, holdoutTaskCount) {
		return VerdictInconclusive
	}
	if candidateLosesOnVisible(visibleCandidate, visibleBaseline) {
		return VerdictNotBeneficial
	}
	if candidateTiesOnVisible(visibleCandidate, visibleBaseline) {
		return VerdictTied
	}
	// The candidate beats the baseline on the visible set from here on.
	if candidateLosesOnHoldout(holdoutCandidate, holdoutBaseline) {
		return VerdictOverfit
	}
	return VerdictBeneficial
}

func comparisonHasNothingToCompare(visibleTaskCount, holdoutTaskCount int) bool {
	return visibleTaskCount == 0 && holdoutTaskCount == 0
}

func candidateLosesOnVisible(visibleCandidate, visibleBaseline Score) bool {
	return scoreCompare(visibleCandidate, visibleBaseline) < 0
}

func candidateTiesOnVisible(visibleCandidate, visibleBaseline Score) bool {
	return scoreCompare(visibleCandidate, visibleBaseline) == 0
}

func candidateLosesOnHoldout(holdoutCandidate, holdoutBaseline Score) bool {
	return scoreCompare(holdoutCandidate, holdoutBaseline) < 0
}

// Compare runs baseline and candidate over the identical visible task set
// and the identical holdout task set, in that order, through evaluator, and
// returns the four scores, the verdict, the grader's digest, the baseline's
// digest and the candidate's identifier.
//
// Compare takes the holdout tasks as a parameter and never resolves them
// itself -- the resolution from digests happens in the calling command
// (cmd/shadow_cmds.go's shadowCompareCmd, via resolveEvalGateHoldouts), so
// this package holds no path to the holdout file and a candidate
// declaration sitting inside this package can reach nothing that names a
// holdout. TestHoldoutResolutionHappensOnlyInTheCommandLayer (cmd package)
// and the plain source-text check the plan's own <verify> command runs both
// prove this package never references a holdout file path.
//
// An expired candidate is refused here and never graded, even if it somehow
// carries a Candidate value (Candidate.Expired is also checked once,
// earlier, at construction in NewCandidate -- this is the second,
// belt-and-braces check at comparison time the plan's own behaviour
// requires). The baseline's digest is captured before and after the four
// runs and compared -- a comparison may not proceed when it changed, though
// because Baseline is an immutable value type this can only ever detect a
// defect in this function itself, never a real mutation from outside.
func Compare(baseline Baseline, candidate Candidate, evaluator FrozenEvaluator, visible []Task, holdout []Task) (Comparison, error) {
	now := time.Now()
	if candidate.Expired(now) {
		return Comparison{}, fmt.Errorf("candidate %q has expired and is refused at comparison time -- an expired candidate is never graded", candidate.ID())
	}

	baselineDigestBefore := baseline.Digest()
	baselineSubject := baselineAsCandidate(baseline)

	visibleBaseline := runScore(evaluator, baselineSubject, visible)
	visibleCandidate := runScore(evaluator, candidate, visible)
	holdoutBaseline := runScore(evaluator, baselineSubject, holdout)
	holdoutCandidate := runScore(evaluator, candidate, holdout)

	baselineDigestAfter := baseline.Digest()
	if baselineDigestAfter != baselineDigestBefore {
		return Comparison{}, fmt.Errorf("baseline digest changed during the comparison -- refusing to report a verdict against a baseline that did not stay frozen")
	}

	verdict := determineVerdict(len(visible), len(holdout), visibleBaseline, visibleCandidate, holdoutBaseline, holdoutCandidate)

	return Comparison{
		CandidateID:      candidate.ID(),
		Verdict:          verdict,
		VisibleBaseline:  visibleBaseline,
		VisibleCandidate: visibleCandidate,
		HoldoutBaseline:  holdoutBaseline,
		HoldoutCandidate: holdoutCandidate,
		EvaluatorDigest:  evaluator.Digest(),
		BaselineDigest:   baselineDigestAfter,
	}, nil
}
