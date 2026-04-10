package colony

import (
	"encoding/json"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// ParallelMode.Valid() tests
// ---------------------------------------------------------------------------

func TestParallelModeValid(t *testing.T) {
	tests := []struct {
		m    ParallelMode
		want bool
	}{
		{ModeInRepo, true},
		{ModeWorktree, true},
		{"", false},
		{"invalid", false},
		{"WORKTREE", false},
		{"InRepo", false},
		{"banana", false},
	}
	for _, tt := range tests {
		name := string(tt.m)
		if name == "" {
			name = "(empty)"
		}
		t.Run(name, func(t *testing.T) {
			if got := tt.m.Valid(); got != tt.want {
				t.Errorf("ParallelMode(%q).Valid() = %v, want %v", tt.m, got, tt.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// ErrInvalidParallelMode is a non-nil error
// ---------------------------------------------------------------------------

func TestErrInvalidParallelMode(t *testing.T) {
	if ErrInvalidParallelMode == nil {
		t.Fatal("ErrInvalidParallelMode must not be nil")
	}
}

// ---------------------------------------------------------------------------
// ColonyState parallel_mode field serialization
// ---------------------------------------------------------------------------

func TestColonyStateParallelModeSerialization(t *testing.T) {
	t.Run("round-trip with mode set", func(t *testing.T) {
		cs := ColonyState{
			Version:         "1",
			ParallelMode:    ModeWorktree,
			PlanGranularity: GranularitySprint,
		}
		data, err := json.Marshal(cs)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}

		var out ColonyState
		if err := json.Unmarshal(data, &out); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}

		if out.ParallelMode != ModeWorktree {
			t.Errorf("ParallelMode = %q, want %q", out.ParallelMode, ModeWorktree)
		}
		if out.PlanGranularity != GranularitySprint {
			t.Errorf("PlanGranularity = %q, want %q", out.PlanGranularity, GranularitySprint)
		}
	})

	t.Run("omits field when zero value", func(t *testing.T) {
		cs := ColonyState{Version: "1"}
		data, err := json.Marshal(cs)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}

		// The key assertion: parallel_mode must NOT appear in the output
		// because omitempty suppresses zero string values.
		s := string(data)
		if strings.Contains(s, "parallel_mode") {
			t.Errorf("zero ParallelMode should be omitted, but got:\n%s", s)
		}
	})

	t.Run("backward compatible: old JSON without parallel_mode", func(t *testing.T) {
		oldJSON := `{"version":"1","goal":"test","state":"READY","current_phase":0,"plan":{"phases":null},"memory":{"phase_learnings":null,"decisions":null,"instincts":null},"errors":{"records":null,"flagged_patterns":null},"signals":null,"graveyards":null,"events":null,"milestone":""}`

		var out ColonyState
		if err := json.Unmarshal([]byte(oldJSON), &out); err != nil {
			t.Fatalf("unmarshal old JSON: %v", err)
		}

		if out.ParallelMode != "" {
			t.Errorf("expected zero ParallelMode from old JSON, got %q", out.ParallelMode)
		}
		if out.Version != "1" {
			t.Errorf("Version = %q, want %q", out.Version, "1")
		}
	})

	t.Run("mode in-repo round-trip", func(t *testing.T) {
		cs := ColonyState{Version: "1", ParallelMode: ModeInRepo}
		data, err := json.Marshal(cs)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}

		var out ColonyState
		if err := json.Unmarshal(data, &out); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if out.ParallelMode != ModeInRepo {
			t.Errorf("ParallelMode = %q, want %q", out.ParallelMode, ModeInRepo)
		}
	})
}
