package cmd

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func TestLifecycleCloseout199Contract(t *testing.T) {
	projection := lifecycleCloseout199Projection(colony.StateEXECUTING, "build")
	projection.Actors.Value = []LifecycleActorFact{{Name: "Mason-67", Caste: "builder", Status: "completed"}}
	projection.Evidence = []colony.LifecycleEvidence{{ID: "receipt-1", Kind: "receipt", Source: "receipts/build.json", Summary: "Build receipt verified"}}
	projection.Verification = []colony.LifecycleVerification{{Name: "focused tests", Passed: true, EvidenceIDs: []string{"receipt-1"}}}
	projection.Changes = []colony.LifecycleChange{{Target: "phase 1", Action: "dispatched"}}
	projection.Signals.Value = []colony.PheromoneSignal{{ID: "focus-1", Type: "FOCUS", Active: true, Content: []byte(`{"text":"Keep lifecycle output focused"}`)}}
	projection.Blockers = []colony.LifecycleIssue{{ID: "open-1", Summary: "Owner confirmation remains open"}}

	result := lifecycleCloseout199Result(projection, colony.OutcomeKindInProgress, colony.LifecycleStateEffectCommitted)
	if err := applyLifecycleCloseout(result, "build", LifecycleCloseoutDetails{Summary: "Phase 1 dispatch finished."}); err != nil {
		t.Fatalf("apply closeout: %v", err)
	}
	closeout := lifecycleCloseout199MustResult(t, result)
	wantSlots := []LifecycleCloseoutSlot{
		LifecycleCloseoutColony,
		LifecycleCloseoutParticipants,
		LifecycleCloseoutWhatHappened,
		LifecycleCloseoutEvidence,
		LifecycleCloseoutStateChanges,
		LifecycleCloseoutStandingInstructions,
		LifecycleCloseoutUnresolved,
		LifecycleCloseoutNextUp,
	}
	if !reflect.DeepEqual(closeout.Slots, wantSlots) {
		t.Fatalf("slots = %#v, want %#v", closeout.Slots, wantSlots)
	}
	if closeout.ProjectionRevision != projection.ProjectionRevision || closeout.OutcomeKind != colony.OutcomeKindInProgress || closeout.StateChanges.Effect != colony.LifecycleStateEffectCommitted {
		t.Fatalf("typed closeout drifted: %#v", closeout)
	}

	visual := renderLifecycleCloseout(closeout, "claude")
	labels := []string{"Colony", "Participants", "What happened", "Evidence", "State changes", "Standing instructions", "Unresolved", "Next Up"}
	last := -1
	for _, label := range labels {
		index := strings.Index(visual, label)
		if index < 0 {
			t.Fatalf("rendered closeout omitted %q:\n%s", label, visual)
		}
		if index <= last {
			t.Fatalf("rendered closeout order drifted at %q:\n%s", label, visual)
		}
		last = index
	}
	for _, forbidden := range []string{"Full Status", "Status Dashboard", "Recent Activity", "Memory Details"} {
		if strings.Contains(strings.ToLower(visual), strings.ToLower(forbidden)) {
			t.Fatalf("focused closeout embedded %q:\n%s", forbidden, visual)
		}
	}
}

func TestLifecycleCloseout199Build(t *testing.T) {
	projection := lifecycleCloseout199Projection(colony.StateEXECUTING, "build")
	projection.Evidence = []colony.LifecycleEvidence{{ID: "build-receipt", Kind: "receipt", Summary: "Worker dispatch receipt"}}
	result := lifecycleCloseout199Result(projection, colony.OutcomeKindInProgress, colony.LifecycleStateEffectCommitted)
	if err := applyLifecycleCloseout(result, "build", LifecycleCloseoutDetails{Summary: "The selected phase was dispatched."}); err != nil {
		t.Fatalf("apply build closeout: %v", err)
	}
	got := lifecycleCloseout199MustResult(t, result)
	if got.Command != "build" || got.WhatHappened.Summary != "The selected phase was dispatched." || got.NextUp.ID != projection.NextAction.ID {
		t.Fatalf("build closeout = %#v", got)
	}
	lifecycleCloseout199RequireSourceCall(t, "compatibility_cmds.go", `applyLifecycleCloseout(buildResult, "build"`)
}

