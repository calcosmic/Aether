package cmd

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func sealOutcome199Facts() LifecycleFacts {
	goal := "Restore honest closure"
	name := "Atlas"
	taskOne := "1.1"
	taskTwo := "2.1"
	state := colony.ColonyState{
		Goal:         &goal,
		ColonyName:   &name,
		State:        colony.StateREADY,
		CurrentPhase: 2,
		Plan: colony.Plan{Phases: []colony.Phase{
			{ID: 1, Name: "Foundation", Status: colony.PhaseCompleted, Tasks: []colony.Task{{ID: &taskOne, Goal: "Build the foundation", Status: colony.TaskCompleted}}},
			{ID: 2, Name: "Closure", Status: colony.PhaseCompleted, Tasks: []colony.Task{{ID: &taskTwo, Goal: "Prove closure", Status: colony.TaskCompleted}}},
		}},
		GateResults: []colony.GateResultEntry{
			{Name: "tests", Passed: true, Timestamp: "2026-09-04T10:00:00Z", Detail: "go test ./... passed"},
			{Name: "security", Passed: true, Timestamp: "2026-09-04T10:01:00Z", Detail: "no blocking findings"},
		},
	}
	confirmed := func(domain, path string) LifecycleFactSource {
		return lifecycleSource(domain, path, LifecycleFactConfirmed, "")
	}
	now := time.Date(2026, 9, 4, 10, 2, 0, 0, time.UTC)
	return LifecycleFacts{
		Root:       "/repo",
		CapturedAt: now,
		State:      LifecycleFact[colony.ColonyState]{Value: state, Source: confirmed("state", ".aether/data/COLONY_STATE.json")},
		Identity: LifecycleFact[LifecycleIdentityFacts]{
			Value:  LifecycleIdentityFacts{Name: name, Goal: goal, Standing: string(state.State)},
			Source: confirmed("identity", ".aether/data/COLONY_STATE.json"),
		},
		Progress: LifecycleFact[LifecycleProgressFacts]{
			Value:  LifecycleProgressFacts{CurrentPhase: state.CurrentPhase, Phases: state.Plan.Phases},
			Source: confirmed("progress", ".aether/data/COLONY_STATE.json"),
		},
		Actors:   LifecycleFact[[]LifecycleActorFact]{Value: []LifecycleActorFact{}, Source: confirmed("actors", ".aether/data/spawn-tree.txt")},
		Signals:  LifecycleFact[[]colony.PheromoneSignal]{Value: []colony.PheromoneSignal{}, Source: confirmed("signals", ".aether/data/pheromones.json")},
		Research: LifecycleFact[LifecycleResearchFacts]{Source: confirmed("research", ".aether/research")},
		Memory: LifecycleFact[LifecycleMemoryFacts]{
			Value:  LifecycleMemoryFacts{State: state.Memory},
			Source: confirmed("memory", ".aether/data/instincts.json"),
		},
		Verification: LifecycleFact[LifecycleVerificationFacts]{
			Value: LifecycleVerificationFacts{
				Gates:     state.GateResults,
				Artifacts: []string{".aether/data/build/phase-2/verification.json", ".aether/data/seal/final-review.json"},
			},
			Source: confirmed("verification", ".aether/data/build"),
		},
		Timing:       LifecycleFact[LifecycleTimingFacts]{Value: LifecycleTimingFacts{CapturedAt: now}, Source: confirmed("timing", ".aether/data/COLONY_STATE.json")},
		ReportedCost: LifecycleFact[LifecycleReportedCostFacts]{Source: confirmed("reported cost", ".aether/data/spend")},
		History:      LifecycleFact[[]string]{Value: []string{"verification completed"}, Source: confirmed("history", ".aether/data/COLONY_STATE.json")},
		Blockers: LifecycleFact[[]colony.FlagEntry]{
			Value:  []colony.FlagEntry{{ID: "owner-checkpoint-1", Type: "owner_confirmation", Description: "Owner accepted the release evidence", Resolved: true}},
			Source: confirmed("blockers", ".aether/data/pending-decisions.json"),
		},
		Evidence: LifecycleFact[LifecycleEvidenceFacts]{Source: confirmed("evidence", ".aether/data/COLONY_STATE.json")},
		Session:  LifecycleFact[colony.SessionFile]{Source: confirmed("session", ".aether/data/session.json")},
	}
}

