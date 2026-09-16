package cmd

import (
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestLifecycleProjectionStateTable(t *testing.T) {
	tests := []struct {
		name        string
		facts       LifecycleFacts
		wantAction  string
		wantRuntime string
		wantChoices []string
		wantClosure string
		wantOutcome colony.OutcomeKind
	}{
		{name: "empty repository", facts: projectionEmptyFacts(), wantAction: "initialize", wantRuntime: `aether init "goal"`, wantClosure: "open", wantOutcome: colony.OutcomeKindNoChange},
		{name: "accepted goal no plan", facts: projectionFacts(projectionState(colony.StateREADY, false, false)), wantAction: "plan", wantRuntime: "aether plan", wantClosure: "open", wantOutcome: colony.OutcomeKindNoChange},
		{name: "accepted plan", facts: projectionFacts(projectionState(colony.StateREADY, true, false)), wantAction: "choose_execution_mode", wantChoices: []string{"build", "run"}, wantClosure: "open", wantOutcome: colony.OutcomeKindNoChange},
		{name: "executing", facts: projectionFacts(projectionState(colony.StateEXECUTING, true, false)), wantAction: "continue", wantRuntime: "aether continue", wantClosure: "open", wantOutcome: colony.OutcomeKindInProgress},
		{name: "paused", facts: projectionFacts(projectionState(colony.StateREADY, true, true)), wantAction: "resume", wantRuntime: "aether resume", wantClosure: "open", wantOutcome: colony.OutcomeKindPaused},
		{name: "blocked", facts: projectionBlockedFacts(), wantAction: "resume", wantRuntime: "aether resume", wantClosure: "open", wantOutcome: colony.OutcomeKindRecoveryRequired},
		{name: "verified complete", facts: projectionFacts(projectionState(colony.StateCOMPLETED, true, false)), wantAction: "seal", wantRuntime: "aether seal", wantClosure: "ready_to_seal", wantOutcome: colony.OutcomeKindCompleted},
		{name: "forced incomplete", facts: projectionSealedFacts(colony.SealDispositionForcedIncomplete), wantAction: "inspect_sealed", wantRuntime: "aether status", wantClosure: "forced_incomplete", wantOutcome: colony.OutcomeKindForcedIncompleteClosure},
		{name: "sealed", facts: projectionSealedFacts(colony.SealDispositionVerified), wantAction: "inspect_sealed", wantRuntime: "aether status", wantClosure: "verified", wantOutcome: colony.OutcomeKindVerifiedCompletion},
		{name: "malformed evidence", facts: projectionMalformedEvidenceFacts(), wantAction: "resume", wantRuntime: "aether resume", wantClosure: "unknown", wantOutcome: colony.OutcomeKindRecoveryRequired},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := projectLifecycle(tc.facts, LifecycleViewFull, "codex")
			if got.NextAction.ID != tc.wantAction || got.NextAction.RuntimeCommand != tc.wantRuntime {
				t.Fatalf("next action = %#v, want id=%q runtime=%q", got.NextAction, tc.wantAction, tc.wantRuntime)
			}
			var choiceIDs []string
			for _, choice := range got.NextAction.Choices {
				choiceIDs = append(choiceIDs, choice.ID)
			}
			if !reflect.DeepEqual(choiceIDs, tc.wantChoices) {
				t.Fatalf("choice ids = %#v, want %#v", choiceIDs, tc.wantChoices)
			}
			if got.Closure.Status != tc.wantClosure || got.OutcomeKind != tc.wantOutcome {
				t.Fatalf("closure/outcome = %#v/%q, want %q/%q", got.Closure, got.OutcomeKind, tc.wantClosure, tc.wantOutcome)
			}
		})
	}
}

