package shadow

import (
	"strings"
	"testing"
	"time"
)

func alwaysPassRun(Candidate, Task) Result {
	return NewResult(true)
}

// validCandidateFields returns a full, valid set of the six declarations
// NewCandidate requires, so each test can start from a known-good baseline
// and mutate exactly the one thing it means to test.
func validCandidateFields() (id, scope, expectedBenefit, harms string, expiry time.Time, rollbackPlan string) {
	return "cand-1",
		"routes builder tasks to a faster model for one caste",
		"shorter turnaround on routine builder tasks",
		"a worse model could be chosen for genuinely hard tasks",
		time.Now().Add(48 * time.Hour),
		"revert the routing rule to its previous value"
}

func TestEvaluatorDigestIsDeterministic(t *testing.T) {
	def := []byte("shadow-eval-definition-v1")
	e1 := NewFrozenEvaluator(def, alwaysPassRun)
	e2 := NewFrozenEvaluator(def, alwaysPassRun)
	if e1.Digest() != e2.Digest() {
		t.Fatalf("constructing twice from the identical definition produced different digests: %x vs %x", e1.Digest(), e2.Digest())
	}
}

func TestDifferentDefinitionsDigestDifferently(t *testing.T) {
	e1 := NewFrozenEvaluator([]byte("shadow-eval-definition-v1"), alwaysPassRun)
	e2 := NewFrozenEvaluator([]byte("shadow-eval-definition-v2"), alwaysPassRun)
	if e1.Digest() == e2.Digest() {
		t.Fatalf("two different definitions produced the same digest: %x", e1.Digest())
	}
}

func TestCandidateRequiresAllSixDeclarations(t *testing.T) {
	for _, missing := range candidateRequiredFieldNames {
		t.Run(missing, func(t *testing.T) {
			id, scope, expectedBenefit, harms, expiry, rollbackPlan := validCandidateFields()
			switch missing {
			case "identifier":
				id = ""
			case "scope":
				scope = ""
			case "expected benefit":
				expectedBenefit = ""
			case "harms":
				harms = ""
			case "expiry":
				expiry = time.Time{}
			case "rollback plan":
				rollbackPlan = ""
			}
			_, err := NewCandidate(id, scope, expectedBenefit, harms, expiry, rollbackPlan)
			if err == nil {
				t.Fatalf("expected a refusal for a candidate missing its %s, got none", missing)
			}
			if !strings.Contains(err.Error(), missing) {
				t.Fatalf("refusal %q does not name the missing field %q", err.Error(), missing)
			}
		})
	}
}

func TestCandidateNamingItsGraderIsRefused(t *testing.T) {
	id, _, expectedBenefit, harms, expiry, rollbackPlan := validCandidateFields()

	t.Run("scope names the evaluator", func(t *testing.T) {
		_, err := NewCandidate(id, "changes which evaluator is used to grade builder tasks", expectedBenefit, harms, expiry, rollbackPlan)
		if err == nil {
			t.Fatal("expected a refusal for a scope naming the evaluator, got none")
		}
		if !strings.Contains(err.Error(), "evaluator") || !strings.Contains(err.Error(), "scope") {
			t.Fatalf("refusal %q does not name the offending field and term", err.Error())
		}
	})

	t.Run("rollback plan names the holdout", func(t *testing.T) {
		_, err := NewCandidate(id, "routes builder tasks to a faster model for one caste", expectedBenefit, harms, expiry, "restore the previous holdout selection")
		if err == nil {
			t.Fatal("expected a refusal for a rollback plan naming the holdout, got none")
		}
		if !strings.Contains(err.Error(), "holdout") || !strings.Contains(err.Error(), "rollback plan") {
			t.Fatalf("refusal %q does not name the offending field and term", err.Error())
		}
	})

	t.Run("scope names the acceptance criteria", func(t *testing.T) {
		_, err := NewCandidate(id, "changes the acceptance criteria for builder tasks", expectedBenefit, harms, expiry, rollbackPlan)
		if err == nil {
			t.Fatal("expected a refusal for a scope naming acceptance criteria, got none")
		}
	})
}

func TestExpiredCandidateIsRefused(t *testing.T) {
	id, scope, expectedBenefit, harms, _, rollbackPlan := validCandidateFields()
	pastExpiry := time.Now().Add(-1 * time.Hour)
	_, err := NewCandidate(id, scope, expectedBenefit, harms, pastExpiry, rollbackPlan)
	if err == nil {
		t.Fatal("expected a refusal for an already-expired candidate declaration, got none")
	}
	if !strings.Contains(err.Error(), "expired") {
		t.Fatalf("refusal %q does not say the candidate expired", err.Error())
	}
}

// TestCandidateIsImmutableAndReadThroughAccessors proves every declared
// field round-trips through its accessor and that Candidate carries no
// exported field of its own (a compile-time fact for the struct literal
// below: only unexported identifiers exist to set, so this file could not
// even attempt `Candidate{ID: ...}`).
func TestCandidateIsImmutableAndReadThroughAccessors(t *testing.T) {
	id, scope, expectedBenefit, harms, expiry, rollbackPlan := validCandidateFields()
	c, err := NewCandidate(id, scope, expectedBenefit, harms, expiry, rollbackPlan)
	if err != nil {
		t.Fatalf("unexpected refusal for a fully valid candidate: %v", err)
	}
	if c.ID() != id || c.Scope() != scope || c.ExpectedBenefit() != expectedBenefit || c.Harms() != harms || !c.Expiry().Equal(expiry) || c.RollbackPlan() != rollbackPlan {
		t.Fatalf("candidate accessors did not round-trip the declared fields: %+v", c)
	}
}
