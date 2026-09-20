package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/shadow"
)

// shadowGraderResolveExampleFixtures resolves one guarded and one unguarded
// example fixture from the REAL committed fixture bank at run time -- never
// by ID, never by a hand-typed guard count (204-14 lands in the wave before
// this plan and changes both numbers; this helper must stay true whatever
// it leaves behind). It fails BY NAME, stating which side was empty, if
// either kind cannot be found -- Test 2b's own non-vacuity requirement.
func shadowGraderResolveExampleFixtures(t *testing.T) (guarded regressionFixture, unguarded regressionFixture) {
	t.Helper()
	bank, err := loadFixtureBank()
	if err != nil {
		t.Fatalf("load the real committed fixture bank: %v", err)
	}
	var foundGuarded, foundUnguarded bool
	for _, f := range bank.Fixtures {
		if f.Guard != nil && !foundGuarded {
			guarded = f
			foundGuarded = true
		}
		if f.Guard == nil && !foundUnguarded {
			unguarded = f
			foundUnguarded = true
		}
		if foundGuarded && foundUnguarded {
			break
		}
	}
	if !foundGuarded {
		t.Fatalf("fixture-bank honesty check failed: found zero GUARDED fixtures in the real committed bank -- the classifier tests below cannot exercise the guarded branch")
	}
	if !foundUnguarded {
		t.Fatalf("fixture-bank honesty check failed: found zero UNGUARDED fixtures in the real committed bank -- the classifier tests below cannot exercise the unguarded branch")
	}
	return guarded, unguarded
}

// shadowGraderTestCandidate builds a valid shadow.Candidate for these tests,
// failing the test (rather than the classifier) on a construction refusal.
func shadowGraderTestCandidate(t *testing.T, id, scope, expectedBenefit, harms string) shadow.Candidate {
	t.Helper()
	c, err := shadow.NewCandidate(id, scope, expectedBenefit, harms, time.Now().Add(48*time.Hour), "revert the declared change")
	if err != nil {
		t.Fatalf("construct test candidate %q: %v", id, err)
	}
	return c
}

// shadowGraderBaselineCandidate builds the same internal "baseline" subject
// pkg/shadow's own baselineAsCandidate constructs, via the one path
// available outside that package: shadow.NewCandidate with the literal
// baseline id. Its declared fields carry no subject-matter overlap with any
// real fixture (they are the same "n/a -- reference point" wording
// baselineAsCandidate itself uses), so this stands in for the real baseline
// subject as far as shadowClassifyAgainstBank can tell -- it only ever
// looks at ID() for the baseline branch.
func shadowGraderBaselineCandidate(t *testing.T) shadow.Candidate {
	t.Helper()
	return shadowGraderTestCandidate(t, shadowBaselineCandidateID,
		"the current behaviour, kept unchanged as the point of comparison",
		"n/a -- this is the reference point, not a proposed change",
		"n/a -- this is the reference point, not a proposed change",
	)
}

// shadowGraderBenignHarms is deliberately built from invented, non-English
// tokens so it can never accidentally overlap a real fixture's own
// Title/Invariant subject-matter words -- used whenever a test needs a
// candidate's Harms() to name nothing at all.
const shadowGraderBenignHarms = "zzqvexnil qqbrimtho fnorpplex declared harms placeholder"

// TestTerseFixtureStillHasAQualifyingSubjectWord is WR-03's fix-locking
// test (204-REVIEW.md): a fixture whose Title and Invariant consist ONLY
// of words shorter than shadowClassifierMinWordLength (5) or words on
// shadowClassifierStopWords still produces at least one subject word, via
// shadowFixtureSubjectWords' fallback pass, so it can never become
// permanently un-addressable by any real candidate. Before this fix, such
// a fixture returned zero subject words, silently and with no error --
// shadowTextNamesFixtureSubject would then return false for every possible
// candidate text, forever.
func TestTerseFixtureStillHasAQualifyingSubjectWord(t *testing.T) {
	terse := regressionFixture{
		ID:        "fixture-terse-title-only-short-words",
		Title:     "seal it now",
		Invariant: "keep it safe here",
	}

	// Fixture-honesty precondition: every word above really is either
	// shorter than the length floor or a declared stop word, so this
	// fixture genuinely exercises the fallback path rather than the
	// ordinary length-filtered one.
	for _, w := range strings.Fields(strings.ToLower(terse.Title + " " + terse.Invariant)) {
		if len(w) >= shadowClassifierMinWordLength && !shadowClassifierStopWords[w] {
			t.Fatalf("fixture setup broken: word %q is %d+ letters and not a stop word -- this fixture does not exercise the fallback", w, shadowClassifierMinWordLength)
		}
	}

	words := shadowFixtureSubjectWords(terse)
	if len(words) == 0 {
		t.Fatal("expected at least one fallback subject word for a fixture whose text is entirely short/stop words, got none -- WR-03 regression")
	}

	// The fallback word must genuinely let a real candidate be credited:
	// prove shadowTextNamesFixtureSubject actually returns true for text
	// naming it, not just that the word list itself is non-empty.
	if !shadowTextNamesFixtureSubject("addresses "+strings.Join(words, " "), terse) {
		t.Fatalf("shadowTextNamesFixtureSubject returned false for text naming the fallback words %v -- the fallback is not actually usable", words)
	}
}