func TestLifecycleCloseout199Run(t *testing.T) {
	projection := lifecycleCloseout199Projection(colony.StateEXECUTING, "run")
	projection.Evidence = []colony.LifecycleEvidence{{ID: "run-report", Kind: "report", Summary: "Bounded autopilot report"}}
	result := lifecycleCloseout199Result(projection, colony.OutcomeKindInProgress, colony.LifecycleStateEffectCommitted)
	if err := applyLifecycleCloseout(result, "run", LifecycleCloseoutDetails{Summary: "Autopilot stopped at its declared boundary."}); err != nil {
		t.Fatalf("apply run closeout: %v", err)
	}
	got := lifecycleCloseout199MustResult(t, result)
	if got.ProjectionRevision != stringValue(result["projection_revision"]) || got.NextUp.RuntimeCommand != stringValue(result[nextActionCommandKey]) {
		t.Fatalf("run closeout disagrees with result: closeout=%#v result=%#v", got, result)
	}
	lifecycleCloseout199RequireSourceCall(t, "compatibility_cmds.go", `applyLifecycleCloseout(result, "run"`)
}

func TestLifecycleCloseout199Pause(t *testing.T) {
	projection := lifecycleCloseout199Projection(colony.StateREADY, "pause")
	projection.OutcomeKind = colony.OutcomeKindPaused
	projection.NextAction = LifecycleProjectedAction{ID: "resume", RuntimeCommand: "aether resume", DisplayCommand: "aether resume", Reason: "A validated handoff is ready."}
	projection.Evidence = []colony.LifecycleEvidence{{ID: "pause-receipt", Kind: "receipt", Summary: "Validated safe-boundary handoff"}}
	result := lifecycleCloseout199Result(projection, colony.OutcomeKindPaused, colony.LifecycleStateEffectCommitted)
	if err := applyLifecycleCloseout(result, "pause", LifecycleCloseoutDetails{Summary: "The colony paused at a safe boundary."}); err != nil {
		t.Fatalf("apply pause closeout: %v", err)
	}
	got := lifecycleCloseout199MustResult(t, result)
	if got.OutcomeKind != colony.OutcomeKindPaused || got.NextUp.RuntimeCommand != "aether resume" || len(got.Evidence) == 0 {
		t.Fatalf("pause closeout = %#v", got)
	}
	lifecycleCloseout199RequireSourceCall(t, "session_flow_cmds.go", `applyLifecycleCloseout(result, "pause"`)
}

func TestLifecycleCloseout199Resume(t *testing.T) {
	projection := lifecycleCloseout199Projection(colony.StateREADY, "resume")
	projection.Evidence = []colony.LifecycleEvidence{{ID: "resume-receipt", Kind: "receipt", Summary: "Confirmed handoff recovery"}}
	result := lifecycleCloseout199Result(projection, colony.OutcomeKindResumed, colony.LifecycleStateEffectCommitted)
	if err := applyLifecycleCloseout(result, "resume", LifecycleCloseoutDetails{Summary: "The validated recovery point was restored."}); err != nil {
		t.Fatalf("apply resume closeout: %v", err)
	}
	got := lifecycleCloseout199MustResult(t, result)
	if got.OutcomeKind != colony.OutcomeKindResumed || got.StateChanges.Effect != colony.LifecycleStateEffectCommitted || got.ProjectionRevision != projection.ProjectionRevision {
		t.Fatalf("resume closeout = %#v", got)
	}
	lifecycleCloseout199RequireSourceCall(t, "session_flow_cmds.go", `applyLifecycleCloseout(result, "resume"`)
}

