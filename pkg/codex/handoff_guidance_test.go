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
