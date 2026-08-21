package codex

import (
	"strings"
	"testing"
)

// TestResponseContractTellsWorkersToRouteJudgementCalls pins the worker-side
// half of the owner-decision relay: a worker blocked on a preference or
// product judgement must record it in open_decisions rather than research
// around it — the wrapper surfaces those questions to the owner at wave
// boundaries, which only works if workers actually write them.
func TestResponseContractTellsWorkersToRouteJudgementCalls(t *testing.T) {
	contract := renderResponseContract(WorkerConfig{Root: ".", Caste: "builder"})
	if !strings.Contains(contract, HandoffOpenDecisionsGuidance) {
		t.Fatalf("response contract missing the open-decisions routing guidance.\ncontract:\n%s", contract)
	}
}

// TestResponseContractOffersNoChangeOutcome: workers can only report an
// honest "nothing needed changing" if the contract tells them the outcome
// exists (ruling D6) — and a rate-limit stop must be reportable as a
// resumable interruption, not a failure (ruling D7).
func TestResponseContractOffersNoChangeOutcome(t *testing.T) {
	for _, caste := range []string{"builder", "probe"} {
		contract := renderResponseContract(WorkerConfig{Root: ".", Caste: caste})
		for _, want := range []string{"completed_no_change", "verified_existing", "NEVER fabricate an edit", "status interrupted"} {
			if !strings.Contains(contract, want) {
				t.Errorf("%s response contract missing %q", caste, want)
			}
		}
	}
}

// TestClaimsNormalizationKeepsHonestStatuses: the in-process claims path used
// to coerce any unknown status to "failed" — a worker truthfully reporting
// completed_no_change was punished for honesty.
func TestClaimsNormalizationKeepsHonestStatuses(t *testing.T) {
	for raw, want := range map[string]string{
		"completed_no_change": "completed_no_change",
		"verified_existing":   "completed_no_change",
		"no_change":           "completed_no_change",
		"interrupted":         "interrupted",
		"rate_limited":        "interrupted",
		"completed":           "completed",
	} {
		claims := workerClaims{Status: raw}
		normalized := normalizeWorkerClaims(claims, WorkerConfig{Root: "."})
		if normalized.Status != want {
			t.Errorf("claims status %q normalized to %q, want %q", raw, normalized.Status, want)
		}
	}
}