func TestLifecycleProjectionIsDeterministicAcrossViews(t *testing.T) {
	facts := projectionFacts(projectionState(colony.StateREADY, true, false))
	first := projectLifecycle(facts, LifecycleViewFull, "codex")
	second := projectLifecycle(facts, LifecycleViewFull, "codex")
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("same facts and revision produced different projections\nfirst=%#v\nsecond=%#v", first, second)
	}
	if first.ProjectionRevision != LifecycleProjectionRevision {
		t.Fatalf("projection revision = %q, want %q", first.ProjectionRevision, LifecycleProjectionRevision)
	}

	for _, view := range []LifecycleView{LifecycleViewCompact, LifecycleViewFocused, LifecycleViewJSON, LifecycleViewVisual} {
		got := projectLifecycle(facts, view, "codex")
		if !reflect.DeepEqual(got.Identity, first.Identity) || !reflect.DeepEqual(got.Phase, first.Phase) ||
			!reflect.DeepEqual(got.Tasks, first.Tasks) || !reflect.DeepEqual(got.Blockers, first.Blockers) ||
			!reflect.DeepEqual(got.NextAction, first.NextAction) {
			t.Fatalf("view %q recomputed semantic fields instead of selecting them", view)
		}
	}
}

func TestLifecycleProjectionSpecAndPlanFactsMatchSnapshot(t *testing.T) {
	state, _ := validCurrentPlanningState(t)
	state.Plan.Revisions = append([]colony.PlanRevision(nil), state.Plan.Revisions...)
	state.Plan.Revisions[len(state.Plan.Revisions)-1].AffectedSemanticIDs = []string{"task:reconcile"}
	facts := lifecycleFactsFromStateSnapshot(state, false, time.Date(2026, 9, 7, 12, 0, 0, 0, time.UTC))

	terminal := projectLifecycle(facts, LifecycleViewVisual, "codex")
	machine := projectLifecycle(facts, LifecycleViewJSON, "codex")
	if !reflect.DeepEqual(terminal.Intent, facts.Intent) || !reflect.DeepEqual(machine.Intent, facts.Intent) {
		t.Fatalf("intent projection was recomputed instead of copied from the snapshot")
	}
	if !reflect.DeepEqual(terminal.Specification, facts.Specification) || !reflect.DeepEqual(machine.Specification, facts.Specification) {
		t.Fatalf("specification projection was recomputed instead of copied from the snapshot")
	}
	if !reflect.DeepEqual(terminal.Planning, facts.Planning) || !reflect.DeepEqual(machine.Planning, facts.Planning) {
		t.Fatalf("planning projection was recomputed instead of copied from the snapshot")
	}
	if terminal.Specification.Value.Approved == terminal.Planning.Value.AcceptedPlan {
		// The fixture intentionally carries an affected accepted plan: approval
		// remains true while build readiness is independently blocked by impact.
		if terminal.Planning.Value.AcceptanceBindingStatus != LifecyclePlanBindingAffected {
			t.Fatalf("approved specification and affected accepted plan were conflated: %+v", terminal.Planning.Value)
		}
	}
}

func TestLifecycleProjectionAcceptedPlanChoicesAreCoequal(t *testing.T) {
	facts := projectionFacts(projectionState(colony.StateREADY, true, false))
	for _, platform := range []struct {
		name      string
		wantBuild string
		wantRun   string
	}{
		{name: "codex", wantBuild: "$ant-build 1", wantRun: "aether run"},
		{name: "claude", wantBuild: "/ant-build 1", wantRun: "/ant-run"},
		{name: "opencode", wantBuild: "/ant-build 1", wantRun: "/ant-run"},
	} {
		t.Run(platform.name, func(t *testing.T) {
			got := projectLifecycle(facts, LifecycleViewFull, platform.name)
			if got.NextAction.ID != "choose_execution_mode" || len(got.NextAction.Choices) != 2 {
				t.Fatalf("accepted plan next action = %#v", got.NextAction)
			}
			build, run := got.NextAction.Choices[0], got.NextAction.Choices[1]
			if build.DisplayCommand != platform.wantBuild || run.DisplayCommand != platform.wantRun {
				t.Fatalf("%s display choices = %q / %q", platform.name, build.DisplayCommand, run.DisplayCommand)
			}
			if build.Rank != run.Rank || build.Recommended || run.Recommended || build.Label == "recommended" || run.Label == "recommended" {
				t.Fatalf("build/run choices are ranked: build=%#v run=%#v", build, run)
			}
			if build.RuntimeCommand != "aether build 1" || run.RuntimeCommand != "aether run" {
				t.Fatalf("platform spelling changed runtime semantics: build=%#v run=%#v", build, run)
			}
		})
	}
}

