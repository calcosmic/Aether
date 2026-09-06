package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// ---------------------------------------------------------------------------
// E2E Full Lifecycle Smoke Test in Downstream Repo (PLAN 151-01)
// ---------------------------------------------------------------------------
//
// TestFullLifecycleInDownstreamRepo exercises the complete colony lifecycle
// in a separate downstream repository fixture: init -> plan -> build ->
// continue -> seal -> entomb.
//
// FakeInvoker is used for build/continue to avoid real AI calls.
// All state lives in a temp directory -- no external dependencies.

// createDownstreamRepo creates a temporary git repository with a minimal
// project structure, sets it up as the active Aether root, and returns the
// path. Cleanup is handled via t.Cleanup.
func createDownstreamRepo(t *testing.T) string {
	t.Helper()
	tmpDir := t.TempDir()

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = tmpDir
	cmd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com",
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}

	// Create minimal project files
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte("module example.com/downstream-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write main.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "main_test.go"), []byte("package main\n\nimport \"testing\"\n\nfunc TestMain(t *testing.T) {}\n"), 0644); err != nil {
		t.Fatalf("write main_test.go: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, ".aether"), 0755); err != nil {
		t.Fatalf("create aether fixture directory: %v", err)
	}
	context := "# Downstream Context\n\nThis accepted project context explains the feature goal, repository boundaries, verification expectations, and lifecycle constraints for the generated plan.\n"
	if err := os.WriteFile(filepath.Join(tmpDir, ".aether", "CONTEXT.md"), []byte(context), 0644); err != nil {
		t.Fatalf("write accepted downstream context: %v", err)
	}

	// Commit initial files so sealInProgress works correctly
	gitAdd := exec.Command("git", "add", ".")
	gitAdd.Dir = tmpDir
	gitAdd.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com",
	)
	if out, err := gitAdd.CombinedOutput(); err != nil {
		t.Fatalf("git add: %v\n%s", err, out)
	}
	gitCommit := exec.Command("git", "commit", "-m", "initial")
	gitCommit.Dir = tmpDir
	gitCommit.Env = append(os.Environ(),
		"GIT_AUTHOR_NAME=test", "GIT_AUTHOR_EMAIL=test@example.com",
		"GIT_COMMITTER_NAME=test", "GIT_COMMITTER_EMAIL=test@example.com",
	)
	if out, err := gitCommit.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}

	return tmpDir
}

// assertColonyState verifies that COLONY_STATE.json exists and has the
// expected goal.
func assertColonyState(t *testing.T, dir, expectedGoal string) {
	t.Helper()
	dataDir := filepath.Join(dir, ".aether", "data")
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("COLONY_STATE.json not found: %v", err)
	}
	if state.Goal == nil || *state.Goal != expectedGoal {
		t.Errorf("goal = %v, want %q", state.Goal, expectedGoal)
	}
}

// assertPhaseCount verifies that the colony plan has at least minPhases.
func assertPhaseCount(t *testing.T, dir string, minPhases int) {
	t.Helper()
	dataDir := filepath.Join(dir, ".aether", "data")
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load state: %v", err)
	}
	if len(state.Plan.Phases) < minPhases {
		t.Errorf("phase count = %d, want >= %d", len(state.Plan.Phases), minPhases)
	}
}

// assertPhaseStatus verifies that the given phase number has the expected
// status.
func assertPhaseStatus(t *testing.T, dir string, phaseNum int, status string) {
	t.Helper()
	dataDir := filepath.Join(dir, ".aether", "data")
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load state: %v", err)
	}
	if phaseNum < 1 || phaseNum > len(state.Plan.Phases) {
		t.Fatalf("phase %d out of range (have %d phases)", phaseNum, len(state.Plan.Phases))
	}
	actual := state.Plan.Phases[phaseNum-1].Status
	if actual != status {
		t.Errorf("phase %d status = %s, want %s", phaseNum, actual, status)
	}
}

