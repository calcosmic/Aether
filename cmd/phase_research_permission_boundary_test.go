package cmd

// This file proves a real plan-time phase_research dispatch resolves at the
// Go worker-adapter permission boundary for caste scout.
//
// The phase's original suites (548 TS tests, full Go suite) all passed while
// every real Scout dispatch failed in production: the only CLI-level
// permission test constructed its internalWorkerDispatchRequest by hand,
// never from a real plannedPhaseResearchDispatches output. A hand-written
// profile literal is exactly what let CR-01/CR-02 hide -- Go's side of this
// boundary was already correct, but nothing proved a real dispatch actually
// reached it faithfully. These tests start from the real dispatch builder,
// cross the JSON boundary (the same hop the TS host crosses), and resolve
// through internalWorkerConfig/ResolvePermissionProfile.

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
)

// realPhaseResearchDispatch builds one real phase_research dispatch the same
// way plan-time code does, and fails the test immediately if the dispatch
// builder's own gates (approval, candidate availability) don't produce
// exactly one dispatch.
func realPhaseResearchDispatch(t *testing.T) codexPlanningDispatch {
	t.Helper()
	candidate := phaseResearchCandidate{ID: 3, Name: "Wire exporter", Description: "Ship the exporter"}
	dispatches := plannedPhaseResearchDispatches(
		t.TempDir(),
		"deep",
		"Build the exporter",
		[]phaseResearchCandidate{candidate},
		codexSurveyContext{},
		false,
		map[int]bool{3: true},
	)
	if len(dispatches) != 1 {
		t.Fatalf("plannedPhaseResearchDispatches returned %d dispatches, want 1", len(dispatches))
	}
	return dispatches[0]
}

// dispatchPermissionProfileOverJSON round-trips a dispatch's permission
// profile through JSON, the same boundary the TS host actually crosses,
// rather than reading dispatch.PermissionProfile directly off the Go struct.
func dispatchPermissionProfileOverJSON(t *testing.T, dispatch codexPlanningDispatch) codex.PermissionProfile {
	t.Helper()
	raw, err := json.Marshal(dispatch)
	if err != nil {
		t.Fatalf("json.Marshal(dispatch) failed: %v", err)
	}
	var decoded struct {
		PermissionProfile codex.PermissionProfile `json:"permission_profile"`
		Brief             string                  `json:"brief"`
	}
	if err := json.Unmarshal(raw, &decoded); err != nil {
		t.Fatalf("json.Unmarshal(dispatch JSON) failed: %v", err)
	}
	return decoded.PermissionProfile
}

func TestPlanResearchDispatchResolvesAtWorkerBoundary(t *testing.T) {
	dispatch := realPhaseResearchDispatch(t)
	profile := dispatchPermissionProfileOverJSON(t, dispatch)

	request := internalWorkerDispatchRequest{
		SchemaVersion:     1,
		Caste:             "scout",
		WorkerName:        dispatch.Name,
		TaskID:            dispatch.TaskID,
		TaskBrief:         dispatch.Brief,
		PermissionProfile: profile,
	}

	config, err := internalWorkerConfig(t.TempDir(), &codex.FakeInvoker{}, request)
	if err != nil {
		t.Fatalf("internalWorkerConfig rejected a real plan-research dispatch's own emitted profile: %v", err)
	}
	if config.PermissionProfile.Name != codex.PermissionWorkspaceWrite {
		t.Fatalf("resolved PermissionProfile.Name = %q, want %q", config.PermissionProfile.Name, codex.PermissionWorkspaceWrite)
	}
}

func TestScoutReadOnlyProfileIsRejectedAtWorkerBoundary(t *testing.T) {
	dispatch := realPhaseResearchDispatch(t)

	// This is the profile shape the stale TS fallback used to produce for
	// scout before CR-01 was fixed: repository_read_only. Every real Scout
	// dispatch hit this rejection in production. The test documents that as
	// intended enforcement, not a bug to soften.
	staleReadOnlyProfile := codex.PermissionProfile{
		SchemaVersion: 1,
		Name:          codex.PermissionRepositoryReadOnly,
		Filesystem:    codex.FilesystemRepositoryReadOnly,
		Shell:         "within_filesystem_boundary",
		Network:       "provider_default",
		Approval:      "never",
	}

	request := internalWorkerDispatchRequest{
		SchemaVersion:     1,
		Caste:             "scout",
		WorkerName:        dispatch.Name,
		TaskID:            dispatch.TaskID,
		TaskBrief:         dispatch.Brief,
		PermissionProfile: staleReadOnlyProfile,
	}

	_, err := internalWorkerConfig(t.TempDir(), &codex.FakeInvoker{}, request)
	if err == nil {
		t.Fatal("internalWorkerConfig accepted a stale repository_read_only profile for scout; expected a mismatch rejection")
	}
	if !strings.Contains(err.Error(), "mismatch") {
		t.Fatalf("internalWorkerConfig error = %q, want it to contain %q", err.Error(), "mismatch")
	}
}

// TestPlanResearchDispatchCarriesFiveSectionMission was
// TestPlanResearchDispatchCarriesSixSectionMission before WIRE-02
// (198.2-08): the "## Hive Wisdom (Pre-existing Knowledge)" heading was
// removed from the output-template instruction because the Scout now
// receives that content directly (a "## Shared Lessons (Cross-Colony
// Patterns)" input section, when the shared store has entries) rather than
// being told to invent or fake a summary of nothing.
func TestPlanResearchDispatchCarriesFiveSectionMission(t *testing.T) {
	dispatch := realPhaseResearchDispatch(t)

	for _, heading := range []string{
		"## Output",
		"## Key Patterns",
		"## External Context",
		"## Gotchas",
		"## Recommended Approach",
		"## Files to Study",
	} {
		if !strings.Contains(dispatch.Brief, heading) {
			t.Errorf("dispatch.Brief missing heading %q:\n%s", heading, dispatch.Brief)
		}
	}

	profile := dispatchPermissionProfileOverJSON(t, dispatch)
	request := internalWorkerDispatchRequest{
		SchemaVersion:     1,
		Caste:             "scout",
		WorkerName:        dispatch.Name,
		TaskID:            dispatch.TaskID,
		TaskBrief:         dispatch.Brief,
		PermissionProfile: profile,
	}

	config, err := internalWorkerConfig(t.TempDir(), &codex.FakeInvoker{}, request)
	if err != nil {
		t.Fatalf("internalWorkerConfig failed: %v", err)
	}
	for _, heading := range []string{
		"## Output",
		"## Key Patterns",
		"## External Context",
		"## Gotchas",
		"## Recommended Approach",
		"## Files to Study",
	} {
		if !strings.Contains(config.TaskBrief, heading) {
			t.Errorf("resolved WorkerConfig.TaskBrief missing heading %q:\n%s", heading, config.TaskBrief)
		}
	}
}