func TestLifecycleProjectionSectionOrderAndUnknownEvidence(t *testing.T) {
	facts := projectionMalformedEvidenceFacts()
	got := projectLifecycle(facts, LifecycleViewFull, "claude")
	wantOrder := []string{"identity", "progress", "actors", "signals", "research", "memory_evidence", "elapsed_cost", "history", "open_items", "next_action"}
	if ids := lifecycleProjectionSectionIDs(got.Sections); !reflect.DeepEqual(ids, wantOrder) {
		t.Fatalf("full section order = %#v, want %#v", ids, wantOrder)
	}
	if got.Provenance != colony.RecoveryProvenanceUnknown || got.Closure.Status != "unknown" {
		t.Fatalf("malformed evidence was promoted to a claim: provenance=%q closure=%#v", got.Provenance, got.Closure)
	}
	if got.OutcomeKind == colony.OutcomeKindCompleted || got.OutcomeKind == colony.OutcomeKindVerifiedCompletion {
		t.Fatalf("malformed evidence became a success outcome: %q", got.OutcomeKind)
	}
	if got.NextAction.DisplayCommand != "/ant-resume" {
		t.Fatalf("ambiguous recovery route = %q, want /ant-resume", got.NextAction.DisplayCommand)
	}

	sealed := projectLifecycle(projectionSealedFacts(colony.SealDispositionVerified), LifecycleViewCompact, "claude")
	if sealed.NextAction.DisplayCommand != "/ant-status" || len(sealed.Alternatives) == 0 || sealed.Alternatives[0].DisplayCommand != "/ant-entomb" {
		t.Fatalf("sealed projection forces archive instead of inspection: next=%#v alternatives=%#v", sealed.NextAction, sealed.Alternatives)
	}
}

func projectionState(state colony.State, withPlan, paused bool) colony.ColonyState {
	goal := "Ship the classic front door"
	name := "Atlas"
	value := colony.ColonyState{
		Goal: &goal, ColonyName: &name, State: state, CurrentPhase: 1,
		Milestone: "v2", Paused: paused,
	}
	if state == colony.StateEXECUTING {
		started := time.Date(2026, 9, 3, 11, 0, 0, 0, time.UTC)
		value.BuildStartedAt = &started
	}
	if withPlan {
		phaseStatus := colony.PhaseReady
		taskStatus := colony.TaskPending
		if state == colony.StateEXECUTING {
			phaseStatus = colony.PhaseInProgress
			taskStatus = colony.TaskInProgress
		}
		if state == colony.StateCOMPLETED {
			phaseStatus = colony.PhaseCompleted
			taskStatus = colony.TaskCompleted
		}
		id := "1-1"
		value.Plan.Phases = []colony.Phase{{
			ID: 1, Name: "Front door", Status: phaseStatus,
			Tasks: []colony.Task{{ID: &id, Goal: "Restore contract", Status: taskStatus}},
		}}
	}
	return value
}

