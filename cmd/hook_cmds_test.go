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

// TestHookPreToolUseAllowsSanctionedScratchDirs is the positive companion to
// TestHookPreToolUseBlocksProtectedPath. It proves the exact-subpath allowlist
// in sanctionedDataWritePrefixes lets a worker write the sanctioned
// scratch subdirectories under .aether/data/ while everything else stays
// blocked — including behavior 6 (a protected file NOT in the allowlist) and
// behavior 7 (a path that merely contains a sanctioned name as a substring,
// not as a directory segment).
func TestHookPreToolUseAllowsSanctionedScratchDirs(t *testing.T) {
	allowed := []struct {
		name string
		rel  string
	}{
		{"planning_dir", filepath.Join(".aether", "data", "planning", "phase-plan.json")},
		{"phase_research_dir", filepath.Join(".aether", "data", "phase-research", "phase-3-research.md")},
		{"survey_dir", filepath.Join(".aether", "data", "survey", "BLUEPRINT.md")},
		{"worker_debug_dir", filepath.Join(".aether", "data", "worker-debug", "worker-1.json")},
		// codex_workflow_cmds.go builds every territory-refresh dispatch output
		// under .aether/data/territory-candidates/<transaction-id>/survey/, so a
		// surveyor ordered there must not be blocked by its own runtime.
		{"territory_candidates_dir", filepath.Join(".aether", "data", "territory-candidates", "territory-1700000000000000000-abcd1234", "survey", "PATHOGENS.md")},
	}
	for _, tt := range allowed {
		t.Run(tt.name, func(t *testing.T) {
			saveGlobalsCmd(t)
			resetRootCmd(t)

			var buf bytes.Buffer
			stdout = &buf
			var errBuf bytes.Buffer
			stderr = &errBuf

			_, tmpDir := newTestStoreCmd(t)
			defer os.RemoveAll(tmpDir)

			target := filepath.Join(tmpDir, tt.rel)
			setHookStdin(t, `{"hook_event_name":"PreToolUse","tool_name":"Write","tool_input":{"file_path":"`+target+`"}}`)

			rootCmd.SetArgs([]string{"hook-pre-tool-use"})
			if err := rootCmd.Execute(); err != nil {
				t.Fatalf("hook-pre-tool-use returned error: %v", err)
			}

			if strings.TrimSpace(buf.String()) != "" {
				t.Fatalf("sanctioned scratch write was blocked: %s", buf.String())
			}
		})
	}

	// Behavior 6: a protected file that is NOT in the allowlist (pheromones.json,
	// a direct child of .aether/data/) must still be blocked. Asserted by name so
	// this does not merely pass incidentally via the untouched blanket case.
	t.Run("behavior_6_pheromones_json_still_blocked", func(t *testing.T) {
		saveGlobalsCmd(t)
		resetRootCmd(t)

		var buf bytes.Buffer
		stdout = &buf
		var errBuf bytes.Buffer
		stderr = &errBuf

		_, tmpDir := newTestStoreCmd(t)
		defer os.RemoveAll(tmpDir)

		target := filepath.Join(tmpDir, ".aether", "data", "pheromones.json")
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
	})

	// Behavior 7: a path that merely contains a sanctioned name as a substring,
	// not as a directory segment, must still be blocked — proving the allowlist
	// match requires full slash-delimited segments, not a bare name match.
	t.Run("behavior_7_substring_not_segment_still_blocked", func(t *testing.T) {
		saveGlobalsCmd(t)
		resetRootCmd(t)

		var buf bytes.Buffer
		stdout = &buf
		var errBuf bytes.Buffer
		stderr = &errBuf

		_, tmpDir := newTestStoreCmd(t)
		defer os.RemoveAll(tmpDir)

		target := filepath.Join(tmpDir, ".aether", "data", "planning-notes.json")
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
	})
}