func TestSealOutcome199VerifiedPreflight(t *testing.T) {
	preflight, err := BuildSealPreflight(sealOutcome199Facts(), SealPreflightRequest{Caller: SealCallerDirectOwner})
	if err != nil {
		t.Fatalf("verified preflight failed: %v", err)
	}
	if err := preflight.Validate(); err != nil {
		t.Fatalf("verified preflight is invalid: %v\n%+v", err, preflight)
	}
	if !preflight.Eligible || preflight.Disposition != colony.SealDispositionVerified || preflight.OutcomeKind != colony.OutcomeKindVerifiedCompletion {
		t.Fatalf("verified discriminator = %+v", preflight)
	}
	if got, want := preflight.CompletedPhaseIDs, []int{1, 2}; !reflect.DeepEqual(got, want) {
		t.Fatalf("completed phases = %#v, want %#v", got, want)
	}
	if got, want := preflight.CompletedTaskIDs, []string{"1.1", "2.1"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("completed tasks = %#v, want %#v", got, want)
	}
	if len(preflight.PassedGates) != 2 || len(preflight.FailedGates) != 0 || len(preflight.SkippedGates) != 0 {
		t.Fatalf("gate partition = passed:%#v failed:%#v skipped:%#v", preflight.PassedGates, preflight.FailedGates, preflight.SkippedGates)
	}
	if len(preflight.Evidence) < 3 || len(preflight.MissingEvidence) != 0 {
		t.Fatalf("verified evidence = present:%#v missing:%#v", preflight.Evidence, preflight.MissingEvidence)
	}
	if len(preflight.OwnerCheckpoints) != 1 || !preflight.OwnerCheckpoints[0].Resolved {
		t.Fatalf("owner checkpoints = %#v", preflight.OwnerCheckpoints)
	}
	if len(preflight.ResidualRisks) != 0 || len(preflight.UnresolvedItems) != 0 || preflight.Rollback != nil || preflight.OwnerReason != "" {
		t.Fatalf("verified preflight contains forced/open fields: %+v", preflight)
	}
	if got := sealPreservedContentIDs(preflight.PreservedContents); !reflect.DeepEqual(got, []string{"active_state", "crowned_record", "findings", "learnings", "lifecycle_evidence", "owner_checkpoints", "rollback", "signals", "wisdom"}) {
		t.Fatalf("preserved contents = %#v", got)
	}
	if preflight.PrimaryNext != "aether status" || preflight.OptionalNext != "aether entomb" {
		t.Fatalf("post-seal actions = %q / %q", preflight.PrimaryNext, preflight.OptionalNext)
	}
}

func TestSealOutcome199IncompleteRefusal(t *testing.T) {
	facts := sealOutcome199Facts()
	facts.State.Value.Plan.Phases[1].Status = colony.PhaseInProgress
	facts.State.Value.Plan.Phases[1].Tasks[0].Status = colony.TaskPending
	facts.Progress.Value = LifecycleProgressFacts{CurrentPhase: 2, Phases: facts.State.Value.Plan.Phases}
	facts.State.Value.GateResults = []colony.GateResultEntry{
		{Name: "tests", Passed: false, Detail: "tests failed"},
		{Name: "accessibility", Passed: false, Detail: "skipped by owner"},
	}
	facts.Verification.Value.Gates = facts.State.Value.GateResults
	facts.Verification.Source = lifecycleSource("verification", ".aether/data/build", LifecycleFactMalformed, "verification report cannot be decoded")
	facts.Blockers.Value = []colony.FlagEntry{{ID: "owner-evidence", Type: "blocker", Description: "Owner acceptance evidence is still queued"}}

	preflight, err := BuildSealPreflight(facts, SealPreflightRequest{Caller: SealCallerDirectOwner})
	if err == nil {
		t.Fatalf("incomplete normal seal unexpectedly became eligible: %+v", preflight)
	}
	if preflight.Eligible || preflight.Disposition != colony.SealDispositionVerified {
		t.Fatalf("refused preflight lost its intended normal disposition: %+v", preflight)
	}
	if !reflect.DeepEqual(preflight.IncompletePhaseIDs, []int{2}) || !reflect.DeepEqual(preflight.IncompleteTaskIDs, []string{"2.1"}) {
		t.Fatalf("incomplete work was not enumerated: phases=%#v tasks=%#v", preflight.IncompletePhaseIDs, preflight.IncompleteTaskIDs)
	}
	if len(preflight.FailedGates) != 1 || preflight.FailedGates[0].Name != "tests" || len(preflight.SkippedGates) != 1 || preflight.SkippedGates[0].Name != "accessibility" {
		t.Fatalf("failed/skipped gates = %#v / %#v", preflight.FailedGates, preflight.SkippedGates)
	}
	if len(preflight.MissingEvidence) == 0 || len(preflight.ResidualRisks) == 0 || len(preflight.UnresolvedItems) < 5 {
		t.Fatalf("refusal hid unresolved evidence: missing=%#v risks=%#v unresolved=%#v", preflight.MissingEvidence, preflight.ResidualRisks, preflight.UnresolvedItems)
	}
	if !strings.Contains(err.Error(), "normal seal requires verified completion") || !strings.Contains(err.Error(), "phase 2") || !strings.Contains(err.Error(), "2.1") {
		t.Fatalf("refusal did not enumerate incomplete work: %v", err)
	}
}