// TestEveryBankFixtureHasAQualifyingSubjectWord is WR-03's suggested
// startup/test-time check: every fixture in the REAL committed bank
// carries at least one word shadowFixtureSubjectWords can return, so no
// currently-committed fixture is silently unaddressable by any candidate.
func TestEveryBankFixtureHasAQualifyingSubjectWord(t *testing.T) {
	bank, err := loadFixtureBank()
	if err != nil {
		t.Fatalf("load the real committed fixture bank: %v", err)
	}
	if len(bank.Fixtures) == 0 {
		t.Fatal("fixture-bank honesty check failed: the real committed bank has zero fixtures")
	}
	var unqualified []string
	for _, f := range bank.Fixtures {
		if len(shadowFixtureSubjectWords(f)) == 0 {
			unqualified = append(unqualified, f.ID)
		}
	}
	if len(unqualified) != 0 {
		t.Fatalf("%d fixture(s) in the committed bank have no qualifying subject word at all, making them permanently unaddressable by any candidate: %v", len(unqualified), unqualified)
	}
}

func TestShadowGraderDistinguishesBeneficialFromHarmful(t *testing.T) {
	guardedFixture, unguardedFixture := shadowGraderResolveExampleFixtures(t)

	t.Run("fixture honesty: at least one guarded and one unguarded fixture exist", func(t *testing.T) {
		if guardedFixture.Guard == nil {
			t.Fatal("resolved 'guarded' fixture does not actually carry a guard")
		}
		if unguardedFixture.Guard != nil {
			t.Fatal("resolved 'unguarded' fixture actually carries a guard")
		}
	})

	t.Run("a candidate naming an unguarded fixture's subject beats the baseline and produces beneficial", func(t *testing.T) {
		subjectWords := shadowFixtureSubjectWords(unguardedFixture)
		if len(subjectWords) == 0 {
			t.Fatalf("resolved unguarded fixture %q has no significant subject words to build a matching candidate from", unguardedFixture.ID)
		}
		candidate := shadowGraderTestCandidate(t, "candidate-beneficial",
			string(canaryScopeProjectKnowledge),
			"addresses "+strings.Join(subjectWords, " "),
			shadowGraderBenignHarms,
		)
		baseline := shadowGraderBaselineCandidate(t)

		// shadowEvaluator() -- the real production entrypoint, over the real
		// committed bank -- rather than a synthetic evaluator, so this test
		// genuinely exercises the wired grader (and fails if it is ever
		// reverted to the always-pass placeholder).
		evaluator := shadowEvaluator()

		visible := []shadow.Task{shadow.NewTask(unguardedFixture.ID)}

		if !evaluator.Run(candidate, visible[0]).Passed() {
			t.Fatalf("expected candidate to pass its own named unguarded fixture %q", unguardedFixture.ID)
		}
		if evaluator.Run(baseline, visible[0]).Passed() {
			t.Fatalf("expected baseline to FAIL an unguarded fixture %q", unguardedFixture.ID)
		}

		comparison, err := shadow.Compare(shadow.NewBaseline([]byte("baseline-def")), candidate, evaluator, visible, nil)
		if err != nil {
			t.Fatalf("compare: %v", err)
		}
		if comparison.Verdict != shadow.VerdictBeneficial {
			t.Fatalf("expected VerdictBeneficial, got %q (visible candidate=%+v baseline=%+v)", comparison.Verdict, comparison.VisibleCandidate, comparison.VisibleBaseline)
		}
	})

	t.Run("a candidate whose harms name a guarded fixture's subject loses to the baseline and produces not_beneficial", func(t *testing.T) {
		subjectWords := shadowFixtureSubjectWords(guardedFixture)
		if len(subjectWords) == 0 {
			t.Fatalf("resolved guarded fixture %q has no significant subject words to build a matching candidate from", guardedFixture.ID)
		}
		candidate := shadowGraderTestCandidate(t, "candidate-harmful",
			string(canaryScopeProjectKnowledge),
			"a generic benefit naming nothing in particular",
			"threatens "+strings.Join(subjectWords, " "),
		)
		baseline := shadowGraderBaselineCandidate(t)

		evaluator := shadowEvaluator()

		visible := []shadow.Task{shadow.NewTask(guardedFixture.ID)}

		if evaluator.Run(candidate, visible[0]).Passed() {
			t.Fatalf("expected candidate to FAIL guarded fixture %q because its harms name that fixture's own subject", guardedFixture.ID)
		}
		if !evaluator.Run(baseline, visible[0]).Passed() {
			t.Fatalf("expected baseline to pass a guarded fixture %q", guardedFixture.ID)
		}

		comparison, err := shadow.Compare(shadow.NewBaseline([]byte("baseline-def")), candidate, evaluator, visible, nil)
		if err != nil {
			t.Fatalf("compare: %v", err)
		}
		if comparison.Verdict != shadow.VerdictNotBeneficial {
			t.Fatalf("expected VerdictNotBeneficial, got %q", comparison.Verdict)
		}
	})

	t.Run("an unresolvable task advantages neither subject", func(t *testing.T) {
		candidate := shadowGraderTestCandidate(t, "candidate-unresolvable",
			string(canaryScopeProjectKnowledge), "addresses nothing in particular", shadowGraderBenignHarms)
		baseline := shadowGraderBaselineCandidate(t)
		evaluator := shadowEvaluator()
		task := shadow.NewTask("fixture-id-that-genuinely-does-not-exist-in-the-real-bank-xyzzy")
		if evaluator.Run(candidate, task).Passed() {
			t.Fatal("expected an unresolvable task to fail for the candidate")
		}
		if evaluator.Run(baseline, task).Passed() {
			t.Fatal("expected an unresolvable task to fail for the baseline too")
		}
	})
}