// TestHookPreToolUseSymlinkInsideSanctionedDirCannotEscape is WR-05's
// regression proof: normalizeHookPath used to be purely lexical, so a
// symlink planted inside an allowlisted scratch subdir (planning/) but
// pointing OUTSIDE it, at protected colony state, would lexically match the
// planning/ prefix and be allowed through -- even though the write actually
// lands on COLONY_STATE.json. Resolving symlinks before allowlist matching
// must catch this.
func TestHookPreToolUseSymlinkInsideSanctionedDirCannotEscape(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	_, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)

	planningDir := filepath.Join(tmpDir, ".aether", "data", "planning")
	if err := os.MkdirAll(planningDir, 0755); err != nil {
		t.Fatalf("mkdir planning dir: %v", err)
	}
	protected := filepath.Join(tmpDir, ".aether", "data", "COLONY_STATE.json")
	if err := os.WriteFile(protected, []byte("{}"), 0644); err != nil {
		t.Fatalf("seed protected file: %v", err)
	}

	// Symlink target must exist for filepath.EvalSymlinks to resolve fully.
	escape := filepath.Join(planningDir, "escape.json")
	if err := os.Symlink(protected, escape); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	setHookStdin(t, `{"hook_event_name":"PreToolUse","tool_name":"Write","tool_input":{"file_path":"`+escape+`"}}`)

	rootCmd.SetArgs([]string{"hook-pre-tool-use"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("hook-pre-tool-use returned error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &result); err != nil {
		t.Fatalf("expected a block decision for the symlink escape, got unparseable output: %v (%q)", err, buf.String())
	}
	if result["decision"] != "block" {
		t.Fatalf("symlink escape from sanctioned planning/ dir onto COLONY_STATE.json was not blocked: %v", result)
	}
}

// TestSanctionedScratchDirsDocumented makes the "four documents describe the
// same sanctioned scratch subpaths as the real enforcement code" claim
// checkable rather than prose that drifts. Deleting any one sanctioned
// subpath from any one of the four documents must fail this test.
func TestSanctionedScratchDirsDocumented(t *testing.T) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		t.Fatalf("findRepoRoot: %v", err)
	}

	docs := []string{
		filepath.Join(repoRoot, ".aether", "rules", "aether-colony.md"),
		filepath.Join(repoRoot, ".claude", "rules", "aether-colony.md"),
		filepath.Join(repoRoot, ".opencode", "OPENCODE.md"),
		filepath.Join(repoRoot, ".aether", "references", "contracts", "protected-local-state-contract.md"),
	}

	for _, doc := range docs {
		content, err := os.ReadFile(doc)
		if err != nil {
			t.Fatalf("read %s: %v", doc, err)
		}
		text := string(content)
		for _, prefix := range sanctionedDataWritePrefixes {
			// sanctionedDataWritePrefixes entries are host-relative match
			// segments ("/.aether/data/planning/"); documentation names the
			// repo-relative path without the leading slash.
			repoRelative := strings.TrimPrefix(prefix, "/")
			if !strings.Contains(text, repoRelative) {
				t.Fatalf("%s does not name sanctioned scratch subpath %q", doc, repoRelative)
			}
		}
	}
}

