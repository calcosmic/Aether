package cmd

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// deterministicCommand represents a command that should never spawn agents.
type deterministicCommand struct {
	name       string
	args       []string
	needsStore bool
	needsGoal  bool
}

// deterministicCommands lists all commands classified as deterministic
// (display, signal, session, admin) that must never dispatch agents.
var deterministicCommands = []deterministicCommand{
	{name: "status", needsStore: true, needsGoal: true},
	{name: "flag-list", needsStore: false, needsGoal: false},
	{name: "pheromones", needsStore: true, needsGoal: false},
	{name: "focus", args: []string{"test focus signal"}, needsStore: true, needsGoal: false},
	{name: "redirect", args: []string{"test redirect signal"}, needsStore: true, needsGoal: false},
	{name: "feedback", args: []string{"test feedback signal"}, needsStore: true, needsGoal: false},
	{name: "history", needsStore: true, needsGoal: false},
	{name: "patrol-check", needsStore: true, needsGoal: false},
	{name: "preferences", args: []string{"--list"}, needsStore: true, needsGoal: false},
	{name: "memory-metrics", needsStore: true, needsGoal: false},
	{name: "memory-details", needsStore: true, needsGoal: false},
	{name: "verify-castes", needsStore: false, needsGoal: false},
	{name: "bump-version", args: []string{"--dry-run", "1.0.0"}, needsStore: false, needsGoal: false},
	{name: "maturity", needsStore: true, needsGoal: true},
	{name: "update", needsStore: false, needsGoal: false},
	{name: "migrate-state", args: []string{"--dry-run"}, needsStore: true, needsGoal: false},
}

// setupDeterministicTestEnv creates a temp directory with store and optional colony state.
func setupDeterministicTestEnv(t *testing.T, needsStore, needsGoal bool) (*storage.Store, string, func()) {
	t.Helper()

	origColonyDataDir := os.Getenv("COLONY_DATA_DIR")
	origAetherRoot := os.Getenv("AETHER_ROOT")
	origStdout := stdout
	origStderr := stderr

	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("mkdir data dir: %v", err)
	}

	os.Setenv("COLONY_DATA_DIR", dataDir)
	os.Setenv("AETHER_ROOT", tmpDir)

	var s *storage.Store
	if needsStore {
		var err error
		s, err = storage.NewStore(dataDir)
		if err != nil {
			t.Fatalf("create store: %v", err)
		}
		store = s
	}

	if needsStore {
		goal := "test colony goal"
		state := colony.ColonyState{Goal: &goal}
		if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
			t.Fatalf("save colony state: %v", err)
		}
	}

	cleanup := func() {
		os.Setenv("COLONY_DATA_DIR", origColonyDataDir)
		os.Setenv("AETHER_ROOT", origAetherRoot)
		stdout = origStdout
		stderr = origStderr
		if s != nil {
			store = nil
		}
	}

	return s, tmpDir, cleanup
}

// countSpawnEntries returns the number of entries in the spawn tree.
func countSpawnEntries(t *testing.T, s *storage.Store) int {
	t.Helper()
	if s == nil {
		return 0
	}
	st := agent.NewSpawnTree(s, "")
	entries, err := st.Parse()
	if err != nil {
		return 0
	}
	return len(entries)
}

func TestDeterministicCommandsNoSpawn(t *testing.T) {
	for _, tc := range deterministicCommands {
		t.Run(tc.name, func(t *testing.T) {
			saveGlobals(t)
			resetRootCmd(t)
			forceJSONOutputModeForTest(t)

			var buf bytes.Buffer
			stdout = &buf
			stderr = &buf

			var s *storage.Store
			var updateFixture *antUpdateFixture
			if tc.name == "update" {
				// Exercise a successful update from a real published payload.
				// An incomplete hub would refuse before reaching the no-spawn check.
				fixture := newAntUpdateFixture(t)
				updateFixture = &fixture
				s = store
			} else {
				var cleanup func()
				s, _, cleanup = setupDeterministicTestEnv(t, tc.needsStore, tc.needsGoal)
				defer cleanup()
			}

			// Capture pre-run spawn state.
			preEntries := countSpawnEntries(t, s)

			// Build args and execute.
			args := append([]string{tc.name}, tc.args...)
			rootCmd.SetArgs(args)
			err := rootCmd.Execute()
			if err != nil {
				t.Fatalf("command %q returned error: %v", tc.name, err)
			}
			if updateFixture != nil {
				antAssertPublishedHome(t, updateFixture.hub, updateFixture.home)
			}

			// Some commands may error gracefully (e.g., no colony) but still not spawn.
			// We only care that they didn't spawn agents.
			postEntries := countSpawnEntries(t, s)

			if postEntries > preEntries {
				t.Errorf("command %q created %d new spawn tree entries (was %d, now %d)",
					tc.name, postEntries-preEntries, preEntries, postEntries)
			}
		})
	}
}

func TestDeterministicCommandCount(t *testing.T) {
	// Ensure we test at least 12 deterministic commands as specified in the plan.
	if len(deterministicCommands) < 12 {
		t.Fatalf("expected at least 12 deterministic commands, got %d", len(deterministicCommands))
	}
}

// mockWorkerInvoker is a test double that records whether Invoke was called.
type mockWorkerInvoker struct {
	called bool
}

func (m *mockWorkerInvoker) IsAvailable(ctx context.Context) bool { return true }
func (m *mockWorkerInvoker) ValidateAgent(path string) error      { return nil }
func (m *mockWorkerInvoker) Invoke(ctx context.Context, cfg codex.WorkerConfig) (codex.WorkerResult, error) {
	m.called = true
	return codex.WorkerResult{Status: "completed"}, nil
}

func TestQuickCommandAttemptsSpawn(t *testing.T) {
	// The quick command is NOT deterministic and SHOULD attempt to spawn.
	// This test verifies our spawn detection works by checking that quick
	// invokes the worker dispatcher (we mock it to avoid real subprocesses).
	saveGlobals(t)
	resetRootCmd(t)
	forceJSONOutputModeForTest(t)

	var buf bytes.Buffer
	stdout = &buf

	_, _, cleanup := setupDeterministicTestEnv(t, true, true)
	defer cleanup()

	// Override the worker invoker factory to inject our mock.
	mock := &mockWorkerInvoker{}
	origFactory := newQuickWorkerInvoker
	newQuickWorkerInvoker = func() codex.WorkerInvoker {
		return mock
	}
	defer func() { newQuickWorkerInvoker = origFactory }()

	rootCmd.SetArgs([]string{"quick", "test question"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("quick returned error: %v", err)
	}

	if !mock.called {
		t.Error("quick command did not attempt to invoke a worker; spawn detection may be broken")
	}
}