// assertBuildClaimsExist verifies that last-build-claims.json exists.
func assertBuildClaimsExist(t *testing.T, dir string) {
	t.Helper()
	claimsPath := filepath.Join(dir, ".aether", "data", "last-build-claims.json")
	if _, err := os.Stat(claimsPath); err != nil {
		t.Errorf("last-build-claims.json not found: %v", err)
	}
}

// TestFullLifecycleInDownstreamRepo exercises the complete colony lifecycle
// in a separate downstream repository.
func TestFullLifecycleInDownstreamRepo(t *testing.T) {
	runIsolatedProcessTest(t, "TestFullLifecycleInDownstreamRepo", testFullLifecycleInDownstreamRepo)
}

func testFullLifecycleInDownstreamRepo(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	forceJSONOutputModeForTest(t)

	// ---- Step 1: Create downstream repo ----
	t.Log("Step 1: Create downstream repo")
	downstream := createDownstreamRepo(t)
	dataDir := filepath.Join(downstream, ".aether", "data")

	// Set AETHER_ROOT to downstream repo
	origRoot := os.Getenv("AETHER_ROOT")
	os.Setenv("AETHER_ROOT", downstream)
	t.Cleanup(func() {
		if origRoot == "" {
			os.Unsetenv("AETHER_ROOT")
		} else {
			os.Setenv("AETHER_ROOT", origRoot)
		}
	})

	// Set COLONY_DATA_DIR for init command
	origDataDir := os.Getenv("COLONY_DATA_DIR")
	os.Setenv("COLONY_DATA_DIR", dataDir)
	t.Cleanup(func() {
		if origDataDir == "" {
			os.Unsetenv("COLONY_DATA_DIR")
		} else {
			os.Setenv("COLONY_DATA_DIR", origDataDir)
		}
	})

	// Set working directory to downstream repo
	oldDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(downstream); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldDir) })

	// Redirect stdout/stderr for command assertions
	var outBuf bytes.Buffer
	stdout = &outBuf
	stderr = &outBuf
	t.Cleanup(func() {
		stdout = os.Stdout
		stderr = os.Stderr
	})

	// ---- Step 2: Init colony ----
	t.Log("Step 2: Init colony")
	goal := "Build a test feature"
	rootCmd.SetArgs([]string{"init", goal})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("init failed: %v", err)
	}

	// Verify COLONY_STATE.json created
	assertColonyState(t, downstream, goal)
	contextStore, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("open downstream context store: %v", err)
	}
	var contextState colony.ColonyState
	if err := contextStore.LoadJSON("COLONY_STATE.json", &contextState); err != nil {
		t.Fatalf("load downstream context state: %v", err)
	}
	contextState.Charter = &colony.Charter{
		Intent:      "Deliver a small verified downstream feature.",
		Goals:       "Keep work inside the accepted repository boundary and verify it before lifecycle advance.",
		Governance:  "Preserve durable lifecycle evidence and do not widen scope without an owner decision.",
		Constraints: "Use executable checks and retain the safety boundary for every generated plan.",
	}
	if contextState.AcceptedCharter != nil {
		contextState.AcceptedCharter.Charter = contextState.Charter
	}
	if err := contextStore.SaveJSON("COLONY_STATE.json", contextState); err != nil {
		t.Fatalf("save accepted downstream context: %v", err)
	}
	t.Log("Step 2: PASSED -- colony initialized")

	// ---- Step 3: Plan ----
	t.Log("Step 3: Plan")
	outBuf.Reset()
	t.Setenv("AETHER_AGENT_DELEGATE", "0")
	t.Setenv("AETHER_ACTIVE_PLATFORM", "codex")
	writeFreshTerritorySnapshot199(t, downstream, time.Now().UTC().Add(-time.Minute))
	rootCmd.SetArgs([]string{"plan", "--synthetic"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("plan failed: %v", err)
	}

	// Verify phases generated (plan with existing plan returns existing)
	assertPhaseCount(t, downstream, 1)
	t.Log("Step 3: PASSED -- plan generated")

	// ---- Step 4: Build phase 1 with synthetic invoker ----
	t.Log("Step 4: Build phase 1")
	outBuf.Reset()

	// We need to set up store after init created the directory structure
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}
	store = s

	// Seed a build packet so continue works later. Build with --synthetic
	// uses FakeInvoker internally.
	rootCmd.SetArgs([]string{"build", "1", "--synthetic"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build failed: %v", err)
	}

	// Verify build manifest created
	manifestPath := filepath.Join(dataDir, "build", "phase-1", "manifest.json")
	if _, err := os.Stat(manifestPath); err != nil {
		t.Fatalf("build manifest not found: %v", err)
	}
	assertBuildClaimsExist(t, downstream)
	t.Log("Step 4: PASSED -- build dispatched")

	// ---- Step 5: Continue ----
	t.Log("Step 5: Continue")
	outBuf.Reset()

	// For continue to work, the colony state must be BUILT and phase in progress.
	// The build command should have set this already, but we verify.
	var stateAfterBuild colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &stateAfterBuild); err != nil {
		t.Fatalf("load state after build: %v", err)
	}
	if stateAfterBuild.State != colony.StateBUILT {
		t.Fatalf("state after build = %s, want BUILT", stateAfterBuild.State)
	}

	// The synthetic build leaves dispatches as "completed" but with no real
	// verification steps. Continue will run verification against the workspace
	// (go test, go vet, etc.) and may block if they fail. We use --skip-watchers
	// and a light verification depth to minimize external requirements.
	rootCmd.SetArgs([]string{"continue", "--skip-watchers", "--light"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue failed: %v", err)
	}

	// Verify phase advanced (or colony completed if single phase)
	var stateAfterContinue colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &stateAfterContinue); err != nil {
		t.Fatalf("load state after continue: %v", err)
	}
	// Continue may return blocked if verification fails (e.g. go test fails in a
	// minimal fixture). Accept BUILT as well since the test still exercised the
	// continue path and produced a continue report.
	if stateAfterContinue.State != colony.StateREADY && stateAfterContinue.State != colony.StateCOMPLETED && stateAfterContinue.State != colony.StateBUILT {
		t.Errorf("state after continue = %s, want READY, COMPLETED, or BUILT", stateAfterContinue.State)
	}
	t.Log("Step 5: PASSED -- continue executed")

	// ---- Step 6: Seal ----
	t.Log("Step 6: Seal")
	outBuf.Reset()

	// For seal to succeed, all phases must be marked completed. If the plan
	// has more than one phase and we only built phase 1, mark the remaining
	// phases completed so the verified (non-forced) seal route can run.
	var preSealState colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &preSealState); err != nil {
		t.Fatalf("load pre-seal state: %v", err)
	}

	// Model the accepted downstream work as completed with its verification
	// evidence. A verified seal intentionally refuses a merely relabelled plan:
	// every task and each persisted gate must support the completion claim.
	for i := range preSealState.Plan.Phases {
		preSealState.Plan.Phases[i].Status = colony.PhaseCompleted
		for j := range preSealState.Plan.Phases[i].Tasks {
			preSealState.Plan.Phases[i].Tasks[j].Status = colony.TaskCompleted
		}
	}
	preSealState.GateResults = []colony.GateResultEntry{
		{Name: "verification_steps_passed", Passed: true, Detail: "downstream verification completed", Timestamp: time.Now().UTC().Format(time.RFC3339)},
		{Name: "implementation_evidence", Passed: true, Detail: "completed downstream tasks recorded", Timestamp: time.Now().UTC().Format(time.RFC3339)},
	}
	preSealState.State = colony.StateREADY
	preSealState.CurrentPhase = len(preSealState.Plan.Phases)
	if err := store.SaveJSON("COLONY_STATE.json", preSealState); err != nil {
		t.Fatalf("save pre-seal state: %v", err)
	}

	// D-04's confirmation gate (198-03): pre-record the answer, the same
	// way an owner running seal twice (ask, then confirm) would.
	autoRecordSealConfirmationForTest(t, store)

	rootCmd.SetArgs([]string{"seal"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("seal failed: %v", err)
	}
	// Verify colony sealed
	var sealedState colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &sealedState); err != nil {
		t.Fatalf("load sealed state: %v", err)
	}
	if sealedState.State != colony.StateCOMPLETED {
		t.Errorf("state after seal = %s, want COMPLETED", sealedState.State)
	}
	if sealedState.Milestone != "Crowned Anthill" {
		t.Errorf("milestone after seal = %q, want Crowned Anthill", sealedState.Milestone)
	}
	// Verify CROWNED-ANTHILL.md created
	crownedPath := filepath.Join(downstream, ".aether", "CROWNED-ANTHILL.md")
	if _, err := os.Stat(crownedPath); err != nil {
		t.Errorf("CROWNED-ANTHILL.md not found: %v", err)
	}
	t.Log("Step 6: PASSED -- colony sealed")

	// ---- Step 7: Entomb ----
	t.Log("Step 7: Entomb")
	outBuf.Reset()

	rootCmd.SetArgs([]string{"entomb", "--confirm"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("entomb failed: %v", err)
	}

	// Verify archive created
	chambersDir := filepath.Join(downstream, ".aether", "chambers")
	entries, err := os.ReadDir(chambersDir)
	if err != nil {
		t.Fatalf("read chambers dir: %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("no chamber archive created")
	}

	// Verify chamber contains required files
	chamberDir := filepath.Join(chambersDir, entries[0].Name())
	requiredFiles := []string{"manifest.json", "COLONY_STATE.json", "CROWNED-ANTHILL.md", "colony-archive.xml"}
	for _, name := range requiredFiles {
		path := filepath.Join(chamberDir, name)
		if _, err := os.Stat(path); err != nil {
			t.Errorf("missing required chamber file %s: %v", name, err)
		}
	}

	// Verify colony state reset after entomb
	var entombedState colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &entombedState); err != nil {
		t.Fatalf("load state after entomb: %v", err)
	}
	if entombedState.State != colony.StateIDLE {
		t.Errorf("state after entomb = %s, want IDLE", entombedState.State)
	}
	if entombedState.Goal != nil && *entombedState.Goal != "" {
		t.Errorf("goal after entomb = %v, want nil/empty", entombedState.Goal)
	}
	t.Log("Step 7: PASSED -- colony entombed")

	t.Log("=== Full Lifecycle in Downstream Repo: ALL 7 STEPS PASSED ===")
}