// TestHookPreToolUseCapturesRawPayloadOnlyWhenCaptureFileIsSet is Phase 173
// (SPAWN-04) Wave 0's proof that the AETHER_HOOK_CAPTURE_FILE recorder is
// off by default and, when on, never changes the hook's allow/deny answer.
// Run twice: once with the env var unset (no file, no capture) and once with
// it set to a path under t.TempDir() (file exists, contents parse as JSON,
// round-tripped tool_name matches). Both runs assert no block decision.
func TestHookPreToolUseCapturesRawPayloadOnlyWhenCaptureFileIsSet(t *testing.T) {
	payload := `{"hook_event_name":"PreToolUse","tool_name":"Task","tool_input":{}}`

	t.Run("capture_off_by_default", func(t *testing.T) {
		saveGlobalsCmd(t)
		resetRootCmd(t)

		var buf bytes.Buffer
		stdout = &buf
		var errBuf bytes.Buffer
		stderr = &errBuf

		_, tmpDir := newTestStoreCmd(t)
		defer os.RemoveAll(tmpDir)

		t.Setenv("AETHER_HOOK_CAPTURE_FILE", "")
		setHookStdin(t, payload)

		rootCmd.SetArgs([]string{"hook-pre-tool-use"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("hook-pre-tool-use returned error: %v", err)
		}

		if strings.TrimSpace(buf.String()) != "" {
			t.Fatalf("expected no stdout (no decision) when capture is off, got %q", buf.String())
		}

		captureCandidate := filepath.Join(tmpDir, "hook-capture.jsonl")
		if _, err := os.Stat(captureCandidate); err == nil {
			t.Fatalf("capture file was created even though AETHER_HOOK_CAPTURE_FILE was unset")
		}
	})

	t.Run("capture_on_when_env_set", func(t *testing.T) {
		saveGlobalsCmd(t)
		resetRootCmd(t)

		var buf bytes.Buffer
		stdout = &buf
		var errBuf bytes.Buffer
		stderr = &errBuf

		_, tmpDir := newTestStoreCmd(t)
		defer os.RemoveAll(tmpDir)

		captureFile := filepath.Join(t.TempDir(), "hook-capture.jsonl")
		t.Setenv("AETHER_HOOK_CAPTURE_FILE", captureFile)
		setHookStdin(t, payload)

		rootCmd.SetArgs([]string{"hook-pre-tool-use"})
		if err := rootCmd.Execute(); err != nil {
			t.Fatalf("hook-pre-tool-use returned error: %v", err)
		}

		if strings.TrimSpace(buf.String()) != "" {
			t.Fatalf("capture must never change the hook's answer, got stdout %q", buf.String())
		}

		data, err := os.ReadFile(captureFile)
		if err != nil {
			t.Fatalf("expected capture file to exist: %v", err)
		}
		var captured map[string]interface{}
		lines := strings.Split(strings.TrimSpace(string(data)), "\n")
		if len(lines) != 1 {
			t.Fatalf("expected exactly one captured line, got %d: %q", len(lines), data)
		}
		if err := json.Unmarshal([]byte(lines[0]), &captured); err != nil {
			t.Fatalf("captured content did not parse as JSON: %v (%q)", err, data)
		}
		if captured["tool_name"] != "Task" {
			t.Fatalf("captured tool_name = %v, want Task", captured["tool_name"])
		}
	})
}

// TestHookCaptureHasNoFileBasedSwitch is 173-REVIEW.md CR-01's regression
// lock: the raw-payload recorder must be switchable ONLY by the
// AETHER_HOOK_CAPTURE_FILE environment variable, never by a file a worker
// could write with its ordinary Write tool. A sentinel file at
// ~/.aether/hook-capture-path briefly existed (2026-08-13) and let capture
// be aimed at protected state or the spawn ledger; this test fails if any
// change makes the hook honor that file again.
func TestHookCaptureHasNoFileBasedSwitch(t *testing.T) {
	payload := `{"hook_event_name":"PreToolUse","tool_name":"Task","tool_input":{}}`

	saveGlobalsCmd(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	_, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)

	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("AETHER_HOOK_CAPTURE_FILE", "")

	captureFile := filepath.Join(t.TempDir(), "hook-capture.jsonl")
	if err := os.MkdirAll(filepath.Join(home, ".aether"), 0o755); err != nil {
		t.Fatalf("could not create fake ~/.aether: %v", err)
	}
	if err := os.WriteFile(filepath.Join(home, ".aether", "hook-capture-path"), []byte(captureFile+"\n"), 0o600); err != nil {
		t.Fatalf("could not write sentinel file: %v", err)
	}

	setHookStdin(t, payload)

	rootCmd.SetArgs([]string{"hook-pre-tool-use"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("hook-pre-tool-use returned error: %v", err)
	}

	if strings.TrimSpace(buf.String()) != "" {
		t.Fatalf("capture must never change the hook's answer, got stdout %q", buf.String())
	}

	if _, err := os.Stat(captureFile); err == nil {
		t.Fatalf("capture file was written via the sentinel file: the file-based switch is back (CR-01)")
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
				CreatedAt: time.Now().UTC().Format(time.RFC3339),
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

// TestHookPreToolUseDeniesUnresolvedRequesterDepth is D-20's direct proof: a
// subagent-originated dispatch (agent_id present) whose agent_type is
// "general-purpose" -- the exact value 173-HOOK-FINDINGS.md observed on the
// real captured inner payload -- cannot be classified by the resolution rule
// (it is indistinguishable from a first-tier worker using the documented
// general-purpose fallback), so the hook must deny rather than guess.
func TestHookPreToolUseDeniesUnresolvedRequesterDepth(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	_, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)

	setHookStdin(t, `{"session_id":"sess_1","agent_id":"ae93ff782863d564f","agent_type":"general-purpose","hook_event_name":"PreToolUse","tool_name":"Agent","tool_input":{"description":"Dispatch leaf agent","prompt":"Reply with the single word: leaf","subagent_type":"general-purpose"}}`)

	rootCmd.SetArgs([]string{"hook-pre-tool-use"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("hook-pre-tool-use returned error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &result); err != nil {
		t.Fatalf("unmarshal hook output: %v (%q)", err, buf.String())
	}
	if result["decision"] != "block" {
		t.Fatalf("decision = %v, want block", result["decision"])
	}
	reason, _ := result["reason"].(string)
	if !strings.Contains(reason, "cannot resolve who is asking") {
		t.Fatalf("reason = %q, want it to name the inability to identify the requester", reason)
	}
}

// TestHookPreToolUseAllowsMainSessionDispatch is the negative control D-20
// requires: a dispatch payload carrying the same tool_name but NO
// agent_id/agent_type field at all -- the coordinator's own dispatch shape,
// exactly as 173-HOOK-FINDINGS.md's first captured payload showed it (no
// agent_id, no agent_type). Without this control, a hook that blocks every
// dispatch unconditionally would also satisfy the deny tests, which would
// stop the coordinator dispatching its own workers and break every build.
func TestHookPreToolUseAllowsMainSessionDispatch(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	_, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)

	setHookStdin(t, `{"session_id":"sess_1","hook_event_name":"PreToolUse","tool_name":"Agent","tool_input":{"description":"Nested dispatch experiment level 1","subagent_type":"general-purpose"}}`)

	rootCmd.SetArgs([]string{"hook-pre-tool-use"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("hook-pre-tool-use returned error: %v", err)
	}

	if strings.TrimSpace(buf.String()) != "" {
		t.Fatalf("main session dispatch was blocked: %s", buf.String())
	}
}

// TestHookPreToolUseAllowsFirstTierWorkerDispatch proves D-01: the
// coordinator's own workers may each call one round of helpers. A subagent
// requester whose agent_type carries the repo's aether-* worker naming
// convention resolves to depth 1, and a depth-1 requester's own dispatch
// (the child would be depth 2) is within the cap.
func TestHookPreToolUseAllowsFirstTierWorkerDispatch(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	_, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)

	setHookStdin(t, `{"session_id":"sess_1","agent_id":"agent_first_tier_1","agent_type":"aether-builder","hook_event_name":"PreToolUse","tool_name":"Agent","tool_input":{"description":"Dispatch a helper","subagent_type":"general-purpose"}}`)

	rootCmd.SetArgs([]string{"hook-pre-tool-use"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("hook-pre-tool-use returned error: %v", err)
	}

	if strings.TrimSpace(buf.String()) != "" {
		t.Fatalf("first-tier worker's own dispatch was blocked: %s", buf.String())
	}
}

// TestHookPreToolUseDeniesPastTheDepthCap is D-01's other half: a
// second-tier helper must not be able to spawn a third level.
//
// Which case the findings actually supported: 173-HOOK-FINDINGS.md's
// capture could not produce a payload that resolves to an authoritative
// depth-2 requester -- agent_type names the REQUESTER's own dispatched
// type, and a first-tier worker following .aether/workers.md's documented
// "subagent_type=general-purpose" fallback verbatim is, on this one field,
// indistinguishable from a second-tier helper doing the same thing. So this
// test exercises the case the findings actually proved: the general-purpose
// value is treated as unresolved (fail-closed) rather than silently
// classified as an allowed depth-1 requester -- which is exactly what would
// let a real depth-2 helper spawn a third level undetected. A genuine
// resolved-depth-2 cap denial is not demonstrated by the capture and is not
// claimed here.
func TestHookPreToolUseDeniesPastTheDepthCap(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	_, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)

	setHookStdin(t, `{"session_id":"sess_1","agent_id":"agent_second_tier_1","agent_type":"general-purpose","hook_event_name":"PreToolUse","tool_name":"Agent","tool_input":{"description":"A second-tier helper attempting to spawn a third level","subagent_type":"general-purpose"}}`)

	rootCmd.SetArgs([]string{"hook-pre-tool-use"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("hook-pre-tool-use returned error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &result); err != nil {
		t.Fatalf("unmarshal hook output: %v (%q)", err, buf.String())
	}
	if result["decision"] != "block" {
		t.Fatalf("decision = %v, want block", result["decision"])
	}
	reason, _ := result["reason"].(string)
	if !strings.Contains(reason, "cannot resolve") && !strings.Contains(reason, "past the cap") {
		t.Fatalf("reason = %q, want it to name the cap or the unresolvability", reason)
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

func TestHookStopAllowsPausedActiveExecution(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)

	goal := "test paused hook stop"
	pausedAt := time.Now().UTC().Format(time.RFC3339)
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 2,
		Paused:       true,
		PausedAt:     &pausedAt,
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

	if got := strings.TrimSpace(buf.String()); got != "" {
		t.Fatalf("expected paused active state to allow stop without stdout, got %q", got)
	}
}

func TestHookStopAllowsImmediatePostResumeStop(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)

	goal := "test resumed hook stop"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateBUILT,
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
	if err := s.SaveJSON("session.json", colony.SessionFile{
		SessionID:     "resume-hook-test",
		LastCommand:   "resume-colony",
		LastCommandAt: time.Now().UTC().Format(time.RFC3339),
		ColonyGoal:    goal,
		CurrentPhase:  2,
		SuggestedNext: "aether continue",
	}); err != nil {
		t.Fatal(err)
	}

	setHookStdin(t, `{"hook_event_name":"Stop","stop_hook_active":false}`)
	rootCmd.SetArgs([]string{"hook-stop"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("hook-stop returned error: %v", err)
	}

	if got := strings.TrimSpace(buf.String()); got != "" {
		t.Fatalf("expected immediate post-resume stop to allow without stdout, got %q", got)
	}
}

func TestHookStopBlocksStalePostResumeStop(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)

	goal := "test stale resumed hook stop"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateBUILT,
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
	if err := s.SaveJSON("session.json", colony.SessionFile{
		SessionID:     "stale-resume-hook-test",
		LastCommand:   "resume-colony",
		LastCommandAt: time.Now().UTC().Add(-time.Hour).Format(time.RFC3339),
		ColonyGoal:    goal,
		CurrentPhase:  2,
		SuggestedNext: "aether continue",
	}); err != nil {
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
	if session.SuggestedNext != "aether status" {
		t.Fatalf("SuggestedNext = %q, want resolver-selected aether status", session.SuggestedNext)
	}
	if !strings.Contains(session.Summary, "manual") {
		t.Fatalf("Summary = %q, want manual trigger context", session.Summary)
	}
}

// TestHookStopNeverBlocksAnAetherSpawnedWorker pins the fix for a live failure:
// this hook ran inside Aether's OWN build workers, blocked one from finishing,
// and advised `aether pause`. The worker followed that advice and paused a
// running Autopilot colony mid-build. The hook exists to stop a PERSON walking
// away mid-phase; applied to a worker it does the opposite of its purpose.
//
// The state below is the exact state that blocks a person -- EXECUTING, not
// paused -- so the only thing separating pass from fail is the worker identity
// the spawn path now sets (pkg/codex.workerProcessEnv).
func TestHookStopNeverBlocksAnAetherSpawnedWorker(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)

	goal := "worker must not be blocked"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{{ID: 1, Name: "Bank the in-progress work", Status: colony.PhaseInProgress}},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	t.Setenv("AETHER_WORKER_NAME", "Weld-32")
	t.Setenv("AETHER_WORKER_CASTE", "builder")

	setHookStdin(t, `{"hook_event_name":"Stop","stop_hook_active":false}`)
	rootCmd.SetArgs([]string{"hook-stop"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("hook-stop returned error: %v", err)
	}

	if out := strings.TrimSpace(buf.String()); out != "" {
		t.Fatalf("hook-stop blocked an Aether-spawned worker; it must stay silent. output: %s", out)
	}
}

// The exemption must be narrow: without a worker identity, the very same state
// still blocks. This is what stops the fix from quietly disabling the hook.
func TestHookStopStillBlocksAPersonInTheSameState(t *testing.T) {
	saveGlobalsCmd(t)
	resetRootCmd(t)

	var buf bytes.Buffer
	stdout = &buf
	var errBuf bytes.Buffer
	stderr = &errBuf

	s, tmpDir := newTestStoreCmd(t)
	defer os.RemoveAll(tmpDir)

	goal := "person must still be blocked"
	state := colony.ColonyState{
		Version:      "1.0",
		Goal:         &goal,
		State:        colony.StateEXECUTING,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{{ID: 1, Name: "Bank the in-progress work", Status: colony.PhaseInProgress}},
		},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	t.Setenv("AETHER_WORKER_NAME", "")
	t.Setenv("AETHER_WORKER_CASTE", "")

	setHookStdin(t, `{"hook_event_name":"Stop","stop_hook_active":false}`)
	rootCmd.SetArgs([]string{"hook-stop"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("hook-stop returned error: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(bytes.TrimSpace(buf.Bytes()), &result); err != nil {
		t.Fatalf("hook-stop stayed silent for a person, so the worker exemption is too wide: %v", err)
	}
	if result["decision"] != "block" {
		t.Fatalf("decision = %v, want block for a person", result["decision"])
	}
}
