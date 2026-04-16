package cmd

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

func setHookStdin(t *testing.T, payload string) {
	t.Helper()
	oldStdin := os.Stdin
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	if _, err := w.Write([]byte(payload)); err != nil {
		t.Fatalf("write hook stdin: %v", err)
	}
	_ = w.Close()
	os.Stdin = r
	t.Cleanup(func() {
		os.Stdin = oldStdin
		_ = r.Close()
	})
}

func TestHookPreToolUseBlocksProtectedPath(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	_, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)

	protected := filepath.Join(tmpDir, ".aether", "data", "COLONY_STATE.json")
	setHookStdin(t, `{"hook_event_name":"PreToolUse","tool_name":"Write","tool_input":{"file_path":"`+protected+`"}}`)

	rootCmd.SetArgs([]string{"hook-pre-tool-use"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("hook-pre-tool-use returned error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &result); err != nil {
		t.Fatalf("unmarshal hook output: %v", err)
	}
	if result["decision"] != "block" {
		t.Fatalf("decision = %v, want block", result["decision"])
	}
	if !strings.Contains(result["reason"].(string), "aether") {
		t.Fatalf("reason = %q, want guidance to use aether CLI", result["reason"])
	}
}

func TestHookPreToolUseBlocksMainBranchWhenRedirectActive(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)

	oldWD, _ := os.Getwd()
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("chdir temp repo: %v", err)
	}
	t.Cleanup(func() { _ = os.Chdir(oldWD) })

	if out, err := exec.Command("git", "init", "-b", "main").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v: %s", err, out)
	}

	goal := "test hook redirects"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{{ID: 1, Name: "Phase One", Status: colony.PhaseInProgress}},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	redirectStrength := 1.0
	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{
				ID:        "sig_1",
				Type:      "REDIRECT",
				Priority:  "high",
				Source:    "user",
				CreatedAt: "2026-04-15T18:00:00Z",
				Active:    true,
				Strength:  &redirectStrength,
				Content:   json.RawMessage(`{"text":"Avoid modifying files directly on the main branch during builds -- all changes go through PRs"}`),
			},
		},
	}
	if err := s.SaveJSON("pheromones.json", pf); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(tmpDir, "internal.go")
	setHookStdin(t, `{"hook_event_name":"PreToolUse","tool_name":"Write","tool_input":{"file_path":"`+target+`"}}`)

	rootCmd.SetArgs([]string{"hook-pre-tool-use"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("hook-pre-tool-use returned error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &result); err != nil {
		t.Fatalf("unmarshal hook output: %v", err)
	}
	if result["decision"] != "block" {
		t.Fatalf("decision = %v, want block", result["decision"])
	}
	if !strings.Contains(result["reason"].(string), "main") {
		t.Fatalf("reason = %q, want main-branch redirect guidance", result["reason"])
	}
}

func TestHookStopBlocksActiveExecution(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)

	goal := "test hook stop"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 2,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase One", Status: colony.PhaseCompleted},
				{ID: 2, Name: "Phase Two", Status: colony.PhaseInProgress},
			},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	setHookStdin(t, `{"hook_event_name":"Stop","stop_hook_active":false}`)
	rootCmd.SetArgs([]string{"hook-stop"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("hook-stop returned error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &result); err != nil {
		t.Fatalf("unmarshal hook output: %v", err)
	}
	if result["decision"] != "block" {
		t.Fatalf("decision = %v, want block", result["decision"])
	}
	if !strings.Contains(result["reason"].(string), "aether continue") {
		t.Fatalf("reason = %q, want continue guidance", result["reason"])
	}
}

func TestHookPreCompactUpdatesSessionSummary(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)

	goal := "test compact snapshot"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Milestone:    "First Mound",
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Phase One", Status: colony.PhaseReady},
			},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	setHookStdin(t, `{"hook_event_name":"PreCompact","trigger":"manual"}`)
	rootCmd.SetArgs([]string{"hook-pre-compact"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("hook-pre-compact returned error: %v", err)
	}

	if strings.TrimSpace(buf.String()) != "" {
		t.Fatalf("expected no stdout from hook-pre-compact, got %q", buf.String())
	}

	var session colony.SessionFile
	if err := s.LoadJSON("session.json", &session); err != nil {
		t.Fatalf("load session.json: %v", err)
	}
	if session.LastCommand != "hook-pre-compact" {
		t.Fatalf("LastCommand = %q, want hook-pre-compact", session.LastCommand)
	}
	if session.SuggestedNext != "aether build 1" {
		t.Fatalf("SuggestedNext = %q, want aether build 1", session.SuggestedNext)
	}
	if !strings.Contains(session.Summary, "manual") {
		t.Fatalf("Summary = %q, want manual trigger context", session.Summary)
	}
}