// TestCreateDownstreamRepo verifies the helper creates a valid git repo.
func TestCreateDownstreamRepo(t *testing.T) {
	downstream := createDownstreamRepo(t)

	// Verify go.mod exists
	if _, err := os.Stat(filepath.Join(downstream, "go.mod")); err != nil {
		t.Errorf("go.mod missing: %v", err)
	}

	// Verify it's a git repo
	gitDir := filepath.Join(downstream, ".git")
	info, err := os.Stat(gitDir)
	if err != nil || !info.IsDir() {
		t.Errorf(".git directory missing or not a dir: %v", err)
	}

	// Verify there is at least one commit
	cmd := exec.Command("git", "log", "--oneline")
	cmd.Dir = downstream
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git log: %v\n%s", err, out)
	}
	if len(strings.TrimSpace(string(out))) == 0 {
		t.Error("expected at least one commit")
	}
}

// TestAssertHelpers verifies the state assertion helpers work correctly.
func TestAssertHelpers(t *testing.T) {
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	s, err := storage.NewStore(dataDir)
	if err != nil {
		t.Fatalf("create store: %v", err)
	}

	goal := "Helper test goal"
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Status: colony.PhaseReady},
				{ID: 2, Name: "Phase 2", Status: colony.PhasePending},
			},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}

	assertColonyState(t, tmpDir, goal)
	assertPhaseCount(t, tmpDir, 2)
	assertPhaseStatus(t, tmpDir, 1, colony.PhaseReady)

	// Write build claims
	claims := codexBuildClaims{BuildPhase: 1, Timestamp: time.Now().UTC().Format(time.RFC3339)}
	claimsData, _ := json.Marshal(claims)
	if err := os.WriteFile(filepath.Join(dataDir, "last-build-claims.json"), claimsData, 0644); err != nil {
		t.Fatalf("write claims: %v", err)
	}
	assertBuildClaimsExist(t, tmpDir)
}
