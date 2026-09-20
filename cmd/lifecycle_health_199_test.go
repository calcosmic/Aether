package cmd

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestLifecycleHealth199Verified(t *testing.T) {
	facts := lifecycleHealth199Facts()
	projection := projectLifecycle(facts, LifecycleViewFocused, "codex")
	result := projectLifecycleHealth(facts, projection)

	assertLifecycleHealth199State(t, result, LifecycleHealthVerified)
	if !reflect.DeepEqual(result.Identity, projection.Identity) || !reflect.DeepEqual(result.NextAction, projection.NextAction) {
		t.Fatal("health/readiness built a competing identity or Next Up answer")
	}
	if !containsLifecycleHealth199(result.Health.EvidenceIDs, "tests") || result.Health.Timestamp != "2026-09-03T11:45:00Z" {
		t.Fatalf("verified health evidence = %#v", result.Health)
	}
	output := stripANSI(renderLifecycleHealth(result, 80))
	for _, want := range []string{"Health: Verified", "Readiness: Verified", "Evidence: tests", "Next Up", "$ant-continue"} {
		if !strings.Contains(output, want) {
			t.Errorf("verified health output missing %q\n%s", want, output)
		}
	}
}

func TestLifecycleHealth199Blocked(t *testing.T) {
	facts := lifecycleHealth199Facts()
	facts.Blockers.Value = []colony.FlagEntry{{ID: "owner-blocker", Type: "blocker", Description: "Owner decision is required"}}
	projection := projectLifecycle(facts, LifecycleViewFocused, "codex")
	result := projectLifecycleHealth(facts, projection)

	assertLifecycleHealth199State(t, result, LifecycleHealthBlocked)
	if !containsLifecycleHealth199(result.Readiness.EvidenceIDs, "owner-blocker") || !containsLifecycleHealth199(result.Readiness.Reasons, "Owner decision is required") {
		t.Fatalf("blocked readiness omitted reason/evidence: %#v", result.Readiness)
	}
}

func TestLifecycleHealth199Degraded(t *testing.T) {
	facts := lifecycleHealth199Facts()
	facts.Memory.Value.Findings = []colony.ReviewLedgerEntry{{
		ID: "finding-open", Status: "open", Severity: colony.ReviewSeverityMedium, Description: "Follow-up remains",
	}}
	projection := projectLifecycle(facts, LifecycleViewFocused, "codex")
	result := projectLifecycleHealth(facts, projection)

	assertLifecycleHealth199State(t, result, LifecycleHealthDegraded)
	if !containsLifecycleHealth199(result.Health.EvidenceIDs, "finding-open") {
		t.Fatalf("degraded health omitted finding evidence: %#v", result.Health)
	}
}

func TestLifecycleHealth199Unavailable(t *testing.T) {
	facts := lifecycleHealth199Facts()
	facts.Verification.Source = lifecycleSource("verification", ".aether/data/build", LifecycleFactUnavailable, "permission denied")
	projection := projectLifecycle(facts, LifecycleViewFocused, "codex")
	result := projectLifecycleHealth(facts, projection)

	assertLifecycleHealth199State(t, result, LifecycleHealthUnavailable)
	if !containsLifecycleHealth199(result.Health.Reasons, "permission denied") {
		t.Fatalf("unavailable health omitted source reason: %#v", result.Health)
	}
}

func TestLifecycleHealth199Unknown(t *testing.T) {
	facts := lifecycleHealth199Facts()
	facts.Verification.Value = LifecycleVerificationFacts{}
	facts.Verification.Source = lifecycleSource("verification", ".aether/data/build", LifecycleFactMissing, "no verification evidence")
	projection := projectLifecycle(facts, LifecycleViewFocused, "codex")
	result := projectLifecycleHealth(facts, projection)

	assertLifecycleHealth199State(t, result, LifecycleHealthUnknown)
	if !containsLifecycleHealth199(result.Readiness.Reasons, "no verification evidence") {
		t.Fatalf("unknown readiness omitted missing-evidence reason: %#v", result.Readiness)
	}
}

func TestLifecycleHealth199ReadOnly(t *testing.T) {
	for _, fixture := range []string{"valid", "missing", "malformed"} {
		t.Run(fixture, func(t *testing.T) {
			root, factStore, watched := seedLifecycleFactsFixture(t, fixture)
			before := fingerprintLifecycleFactSurfaces(t, root, watched)
			now, _ := time.Parse(time.RFC3339, lifecycleFactsNow)
			facts, err := loadLifecycleFacts(root, factStore, now)
			if err != nil {
				t.Fatalf("load lifecycle facts: %v", err)
			}
			projection := projectLifecycle(facts, LifecycleViewFocused, "codex")
			result := projectLifecycleHealth(facts, projection)
			_ = renderLifecycleHealth(result, 80)
			after := fingerprintLifecycleFactSurfaces(t, root, watched)
			if !reflect.DeepEqual(before, after) {
				t.Fatalf("%s health/readiness mutated workspace or hub\nbefore: %#v\nafter: %#v", fixture, before, after)
			}
		})
	}
}

func lifecycleHealth199Facts() LifecycleFacts {
	state := projectionState(colony.StateBUILT, true, false)
	state.State = colony.StateBUILT
	facts := projectionFacts(state)
	facts.Verification = LifecycleFact[LifecycleVerificationFacts]{
		Value: LifecycleVerificationFacts{Gates: []colony.GateResultEntry{{
			Name: "tests", Passed: true, Timestamp: "2026-09-03T11:45:00Z", Detail: "passed",
		}}},
		Source: lifecycleSource("verification", ".aether/data/build", LifecycleFactConfirmed, ""),
	}
	facts.Blockers = LifecycleFact[[]colony.FlagEntry]{
		Value:  []colony.FlagEntry{},
		Source: lifecycleSource("blockers", ".aether/data/pending-decisions.json", LifecycleFactConfirmed, ""),
	}
	facts.Memory.Value.Findings = nil
	return facts
}

func assertLifecycleHealth199State(t *testing.T, result LifecycleHealthProjection, want LifecycleHealthState) {
	t.Helper()
	if result.Health.Status != want || result.Readiness.Status != want {
		t.Fatalf("health/readiness = %q/%q, want %q\nresult=%#v", result.Health.Status, result.Readiness.Status, want, result)
	}
}

func containsLifecycleHealth199(values []string, want string) bool {
	for _, value := range values {
		if strings.Contains(value, want) {
			return true
		}
	}
	return false
}