func projectionFacts(state colony.ColonyState) LifecycleFacts {
	now := time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)
	stateSource := lifecycleSource("state", ".aether/data/COLONY_STATE.json", LifecycleFactConfirmed, "")
	identity := LifecycleIdentityFacts{Goal: lifecycleString(state.Goal), Name: lifecycleString(state.ColonyName), Standing: string(state.State), Milestone: state.Milestone}
	return LifecycleFacts{
		CapturedAt:   now,
		State:        LifecycleFact[colony.ColonyState]{Value: state, Source: stateSource},
		Identity:     LifecycleFact[LifecycleIdentityFacts]{Value: identity, Source: lifecycleDerivedSource("identity", stateSource)},
		Progress:     LifecycleFact[LifecycleProgressFacts]{Value: LifecycleProgressFacts{CurrentPhase: state.CurrentPhase, Phases: state.Plan.Phases}, Source: lifecycleDerivedSource("progress", stateSource)},
		Actors:       LifecycleFact[[]LifecycleActorFact]{Value: []LifecycleActorFact{}, Source: lifecycleSource("actors", ".aether/data/spawn-tree.txt", LifecycleFactConfirmed, "")},
		Signals:      LifecycleFact[[]colony.PheromoneSignal]{Value: []colony.PheromoneSignal{}, Source: lifecycleSource("signals", ".aether/data/pheromones.json", LifecycleFactConfirmed, "")},
		Research:     LifecycleFact[LifecycleResearchFacts]{Source: lifecycleSource("research", ".aether/research", LifecycleFactConfirmed, "")},
		Memory:       LifecycleFact[LifecycleMemoryFacts]{Value: LifecycleMemoryFacts{State: state.Memory}, Source: lifecycleSource("memory", ".aether/data/instincts.json", LifecycleFactConfirmed, "")},
		Verification: LifecycleFact[LifecycleVerificationFacts]{Value: LifecycleVerificationFacts{Gates: state.GateResults}, Source: lifecycleSource("verification", ".aether/data/build", LifecycleFactConfirmed, "")},
		Timing:       LifecycleFact[LifecycleTimingFacts]{Value: LifecycleTimingFacts{CapturedAt: now}, Source: lifecycleDerivedSource("timing", stateSource)},
		ReportedCost: LifecycleFact[LifecycleReportedCostFacts]{Source: lifecycleSource("reported cost", ".aether/data/spend", LifecycleFactMissing, "not reported")},
		History:      LifecycleFact[[]string]{Value: state.Events, Source: lifecycleDerivedSource("history", stateSource)},
		Blockers:     LifecycleFact[[]colony.FlagEntry]{Value: []colony.FlagEntry{}, Source: lifecycleSource("blockers", ".aether/data/pending-decisions.json", LifecycleFactConfirmed, "")},
		Evidence:     LifecycleFact[LifecycleEvidenceFacts]{Value: LifecycleEvidenceFacts{Receipt: state.LifecycleReceipt, Recovery: state.RecoveryProvenance, Seal: state.SealOutcome, Archive: state.ArchiveReference}, Source: lifecycleSource("evidence", ".aether/data/COLONY_STATE.json", LifecycleFactConfirmed, "")},
		Session:      LifecycleFact[colony.SessionFile]{Source: lifecycleSource("session", ".aether/data/session.json", LifecycleFactConfirmed, "")},
	}
}

func projectionEmptyFacts() LifecycleFacts {
	facts := projectionFacts(colony.ColonyState{})
	missing := lifecycleSource("state", ".aether/data/COLONY_STATE.json", LifecycleFactMissing, "source does not exist")
	facts.State.Source = missing
	facts.Identity.Source = lifecycleDerivedSource("identity", missing)
	facts.Progress.Source = lifecycleDerivedSource("progress", missing)
	facts.Evidence.Source = lifecycleDerivedSource("evidence", missing)
	return facts
}

func projectionBlockedFacts() LifecycleFacts {
	facts := projectionFacts(projectionState(colony.StateEXECUTING, true, false))
	facts.Blockers.Value = []colony.FlagEntry{{ID: "block-1", Type: "blocker", Description: "evidence conflict"}}
	return facts
}

func projectionSealedFacts(disposition colony.SealDisposition) LifecycleFacts {
	state := projectionState(colony.StateCOMPLETED, true, false)
	state.SealOutcome = &colony.SealOutcome{Disposition: disposition}
	facts := projectionFacts(state)
	facts.Evidence.Value.Seal = state.SealOutcome
	return facts
}