func TestSealOutcome199ForceAuthority(t *testing.T) {
	facts := sealOutcome199Facts()
	facts.State.Value.Plan.Phases[1].Status = colony.PhaseInProgress
	facts.State.Value.Plan.Phases[1].Tasks[0].Status = colony.TaskPending
	facts.Progress.Value.Phases = facts.State.Value.Plan.Phases

	for _, tc := range []struct {
		name    string
		request SealPreflightRequest
		want    string
	}{
		{name: "blank reason", request: SealPreflightRequest{Caller: SealCallerDirectOwner, Force: true}, want: "nonblank --reason"},
		{name: "wrapper", request: SealPreflightRequest{Caller: SealCallerWrapper, Force: true, Reason: "owner asked"}, want: "direct owner"},
		{name: "finalizer", request: SealPreflightRequest{Caller: SealCallerFinalizer, Force: true, Reason: "owner asked"}, want: "direct owner"},
		{name: "autopilot", request: SealPreflightRequest{Caller: SealCallerAutopilot, Force: true, Reason: "owner asked"}, want: "direct owner"},
		{name: "worker", request: SealPreflightRequest{Caller: SealCallerWorker, Force: true, Reason: "owner asked"}, want: "direct owner"},
		{name: "recovery", request: SealPreflightRequest{Caller: SealCallerRecovery, Force: true, Reason: "owner asked"}, want: "direct owner"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			preflight, err := BuildSealPreflight(facts, tc.request)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("force authority result = %+v, err=%v, want error containing %q", preflight, err, tc.want)
			}
			if preflight.Eligible {
				t.Fatalf("unauthorized force became eligible: %+v", preflight)
			}
		})
	}

	preflight, err := BuildSealPreflight(facts, SealPreflightRequest{
		Caller: SealCallerDirectOwner,
		Force:  true,
		Reason: "owner accepts the unfinished closure record",
	})
	if err != nil {
		t.Fatalf("direct owner force failed: %v", err)
	}
	if err := preflight.Validate(); err != nil {
		t.Fatalf("forced preflight invalid: %v\n%+v", err, preflight)
	}
	if !preflight.Eligible || preflight.Disposition != colony.SealDispositionForcedIncomplete || preflight.OutcomeKind != colony.OutcomeKindForcedIncompleteClosure {
		t.Fatalf("forced discriminator = %+v", preflight)
	}
	if preflight.OwnerReason != "owner accepts the unfinished closure record" || preflight.Rollback == nil || len(preflight.Rollback.Evidence) == 0 {
		t.Fatalf("forced authority/rollback not recorded: %+v", preflight)
	}
	if len(preflight.UnresolvedItems) == 0 || len(preflight.IncompletePhaseIDs) != 1 || len(preflight.IncompleteTaskIDs) != 1 {
		t.Fatalf("forced closure hid residual work: %+v", preflight)
	}
}

func TestSealOutcome199ConfirmCopy(t *testing.T) {
	verified, err := BuildSealPreflight(sealOutcome199Facts(), SealPreflightRequest{Caller: SealCallerDirectOwner})
	if err != nil {
		t.Fatal(err)
	}
	if got, want := SealConfirmationCopy(verified), "Seal this verified colony and write its Crowned Anthill record? [y/N]"; got != want {
		t.Fatalf("verified confirmation = %q, want %q", got, want)
	}

	facts := sealOutcome199Facts()
	facts.State.Value.Plan.Phases[1].Status = colony.PhaseInProgress
	facts.State.Value.Plan.Phases[1].Tasks[0].Status = colony.TaskPending
	facts.Progress.Value.Phases = facts.State.Value.Plan.Phases
	forced, err := BuildSealPreflight(facts, SealPreflightRequest{Caller: SealCallerDirectOwner, Force: true, Reason: "owner override"})
	if err != nil {
		t.Fatal(err)
	}
	want := "Force-seal this incomplete colony with 2 unresolved item(s)? This records an owner override; it does not verify completion. [y/N]"
	if got := SealConfirmationCopy(forced); got != want {
		t.Fatalf("forced confirmation = %q, want %q", got, want)
	}
}

func sealPreservedContentIDs(contents []SealPreservedContent) []string {
	ids := make([]string, 0, len(contents))
	for _, content := range contents {
		ids = append(ids, content.ID)
	}
	return ids
}
