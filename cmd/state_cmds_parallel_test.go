package cmd

import (
	"bytes"
	"os"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// ---------------------------------------------------------------------------
// state-mutate --field parallel_mode tests
// ---------------------------------------------------------------------------

func TestStateMutateParallelModeField(t *testing.T) {
	t.Run("set worktree", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		var buf bytes.Buffer
		stdout = &buf

		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		goal := "test"
		state := colony.ColonyState{
			Version: "3.0",
			Goal:    &goal,
			State:   colony.StateREADY,
		}
		s.SaveJSON("COLONY_STATE.json", state)

		rootCmd.SetArgs([]string{"state-mutate", "--field", "parallel_mode", "--value", "worktree"})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		env := parseEnvelope(t, buf.String())
		if env["ok"] != true {
			t.Fatalf("expected ok, got: %v", env)
		}

		var loaded colony.ColonyState
		s.LoadJSON("COLONY_STATE.json", &loaded)
		if loaded.ParallelMode != colony.ModeWorktree {
			t.Errorf("persisted parallel_mode = %q, want %q", loaded.ParallelMode, colony.ModeWorktree)
		}
	})

	t.Run("set in-repo", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		var buf bytes.Buffer
		stdout = &buf

		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		goal := "test"
		state := colony.ColonyState{
			Version: "3.0",
			Goal:    &goal,
			State:   colony.StateREADY,
		}
		s.SaveJSON("COLONY_STATE.json", state)

		rootCmd.SetArgs([]string{"state-mutate", "--field", "parallel_mode", "--value", "in-repo"})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		env := parseEnvelope(t, buf.String())
		if env["ok"] != true {
			t.Fatalf("expected ok, got: %v", env)
		}

		var loaded colony.ColonyState
		s.LoadJSON("COLONY_STATE.json", &loaded)
		if loaded.ParallelMode != colony.ModeInRepo {
			t.Errorf("persisted parallel_mode = %q, want %q", loaded.ParallelMode, colony.ModeInRepo)
		}
	})

	t.Run("overwrite existing value", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		var buf bytes.Buffer
		stdout = &buf

		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		goal := "test"
		state := colony.ColonyState{
			Version:      "3.0",
			Goal:         &goal,
			State:        colony.StateREADY,
			ParallelMode: colony.ModeInRepo,
		}
		s.SaveJSON("COLONY_STATE.json", state)

		rootCmd.SetArgs([]string{"state-mutate", "--field", "parallel_mode", "--value", "worktree"})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		var loaded colony.ColonyState
		s.LoadJSON("COLONY_STATE.json", &loaded)
		if loaded.ParallelMode != colony.ModeWorktree {
			t.Errorf("persisted parallel_mode = %q, want %q (overwrite)", loaded.ParallelMode, colony.ModeWorktree)
		}
	})
}

// ---------------------------------------------------------------------------
// state-mutate --field parallel_mode invalid value
// ---------------------------------------------------------------------------

func TestStateMutateParallelModeInvalid(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stderr = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "test"
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"state-mutate", "--field", "parallel_mode", "--value", "invalid"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected cobra error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != false {
		t.Errorf("expected ok=false for invalid parallel_mode, got: %v", env)
	}
	if env["code"] != float64(1) {
		t.Errorf("expected code 1, got: %v", env["code"])
	}

	// Verify state was NOT mutated
	var loaded colony.ColonyState
	s.LoadJSON("COLONY_STATE.json", &loaded)
	if loaded.ParallelMode != "" {
		t.Errorf("parallel_mode should remain empty after invalid mutation, got %q", loaded.ParallelMode)
	}
}

// ---------------------------------------------------------------------------
// state-read-field --field parallel_mode tests
// ---------------------------------------------------------------------------

func TestStateReadFieldParallelMode(t *testing.T) {
	t.Run("read worktree value", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		var buf bytes.Buffer
		stdout = &buf

		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		goal := "test"
		state := colony.ColonyState{
			Version:      "3.0",
			Goal:         &goal,
			State:        colony.StateREADY,
			ParallelMode: colony.ModeWorktree,
		}
		s.SaveJSON("COLONY_STATE.json", state)

		rootCmd.SetArgs([]string{"state-read-field", "--field", "parallel_mode"})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		env := parseEnvelope(t, buf.String())
		if env["ok"] != true {
			t.Fatalf("expected ok, got: %v", env)
		}
		result := env["result"].(map[string]interface{})
		if result["field"] != "parallel_mode" {
			t.Errorf("expected field 'parallel_mode', got %v", result["field"])
		}
		if result["value"] != "worktree" {
			t.Errorf("expected value 'worktree', got %v", result["value"])
		}
	})

	t.Run("read empty default", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		var buf bytes.Buffer
		stdout = &buf

		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		goal := "test"
		state := colony.ColonyState{
			Version: "3.0",
			Goal:    &goal,
			State:   colony.StateREADY,
		}
		s.SaveJSON("COLONY_STATE.json", state)

		rootCmd.SetArgs([]string{"state-read-field", "--field", "parallel_mode"})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		env := parseEnvelope(t, buf.String())
		result := env["result"].(map[string]interface{})
		if result["value"] != "" {
			t.Errorf("expected empty value for unset parallel_mode, got %v", result["value"])
		}
	})

	t.Run("read in-repo value", func(t *testing.T) {
		saveGlobals(t)
		resetRootCmd(t)
		var buf bytes.Buffer
		stdout = &buf

		s, tmpDir := newTestStore(t)
		defer os.RemoveAll(tmpDir)
		store = s

		goal := "test"
		state := colony.ColonyState{
			Version:      "3.0",
			Goal:         &goal,
			State:        colony.StateREADY,
			ParallelMode: colony.ModeInRepo,
		}
		s.SaveJSON("COLONY_STATE.json", state)

		rootCmd.SetArgs([]string{"state-read-field", "--field", "parallel_mode"})
		err := rootCmd.Execute()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		env := parseEnvelope(t, buf.String())
		result := env["result"].(map[string]interface{})
		if result["value"] != "in-repo" {
			t.Errorf("expected value 'in-repo', got %v", result["value"])
		}
	})
}

// ---------------------------------------------------------------------------
// state-mutate expression mode for parallel_mode
// ---------------------------------------------------------------------------

func TestStateMutateParallelModeExpression(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	goal := "test"
	state := colony.ColonyState{
		Version: "3.0",
		Goal:    &goal,
		State:   colony.StateREADY,
	}
	s.SaveJSON("COLONY_STATE.json", state)

	rootCmd.SetArgs([]string{"state-mutate", `.parallel_mode = "worktree"`})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok, got: %v", env)
	}

	// Verify the raw JSON expression set the field correctly
	var loaded colony.ColonyState
	s.LoadJSON("COLONY_STATE.json", &loaded)
	if loaded.ParallelMode != colony.ModeWorktree {
		t.Errorf("persisted parallel_mode via expression = %q, want %q", loaded.ParallelMode, colony.ModeWorktree)
	}
}