func TestOverfitCandidateIsRefusedAtTheGate(t *testing.T) {
	guardedFixture, unguardedFixture := shadowGraderResolveExampleFixtures(t)

	visibleWords := shadowFixtureSubjectWords(unguardedFixture)
	holdoutWords := shadowFixtureSubjectWords(guardedFixture)
	if len(visibleWords) == 0 || len(holdoutWords) == 0 {
		t.Fatal("resolved example fixtures have no significant subject words to build this scenario from")
	}
	// Filter out any holdout word that would ALSO name the visible fixture's
	// own subject -- two distinct real fixtures can share a generic word
	// (e.g. both titles mentioning "guard"), and the harms text must name
	// ONLY the holdout fixture's subject for this scenario to hold.
	var safeHoldoutWords []string
	for _, w := range holdoutWords {
		if !shadowTextNamesFixtureSubject(w, unguardedFixture) {
			safeHoldoutWords = append(safeHoldoutWords, w)
		}
	}
	if len(safeHoldoutWords) == 0 {
		t.Fatalf("every significant word of guarded fixture %q also names visible fixture %q's own subject -- pick different example fixtures", guardedFixture.ID, unguardedFixture.ID)
	}

	candidate := shadowGraderTestCandidate(t, "candidate-overfit",
		string(canaryScopeRouting),
		"addresses "+strings.Join(visibleWords, " "),
		"threatens "+strings.Join(safeHoldoutWords, " "),
	)
	// Defensive non-collision check: the harms text must not also match the
	// visible (unguarded) fixture, or the "wins on visible" half of this
	// scenario would not hold. Extremely unlikely for two distinct real
	// fixtures, but asserted rather than assumed.
	if shadowTextNamesFixtureSubject(candidate.Harms(), unguardedFixture) {
		t.Fatalf("test setup collision: candidate harms (from guarded fixture %q) also name the visible fixture %q's subject -- pick different example fixtures", guardedFixture.ID, unguardedFixture.ID)
	}

	evaluator := shadowEvaluator()

	visible := []shadow.Task{shadow.NewTask(unguardedFixture.ID)}
	holdout := []shadow.Task{shadow.NewTask(guardedFixture.ID)}

	comparison, err := shadow.Compare(shadow.NewBaseline([]byte("baseline-def")), candidate, evaluator, visible, holdout)
	if err != nil {
		t.Fatalf("compare: %v", err)
	}
	if comparison.Verdict != shadow.VerdictOverfit {
		t.Fatalf("expected VerdictOverfit, got %q (visible candidate=%+v baseline=%+v; holdout candidate=%+v baseline=%+v)",
			comparison.Verdict, comparison.VisibleCandidate, comparison.VisibleBaseline, comparison.HoldoutCandidate, comparison.HoldoutBaseline)
	}

	if _, err := admitCandidateToCanary(candidate, comparison); err == nil {
		t.Fatal("expected an overfit candidate to be refused at the gate")
	} else if !strings.Contains(err.Error(), "overfit") {
		t.Fatalf("refusal does not name overfit: %v", err)
	}
}

func TestEvaluatorDigestIsStableAndChanged(t *testing.T) {
	first := shadowEvaluator().Digest()
	second := shadowEvaluator().Digest()
	if first != second {
		t.Fatalf("shadowEvaluator().Digest() is not stable across two calls: %x vs %x", first, second)
	}

	criteria := shadow.NewAcceptanceCriteria([]byte(shadowAcceptanceCriteriaDefinition))
	criteriaDigest := criteria.Digest()
	oldDef := append([]byte("shadow-evaluator-v1"), criteriaDigest[:]...)
	oldDigest := shadow.NewFrozenEvaluator(oldDef, func(shadow.Candidate, shadow.Task) shadow.Result {
		return shadow.NewResult(true)
	}).Digest()

	if first == oldDigest {
		t.Fatalf("shadowEvaluator().Digest() did not change from the previous always-pass v1 definition: %x", first)
	}
}