func projectionMalformedEvidenceFacts() LifecycleFacts {
	facts := projectionFacts(projectionState(colony.StateREADY, true, false))
	facts.Evidence.Source = lifecycleSource("evidence", ".aether/data/COLONY_STATE.json", LifecycleFactMalformed, "invalid receipt")
	return facts
}

func lifecycleProjectionSectionIDs(sections []LifecycleProjectionSection) []string {
	ids := make([]string, 0, len(sections))
	for _, section := range sections {
		ids = append(ids, section.ID)
	}
	return ids
}

func TestCodexAntSkillRuntimeIdentity(t *testing.T) {
	facts := projectionFacts(projectionState(colony.StateREADY, true, false))
	got := projectLifecycle(facts, LifecycleViewJSON, "codex")
	if got.NextAction.ID != "choose_execution_mode" || len(got.NextAction.Choices) != 2 {
		t.Fatalf("action = %+v", got.NextAction)
	}
	build := got.NextAction.Choices[0]
	if build.ID != "build" || build.RuntimeCommand != "aether build 1" || build.DisplayCommand != "$ant-build 1" {
		t.Errorf("build = %+v", build)
	}
	raw, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(raw), `"runtime_command":"aether build 1"`) {
		t.Errorf("machine route missing: %s", raw)
	}
	t.Logf("accepted plan action: %+v", got.NextAction)
	for _, tc := range []struct{ runtime, display string }{
		{`aether init "Ship café 🐜"`, `$ant-init "Ship café 🐜"`},
		{"aether oracle  'quoted topic'  --flag=é ", "$ant-oracle  'quoted topic'  --flag=é "},
		{"aether build 1 --force", "$ant-build 1 --force"},
		{"aether run", "aether run"}, {"aether status", "aether status"},
		{"aether spec", "aether spec"}, {"aether publish --channel dev", "aether publish --channel dev"},
		{"aether maintenance skills inspect", "aether maintenance skills inspect"},
		{"aether plan-finalize --completion-file packet.json", "aether plan-finalize --completion-file packet.json"},
		{"AETHER_OUTPUT_MODE=json aether plan", "AETHER_OUTPUT_MODE=json aether plan"},
		{"AETHER_OUTPUT_MODE=visual aether build 1", "AETHER_OUTPUT_MODE=visual aether build 1"},
		{"FOO=bar AETHER_OUTPUT_MODE=json aether plan-finalize", "FOO=bar AETHER_OUTPUT_MODE=json aether plan-finalize"},
	} {
		t.Run(tc.runtime, func(t *testing.T) {
			action := lifecycleAction("stable-id", tc.runtime, "reason", nil)
			action, alternatives := lifecycleApplyPlatform(action, []LifecycleActionChoice{lifecycleChoice("alternative-id", tc.runtime, "reason")}, "codex")
			if action.ID != "stable-id" || action.RuntimeCommand != tc.runtime || action.DisplayCommand != tc.display {
				t.Errorf("action changed = %+v", action)
			}
			if alternatives[0].ID != "alternative-id" || alternatives[0].RuntimeCommand != tc.runtime || alternatives[0].DisplayCommand != tc.display {
				t.Errorf("alternative changed = %+v", alternatives[0])
			}
			if hint := translateHintCommandsForPlatform(tc.runtime, "codex"); hint != tc.display {
				t.Errorf("hint = %q, want %q", hint, tc.display)
			}
		})
	}
	// Only platform/display fields may differ; sources, IDs, evidence and state facts stay identical.
	other := projectLifecycle(facts, LifecycleViewJSON, "claude")
	other.Platform = got.Platform
	other.NextAction.DisplayCommand = got.NextAction.DisplayCommand
	for i := range other.NextAction.Choices {
		other.NextAction.Choices[i].DisplayCommand = got.NextAction.Choices[i].DisplayCommand
	}
	for i := range other.Alternatives {
		other.Alternatives[i].DisplayCommand = got.Alternatives[i].DisplayCommand
	}
	if !reflect.DeepEqual(other, got) {
		t.Fatal("display spelling changed semantic projection fields")
	}
}
