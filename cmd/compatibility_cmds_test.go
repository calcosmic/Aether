package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

func TestQueenPromoteLegacyPositionalSanitizesContent(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	hubDir := filepath.Join(tmpDir, "hub")
	origHub := os.Getenv("AETHER_HUB_DIR")
	os.Setenv("AETHER_HUB_DIR", hubDir)
	t.Cleanup(func() { os.Setenv("AETHER_HUB_DIR", origHub) })

	rootCmd.SetArgs([]string{"queen-promote", "pattern", "Line one\n## injected"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got %v", env["ok"])
	}

	data, err := os.ReadFile(filepath.Join(hubDir, "QUEEN.md"))
	if err != nil {
		t.Fatalf("read QUEEN.md: %v", err)
	}
	text := string(data)
	if !strings.Contains(text, "## Patterns") {
		t.Fatalf("QUEEN.md missing Patterns section:\n%s", text)
	}
	if strings.Contains(text, "\n## injected") {
		t.Fatalf("unsanitized injected header present:\n%s", text)
	}
	if !strings.Contains(text, "Line one ## injected") {
		t.Fatalf("sanitized content missing:\n%s", text)
	}
}

func TestCharterWriteLegacyFlags(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	hubDir := filepath.Join(tmpDir, "hub")
	origHub := os.Getenv("AETHER_HUB_DIR")
	os.Setenv("AETHER_HUB_DIR", hubDir)
	t.Cleanup(func() { os.Setenv("AETHER_HUB_DIR", origHub) })

	rootCmd.SetArgs([]string{
		"charter-write",
		"--intent", "Ship\ncleanly",
		"--vision", "Great colony system",
		"--governance", "Go is source of truth",
		"--goals", "parity, resilience",
	})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(hubDir, "QUEEN.md"))
	if err != nil {
		t.Fatalf("read QUEEN.md: %v", err)
	}
	text := string(data)
	for _, want := range []string{
		"## Colony Charter",
		"- **Intent:** Ship cleanly",
		"- **Vision:** Great colony system",
		"- **Governance:** Go is source of truth",
		"- **Goals:** parity, resilience",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("QUEEN.md missing %q:\n%s", want, text)
		}
	}
}