func TestLifecycleCloseout199Refusal(t *testing.T) {
	projection := lifecycleCloseout199Projection(colony.StateREADY, "run")
	projection.NextAction = LifecycleProjectedAction{ID: "status", RuntimeCommand: "aether status", DisplayCommand: "aether status", Reason: "Inspect the retained project before retrying."}
	result := lifecycleCloseout199Result(projection, colony.OutcomeKindRefused, colony.LifecycleStateEffectNone)
	if err := applyLifecycleCloseout(result, "run", LifecycleCloseoutDetails{Summary: "Autopilot refused an unavailable preflight."}); err != nil {
		t.Fatalf("apply refusal closeout: %v", err)
	}
	got := lifecycleCloseout199MustResult(t, result)
	if got.OutcomeKind != colony.OutcomeKindRefused || got.StateChanges.Effect != colony.LifecycleStateEffectNone || got.NextUp.RuntimeCommand != "aether status" {
		t.Fatalf("refusal closeout claims mutation or unsafe action: %#v", got)
	}
	if strings.Contains(renderLifecycleCloseout(got, "codex"), "Full Status") {
		t.Fatal("refusal closeout embedded the status dashboard")
	}
}

func TestLifecycleCloseout199Replay(t *testing.T) {
	projection := lifecycleCloseout199Projection(colony.StateREADY, "pause")
	projection.OutcomeKind = colony.OutcomeKindPaused
	projection.NextAction = LifecycleProjectedAction{ID: "resume", RuntimeCommand: "aether resume", DisplayCommand: "aether resume", Reason: "The existing handoff remains valid."}
	result := lifecycleCloseout199Result(projection, colony.OutcomeKindPaused, colony.LifecycleStateEffectCommitted)
	details := LifecycleCloseoutDetails{Summary: "The existing pause receipt was replayed."}
	if err := applyLifecycleCloseout(result, "pause", details); err != nil {
		t.Fatalf("first replay closeout: %v", err)
	}
	first := lifecycleCloseout199MustResult(t, result)
	if err := applyLifecycleCloseout(result, "pause", details); err != nil {
		t.Fatalf("second replay closeout: %v", err)
	}
	second := lifecycleCloseout199MustResult(t, result)
	if !reflect.DeepEqual(first, second) || renderLifecycleCloseout(first, "codex") != renderLifecycleCloseout(second, "codex") {
		t.Fatalf("replay changed closeout:\nfirst=%#v\nsecond=%#v", first, second)
	}
}

func TestLifecycleCloseout199ZeroWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "COLONY_STATE.json")
	if err := os.WriteFile(path, []byte(`{"state":"READY","sentinel":"unchanged"}`), 0o600); err != nil {
		t.Fatalf("write sentinel: %v", err)
	}
	before := snapshotProjectDataTree(t, dir)

	projection := lifecycleCloseout199Projection(colony.StateREADY, "run")
	result := lifecycleCloseout199Result(projection, colony.OutcomeKindNoChange, colony.LifecycleStateEffectNone)
	if err := applyLifecycleCloseout(result, "run", LifecycleCloseoutDetails{Summary: "Preview completed without changing the colony."}); err != nil {
		t.Fatalf("apply zero-write closeout: %v", err)
	}
	_ = renderLifecycleCloseout(lifecycleCloseout199MustResult(t, result), "claude")
	after := snapshotProjectDataTree(t, dir)
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("closeout construction or rendering wrote project data:\nbefore=%#v\nafter=%#v", before, after)
	}
}

