package cmd

import (
	"encoding/json"
	"testing"
)

func TestAutopilotStateCompatibility(t *testing.T) {
	legacyJSON := []byte(`{
		"initialized_at":"2026-01-01T00:00:00Z",
		"total_phases":3,
		"current_phase":1,
		"status":"running",
		"reason":"",
		"headless":true,
		"replan_interval":2,
		"phases":[{"phase":1,"status":"completed","at":"2026-01-01T00:05:00Z"}],
		"last_updated":"2026-01-01T00:05:00Z"
	}`)

	var state autopilotState
	if err := json.Unmarshal(legacyJSON, &state); err != nil {
		t.Fatalf("decode legacy state: %v", err)
	}
	if state.SchemaVersion != 0 || state.LastReport != nil {
		t.Fatalf("legacy optional fields changed: schema=%d report=%+v", state.SchemaVersion, state.LastReport)
	}
	if state.TotalPhases != 3 || state.CurrentPhase != 1 || len(state.Phases) != 1 {
		t.Fatalf("legacy state fields not preserved: %+v", state)
	}
	if got := state.Phases[0]; got.Phase != 1 || got.Status != "completed" {
		t.Fatalf("legacy phase status not preserved: %+v", got)
	}

	state.SchemaVersion = autopilotStateSchemaVersion
	state.LastReport = &autopilotInvocationReport{
		SchemaVersion: autopilotReportSchemaVersion,
		InvocationID:  "run-compatibility-test",
		Outcome:       "completed",
		Next:          "aether seal",
	}
	raw, err := json.Marshal(state)
	if err != nil {
		t.Fatalf("encode current state: %v", err)
	}

	var roundTrip autopilotState
	if err := json.Unmarshal(raw, &roundTrip); err != nil {
		t.Fatalf("decode current state: %v", err)
	}
	if roundTrip.SchemaVersion != autopilotStateSchemaVersion || roundTrip.LastReport == nil {
		t.Fatalf("current optional fields not preserved: %+v", roundTrip)
	}
	if roundTrip.LastReport.InvocationID != "run-compatibility-test" || roundTrip.LastReport.Next != "aether seal" {
		t.Fatalf("current report fields not preserved: %+v", roundTrip.LastReport)
	}
	if roundTrip.TotalPhases != 3 || roundTrip.Phases[0].Status != "completed" {
		t.Fatalf("legacy fields lost after current round trip: %+v", roundTrip)
	}
}