func TestMemoryCaptureSupportsPositionalContent(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	rootCmd.SetArgs([]string{"memory-capture", "use positional content"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	if env["ok"] != true {
		t.Fatalf("expected ok:true, got %v", env["ok"])
	}
}

func TestHivePromoteDefaultsDomainForLegacyPlaybooks(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)
	var buf bytes.Buffer
	stdout = &buf

	s, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	store = s

	hubDir := filepath.Join(tmpDir, "hub")
	origHub := os.Getenv("AETHER_HUB_DIR")
	os.Setenv("AETHER_HUB_DIR", hubDir)
	t.Cleanup(func() { os.Setenv("AETHER_HUB_DIR", origHub) })

	rootCmd.SetArgs([]string{"hive-promote", "--text", "Prefer focused fixes", "--source-repo", "aether", "--confidence", "0.9"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(hubDir, "hive", "wisdom.json"))
	if err != nil {
		t.Fatalf("read wisdom.json: %v", err)
	}
	if !strings.Contains(string(data), `"domain": "general"`) {
		t.Fatalf("wisdom.json missing default general domain:\n%s", string(data))
	}
}

func TestResumeAliasRestoresResumeColonyFlow(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	goal := "Resume via alias"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Execution", Status: colony.PhaseInProgress},
				{ID: 2, Name: "Verification", Status: colony.PhaseReady},
			},
		},
	})

	if err := store.SaveJSON("session.json", colony.SessionFile{
		SessionID:      "resume-alias",
		StartedAt:      "2026-04-17T10:00:00Z",
		ColonyGoal:     goal,
		CurrentPhase:   1,
		SuggestedNext:  "aether continue",
		ContextCleared: true,
		Summary:        "Paused with a handoff",
	}); err != nil {
		t.Fatalf("seed session: %v", err)
	}

	handoffPath := filepath.Join(os.Getenv("AETHER_ROOT"), ".aether", "HANDOFF.md")
	if err := os.MkdirAll(filepath.Dir(handoffPath), 0755); err != nil {
		t.Fatalf("mkdir handoff dir: %v", err)
	}
	if err := os.WriteFile(handoffPath, []byte("# Colony Session — Paused Colony\n\nResume from here.\n"), 0644); err != nil {
		t.Fatalf("seed handoff: %v", err)
	}

	rootCmd.SetArgs([]string{"resume"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("resume returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if result["resumed"] != true {
		t.Fatalf("expected resumed:true, got %v", result)
	}
	if result["handoff_found"] != true {
		t.Fatalf("expected handoff_found:true, got %v", result["handoff_found"])
	}
}

func TestSwarmCompatibilityWatchReportsActiveWorkers(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	goal := "Watch worker activity"
	now := time.Now().UTC()
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          colony.StateEXECUTING,
		Scope:          colony.ScopeMeta,
		CurrentPhase:   1,
		BuildStartedAt: &now,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Execution", Status: colony.PhaseInProgress},
			},
		},
	})

	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	if err := spawnTree.RecordSpawn("Queen", "builder", "Hammer-1", "Fix the remaining issue", 1); err != nil {
		t.Fatalf("record spawn: %v", err)
	}
	if err := spawnTree.UpdateStatus("Hammer-1", "active", "Awaiting continue verification"); err != nil {
		t.Fatalf("mark active: %v", err)
	}
	if err := spawnTree.RecordSpawn("Queen", "watcher", "Keen-2", "Verify the fix", 1); err != nil {
		t.Fatalf("record completed spawn: %v", err)
	}
	if err := spawnTree.UpdateStatus("Keen-2", "completed", "Verified"); err != nil {
		t.Fatalf("mark completed: %v", err)
	}

	rootCmd.SetArgs([]string{"swarm", "--watch"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("swarm --watch returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if result["mode"] != "watch" {
		t.Fatalf("mode = %v, want watch", result["mode"])
	}
	if result["active_count"] != float64(1) {
		t.Fatalf("active_count = %v, want 1", result["active_count"])
	}
	if result["scope"] != "meta" {
		t.Fatalf("scope = %v, want meta", result["scope"])
	}
	if result["completed_count"] != float64(1) {
		t.Fatalf("completed_count = %v, want 1", result["completed_count"])
	}
	if result["recent_count"] != float64(1) {
		t.Fatalf("recent_count = %v, want 1", result["recent_count"])
	}
	if live, _ := result["live_refresh"].(bool); live {
		t.Fatalf("expected snapshot-style swarm watch result, got live_refresh=true")
	}
	recentWorkers := result["recent_workers"].([]interface{})
	if len(recentWorkers) != 1 {
		t.Fatalf("recent_workers len = %d, want 1", len(recentWorkers))
	}
	recentWorker := recentWorkers[0].(map[string]interface{})
	if recentWorker["name"] != "Keen-2" {
		t.Fatalf("recent worker name = %v, want Keen-2", recentWorker["name"])
	}
}

func TestSwarmCompatibilityWatchPrefersCurrentRunWorkers(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	goal := "Watch current run only"
	now := time.Now().UTC()
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          colony.StateEXECUTING,
		Scope:          colony.ScopeMeta,
		CurrentPhase:   1,
		BuildStartedAt: &now,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Execution", Status: colony.PhaseInProgress},
			},
		},
	})

	spawnData := strings.Join([]string{
		fmt.Sprintf("%s|Queen|builder|Ghost-41|Old worker|1|spawned", now.Add(-110*time.Second).Format(time.RFC3339)),
		fmt.Sprintf("%s|Ghost-41|active|Old active", now.Add(-100*time.Second).Format(time.RFC3339)),
		fmt.Sprintf("%s|Queen|builder|Hammer-1|Current worker|1|spawned", now.Add(-10*time.Second).Format(time.RFC3339)),
		fmt.Sprintf("%s|Hammer-1|active|Current active", now.Add(-5*time.Second).Format(time.RFC3339)),
	}, "\n") + "\n"
	if err := store.AtomicWrite("spawn-tree.txt", []byte(spawnData)); err != nil {
		t.Fatalf("write spawn-tree.txt: %v", err)
	}

	runState := fmt.Sprintf(`{
  "current_run_id": "run-current",
  "runs": [
    {
      "id": "run-old",
      "command": "build",
      "started_at": %q,
      "ended_at": %q,
      "status": "completed"
    },
    {
      "id": "run-current",
      "command": "build",
      "started_at": %q,
      "status": "active"
    }
  ]
}
`, now.Add(-2*time.Minute).Format(time.RFC3339), now.Add(-90*time.Second).Format(time.RFC3339), now.Add(-30*time.Second).Format(time.RFC3339))
	if err := store.AtomicWrite("spawn-runs.json", []byte(runState)); err != nil {
		t.Fatalf("write spawn-runs.json: %v", err)
	}

	rootCmd.SetArgs([]string{"swarm", "--watch"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("swarm --watch returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if result["active_count"] != float64(1) {
		t.Fatalf("active_count = %v, want 1", result["active_count"])
	}

	workers := result["active_workers"].([]interface{})
	if len(workers) != 1 {
		t.Fatalf("active_workers len = %d, want 1", len(workers))
	}
	worker := workers[0].(map[string]interface{})
	if worker["name"] != "Hammer-1" {
		t.Fatalf("worker name = %v, want Hammer-1", worker["name"])
	}
	recentWorkers := result["recent_workers"].([]interface{})
	if len(recentWorkers) != 0 {
		t.Fatalf("recent_workers len = %d, want 0", len(recentWorkers))
	}
}

func TestAutopilotSuccessStatusCountsAsCompleted(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	_, tmpDir := newTestStore(t)
	defer os.RemoveAll(tmpDir)
	var buf bytes.Buffer
	stdout = &buf

	rootCmd.SetArgs([]string{"autopilot-init", "--phases", "2"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("autopilot-init returned error: %v", err)
	}

	buf.Reset()
	rootCmd.SetArgs([]string{"autopilot-update", "--phase", "1", "--status", "success"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("autopilot-update returned error: %v", err)
	}

	buf.Reset()
	rootCmd.SetArgs([]string{"autopilot-check-replan", "--interval", "1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("autopilot-check-replan returned error: %v", err)
	}

	env := parseEnvelope(t, buf.String())
	result := env["result"].(map[string]interface{})
	if result["replan"] != true {
		t.Fatalf("expected replan:true after success status normalization, got %v", result)
	}
}

func TestWatchCompatibilityWritesArtifacts(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	goal := "Watch live workers"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{{ID: 1, Name: "Execution", Status: colony.PhaseInProgress}},
		},
	})

	spawnTree := agent.NewSpawnTree(store, "spawn-tree.txt")
	if err := spawnTree.RecordSpawn("Queen", "builder", "Hammer-9", "Investigate release issue", 1); err != nil {
		t.Fatalf("record spawn: %v", err)
	}
	if err := spawnTree.UpdateStatus("Hammer-9", "active", "Running"); err != nil {
		t.Fatalf("mark active: %v", err)
	}

	rootCmd.SetArgs([]string{"watch"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("watch returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if result["mode"] != "watch" {
		t.Fatalf("mode = %v, want watch", result["mode"])
	}
	if result["active_count"] != float64(1) {
		t.Fatalf("active_count = %v, want 1", result["active_count"])
	}

	statusPath := filepath.Join(dataDir, "watch-status.txt")
	progressPath := filepath.Join(dataDir, "watch-progress.txt")
	if _, err := os.Stat(statusPath); err != nil {
		t.Fatalf("watch-status.txt missing: %v", err)
	}
	if _, err := os.Stat(progressPath); err != nil {
		t.Fatalf("watch-progress.txt missing: %v", err)
	}
}

func TestSwarmCompatibilityWatchShowsRecoveryGuidance(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	goal := "Recover blocked phase from watch"
	now := time.Now().UTC()
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:        "3.0",
		Goal:           &goal,
		State:          colony.StateEXECUTING,
		CurrentPhase:   1,
		BuildStartedAt: &now,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Execution", Status: colony.PhaseInProgress},
			},
		},
	})
	seedBlockedContinueReport(t, dataDir, 1, now.Add(time.Second), "Recover the blocked task before rerunning verification", "aether build 1 --task 1.1", codexContinueRecoveryPlan{
		RedispatchTasks:   []string{"1.1"},
		RedispatchCommand: "aether build 1 --task 1.1",
	})

	rootCmd.SetArgs([]string{"swarm", "--watch"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("swarm --watch returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if result["next"] != "aether build 1 --task 1.1" {
		t.Fatalf("next = %v, want targeted recovery command", result["next"])
	}
	if result["recovery_summary"] != "Recover the blocked task before rerunning verification" {
		t.Fatalf("recovery_summary = %v, want blocked recovery summary", result["recovery_summary"])
	}
	if result["continue_report"] != ".aether/data/build/phase-1/continue.json" {
		t.Fatalf("continue_report = %v, want continue report path", result["continue_report"])
	}
}

func TestOracleCompatibilityRunsAutonomousLoop(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/oracle-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	agentsDir := filepath.Join(root, ".codex", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("mkdir agents: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentsDir, "aether-oracle.toml"), validCodexAgentTOML("aether-oracle", "oracle"), 0644); err != nil {
		t.Fatalf("write oracle agent: %v", err)
	}

	originalInvoker := newOracleWorkerInvoker
	newOracleWorkerInvoker = func() codex.WorkerInvoker { return &oracleCompletingInvoker{} }
	defer func() { newOracleWorkerInvoker = originalInvoker }()

	rootCmd.SetArgs([]string{"oracle", "release parity"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("oracle start returned error: %v", err)
	}

	startEnv := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	startResult := startEnv["result"].(map[string]interface{})
	if startResult["started"] != true {
		t.Fatalf("expected started:true, got %v", startResult)
	}
	if startResult["status"] != "complete" {
		t.Fatalf("status = %v, want complete", startResult["status"])
	}
	if startResult["autonomous"] != true {
		t.Fatalf("expected autonomous:true, got %v", startResult["autonomous"])
	}
	if startResult["detected_type"] != "go" {
		t.Fatalf("detected_type = %v, want go", startResult["detected_type"])
	}
	if startResult["answered_count"] != startResult["question_count"] {
		t.Fatalf("answered_count = %v, question_count = %v", startResult["answered_count"], startResult["question_count"])
	}
	if _, err := os.Stat(filepath.Join(root, ".aether", "oracle", ".loop-active")); !os.IsNotExist(err) {
		t.Fatalf("expected loop marker to be removed after completion, got err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".aether", "oracle", "responses")); err != nil {
		t.Fatalf("expected oracle responses dir to exist, got %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, ".aether", "oracle", "discoveries")); err != nil {
		t.Fatalf("expected oracle discoveries dir to exist, got %v", err)
	}

	stdout.(*bytes.Buffer).Reset()
	rootCmd.SetArgs([]string{"oracle", "status"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("oracle status returned error: %v", err)
	}

	statusEnv := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	statusResult := statusEnv["result"].(map[string]interface{})
	if statusResult["has_state"] != true || statusResult["has_plan"] != true {
		t.Fatalf("expected oracle workspace files to exist, got %v", statusResult)
	}
	if statusResult["status"] != "complete" {
		t.Fatalf("status = %v, want complete", statusResult["status"])
	}
}

func TestOracleCompatibilityStopCommandWritesMarker(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	stdout.(*bytes.Buffer).Reset()
	rootCmd.SetArgs([]string{"oracle", "stop"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("oracle stop returned error: %v", err)
	}

	stopEnv := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	stopResult := stopEnv["result"].(map[string]interface{})
	if stopResult["stopped"] != true {
		t.Fatalf("expected stopped:true, got %v", stopResult)
	}
	if _, err := os.Stat(filepath.Join(root, ".aether", "oracle", ".stop")); err != nil {
		t.Fatalf("expected .stop marker, got %v", err)
	}
}

func TestOracleCompatibilityStopKillsControllerProcessTree(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process-tree termination test is Unix-only")
	}

	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	oracleDir := filepath.Join(root, ".aether", "oracle")
	if err := os.MkdirAll(oracleDir, 0755); err != nil {
		t.Fatalf("mkdir oracle dir: %v", err)
	}

	cmd := exec.Command("sh", "-c", "sleep 30 & wait")
	if err := cmd.Start(); err != nil {
		t.Fatalf("start oracle controller fixture: %v", err)
	}
	t.Cleanup(func() {
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
	})

	var tree []int
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		var err error
		tree, err = oracleProcessTree(cmd.Process.Pid)
		if err == nil && len(tree) >= 2 {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if len(tree) < 2 {
		t.Fatalf("expected parent+child process tree, got %v", tree)
	}

	state := oracleStateFile{
		Version:       "1.1",
		Topic:         "stop test",
		Status:        "active",
		Phase:         "survey",
		Iteration:     1,
		MaxIterations: 8,
		Platform:      "codex",
		ControllerPID: cmd.Process.Pid,
	}
	if err := writeOracleStateFile(filepath.Join(oracleDir, "state.json"), state); err != nil {
		t.Fatalf("write oracle state: %v", err)
	}
	if err := os.WriteFile(filepath.Join(oracleDir, ".loop-active"), []byte("active\n"), 0644); err != nil {
		t.Fatalf("write loop marker: %v", err)
	}

	result, err := runOracleCompatibility(root, []string{"stop"})
	if err != nil {
		t.Fatalf("oracle stop returned error: %v", err)
	}
	if result["stopped"] != true {
		t.Fatalf("expected stopped:true, got %v", result)
	}

	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		alive := false
		for _, pid := range tree {
			if oracleProcessExists(pid) {
				alive = true
				break
			}
		}
		if !alive {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}

	t.Fatalf("expected oracle stop to kill process tree, still alive: %v", tree)
}

func TestOracleCompatibilityStopsAtIterationBoundary(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/oracle-stop-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	agentsDir := filepath.Join(root, ".codex", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("mkdir agents: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentsDir, "aether-oracle.toml"), validCodexAgentTOML("aether-oracle", "oracle"), 0644); err != nil {
		t.Fatalf("write oracle agent: %v", err)
	}

	originalInvoker := newOracleWorkerInvoker
	newOracleWorkerInvoker = func() codex.WorkerInvoker { return &oracleStopSignalInvoker{} }
	defer func() { newOracleWorkerInvoker = originalInvoker }()

	rootCmd.SetArgs([]string{"oracle", "manual stop test"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("oracle start returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if result["status"] != "stopped" {
		t.Fatalf("status = %v, want stopped", result["status"])
	}
	if result["stop_reason"] != "manual_stop" {
		t.Fatalf("stop_reason = %v, want manual_stop", result["stop_reason"])
	}
}

func TestOracleCompatibilityPersistsWorkerErrorReason(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/oracle-error-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	agentsDir := filepath.Join(root, ".codex", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("mkdir agents: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentsDir, "aether-oracle.toml"), validCodexAgentTOML("aether-oracle", "oracle"), 0644); err != nil {
		t.Fatalf("write oracle agent: %v", err)
	}

	originalInvoker := newOracleWorkerInvoker
	newOracleWorkerInvoker = func() codex.WorkerInvoker { return &oracleErrorInvoker{} }
	defer func() { newOracleWorkerInvoker = originalInvoker }()

	rootCmd.SetArgs([]string{"oracle", "worker error test"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("oracle start returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if result["status"] != "blocked" {
		t.Fatalf("status = %v, want blocked", result["status"])
	}
	if result["stop_reason"] != "worker_error" {
		t.Fatalf("stop_reason = %v, want worker_error", result["stop_reason"])
	}
	summary := result["summary"].(string)
	if !strings.Contains(summary, "invalid JSON") {
		t.Fatalf("summary = %q, want invalid JSON reason", summary)
	}
	if !strings.Contains(summary, ".aether/oracle/discoveries/iteration-01.json") {
		t.Fatalf("summary missing discovery artifact path: %q", summary)
	}
	if _, err := os.Stat(filepath.Join(root, ".aether", "oracle", "discoveries", "iteration-01.json")); err != nil {
		t.Fatalf("expected discovery artifact, got %v", err)
	}
}

func TestOracleCompatibilityRetriesRecoverableWorkerFailure(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/oracle-retry-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	agentsDir := filepath.Join(root, ".codex", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("mkdir agents: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentsDir, "aether-oracle.toml"), validCodexAgentTOML("aether-oracle", "oracle"), 0644); err != nil {
		t.Fatalf("write oracle agent: %v", err)
	}

	retrying := &oracleRetryInvoker{}
	originalInvoker := newOracleWorkerInvoker
	newOracleWorkerInvoker = func() codex.WorkerInvoker { return retrying }
	defer func() { newOracleWorkerInvoker = originalInvoker }()

	rootCmd.SetArgs([]string{"oracle", "retryable oracle topic"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("oracle start returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if result["status"] != "complete" {
		t.Fatalf("status = %v, want complete", result["status"])
	}
	if retrying.calls < 2 {
		t.Fatalf("calls = %d, want at least 2", retrying.calls)
	}

	data, err := os.ReadFile(filepath.Join(root, ".aether", "oracle", "discoveries", "iteration-01.json"))
	if err != nil {
		t.Fatalf("read discovery artifact: %v", err)
	}
	if !strings.Contains(string(data), "\"attempt\": 2") {
		t.Fatalf("expected canonical artifact to record the successful retry attempt, got:\n%s", string(data))
	}
}

func TestOracleCompatibilityAppliesSurveyAttemptPolicy(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/oracle-policy-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	agentsDir := filepath.Join(root, ".codex", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("mkdir agents: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentsDir, "aether-oracle.toml"), validCodexAgentTOML("aether-oracle", "oracle"), 0644); err != nil {
		t.Fatalf("write oracle agent: %v", err)
	}

	capturing := &oracleCapturingInvoker{}
	originalInvoker := newOracleWorkerInvoker
	newOracleWorkerInvoker = func() codex.WorkerInvoker { return capturing }
	defer func() { newOracleWorkerInvoker = originalInvoker }()

	rootCmd.SetArgs([]string{"oracle", "release parity"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("oracle start returned error: %v", err)
	}
	if len(capturing.configs) == 0 {
		t.Fatal("expected at least one oracle worker invocation")
	}

	first := capturing.configs[0]
	if first.Timeout != 3*time.Minute {
		t.Fatalf("survey timeout = %v, want 3m", first.Timeout)
	}
	if !containsString(first.ConfigOverrides, `model_reasoning_effort="low"`) {
		t.Fatalf("config overrides = %v, want survey low reasoning override", first.ConfigOverrides)
	}
	if first.ResponsePath == "" {
		t.Fatal("expected controller-managed oracle response path to be set")
	}
	if strings.Contains(first.TaskBrief, "Read .aether/utils/oracle/oracle.md") || strings.Contains(first.TaskBrief, "Update the complete .aether/oracle/state.json") {
		t.Fatalf("oracle task brief still tells workers to rewrite workspace files:\n%s", first.TaskBrief)
	}
	if !strings.Contains(first.TaskBrief, "Response File:") {
		t.Fatalf("oracle task brief missing response file contract:\n%s", first.TaskBrief)
	}
}

func TestOracleCompatibilityWritesHeartbeatWhileRunning(t *testing.T) {
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/oracle-heartbeat-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	agentsDir := filepath.Join(root, ".codex", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("mkdir agents: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentsDir, "aether-oracle.toml"), validCodexAgentTOML("aether-oracle", "oracle"), 0644); err != nil {
		t.Fatalf("write oracle agent: %v", err)
	}

	originalInvoker := newOracleWorkerInvoker
	newOracleWorkerInvoker = func() codex.WorkerInvoker { return &oracleSlowCompletingInvoker{delay: 80 * time.Millisecond} }
	defer func() { newOracleWorkerInvoker = originalInvoker }()

	originalPolicy := oracleAttemptPolicyForPhase
	oracleAttemptPolicyForPhase = func(phase string, attempt int) oracleAttemptPolicy {
		return oracleAttemptPolicy{
			ReasoningEffort: "low",
			Timeout:         250 * time.Millisecond,
			Heartbeat:       10 * time.Millisecond,
		}
	}
	defer func() { oracleAttemptPolicyForPhase = originalPolicy }()

	done := make(chan error, 1)
	go func() {
		_, err := runOracleCompatibility(root, []string{"heartbeat test"})
		done <- err
	}()

	statePath := filepath.Join(root, ".aether", "oracle", "state.json")
	deadline := time.Now().Add(300 * time.Millisecond)
	heartbeatSeen := false
	for time.Now().Before(deadline) {
		var state oracleStateFile
		data, err := os.ReadFile(statePath)
		if err == nil && json.Unmarshal(data, &state) == nil {
			if strings.Contains(state.Summary, "elapsed") {
				heartbeatSeen = true
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
	}

	if err := <-done; err != nil {
		t.Fatalf("runOracleCompatibility returned error: %v", err)
	}
	if !heartbeatSeen {
		t.Fatal("expected oracle heartbeat update while worker was still running")
	}
}

func TestOracleCompatibilityShortCircuitsOnValidResponseFile(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.com/oracle-response-shortcut-test\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write go.mod: %v", err)
	}
	agentsDir := filepath.Join(root, ".codex", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("mkdir agents: %v", err)
	}
	if err := os.WriteFile(filepath.Join(agentsDir, "aether-oracle.toml"), validCodexAgentTOML("aether-oracle", "oracle"), 0644); err != nil {
		t.Fatalf("write oracle agent: %v", err)
	}

	responseFirst := &oracleResponseFirstInvoker{}
	originalInvoker := newOracleWorkerInvoker
	newOracleWorkerInvoker = func() codex.WorkerInvoker { return responseFirst }
	defer func() { newOracleWorkerInvoker = originalInvoker }()

	originalPolicy := oracleAttemptPolicyForPhase
	oracleAttemptPolicyForPhase = func(phase string, attempt int) oracleAttemptPolicy {
		return oracleAttemptPolicy{
			ReasoningEffort: "low",
			Timeout:         250 * time.Millisecond,
			Heartbeat:       10 * time.Millisecond,
		}
	}
	defer func() { oracleAttemptPolicyForPhase = originalPolicy }()

	rootCmd.SetArgs([]string{"oracle", "release parity"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("oracle start returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if result["status"] != "complete" {
		t.Fatalf("status = %v, want complete", result["status"])
	}
	if atomic.LoadInt32(&responseFirst.cancelled) == 0 {
		t.Fatal("expected controller to cancel at least one worker after a valid response file appeared")
	}
}

func TestOracleCompatibilityRejectsDuplicateActiveLoop(t *testing.T) {
	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	oracleDir := filepath.Join(root, ".aether", "oracle")
	if err := os.MkdirAll(oracleDir, 0755); err != nil {
		t.Fatalf("mkdir oracle dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(oracleDir, ".loop-active"), []byte("active\n"), 0644); err != nil {
		t.Fatalf("write loop marker: %v", err)
	}
	if err := writeOracleStateFile(filepath.Join(oracleDir, "state.json"), oracleStateFile{
		Version:   "1.1",
		Topic:     "existing oracle loop",
		Phase:     "survey",
		Iteration: 1,
		Status:    "active",
		Platform:  "codex",
	}); err != nil {
		t.Fatalf("write oracle state: %v", err)
	}

	_, err := runOracleCompatibility(root, []string{"new topic"})
	if err == nil {
		t.Fatal("expected duplicate active loop error, got nil")
	}
	if !strings.Contains(err.Error(), "already active") {
		t.Fatalf("error = %v, want duplicate active loop message", err)
	}
}

type oracleCompletingInvoker struct{}

func (i *oracleCompletingInvoker) Invoke(ctx context.Context, cfg codex.WorkerConfig) (codex.WorkerResult, error) {
	if err := writeOracleTestResponse(cfg, oracleWorkerResponse{
		Status:     "answered",
		Confidence: 90,
		Summary:    "Autonomous oracle loop produced a source-backed answer.",
		Findings: []oracleWorkerFinding{{
			Text: "Autonomous oracle loop produced a source-backed answer.",
			Evidence: []oracleWorkerEvidence{{
				Title:    "Oracle loop implementation",
				Location: "cmd/oracle_loop.go",
				Type:     "codebase",
			}},
		}},
		Recommendation: "Continue until all planned questions are answered.",
	}); err != nil {
		return codex.WorkerResult{}, err
	}

	return codex.WorkerResult{
		WorkerName: cfg.WorkerName,
		Caste:      cfg.Caste,
		TaskID:     cfg.TaskID,
		Status:     "completed",
		Summary:    "Oracle iteration completed with fully answered questions.",
	}, nil
}

func (i *oracleCompletingInvoker) IsAvailable(ctx context.Context) bool { return true }
func (i *oracleCompletingInvoker) ValidateAgent(path string) error      { return nil }

type oracleStopSignalInvoker struct{}

func (i *oracleStopSignalInvoker) Invoke(ctx context.Context, cfg codex.WorkerConfig) (codex.WorkerResult, error) {
	paths := oracleWorkspacePaths(cfg.Root)
	now := time.Now().UTC().Format(time.RFC3339)
	if err := writeOracleTestResponse(cfg, oracleWorkerResponse{
		Status:     "partial",
		Confidence: 40,
		Summary:    "Progress made before a manual stop request.",
		Findings: []oracleWorkerFinding{{
			Text: "Progress made before a manual stop request.",
			Evidence: []oracleWorkerEvidence{{
				Title:    "Oracle loop implementation",
				Location: "cmd/oracle_loop.go",
				Type:     "codebase",
			}},
		}},
		Gaps: []string{"Remaining questions after stop request"},
	}); err != nil {
		return codex.WorkerResult{}, err
	}
	if err := os.WriteFile(paths.StopPath, []byte(now+"\n"), 0644); err != nil {
		return codex.WorkerResult{}, err
	}

	return codex.WorkerResult{
		WorkerName: cfg.WorkerName,
		Caste:      cfg.Caste,
		TaskID:     cfg.TaskID,
		Status:     "completed",
		Summary:    "Oracle iteration completed and a stop signal was emitted.",
	}, nil
}

func (i *oracleStopSignalInvoker) IsAvailable(ctx context.Context) bool { return true }
func (i *oracleStopSignalInvoker) ValidateAgent(path string) error      { return nil }

type oracleErrorInvoker struct{}

func (i *oracleErrorInvoker) Invoke(ctx context.Context, cfg codex.WorkerConfig) (codex.WorkerResult, error) {
	return codex.WorkerResult{
		WorkerName: cfg.WorkerName,
		Caste:      cfg.Caste,
		TaskID:     cfg.TaskID,
		Status:     "failed",
		RawOutput:  "worker emitted malformed output",
		Error:      fmt.Errorf("parse worker output: invalid JSON"),
	}, nil
}

func (i *oracleErrorInvoker) IsAvailable(ctx context.Context) bool { return true }
func (i *oracleErrorInvoker) ValidateAgent(path string) error      { return nil }

type oracleCapturingInvoker struct {
	configs []codex.WorkerConfig
}

func (i *oracleCapturingInvoker) Invoke(ctx context.Context, cfg codex.WorkerConfig) (codex.WorkerResult, error) {
	i.configs = append(i.configs, cfg)
	if err := writeOracleTestResponse(cfg, oracleWorkerResponse{
		Status:     "answered",
		Confidence: 86,
		Summary:    "Oracle worker invocation captured the expected policy overrides.",
		Findings: []oracleWorkerFinding{{
			Text: "Oracle worker invocation captured the expected policy overrides.",
			Evidence: []oracleWorkerEvidence{{
				Title:    "Oracle policy capture test",
				Location: "cmd/oracle_loop.go",
				Type:     "codebase",
			}},
		}},
	}); err != nil {
		return codex.WorkerResult{}, err
	}

	return codex.WorkerResult{
		WorkerName: cfg.WorkerName,
		Caste:      cfg.Caste,
		TaskID:     cfg.TaskID,
		Status:     "completed",
		Summary:    "Oracle policy capture completed.",
	}, nil
}

func (i *oracleCapturingInvoker) IsAvailable(ctx context.Context) bool { return true }
func (i *oracleCapturingInvoker) ValidateAgent(path string) error      { return nil }

type oracleSlowCompletingInvoker struct {
	delay time.Duration
}

func (i *oracleSlowCompletingInvoker) Invoke(ctx context.Context, cfg codex.WorkerConfig) (codex.WorkerResult, error) {
	timer := time.NewTimer(i.delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return codex.WorkerResult{
			WorkerName: cfg.WorkerName,
			Caste:      cfg.Caste,
			TaskID:     cfg.TaskID,
			Status:     "failed",
			Error:      ctx.Err(),
		}, nil
	case <-timer.C:
	}

	if err := writeOracleTestResponse(cfg, oracleWorkerResponse{
		Status:     "answered",
		Confidence: 87,
		Summary:    "Heartbeat test completed after a deliberately slow worker pass.",
		Findings: []oracleWorkerFinding{{
			Text: "Heartbeat test completed after a deliberately slow worker pass.",
			Evidence: []oracleWorkerEvidence{{
				Title:    "Oracle heartbeat test",
				Location: "cmd/oracle_loop.go",
				Type:     "codebase",
			}},
		}},
	}); err != nil {
		return codex.WorkerResult{}, err
	}

	return codex.WorkerResult{
		WorkerName: cfg.WorkerName,
		Caste:      cfg.Caste,
		TaskID:     cfg.TaskID,
		Status:     "completed",
		Summary:    "Oracle slow worker completed.",
	}, nil
}

func (i *oracleSlowCompletingInvoker) IsAvailable(ctx context.Context) bool { return true }
func (i *oracleSlowCompletingInvoker) ValidateAgent(path string) error      { return nil }

type oracleRetryInvoker struct {
	calls int
}

func (i *oracleRetryInvoker) Invoke(ctx context.Context, cfg codex.WorkerConfig) (codex.WorkerResult, error) {
	i.calls++
	if i.calls == 1 {
		return codex.WorkerResult{
			WorkerName: cfg.WorkerName,
			Caste:      cfg.Caste,
			TaskID:     cfg.TaskID,
			Status:     "failed",
			Summary:    "first attempt failed",
			Error:      fmt.Errorf("parse worker output: truncated JSON"),
			RawOutput:  "malformed output",
		}, nil
	}

	if err := writeOracleTestResponse(cfg, oracleWorkerResponse{
		Status:     "answered",
		Confidence: 88,
		Summary:    "Retry path completed the oracle iteration successfully.",
		Findings: []oracleWorkerFinding{{
			Text: "Retry path completed the oracle iteration successfully.",
			Evidence: []oracleWorkerEvidence{{
				Title:    "Oracle retry test",
				Location: "cmd/oracle_loop.go",
				Type:     "codebase",
			}},
		}},
	}); err != nil {
		return codex.WorkerResult{}, err
	}

	return codex.WorkerResult{
		WorkerName: cfg.WorkerName,
		Caste:      cfg.Caste,
		TaskID:     cfg.TaskID,
		Status:     "completed",
		Summary:    "Oracle retry completed successfully.",
	}, nil
}

func (i *oracleRetryInvoker) IsAvailable(ctx context.Context) bool { return true }
func (i *oracleRetryInvoker) ValidateAgent(path string) error      { return nil }

type oracleResponseFirstInvoker struct {
	cancelled int32
}

func (i *oracleResponseFirstInvoker) Invoke(ctx context.Context, cfg codex.WorkerConfig) (codex.WorkerResult, error) {
	if err := writeOracleTestResponse(cfg, oracleWorkerResponse{
		Status:     "answered",
		Confidence: 85,
		Summary:    "Response file was written before the worker finished its normal final message path.",
		Findings: []oracleWorkerFinding{{
			Text: "The controller can consume a valid response file before the nested worker emits its final claims JSON.",
			Evidence: []oracleWorkerEvidence{{
				Title:    "Oracle response shortcut test",
				Location: "cmd/oracle_loop.go",
				Type:     "codebase",
			}},
		}},
	}); err != nil {
		return codex.WorkerResult{}, err
	}

	<-ctx.Done()
	atomic.AddInt32(&i.cancelled, 1)
	return codex.WorkerResult{
		WorkerName: cfg.WorkerName,
		Caste:      cfg.Caste,
		TaskID:     cfg.TaskID,
		Status:     "failed",
		Error:      ctx.Err(),
	}, nil
}

func (i *oracleResponseFirstInvoker) IsAvailable(ctx context.Context) bool { return true }
func (i *oracleResponseFirstInvoker) ValidateAgent(path string) error      { return nil }

func writeOracleTestResponse(cfg codex.WorkerConfig, response oracleWorkerResponse) error {
	if strings.TrimSpace(cfg.ResponsePath) == "" {
		return fmt.Errorf("missing oracle response path")
	}
	data, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(cfg.ResponsePath), 0755); err != nil {
		return err
	}
	return os.WriteFile(cfg.ResponsePath, append(data, '\n'), 0644)
}

func TestRunCompatibilityDryRunPlansLifecycle(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	goal := "Ship the release"
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Status: colony.PhaseReady, Tasks: []colony.Task{{ID: ptrString("1.1"), Goal: "Do work", Status: colony.TaskPending}}},
				{ID: 2, Name: "Phase 2", Status: colony.PhasePending, Tasks: []colony.Task{{ID: ptrString("2.1"), Goal: "More work", Status: colony.TaskPending}}},
			},
		},
	})

	rootCmd.SetArgs([]string{"run", "--dry-run", "--max-phases", "1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("run --dry-run returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if result["dry_run"] != true {
		t.Fatalf("expected dry_run:true, got %v", result)
	}
	if result["stopped_reason"] != "max_phases_reached" {
		t.Fatalf("stopped_reason = %v, want max_phases_reached", result["stopped_reason"])
	}
	steps := result["steps"].([]interface{})
	if len(steps) != 2 {
		t.Fatalf("expected 2 dry-run steps, got %d", len(steps))
	}
}

func TestRunCompatibilityExecutesSinglePhase(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(dataDir))
	withWorkingDir(t, root)

	agentsDir := filepath.Join(root, ".codex", "agents")
	if err := os.MkdirAll(agentsDir, 0755); err != nil {
		t.Fatalf("mkdir agents: %v", err)
	}
	for _, agentName := range []string{"aether-builder.toml", "aether-watcher.toml"} {
		role := strings.TrimSuffix(strings.TrimPrefix(agentName, "aether-"), ".toml")
		if err := os.WriteFile(filepath.Join(agentsDir, agentName), validCodexAgentTOML(strings.TrimSuffix(agentName, ".toml"), role), 0644); err != nil {
			t.Fatalf("write %s: %v", agentName, err)
		}
	}

	goal := "Run one autopilot phase"
	now := time.Now().UTC()
	createTestColonyState(t, dataDir, colony.ColonyState{
		Version:       "3.0",
		Goal:          &goal,
		State:         colony.StateREADY,
		CurrentPhase:  1,
		InitializedAt: &now,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase 1", Status: colony.PhaseReady, Tasks: []colony.Task{{ID: ptrString("1.1"), Goal: "Implement it", Status: colony.TaskPending}}},
			},
		},
	})

	rootCmd.SetArgs([]string{"run", "--max-phases", "1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("run returned error: %v", err)
	}

	env := parseEnvelope(t, stdout.(*bytes.Buffer).String())
	result := env["result"].(map[string]interface{})
	if result["completed"] != false {
		t.Fatalf("expected completed:false when run stops on a simulated build, got %v", result)
	}
	if result["stopped_reason"] != "blocked" {
		t.Fatalf("stopped_reason = %v, want blocked", result["stopped_reason"])
	}
	if result["next"] != "aether build 1 --task 1.1" {
		t.Fatalf("next = %v, want task-scoped redispatch", result["next"])
	}
	if result["phases_completed"] != float64(0) {
		t.Fatalf("phases_completed = %v, want 0", result["phases_completed"])
	}

	var autopilot autopilotState
	if err := store.LoadJSON(autopilotStatePath, &autopilot); err != nil {
		t.Fatalf("load autopilot state: %v", err)
	}
	if autopilot.Status != "paused" {
		t.Fatalf("autopilot status = %q, want paused", autopilot.Status)
	}
}
