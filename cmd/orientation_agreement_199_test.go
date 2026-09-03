package cmd

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

type orientationAgreement199Shared struct {
	ProjectionRevision string
	Identity           LifecycleFact[LifecycleIdentityFacts]
	Goal               LifecycleFact[string]
	Standing           LifecycleFact[string]
	Phase              LifecycleFact[LifecyclePhaseProjection]
	Blockers           []colony.LifecycleIssue
	OwnerDecisions     []colony.LifecycleDecision
	NextAction         LifecycleProjectedAction
}

func TestOrientationViewsAgree199(t *testing.T) {
	assertOrientationCommandsLoadFactsOnce199(t)
	root, _, watched := seedLifecycleFactsFixture(t, "missing")
	before := fingerprintLifecycleFactSurfaces(t, root, watched)

	fixtures := []struct {
		name  string
		facts LifecycleFacts
	}{
		{name: "empty", facts: projectionEmptyFacts()},
		{name: "planned", facts: projectionFacts(projectionState(colony.StateREADY, true, false))},
		{name: "executing", facts: projectionFacts(projectionState(colony.StateEXECUTING, true, false))},
		{name: "paused", facts: projectionFacts(projectionState(colony.StateREADY, true, true))},
		{name: "blocked", facts: projectionBlockedFacts()},
		{name: "forced-incomplete", facts: projectionSealedFacts(colony.SealDispositionForcedIncomplete)},
		{name: "sealed", facts: projectionSealedFacts(colony.SealDispositionVerified)},
	}

	for _, fixture := range fixtures {
		t.Run(fixture.name, func(t *testing.T) {
			facts := fixture.facts
			facts.Blockers.Value = append(facts.Blockers.Value, colony.FlagEntry{
				ID: "decision-199", Type: "decision", Description: "Choose the retained review route",
			})

			status := projectLifecycle(facts, LifecycleViewFull, "codex")
			status.Command = "status"
			focused := projectLifecycle(facts, LifecycleViewFocused, "codex")

			phaseDetail, detailMessage := buildLifecyclePhaseProjection(facts, focused, lifecyclePhaseSelectionDetail, 0, false)
			phaseList, listMessage := buildLifecyclePhaseProjection(facts, focused, lifecyclePhaseSelectionList, 0, false)
			phaseAll, allMessage := buildLifecyclePhaseProjection(facts, focused, lifecyclePhaseSelectionAll, 0, false)
			history := buildLifecycleHistoryProjection(facts, focused, "", 20)
			if fixture.name != "empty" && (detailMessage != "" || listMessage != "" || allMessage != "") {
				t.Fatalf("phase builder rejected %s fixture: detail=%q list=%q all=%q", fixture.name, detailMessage, listMessage, allMessage)
			}

			want := orientationAgreement199FromProjection(status)
			for name, got := range map[string]orientationAgreement199Shared{
				"phase detail": orientationAgreement199FromPhase(t, phaseDetail),
				"phase list":   orientationAgreement199FromPhase(t, phaseList),
				"phase all":    orientationAgreement199FromPhase(t, phaseAll),
				"history":      orientationAgreement199FromHistory(history),
			} {
				assertOrientationAgreement199(t, name, want, got)
			}

			_ = renderLifecycleStatus(status, 88)
			_ = renderLifecyclePhase(phaseDetail, 88)
			_ = renderLifecyclePhase(phaseList, 88)
			_ = renderLifecyclePhase(phaseAll, 88)
			_ = renderLifecycleHistory(history)
			assertOrientationAgreement199(t, "history after rendering", want, orientationAgreement199FromHistory(history))
		})
	}

	after := fingerprintLifecycleFactSurfaces(t, root, watched)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("orientation builders or renderers mutated workspace or hub\nbefore: %#v\nafter: %#v", before, after)
	}
}

func orientationAgreement199FromProjection(projection LifecycleProjection) orientationAgreement199Shared {
	return orientationAgreement199Shared{
		ProjectionRevision: projection.ProjectionRevision,
		Identity:           projection.Identity,
		Goal:               projection.Goal,
		Standing:           projection.Standing,
		Phase:              projection.Phase,
		Blockers:           projection.Blockers,
		OwnerDecisions:     projection.OwnerDecisions,
		NextAction:         projection.NextAction,
	}
}

func orientationAgreement199FromHistory(result LifecycleHistoryResult) orientationAgreement199Shared {
	return orientationAgreement199Shared{
		ProjectionRevision: result.ProjectionRevision,
		Identity:           result.Identity,
		Goal:               result.Goal,
		Standing:           result.Standing,
		Phase:              result.Phase,
		Blockers:           result.Blockers,
		OwnerDecisions:     result.OwnerDecisions,
		NextAction:         result.NextAction,
	}
}

func orientationAgreement199FromPhase(t *testing.T, result LifecyclePhaseResult) orientationAgreement199Shared {
	t.Helper()
	payload, err := json.Marshal(result)
	if err != nil {
		t.Fatalf("marshal phase result: %v", err)
	}
	var wire struct {
		ProjectionRevision string                                  `json:"projection_revision"`
		Identity           LifecycleFact[LifecycleIdentityFacts]   `json:"identity"`
		Goal               LifecycleFact[string]                   `json:"goal"`
		Standing           LifecycleFact[string]                   `json:"standing"`
		Phase              LifecycleFact[LifecyclePhaseProjection] `json:"current_phase"`
		Blockers           []colony.LifecycleIssue                 `json:"blockers"`
		OwnerDecisions     []colony.LifecycleDecision              `json:"owner_decisions"`
		NextAction         LifecycleProjectedAction                `json:"next_action"`
	}
	if err := json.Unmarshal(payload, &wire); err != nil {
		t.Fatalf("decode phase shared fields: %v", err)
	}
	return orientationAgreement199Shared{
		ProjectionRevision: wire.ProjectionRevision,
		Identity:           wire.Identity,
		Goal:               wire.Goal,
		Standing:           wire.Standing,
		Phase:              wire.Phase,
		Blockers:           wire.Blockers,
		OwnerDecisions:     wire.OwnerDecisions,
		NextAction:         wire.NextAction,
	}
}

func assertOrientationAgreement199(t *testing.T, name string, want, got orientationAgreement199Shared) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s disagrees with status for the same projection revision\nwant: %#v\n got: %#v", name, want, got)
	}
}

func assertOrientationCommandsLoadFactsOnce199(t *testing.T) {
	t.Helper()
	for _, path := range []string{"status.go", "phase.go", "history.go"} {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		if calls := strings.Count(string(data), "loadLifecycleFacts("); calls != 1 {
			t.Fatalf("%s calls loadLifecycleFacts %d times, want exactly once", path, calls)
		}
	}
}
