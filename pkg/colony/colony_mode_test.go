package colony

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestColonyModeValidAndEffective(t *testing.T) {
	tests := []struct {
		name      string
		mode      ColonyMode
		wantValid bool
		want      ColonyMode
	}{
		{name: "colony", mode: ColonyModeColony, wantValid: true, want: ColonyModeColony},
		{name: "orchestrator", mode: ColonyModeOrchestrator, wantValid: true, want: ColonyModeOrchestrator},
		{name: "empty legacy value", mode: "", wantValid: false, want: ColonyModeColony},
		{name: "unknown value", mode: ColonyMode("experimental"), wantValid: false, want: ColonyModeColony},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.mode.Valid(); got != tt.wantValid {
				t.Fatalf("Valid() = %v, want %v", got, tt.wantValid)
			}
			if got := tt.mode.Effective(); got != tt.want {
				t.Fatalf("Effective() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestColonyStateColonyModeSerialization(t *testing.T) {
	t.Run("round-trip with orchestrator mode set", func(t *testing.T) {
		state := ColonyState{
			Version:    "1",
			ColonyMode: ColonyModeOrchestrator,
		}

		data, err := json.Marshal(state)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if !strings.Contains(string(data), `"colony_mode":"orchestrator"`) {
			t.Fatalf("expected colony_mode in JSON, got %s", data)
		}

		var decoded ColonyState
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if decoded.ColonyMode != ColonyModeOrchestrator {
			t.Fatalf("ColonyMode = %q, want %q", decoded.ColonyMode, ColonyModeOrchestrator)
		}
		if got := decoded.EffectiveColonyMode(); got != ColonyModeOrchestrator {
			t.Fatalf("EffectiveColonyMode() = %q, want %q", got, ColonyModeOrchestrator)
		}
	})

	t.Run("omits zero mode", func(t *testing.T) {
		data, err := json.Marshal(ColonyState{Version: "1"})
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		if strings.Contains(string(data), "colony_mode") {
			t.Fatalf("zero ColonyMode should be omitted, got %s", data)
		}
	})

	t.Run("legacy JSON defaults effectively to colony mode", func(t *testing.T) {
		oldJSON := `{"version":"1","state":"READY","current_phase":0,"plan":{"phases":null},"memory":{"phase_learnings":null,"decisions":null,"instincts":null},"errors":{"records":null,"flagged_patterns":null},"signals":null,"graveyards":null,"events":null,"milestone":""}`

		var decoded ColonyState
		if err := json.Unmarshal([]byte(oldJSON), &decoded); err != nil {
			t.Fatalf("unmarshal old JSON: %v", err)
		}
		if decoded.ColonyMode != "" {
			t.Fatalf("legacy ColonyMode = %q, want zero value", decoded.ColonyMode)
		}
		if got := decoded.EffectiveColonyMode(); got != ColonyModeColony {
			t.Fatalf("EffectiveColonyMode() = %q, want %q", got, ColonyModeColony)
		}
	})

	t.Run("unknown mode is invalid but effectively colony mode", func(t *testing.T) {
		var decoded ColonyState
		if err := json.Unmarshal([]byte(`{"version":"1","colony_mode":"experimental"}`), &decoded); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if decoded.ColonyMode.Valid() {
			t.Fatalf("expected unknown ColonyMode %q to be invalid", decoded.ColonyMode)
		}
		if got := decoded.EffectiveColonyMode(); got != ColonyModeColony {
			t.Fatalf("EffectiveColonyMode() = %q, want %q", got, ColonyModeColony)
		}
	})
}
