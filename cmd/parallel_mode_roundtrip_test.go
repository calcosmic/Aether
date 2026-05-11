package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// ---------------------------------------------------------------------------
// Integration test: full parallel-mode state round-trip
//
// Exercises the complete lifecycle: set, persist, read back, survive an
// unrelated state mutation, change mode, verify default when unset.
// ---------------------------------------------------------------------------

func TestParallelModeRoundTrip(t *testing.T) {
	forceJSONOutputModeForTest(t)

	t.Run("set worktree then get returns worktree", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		var buf bytes.Buffer
		stdout = &buf

		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		goal := "test round-trip"
		state := colony.ColonyState{
			Version: "3.0",
			Goal:    &goal,
			State:   colony.StateREADY,
		}
		s.SaveJSON("COLONY_STATE.json", state)

		// Step 1: Set parallel-mode to worktree
		rootCmd.SetArgs([]string{"parallel-mode", "set", "--mode", "worktree"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("set worktree: %v", err)
		}
		env := parseEnvelope(t, buf.String())
		if env["ok"] != true {
			t.Fatalf("set: expected ok, got: %v", env)
		}
		result := env["result"].(map[string]interface{})
		if result["mode"] != "worktree" {
			t.Errorf("set: expected mode 'worktree', got %v", result["mode"])
		}

		// Step 2: Get should return worktree from state
		buf.Reset()
		rootCmd.SetArgs([]string{"parallel-mode", "get"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("get after set: %v", err)
		}
		env = parseEnvelope(t, buf.String())
		result = env["result"].(map[string]interface{})
		if result["mode"] != "worktree" {
			t.Errorf("get: expected mode 'worktree', got %v", result["mode"])
		}
		if result["source"] != "state" {
			t.Errorf("get: expected source 'state', got %v", result["source"])
		}
	})

	t.Run("survives unrelated state mutation", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		var buf bytes.Buffer
		stdout = &buf

		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		goal := "test survive mutation"
		state := colony.ColonyState{
			Version:      "3.0",
			Goal:         &goal,
			State:        colony.StateREADY,
			ParallelMode: colony.ModeWorktree,
		}
		s.SaveJSON("COLONY_STATE.json", state)

		// Step 1: Mutate an unrelated field (colony_depth)
		rootCmd.SetArgs([]string{"state-mutate", "--field", "colony_depth", "--value", "deep"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("state-mutate colony_depth: %v", err)
		}
		env := parseEnvelope(t, buf.String())
		if env["ok"] != true {
			t.Fatalf("state-mutate: expected ok, got: %v", env)
		}

		// Step 2: Verify parallel-mode survived the mutation
		buf.Reset()
		rootCmd.SetArgs([]string{"parallel-mode", "get"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("get after mutation: %v", err)
		}
		env = parseEnvelope(t, buf.String())
		result := env["result"].(map[string]interface{})
		if result["mode"] != "worktree" {
			t.Errorf("mode should survive unrelated mutation: got %v, want 'worktree'", result["mode"])
		}

		// Step 3: Also verify the mutation itself persisted
		var loaded colony.ColonyState
		s.LoadJSON("COLONY_STATE.json", &loaded)
		if loaded.ColonyDepth != "deep" {
			t.Errorf("colony_depth mutation lost: got %q, want 'deep'", loaded.ColonyDepth)
		}
		if loaded.ParallelMode != colony.ModeWorktree {
			t.Errorf("parallel_mode lost after unrelated mutation: got %q, want 'worktree'", loaded.ParallelMode)
		}
	})

	t.Run("set in-repo after worktree", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		var buf bytes.Buffer
		stdout = &buf

		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		goal := "test switch mode"
		state := colony.ColonyState{
			Version:      "3.0",
			Goal:         &goal,
			State:        colony.StateREADY,
			ParallelMode: colony.ModeWorktree,
		}
		s.SaveJSON("COLONY_STATE.json", state)

		// Step 1: Switch to in-repo
		rootCmd.SetArgs([]string{"parallel-mode", "set", "--mode", "in-repo"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("set in-repo: %v", err)
		}

		// Step 2: Get should return in-repo
		buf.Reset()
		rootCmd.SetArgs([]string{"parallel-mode", "get"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("get after switch: %v", err)
		}
		env := parseEnvelope(t, buf.String())
		result := env["result"].(map[string]interface{})
		if result["mode"] != "in-repo" {
			t.Errorf("after switch: expected 'in-repo', got %v", result["mode"])
		}
		if result["source"] != "state" {
			t.Errorf("after switch: expected source 'state', got %v", result["source"])
		}
	})

	t.Run("default is in-repo when unset", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		var buf bytes.Buffer
		stdout = &buf

		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		goal := "test default"
		state := colony.ColonyState{
			Version: "3.0",
			Goal:    &goal,
			State:   colony.StateREADY,
			// ParallelMode deliberately omitted
		}
		s.SaveJSON("COLONY_STATE.json", state)

		rootCmd.SetArgs([]string{"parallel-mode", "get"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("get default: %v", err)
		}
		env := parseEnvelope(t, buf.String())
		result := env["result"].(map[string]interface{})
		if result["mode"] != "in-repo" {
			t.Errorf("default mode: expected 'in-repo', got %v", result["mode"])
		}
		if result["source"] != "default" {
			t.Errorf("default source: expected 'default', got %v", result["source"])
		}
	})

	t.Run("default when no state file exists", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		var buf bytes.Buffer
		stdout = &buf

		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s
		// No COLONY_STATE.json saved at all

		rootCmd.SetArgs([]string{"parallel-mode", "get"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("get no state: %v", err)
		}
		env := parseEnvelope(t, buf.String())
		result := env["result"].(map[string]interface{})
		if result["mode"] != "in-repo" {
			t.Errorf("no-state default: expected 'in-repo', got %v", result["mode"])
		}
		if result["source"] != "default" {
			t.Errorf("no-state source: expected 'default', got %v", result["source"])
		}
	})

	t.Run("invalid mode rejected", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		var buf bytes.Buffer
		stderr = &buf

		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		goal := "test invalid"
		state := colony.ColonyState{
			Version: "3.0",
			Goal:    &goal,
			State:   colony.StateREADY,
		}
		s.SaveJSON("COLONY_STATE.json", state)

		rootCmd.SetArgs([]string{"parallel-mode", "set", "--mode", "invalid"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("cobra error: %v", err)
		}
		env := parseEnvelope(t, buf.String())
		if env["ok"] != false {
			t.Errorf("invalid mode: expected ok=false, got %v", env["ok"])
		}
		if env["code"] != float64(1) {
			t.Errorf("invalid mode: expected code 1, got %v", env["code"])
		}

		// Verify state was NOT modified
		var loaded colony.ColonyState
		s.LoadJSON("COLONY_STATE.json", &loaded)
		if loaded.ParallelMode != "" {
			t.Errorf("invalid mode should not mutate state, got %q", loaded.ParallelMode)
		}
	})

	t.Run("set without state file fails", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		var buf bytes.Buffer
		stderr = &buf

		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s
		// No COLONY_STATE.json

		rootCmd.SetArgs([]string{"parallel-mode", "set", "--mode", "worktree"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("cobra error: %v", err)
		}
		env := parseEnvelope(t, buf.String())
		if env["ok"] != false {
			t.Errorf("no state file: expected ok=false, got %v", env["ok"])
		}
		if env["code"] != float64(1) {
			t.Errorf("no state file: expected code 1, got %v", env["code"])
		}
	})

	t.Run("JSON serialization round-trip", func(t *testing.T) {
		goal := "test json round-trip"
		original := colony.ColonyState{
			Version:      "3.0",
			Goal:         &goal,
			State:        colony.StateREADY,
			ParallelMode: colony.ModeWorktree,
		}

		data, err := json.Marshal(original)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}

		var decoded colony.ColonyState
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}

		if decoded.ParallelMode != colony.ModeWorktree {
			t.Errorf("JSON round-trip: got %q, want %q", decoded.ParallelMode, colony.ModeWorktree)
		}

		// Also verify empty mode round-trips correctly
		emptyState := colony.ColonyState{
			Version: "3.0",
			Goal:    &goal,
			State:   colony.StateREADY,
			// ParallelMode deliberately omitted (zero value)
		}
		data, err = json.Marshal(emptyState)
		if err != nil {
			t.Fatalf("marshal empty: %v", err)
		}
		var emptyDecoded colony.ColonyState
		if err := json.Unmarshal(data, &emptyDecoded); err != nil {
			t.Fatalf("unmarshal empty: %v", err)
		}
		if emptyDecoded.ParallelMode != "" {
			t.Errorf("empty mode round-trip: got %q, want empty", emptyDecoded.ParallelMode)
		}
	})

	t.Run("parallel_mode Valid method", func(t *testing.T) {
		validModes := []colony.ParallelMode{
			colony.ModeInRepo,
			colony.ModeWorktree,
		}
		for _, m := range validModes {
			if !m.Valid() {
				t.Errorf("ParallelMode(%q).Valid() = false, want true", m)
			}
		}

		invalidModes := []colony.ParallelMode{
			"",
			"WORKTREE",
			"In-Repo",
			"parallel",
			"worktrees",
		}
		for _, m := range invalidModes {
			if m.Valid() {
				t.Errorf("ParallelMode(%q).Valid() = true, want false", m)
			}
		}
	})

	t.Run("survives plan_granularity mutation", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		var buf bytes.Buffer
		stdout = &buf

		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		goal := "test survive granularity mutation"
		state := colony.ColonyState{
			Version:         "3.0",
			Goal:            &goal,
			State:           colony.StateREADY,
			ParallelMode:    colony.ModeWorktree,
			PlanGranularity: colony.GranularitySprint,
		}
		s.SaveJSON("COLONY_STATE.json", state)

		// Mutate plan_granularity to quarter
		rootCmd.SetArgs([]string{"state-mutate", "--field", "plan_granularity", "--value", "quarter"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("state-mutate plan_granularity: %v", err)
		}

		// Verify both fields are correct
		var loaded colony.ColonyState
		s.LoadJSON("COLONY_STATE.json", &loaded)
		if loaded.PlanGranularity != colony.GranularityQuarter {
			t.Errorf("plan_granularity: got %q, want %q", loaded.PlanGranularity, colony.GranularityQuarter)
		}
		if loaded.ParallelMode != colony.ModeWorktree {
			t.Errorf("parallel_mode lost after plan_granularity mutation: got %q, want 'worktree'", loaded.ParallelMode)
		}
	})

	t.Run("full lifecycle: default to worktree to in-repo to default via clear", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		var buf bytes.Buffer
		stdout = &buf

		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		goal := "test full lifecycle"
		state := colony.ColonyState{
			Version: "3.0",
			Goal:    &goal,
			State:   colony.StateREADY,
		}
		s.SaveJSON("COLONY_STATE.json", state)

		// Phase 1: default is in-repo
		rootCmd.SetArgs([]string{"parallel-mode", "get"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("phase 1 get: %v", err)
		}
		env := parseEnvelope(t, buf.String())
		result := env["result"].(map[string]interface{})
		if result["mode"] != "in-repo" || result["source"] != "default" {
			t.Errorf("phase 1: expected default in-repo, got mode=%v source=%v", result["mode"], result["source"])
		}

		// Phase 2: set to worktree
		buf.Reset()
		rootCmd.SetArgs([]string{"parallel-mode", "set", "--mode", "worktree"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("phase 2 set: %v", err)
		}

		buf.Reset()
		rootCmd.SetArgs([]string{"parallel-mode", "get"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("phase 2 get: %v", err)
		}
		env = parseEnvelope(t, buf.String())
		result = env["result"].(map[string]interface{})
		if result["mode"] != "worktree" {
			t.Errorf("phase 2: expected worktree, got %v", result["mode"])
		}

		// Phase 3: switch to in-repo
		buf.Reset()
		rootCmd.SetArgs([]string{"parallel-mode", "set", "--mode", "in-repo"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("phase 3 set: %v", err)
		}

		buf.Reset()
		rootCmd.SetArgs([]string{"parallel-mode", "get"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("phase 3 get: %v", err)
		}
		env = parseEnvelope(t, buf.String())
		result = env["result"].(map[string]interface{})
		if result["mode"] != "in-repo" {
			t.Errorf("phase 3: expected in-repo, got %v", result["mode"])
		}
		if result["source"] != "state" {
			t.Errorf("phase 3: expected source 'state', got %v", result["source"])
		}
	})
}