func TestLifecycleCloseout199Init(t *testing.T) {
	projection := projectLifecycle(projectionFacts(projectionState(colony.StateREADY, false, false)), LifecycleViewFocused, "codex")
	projection.Command = "init"
	result := lifecycleCloseout199Result(projection, colony.OutcomeKindCompleted, colony.LifecycleStateEffectCommitted)
	if err := applyLifecycleCloseout(result, "init", LifecycleCloseoutDetails{
		Summary: "The owner-provided charter was accepted and the colony was created.",
		Evidence: []colony.LifecycleEvidence{
			{ID: "accepted-charter", Kind: "charter", Source: "COLONY_STATE.json", Summary: "Persisted accepted charter"},
			{ID: "territory", Kind: "territory", Source: "territory-snapshot.json", Summary: "Territory freshness result"},
		},
	}); err != nil {
		t.Fatalf("apply init closeout: %v", err)
	}
	got := lifecycleCloseout199MustResult(t, result)
	visual := renderLifecycleCloseout(got, "claude")
	if got.Colony.Goal != "Ship the classic front door" || got.StateChanges.Effect != colony.LifecycleStateEffectCommitted || !strings.Contains(visual, "/ant-plan") {
		t.Fatalf("init closeout lost accepted intent, commit truth, or exact next action:\n%#v\n%s", got, visual)
	}
	if strings.Contains(visual, "aether plan") || got.ProjectionRevision != stringValue(result["projection_revision"]) {
		t.Fatalf("init closeout has platform/revision drift:\n%#v\n%s", got, visual)
	}
	lifecycleCloseout199RequireSourceCall(t, "init_cmd.go", `applyLifecycleCloseout(result, "init"`)
}

func TestLifecycleCloseout199Colonize(t *testing.T) {
	projection := projectLifecycle(projectionFacts(projectionState(colony.StateREADY, false, false)), LifecycleViewFocused, "codex")
	projection.Command = "colonize"
	projection.Evidence = []colony.LifecycleEvidence{{ID: "territory-snapshot", Kind: "territory", Source: "territory-snapshot.json", Summary: "Verified immutable territory snapshot"}}
	result := lifecycleCloseout199Result(projection, colony.OutcomeKindCompleted, colony.LifecycleStateEffectCommitted)
	if err := applyLifecycleCloseout(result, "colonize", LifecycleCloseoutDetails{
		Summary:  "The surveyors published the verified territory snapshot.",
		Evidence: []colony.LifecycleEvidence{{ID: "territory-receipt", Kind: "receipt", Summary: "Committed territory transaction receipt"}},
	}); err != nil {
		t.Fatalf("apply colonize closeout: %v", err)
	}
	got := lifecycleCloseout199MustResult(t, result)
	if got.NextUp.RuntimeCommand != "aether plan" || len(got.Evidence) != 2 || got.StateChanges.Effect != colony.LifecycleStateEffectCommitted {
		t.Fatalf("colonize closeout = %#v", got)
	}
	visual := renderLifecycleCloseout(got, "claude")
	if !strings.Contains(visual, "/ant-plan") || strings.Contains(strings.ToLower(visual), "status dashboard") {
		t.Fatalf("colonize focused closeout drifted:\n%s", visual)
	}
	lifecycleCloseout199RequireSourceCall(t, "codex_colonize_finalize.go", `applyLifecycleCloseout(result, "colonize"`)
}

func TestLifecycleCloseout199Plan(t *testing.T) {
	projection := lifecycleCloseout199Projection(colony.StateREADY, "plan")
	projection.Evidence = []colony.LifecycleEvidence{{ID: "plan-revision", Kind: "plan", Source: "COLONY_STATE.json", Summary: "Accepted plan revision"}}
	result := lifecycleCloseout199Result(projection, colony.OutcomeKindCompleted, colony.LifecycleStateEffectCommitted)
	if err := applyLifecycleCloseout(result, "plan", LifecycleCloseoutDetails{
		Summary:  "Scout and Route-Setter evidence produced an accepted plan.",
		Evidence: []colony.LifecycleEvidence{{ID: "plan-receipt", Kind: "receipt", Summary: "Committed planning receipt"}},
		Blockers: []colony.LifecycleIssue{{ID: "gap-1", Summary: "One recorded planning gap remains visible"}},
	}); err != nil {
		t.Fatalf("apply plan closeout: %v", err)
	}
	got := lifecycleCloseout199MustResult(t, result)
	if len(got.NextUp.Choices) != 2 || got.NextUp.Choices[0].ID != "build" || got.NextUp.Choices[1].ID != "run" {
		t.Fatalf("accepted plan lost coequal build/run choice: %#v", got.NextUp)
	}
	if !lifecycleCloseoutHasOpenItems(got.Unresolved) || got.ProjectionRevision != projection.ProjectionRevision {
		t.Fatalf("plan closeout lost gap or revision: %#v", got)
	}
	lifecycleCloseout199RequireSourceCall(t, "codex_plan_finalize.go", `applyLifecycleCloseout(result, "plan"`)
}

func TestLifecycleCloseout199FrontDoorRefusal(t *testing.T) {
	dir := t.TempDir()
	statePath := filepath.Join(dir, "COLONY_STATE.json")
	if err := os.WriteFile(statePath, []byte(`{"goal":"existing","state":"READY"}`), 0o600); err != nil {
		t.Fatalf("write refusal fixture: %v", err)
	}
	before := snapshotProjectDataTree(t, dir)

	projection := lifecycleCloseout199Projection(colony.StateREADY, "init")
	projection.NextAction = LifecycleProjectedAction{ID: "status", RuntimeCommand: "aether status", DisplayCommand: "aether status", Reason: "Inspect the active colony before replacing it."}
	result := lifecycleCloseout199Result(projection, colony.OutcomeKindRefused, colony.LifecycleStateEffectNone)
	if err := applyLifecycleCloseout(result, "init", LifecycleCloseoutDetails{Summary: "An active colony already exists; no files were changed."}); err != nil {
		t.Fatalf("apply init refusal: %v", err)
	}
	visual := renderLifecycleCloseout(lifecycleCloseout199MustResult(t, result), "claude")
	after := snapshotProjectDataTree(t, dir)
	if !reflect.DeepEqual(before, after) || result["state_effect"] != colony.LifecycleStateEffectNone {
		t.Fatalf("front-door refusal wrote state:\nbefore=%#v\nafter=%#v result=%#v", before, after, result)
	}
	if !strings.Contains(visual, "/ant-status") || strings.Contains(visual, "/ant-plan") {
		t.Fatalf("front-door refusal offered unsafe action:\n%s", visual)
	}
	for path, fragment := range map[string]string{
		"init_cmd.go":                `lifecycleCloseoutRefusalForState(`,
		"codex_colonize_finalize.go": `lifecycleCloseoutRefusalFromDisk(`,
		"codex_plan_finalize.go":     `lifecycleCloseoutRefusalFromDisk(`,
	} {
		lifecycleCloseout199RequireSourceCall(t, path, fragment)
	}
}

func lifecycleCloseout199Projection(state colony.State, command string) LifecycleProjection {
	facts := projectionFacts(projectionState(state, true, false))
	projection := projectLifecycle(facts, LifecycleViewFocused, "codex")
	projection.Command = command
	return projection
}

func lifecycleCloseout199Result(projection LifecycleProjection, outcome colony.OutcomeKind, effect colony.LifecycleStateEffect) map[string]interface{} {
	result := map[string]interface{}{
		"projection_revision": projection.ProjectionRevision,
		"outcome_kind":        outcome,
		"state_effect":        effect,
	}
	answer := nextAction{Command: projection.NextAction.RuntimeCommand, Recommendation: projection.NextAction.Reason, Projection: &projection}
	applyNextActionToResult(result, answer)
	return result
}

func lifecycleCloseout199MustResult(t *testing.T, result map[string]interface{}) LifecycleCloseout {
	t.Helper()
	closeout, ok := lifecycleCloseoutFromResult(result)
	if !ok {
		t.Fatalf("result has no typed lifecycle closeout: %#v", result)
	}
	return closeout
}

func lifecycleCloseout199RequireSourceCall(t *testing.T, path, fragment string) {
	t.Helper()
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if !strings.Contains(string(source), fragment) {
		t.Fatalf("%s does not wire the shared closeout with %q", path, fragment)
	}
}
