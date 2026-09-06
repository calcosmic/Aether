package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

type cliBlackBox struct {
	binary       string
	adapter      string
	repo         string
	home         string
	env          []string
	sourceRoot   string
	sourceStatus string
}

type cliBlackBoxResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// sharedBlackBoxBinaries builds the CLI and the deterministic adapter ONCE for
// the whole package and hands every black-box test the same two paths.
//
// Before this, newCLIBlackBox built both binaries per test into a per-test
// GOCACHE under t.TempDir(). A fresh GOCACHE means a cold compile every time:
// measured on this repo, a cold build of ./cmd/aether takes 16.4s against 1.1s
// warm -- so fifteen black-box tests spent roughly four of the cmd package's
// twelve minutes compiling the same program fifteen times. That is not a test
// of Aether; it is a re-test of the Go toolchain.
//
// What is shared is ONLY the compiled artifact, which every test executes and
// none modifies. Each test still gets its own home, repo, tmp and environment
// from t.TempDir(), so the isolation that tests actually depend on is
// unchanged.
//
// The build deliberately inherits the developer's real GOCACHE rather than
// making its own: a warm cache is the entire point, and the build is the same
// `go build` a developer runs by hand.
var sharedBlackBoxBinaries struct {
	once    sync.Once
	dir     string
	binary  string
	adapter string
	err     error
}

func blackBoxBinaries(t *testing.T, sourceRoot string) (string, string) {
	t.Helper()
	sharedBlackBoxBinaries.once.Do(func() {
		dir, err := os.MkdirTemp("", "aether-blackbox-bin-")
		if err != nil {
			sharedBlackBoxBinaries.err = fmt.Errorf("create shared black-box bin dir: %w", err)
			return
		}
		sharedBlackBoxBinaries.dir = dir
		sharedBlackBoxBinaries.binary = filepath.Join(dir, "aether")
		sharedBlackBoxBinaries.adapter = filepath.Join(dir, "deterministic-adapter")
		for _, target := range []struct {
			name        string
			output      string
			packagePath string
		}{
			{name: "aether", output: sharedBlackBoxBinaries.binary, packagePath: "./cmd/aether"},
			{name: "deterministic adapter", output: sharedBlackBoxBinaries.adapter, packagePath: "./cmd/testdata/adapter-fixture"},
		} {
			build := exec.Command("go", "build", "-o", target.output, target.packagePath)
			build.Dir = sourceRoot
			if output, err := build.CombinedOutput(); err != nil {
				sharedBlackBoxBinaries.err = fmt.Errorf("build black-box %s: %w\n%s", target.name, err, output)
				return
			}
		}
	})
	if sharedBlackBoxBinaries.err != nil {
		t.Fatalf("%v", sharedBlackBoxBinaries.err)
	}
	return sharedBlackBoxBinaries.binary, sharedBlackBoxBinaries.adapter
}

func newCLIBlackBox(t *testing.T) *cliBlackBox {
	t.Helper()
	sourceRoot := findTestModuleRoot(t)
	fixtureRoot := t.TempDir()
	home := filepath.Join(fixtureRoot, "home")
	repo := filepath.Join(fixtureRoot, "repo")
	tmpDir := filepath.Join(fixtureRoot, "tmp")
	for _, dir := range []string{home, repo, tmpDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("create black-box directory %s: %v", dir, err)
		}
	}

	binary, adapter := blackBoxBinaries(t, sourceRoot)

	harness := &cliBlackBox{
		binary:     binary,
		adapter:    adapter,
		repo:       repo,
		home:       home,
		sourceRoot: sourceRoot,
		env: replaceProcessEnv(os.Environ(), map[string]string{
			"AETHER_HUB_DIR":     "",
			"AETHER_OUTPUT_MODE": "json",
			"AETHER_ROOT":        repo,
			"CODEX_HOME":         filepath.Join(home, ".codex"),
			"COLONY_DATA_DIR":    filepath.Join(repo, ".aether", "data"),
			"HOME":               home,
			"NO_COLOR":           "1",
			"TMPDIR":             tmpDir,
			"USERPROFILE":        home,
		}),
	}
	harness.sourceStatus = harness.gitSourceStatus(t)
	return harness
}

func goEnvValue(t *testing.T, root, key string) string {
	t.Helper()
	command := exec.Command("go", "env", key)
	command.Dir = root
	output, err := command.Output()
	if err != nil {
		t.Fatalf("read go env %s: %v", key, err)
	}
	return strings.TrimSpace(string(output))
}

func (h *cliBlackBox) run(t *testing.T, args ...string) cliBlackBoxResult {
	return h.runWithEnv(t, nil, args...)
}

func (h *cliBlackBox) runWithEnv(t *testing.T, env map[string]string, args ...string) cliBlackBoxResult {
	t.Helper()
	command := exec.Command(h.binary, args...)
	command.Dir = h.repo
	command.Env = replaceProcessEnv(h.env, env)
	var stdoutBuffer, stderrBuffer bytes.Buffer
	command.Stdout = &stdoutBuffer
	command.Stderr = &stderrBuffer
	err := command.Run()
	exitCode := 0
	if err != nil {
		var exitErr *exec.ExitError
		if !errors.As(err, &exitErr) {
			t.Fatalf("run aether %s: %v", strings.Join(args, " "), err)
		}
		exitCode = exitErr.ExitCode()
	}
	return cliBlackBoxResult{
		Stdout:   stdoutBuffer.String(),
		Stderr:   stderrBuffer.String(),
		ExitCode: exitCode,
	}
}

func (h *cliBlackBox) prepareBuildFixture(t *testing.T) string {
	t.Helper()
	if err := os.RemoveAll(h.repo); err != nil {
		t.Fatalf("reset black-box repository: %v", err)
	}
	dataDir := filepath.Join(h.repo, ".aether", "data")
	agentDir := filepath.Join(h.repo, ".codex", "agents")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("create black-box data directory: %v", err)
	}
	if err := os.MkdirAll(agentDir, 0755); err != nil {
		t.Fatalf("create black-box agent directory: %v", err)
	}
	entries, err := os.ReadDir(filepath.Join(h.sourceRoot, ".codex", "agents"))
	if err != nil {
		t.Fatalf("read source agent definitions: %v", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".toml" {
			continue
		}
		data, readErr := os.ReadFile(filepath.Join(h.sourceRoot, ".codex", "agents", entry.Name()))
		if readErr != nil {
			t.Fatalf("read agent definition %s: %v", entry.Name(), readErr)
		}
		if writeErr := os.WriteFile(filepath.Join(agentDir, entry.Name()), data, 0644); writeErr != nil {
			t.Fatalf("write agent definition %s: %v", entry.Name(), writeErr)
		}
	}

	goal := "Prove deterministic external worker execution"
	taskID := "1.1"
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		ColonyDepth:  "quick",
		CurrentPhase: 0,
		Plan: colony.Plan{Phases: []colony.Phase{
			{
				ID:              1,
				Name:            "External adapter proof",
				Description:     "Create app.txt through a provider-backed worker process",
				Status:          colony.PhaseReady,
				SuccessCriteria: []string{"app.txt exists with deterministic content"},
				EvidenceRequirements: []colony.CriterionEvidenceRequirement{
					{
						Criterion: "app.txt exists with deterministic content",
						Artifacts: []string{"app.txt"},
					},
				},
				Tasks: []colony.Task{
					{ID: &taskID, Goal: "Implement app.txt with deterministic content", Status: colony.TaskPending},
				},
			},
		}},
	}
	stateData, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		t.Fatalf("marshal black-box state: %v", err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "COLONY_STATE.json"), stateData, 0644); err != nil {
		t.Fatalf("write black-box state: %v", err)
	}
	if err := os.WriteFile(filepath.Join(h.repo, ".gitignore"), []byte(".aether/\n"), 0644); err != nil {
		t.Fatalf("write black-box .gitignore: %v", err)
	}
	h.runGit(t, "init", "-q")
	h.runGit(t, "add", ".gitignore", ".codex")
	h.runGit(t, "-c", "user.name=Aether Test", "-c", "user.email=aether@example.invalid", "commit", "-qm", "fixture baseline")
	logPath := filepath.Join(filepath.Dir(h.repo), "adapter-invocations.jsonl")
	if err := os.Remove(logPath); err != nil && !os.IsNotExist(err) {
		t.Fatalf("reset adapter invocation log: %v", err)
	}
	return logPath
}

func (h *cliBlackBox) runGit(t *testing.T, args ...string) {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = h.repo
	command.Env = append([]string{}, h.env...)
	if output, err := command.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
}

func (h *cliBlackBox) loadColonyState(t *testing.T) colony.ColonyState {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(h.repo, ".aether", "data", "COLONY_STATE.json"))
	if err != nil {
		t.Fatalf("read black-box colony state: %v", err)
	}
	var state colony.ColonyState
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("parse black-box colony state: %v", err)
	}
	return state
}

func (h *cliBlackBox) loadVerificationReport(t *testing.T, phase int) codexContinueVerificationReport {
	t.Helper()
	path := filepath.Join(h.repo, ".aether", "data", "build", fmt.Sprintf("phase-%d", phase), "verification.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read black-box verification report: %v", err)
	}
	var report codexContinueVerificationReport
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatalf("parse black-box verification report: %v", err)
	}
	return report
}

func (h *cliBlackBox) loadBuildAttempt(t *testing.T, phase int) buildAttemptRecord {
	t.Helper()
	pointerPath := filepath.Join(h.repo, ".aether", "data", "build", fmt.Sprintf("phase-%d", phase), "latest-attempt.json")
	data, err := os.ReadFile(pointerPath)
	if err != nil {
		t.Fatalf("read black-box build attempt pointer: %v", err)
	}
	var pointer latestBuildAttemptPointer
	if err := json.Unmarshal(data, &pointer); err != nil {
		t.Fatalf("parse black-box build attempt pointer: %v", err)
	}
	rel := strings.TrimPrefix(filepath.ToSlash(pointer.Path), ".aether/data/")
	data, err = os.ReadFile(filepath.Join(h.repo, ".aether", "data", filepath.FromSlash(rel)))
	if err != nil {
		t.Fatalf("read black-box build attempt: %v", err)
	}
	var attempt buildAttemptRecord
	if err := json.Unmarshal(data, &attempt); err != nil {
		t.Fatalf("parse black-box build attempt: %v", err)
	}
	return attempt
}

func (h *cliBlackBox) loadBuildAttemptByID(t *testing.T, phase int, attemptID string) buildAttemptRecord {
	t.Helper()
	path := filepath.Join(h.repo, ".aether", "data", "build", fmt.Sprintf("phase-%d", phase), "attempts", attemptID+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read black-box build attempt %s: %v", attemptID, err)
	}
	var attempt buildAttemptRecord
	if err := json.Unmarshal(data, &attempt); err != nil {
		t.Fatalf("parse black-box build attempt %s: %v", attemptID, err)
	}
	return attempt
}

func (h *cliBlackBox) assertSourceUnchanged(t *testing.T) {
	t.Helper()
	if after := h.gitSourceStatus(t); after != h.sourceStatus {
		t.Fatalf("black-box command changed the source checkout\nbefore:\n%s\nafter:\n%s", h.sourceStatus, after)
	}
}

func (h *cliBlackBox) gitSourceStatus(t *testing.T) string {
	t.Helper()
	command := exec.Command("git", "status", "--short")
	command.Dir = h.sourceRoot
	output, err := command.Output()
	if err != nil {
		t.Fatalf("read source status: %v", err)
	}
	return string(output)
}

func findTestModuleRoot(t *testing.T) string {
	t.Helper()
	current, err := os.Getwd()
	if err != nil {
		t.Fatalf("get test working directory: %v", err)
	}
	for {
		if data, readErr := os.ReadFile(filepath.Join(current, "go.mod")); readErr == nil && strings.Contains(string(data), "github.com/calcosmic/Aether") {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			t.Fatal("could not find Aether module root")
		}
		current = parent
	}
}

func replaceProcessEnv(base []string, replacements map[string]string) []string {
	result := make([]string, 0, len(base)+len(replacements))
	for _, entry := range base {
		key, _, ok := strings.Cut(entry, "=")
		if ok {
			if _, replaced := replacements[key]; replaced {
				continue
			}
		}
		result = append(result, entry)
	}
	for key, value := range replacements {
		result = append(result, key+"="+value)
	}
	return result
}

func TestCLIErrorEnvelopeExitsNonZero(t *testing.T) {
	t.Parallel()
	harness := newCLIBlackBox(t)

	result := harness.run(t, "flag-add")
	if result.ExitCode != 1 {
		t.Fatalf("exit code = %d, want 1\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}
	var envelope struct {
		OK    bool   `json:"ok"`
		Error string `json:"error"`
		Code  int    `json:"code"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(result.Stderr)), &envelope); err != nil {
		t.Fatalf("stderr is not one JSON error envelope: %v\n%s", err, result.Stderr)
	}
	if envelope.OK || envelope.Code != 1 || !strings.Contains(envelope.Error, "flag title is required") {
		t.Fatalf("unexpected error envelope: %+v", envelope)
	}
	if _, err := os.Stat(filepath.Join(harness.repo, ".aether", "data", "pending-decisions.json")); !os.IsNotExist(err) {
		t.Fatalf("failed command mutated pending decisions: %v", err)
	}

	success := harness.run(t, "version")
	if success.ExitCode != 0 {
		t.Fatalf("successful command exit code = %d, want 0\nstdout:\n%s\nstderr:\n%s", success.ExitCode, success.Stdout, success.Stderr)
	}
	harness.assertSourceUnchanged(t)
}

func TestCLIExternalAdapterBuildContract(t *testing.T) {
	t.Parallel()
	harness := newCLIBlackBox(t)
	modes := []struct {
		name        string
		mode        string
		wantSuccess bool
		wantPartial bool
		wantError   string
	}{
		{name: "writes claimed output", mode: "success", wantSuccess: true},
		{name: "rejects no-op completion", mode: "no-op", wantError: "completed without observed file changes"},
		{name: "rejects process crash", mode: "crash", wantError: "exit status 17"},
		{name: "rejects malformed claims", mode: "malformed", wantError: "parse worker output"},
		{name: "rejects timeout", mode: "timeout", wantError: "worker timeout"},
		{name: "preserves partial write without advancing", mode: "partial", wantPartial: true, wantError: "exit status 18"},
	}
	for _, tc := range modes {
		t.Run(tc.name, func(t *testing.T) {
			logPath := harness.prepareBuildFixture(t)
			// Only the timeout case wants a deadline the adapter will miss (it
			// sleeps 10s). Every other case must finish, and 1s is too tight
			// for a real subprocess under a fully loaded parallel test run.
			workerTimeout := "5s"
			if tc.mode == "timeout" {
				workerTimeout = "1s"
			}
			result := harness.runWithEnv(t, map[string]string{
				"AETHER_ACTIVE_PLATFORM":     "codex",
				"AETHER_CODEX_PATH":          harness.adapter,
				"AETHER_CODEX_REAL_DISPATCH": "real",
				"AETHER_TEST_ADAPTER_LOG":    logPath,
				"AETHER_TEST_ADAPTER_MODE":   tc.mode,
				"AETHER_WORKER_PLATFORM":     "codex",
			}, "build", "1", "--light", "--worker-timeout", workerTimeout)

			state := harness.loadColonyState(t)
			attempt := harness.loadBuildAttempt(t, 1)
			if tc.wantSuccess {
				if result.ExitCode != 0 {
					t.Fatalf("build exit code = %d, want 0\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
				}
				if state.State != colony.StateBUILT {
					t.Fatalf("state = %s, want BUILT", state.State)
				}
				if attempt.Status != buildAttemptBuilt || len(attempt.History) < 4 || attempt.Claims == nil || len(attempt.Claims.ArtifactEvidence) == 0 {
					t.Fatalf("successful build attempt lacks durable terminal evidence: %+v", attempt)
				}
				data, err := os.ReadFile(filepath.Join(harness.repo, "app.txt"))
				if err != nil || string(data) != "created by deterministic adapter\n" {
					t.Fatalf("app.txt = %q, %v", data, err)
				}
			} else {
				if result.ExitCode == 0 {
					t.Fatalf("failed adapter mode exited 0\nstdout:\n%s\nstderr:\n%s", result.Stdout, result.Stderr)
				}
				if state.State == colony.StateBUILT {
					t.Fatalf("failed adapter mode advanced state to BUILT")
				}
				if attempt.Status != buildAttemptFailed || attempt.RecoveryCommand != "aether build 1 --force" || attempt.Error == "" {
					t.Fatalf("failed build attempt lacks recovery evidence: %+v", attempt)
				}
				if !strings.Contains(result.Stderr, tc.wantError) {
					t.Fatalf("stderr missing %q\n%s", tc.wantError, result.Stderr)
				}
			}
			if tc.wantPartial {
				matches, err := filepath.Glob(filepath.Join(harness.repo, "partial-*.txt"))
				if err != nil || len(matches) == 0 {
					t.Fatalf("partial worker output was not preserved: matches=%v err=%v", matches, err)
				}
			}
			logData, err := os.ReadFile(logPath)
			if err != nil || len(strings.TrimSpace(string(logData))) == 0 {
				t.Fatalf("external adapter did not record an invocation: %v", err)
			}
			// Phase 193 (D-08): the build side no longer dispatches an
			// implicit watcher without an explicit Queen proposal (none was
			// made here) -- agent review now lives in `continue`.
			if tc.wantSuccess && !bytes.Contains(logData, []byte(`"caste":"builder"`)) {
				t.Fatalf("successful build did not execute a builder process:\n%s", logData)
			}
			harness.assertSourceUnchanged(t)
		})
	}
}

func TestCLIInternalWorkerAdapterOwnsProviderSelectionAndClaimsParsing(t *testing.T) {
	t.Parallel()
	harness := newCLIBlackBox(t)
	harness.prepareBuildFixture(t)
	tmpRoot := filepath.Join(filepath.Dir(harness.repo), "tmp")
	requestDir, err := os.MkdirTemp(tmpRoot, "aether-worker-request-blackbox-")
	if err != nil {
		t.Fatalf("create internal worker request directory: %v", err)
	}
	requestPath := filepath.Join(requestDir, "worker-request.json")
	request := map[string]interface{}{
		"schema_version":     1,
		"caste":              "builder",
		"worker_name":        "Builder-Adapter",
		"task_id":            "adapter-1",
		"task":               "Create deterministic adapter output",
		"task_brief":         "# Task adapter-1\n\nCreate deterministic adapter output.",
		"timeout_ms":         5000,
		"permission_profile": codex.PermissionProfileForCaste("Builder"),
	}
	data, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("marshal internal worker request: %v", err)
	}
	if err := os.WriteFile(requestPath, data, 0600); err != nil {
		t.Fatalf("write internal worker request: %v", err)
	}
	baseEnv := map[string]string{
		"AETHER_ACTIVE_PLATFORM":     "codex",
		"AETHER_CODEX_PATH":          harness.adapter,
		"AETHER_CODEX_REAL_DISPATCH": "real",
		"AETHER_WORKER_PLATFORM":     "codex",
	}

	success := harness.runWithEnv(t, baseEnv, "internal-worker-adapter", "--request-file", requestPath)
	assertBlackBoxSuccess(t, "internal worker adapter", success)
	var envelope struct {
		Result internalWorkerAdapterResponse `json:"result"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(success.Stdout)), &envelope); err != nil {
		t.Fatalf("parse internal worker adapter response: %v\n%s", err, success.Stdout)
	}
	if envelope.Result.ExecutionOwner != "go-adapter" || envelope.Result.Platform != "codex" {
		t.Fatalf("unexpected adapter owner/platform: %+v", envelope.Result)
	}
	if envelope.Result.Worker == nil || envelope.Result.Worker.Status != "completed" || envelope.Result.Worker.TaskID != "adapter-1" {
		t.Fatalf("typed worker result missing completion identity: %+v", envelope.Result.Worker)
	}

	malformedEnv := make(map[string]string, len(baseEnv)+1)
	for key, value := range baseEnv {
		malformedEnv[key] = value
	}
	malformedEnv["AETHER_TEST_ADAPTER_MODE"] = "malformed"
	malformed := harness.runWithEnv(t, malformedEnv, "internal-worker-adapter", "--request-file", requestPath)
	assertBlackBoxSuccess(t, "malformed internal worker result", malformed)
	if err := json.Unmarshal([]byte(strings.TrimSpace(malformed.Stdout)), &envelope); err != nil {
		t.Fatalf("parse malformed adapter response: %v\n%s", err, malformed.Stdout)
	}
	if envelope.Result.Worker == nil || envelope.Result.Worker.Status != "failed" || envelope.Result.Worker.Status == "completed" {
		t.Fatalf("malformed provider output was not failed closed: %+v", envelope.Result.Worker)
	}

	pinnedUnavailable := harness.runWithEnv(t, map[string]string{
		"AETHER_ACTIVE_PLATFORM":     "codex",
		"AETHER_CLAUDE_PATH":         filepath.Join(tmpRoot, "missing-claude"),
		"AETHER_CODEX_PATH":          harness.adapter,
		"AETHER_CODEX_REAL_DISPATCH": "real",
		"AETHER_WORKER_PLATFORM":     "claude",
	}, "internal-worker-adapter", "--request-file", requestPath)
	if pinnedUnavailable.ExitCode == 0 {
		t.Fatalf("unavailable explicit platform pin silently fell back\nstdout:\n%s", pinnedUnavailable.Stdout)
	}
	if !strings.Contains(strings.ToLower(pinnedUnavailable.Stderr), "claude") {
		t.Fatalf("unavailable pin diagnostic did not identify claude:\n%s", pinnedUnavailable.Stderr)
	}

	unsupported := harness.runWithEnv(t, map[string]string{
		"AETHER_CODEX_PATH":          harness.adapter,
		"AETHER_CODEX_REAL_DISPATCH": "real",
		"AETHER_WORKER_PLATFORM":     "banana",
	}, "internal-worker-adapter", "--request-file", requestPath)
	if unsupported.ExitCode == 0 || !strings.Contains(unsupported.Stderr, "unsupported AETHER_WORKER_PLATFORM") {
		t.Fatalf("unsupported provider pin did not fail explicitly\nstdout:\n%s\nstderr:\n%s", unsupported.Stdout, unsupported.Stderr)
	}
	harness.assertSourceUnchanged(t)
}

func TestCLIContinueEnforcesFreshCriterionEvidence(t *testing.T) {
	t.Parallel()
	harness := newCLIBlackBox(t)
	adapterEnv := map[string]string{
		"AETHER_ACTIVE_PLATFORM":     "codex",
		"AETHER_CODEX_PATH":          harness.adapter,
		"AETHER_CODEX_REAL_DISPATCH": "real",
		"AETHER_TEST_ADAPTER_MODE":   "success",
		"AETHER_WORKER_PLATFORM":     "codex",
	}
	for _, tc := range []struct {
		name      string
		mutate    func(t *testing.T, path string)
		wantPass  bool
		wantIssue string
	}{
		{name: "unchanged claimed artifact advances", wantPass: true},
		{
			name: "deleted claimed artifact blocks",
			mutate: func(t *testing.T, path string) {
				t.Helper()
				if err := os.Remove(path); err != nil {
					t.Fatalf("remove claimed artifact: %v", err)
				}
			},
			wantIssue: "cannot be verified",
		},
		{
			name: "changed claimed artifact blocks",
			mutate: func(t *testing.T, path string) {
				t.Helper()
				if err := os.WriteFile(path, []byte("changed after build\n"), 0644); err != nil {
					t.Fatalf("change claimed artifact: %v", err)
				}
			},
			wantIssue: "changed after build evidence",
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			logPath := harness.prepareBuildFixture(t)
			env := make(map[string]string, len(adapterEnv)+1)
			for key, value := range adapterEnv {
				env[key] = value
			}
			env["AETHER_TEST_ADAPTER_LOG"] = logPath
			// 30s, not 1s: this build is expected to SUCCEED, so the timeout is
			// only a backstop. A 1s cap made the subtest fail intermittently
			// under full-package load, when the fake adapter needed longer than
			// a second to be scheduled. The deliberate timeout-rejection case is
			// covered separately by the "rejects timeout" adapter mode above,
			// which still uses 1s.
			build := harness.runWithEnv(t, env, "build", "1", "--light", "--worker-timeout", "30s")
			if build.ExitCode != 0 {
				t.Fatalf("build failed before continue: exit=%d\nstdout:\n%s\nstderr:\n%s", build.ExitCode, build.Stdout, build.Stderr)
			}
			artifact := filepath.Join(harness.repo, "app.txt")
			if tc.mutate != nil {
				tc.mutate(t, artifact)
			}

			result := harness.runWithEnv(t, env, "continue", "--skip-watchers", "--verification-depth", "light")
			report := harness.loadVerificationReport(t, 1)
			state := harness.loadColonyState(t)
			if tc.wantPass {
				if result.ExitCode != 0 || !report.Passed || !report.CriteriaEnforced || !report.CriteriaPassed {
					t.Fatalf("continue did not pass fresh evidence: exit=%d report=%+v\nstdout:\n%s\nstderr:\n%s", result.ExitCode, report, result.Stdout, result.Stderr)
				}
				if len(state.Plan.Phases) != 1 || state.Plan.Phases[0].Status != colony.PhaseCompleted || state.State != colony.StateCOMPLETED {
					t.Fatalf("fresh evidence did not complete phase: state=%s phases=%+v", state.State, state.Plan.Phases)
				}
			} else {
				issues := strings.Join(report.BlockingIssues, "\n")
				if report.Passed || report.CriteriaPassed || !strings.Contains(issues, tc.wantIssue) {
					t.Fatalf("continue report = %+v, want blocked issue %q\nstdout:\n%s\nstderr:\n%s", report, tc.wantIssue, result.Stdout, result.Stderr)
				}
				if len(state.Plan.Phases) != 1 || state.Plan.Phases[0].Status == colony.PhaseCompleted || state.State == colony.StateCOMPLETED {
					t.Fatalf("stale evidence advanced phase: state=%s phases=%+v", state.State, state.Plan.Phases)
				}
			}
			harness.assertSourceUnchanged(t)
		})
	}
}

func TestCLIInterruptedBuildResumesThroughForceRedispatch(t *testing.T) {
	t.Parallel()
	harness := newCLIBlackBox(t)
	logPath := harness.prepareBuildFixture(t)
	env := map[string]string{
		"AETHER_ACTIVE_PLATFORM":     "codex",
		"AETHER_CODEX_PATH":          harness.adapter,
		"AETHER_CODEX_REAL_DISPATCH": "real",
		"AETHER_TEST_ADAPTER_LOG":    logPath,
		"AETHER_TEST_ADAPTER_MODE":   "partial-timeout",
		"AETHER_WORKER_PLATFORM":     "codex",
	}
	command := exec.Command(harness.binary, "build", "1", "--light", "--worker-timeout", "30s")
	command.Dir = harness.repo
	command.Env = replaceProcessEnv(harness.env, env)
	var stdoutBuffer, stderrBuffer bytes.Buffer
	command.Stdout = &stdoutBuffer
	command.Stderr = &stderrBuffer
	if err := command.Start(); err != nil {
		t.Fatalf("start interrupted build: %v", err)
	}
	partialPath := filepath.Join(harness.repo, "interrupted-worker-output.txt")
	waitForBlackBoxFile(t, partialPath, 10*time.Second)
	if err := command.Process.Kill(); err != nil {
		t.Fatalf("kill interrupted build: %v", err)
	}
	_ = command.Wait()
	killFixtureAdapterProcesses(t, logPath)

	interruptedState := harness.loadColonyState(t)
	if interruptedState.State != colony.StateEXECUTING || interruptedState.BuildStartedAt == nil || interruptedState.Plan.Phases[0].Status != colony.PhaseInProgress {
		t.Fatalf("killed build state = %+v, want active uncompleted phase", interruptedState)
	}
	interruptedAttempt := harness.loadBuildAttempt(t, 1)
	if interruptedAttempt.Status != buildAttemptDispatching {
		t.Fatalf("killed attempt status = %q, want dispatching", interruptedAttempt.Status)
	}
	if _, err := os.Stat(partialPath); err != nil {
		t.Fatalf("partial worker output was not preserved: %v", err)
	}

	resume := harness.runWithEnv(t, env, "resume")
	if resume.ExitCode != 0 || !strings.Contains(resume.Stdout, "aether build 1 --force") || !strings.Contains(resume.Stdout, `"build_attempt"`) {
		t.Fatalf("resume did not expose valid interrupted-build recovery: exit=%d\nstdout:\n%s\nstderr:\n%s", resume.ExitCode, resume.Stdout, resume.Stderr)
	}
	resumedState := harness.loadColonyState(t)
	if resumedState.State != colony.StateREADY || resumedState.BuildStartedAt != nil || resumedState.Plan.Phases[0].Status != colony.PhaseInProgress || resumedState.RecoveryProvenance == nil {
		t.Fatalf("resumed interrupted state = %+v, want READY recovery orientation with the active phase retained", resumedState)
	}
	resumedAttempt := harness.loadBuildAttempt(t, 1)
	if resumedAttempt.ID != interruptedAttempt.ID || resumedAttempt.Status != buildAttemptDispatching {
		t.Fatalf("resume rewrote the interrupted build attempt before explicit redispatch: %+v", resumedAttempt)
	}

	env["AETHER_TEST_ADAPTER_MODE"] = "success"
	retry := harness.runWithEnv(t, env, "build", "1", "--force", "--light", "--worker-timeout", "30s")
	if retry.ExitCode != 0 {
		t.Fatalf("force redispatch failed: exit=%d\nstdout:\n%s\nstderr:\n%s", retry.ExitCode, retry.Stdout, retry.Stderr)
	}
	oldAttempt := harness.loadBuildAttemptByID(t, 1, interruptedAttempt.ID)
	if oldAttempt.Status != buildAttemptInterrupted || !strings.Contains(oldAttempt.Error, "force redispatch") {
		t.Fatalf("old attempt was not preserved as interrupted: %+v", oldAttempt)
	}
	newAttempt := harness.loadBuildAttempt(t, 1)
	if newAttempt.ID == oldAttempt.ID || newAttempt.Status != buildAttemptBuilt {
		t.Fatalf("retry attempt = %+v, want distinct built attempt", newAttempt)
	}
	finalState := harness.loadColonyState(t)
	if finalState.State != colony.StateBUILT || finalState.Plan.Phases[0].Status != colony.PhaseInProgress {
		t.Fatalf("retry state = %+v, want BUILT awaiting verification", finalState)
	}
	harness.assertSourceUnchanged(t)
}

func TestCLIPlanOnlyCompletionIsBoundAndIdempotent(t *testing.T) {
	t.Parallel()
	harness := newCLIBlackBox(t)
	harness.prepareBuildFixture(t)

	plan := harness.run(t, "build", "1", "--plan-only", "--light")
	if plan.ExitCode != 0 {
		t.Fatalf("plan-only build failed: exit=%d\nstdout:\n%s\nstderr:\n%s", plan.ExitCode, plan.Stdout, plan.Stderr)
	}
	var planEnvelope struct {
		OK     bool `json:"ok"`
		Result struct {
			DispatchManifest codexBuildManifest `json:"dispatch_manifest"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(plan.Stdout), &planEnvelope); err != nil {
		t.Fatalf("parse plan-only output: %v\n%s", err, plan.Stdout)
	}
	manifest := planEnvelope.Result.DispatchManifest
	if !planEnvelope.OK || manifest.AttemptID == "" || manifest.AttemptPath == "" {
		t.Fatalf("plan-only output lacks durable attempt identity: %+v", planEnvelope)
	}
	if err := os.WriteFile(filepath.Join(harness.repo, "app.txt"), []byte("created by external wrapper\n"), 0644); err != nil {
		t.Fatalf("write wrapper artifact: %v", err)
	}
	workerResults := make([]codexExternalBuildWorkerResult, 0, len(manifest.Dispatches))
	for _, dispatch := range manifest.Dispatches {
		worker := codexExternalBuildWorkerResult{
			Stage:         dispatch.Stage,
			Wave:          dispatch.Wave,
			ExecutionWave: normalizedDispatchWave(dispatch),
			Caste:         dispatch.Caste,
			Name:          dispatch.Name,
			TaskID:        dispatch.TaskID,
			Status:        "completed",
			Summary:       dispatch.Name + " completed through compiled wrapper contract",
			Handoff: codex.WorkerHandoff{
				CommandsRun:            []string{"go test ./..."},
				VerificationStatus:     "pass",
				NextWorkerInstructions: []string{"work complete"},
			},
		}
		if dispatch.Caste == "builder" {
			worker.FilesCreated = []string{"app.txt"}
		}
		workerResults = append(workerResults, worker)
	}
	completion := codexExternalBuildCompletion{DispatchManifest: &manifest, Dispatches: workerResults}
	completionPath := filepath.Join(filepath.Dir(harness.repo), "build-completion.json")
	writeBlackBoxCompletion(t, completionPath, completion)

	first := harness.run(t, "build-finalize", "1", "--completion-file", completionPath)
	if first.ExitCode != 0 || !jsonResultBool(first.Stdout, "idempotent", false) {
		t.Fatalf("first finalization failed or was not a real commit: exit=%d\nstdout:\n%s\nstderr:\n%s", first.ExitCode, first.Stdout, first.Stderr)
	}
	second := harness.run(t, "build-finalize", "1", "--completion-file", completionPath)
	if second.ExitCode != 0 || !jsonResultBool(second.Stdout, "idempotent", true) {
		t.Fatalf("completion replay was not idempotent: exit=%d\nstdout:\n%s\nstderr:\n%s", second.ExitCode, second.Stdout, second.Stderr)
	}
	attempt := harness.loadBuildAttempt(t, 1)
	if attempt.ID != manifest.AttemptID || attempt.Status != buildAttemptBuilt || attempt.CompletionSHA256 == "" {
		t.Fatalf("compiled finalizer did not commit the issued attempt: %+v", attempt)
	}

	completion.Dispatches[0].Summary += " tampered"
	writeBlackBoxCompletion(t, completionPath, completion)
	tampered := harness.run(t, "build-finalize", "1", "--completion-file", completionPath)
	if tampered.ExitCode == 0 || !strings.Contains(tampered.Stderr, "does not match the result already bound") {
		t.Fatalf("tampered completion was not rejected: exit=%d\nstdout:\n%s\nstderr:\n%s", tampered.ExitCode, tampered.Stdout, tampered.Stderr)
	}
	harness.assertSourceUnchanged(t)
}

func TestCLIVersionedPlanRevisionSurvivesRestartAndBindsNextBuild(t *testing.T) {
	t.Parallel()
	harness := newCLIBlackBox(t)
	dataDir := filepath.Join(harness.repo, ".aether", "data")
	oracleDir := filepath.Join(harness.repo, ".aether", "oracle")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatalf("create revision state fixture: %v", err)
	}
	if err := os.MkdirAll(oracleDir, 0755); err != nil {
		t.Fatalf("create revision fixture: %v", err)
	}
	goal := "Adapt the remaining implementation after research"
	doneTaskID := "1.1"
	futureTaskID := "2.1"
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 2,
		Plan: colony.Plan{
			EvidencePolicy: colony.PlanEvidenceBoundV1,
			Phases: []colony.Phase{
				{ID: 1, Name: "Completed foundation", Description: "Accepted work", Status: colony.PhaseCompleted, Tasks: []colony.Task{{ID: &doneTaskID, Goal: "Build foundation", Status: colony.TaskCompleted}}},
				{ID: 2, Name: "Invalidated approach", Description: "Research made this obsolete", Status: colony.PhaseReady, Tasks: []colony.Task{{ID: &futureTaskID, Goal: "Use old approach", Status: colony.TaskPending}}},
			},
		},
		Memory: colony.Memory{PhaseLearnings: []colony.PhaseLearning{}, Decisions: []colony.Decision{}, Instincts: []colony.Instinct{}},
		Errors: colony.Errors{Records: []colony.ErrorRecord{}, FlaggedPatterns: []colony.FlaggedPattern{}},
		Events: []string{},
	}
	stateData, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataDir, "COLONY_STATE.json"), stateData, 0644); err != nil {
		t.Fatalf("write revision state: %v", err)
	}
	if err := os.WriteFile(filepath.Join(oracleDir, "synthesis.md"), []byte("# Oracle synthesis\nThe original dependency assumption is false.\n"), 0644); err != nil {
		t.Fatalf("write Oracle evidence: %v", err)
	}
	completedBefore, _ := json.Marshal(state.Plan.Phases[0])

	revise := harness.run(t,
		"plan", "--refresh", "--synthetic", "--depth", "fast", "--accept",
		"--revision-type", "research",
		"--revision-reason", "Oracle disproved the original dependency assumption",
		"--revision-evidence", ".aether/oracle/synthesis.md",
	)
	if revise.ExitCode != 0 {
		t.Fatalf("compiled plan revision failed: exit=%d\nstdout:\n%s\nstderr:\n%s", revise.ExitCode, revise.Stdout, revise.Stderr)
	}
	revised := harness.loadColonyState(t)
	completedAfter, _ := json.Marshal(revised.Plan.Phases[0])
	if string(completedBefore) != string(completedAfter) {
		t.Fatalf("completed phase changed across compiled revision\nbefore=%s\nafter=%s", completedBefore, completedAfter)
	}
	active, ok := activePlanRevision(revised.Plan)
	if !ok || active.Number != 2 || active.ReasonType != colony.PlanRevisionResearch {
		t.Fatalf("compiled revision history = %+v, active=%+v ok=%v", revised.Plan.Revisions, active, ok)
	}
	if revised.CurrentPhase != 2 || len(revised.Plan.Phases) < 2 || revised.Plan.Phases[1].Status != colony.PhaseReady {
		t.Fatalf("compiled revision did not activate replacement future work: %+v", revised)
	}

	resume := harness.run(t, "resume")
	if resume.ExitCode != 0 || !strings.Contains(resume.Stdout, active.ID) || !strings.Contains(resume.Stdout, "Oracle disproved") {
		t.Fatalf("new process did not restore active revision context: exit=%d\nstdout:\n%s\nstderr:\n%s", resume.ExitCode, resume.Stdout, resume.Stderr)
	}
	buildPlan := harness.run(t, "build", "2", "--plan-only", "--light")
	if buildPlan.ExitCode != 0 {
		t.Fatalf("build plan for revised phase failed: exit=%d\nstdout:\n%s\nstderr:\n%s", buildPlan.ExitCode, buildPlan.Stdout, buildPlan.Stderr)
	}
	var buildEnvelope struct {
		Result struct {
			Manifest codexBuildManifest `json:"dispatch_manifest"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(buildPlan.Stdout), &buildEnvelope); err != nil {
		t.Fatalf("parse revised build manifest: %v\n%s", err, buildPlan.Stdout)
	}
	if buildEnvelope.Result.Manifest.PlanRevisionID != active.ID {
		t.Fatalf("next build bound to revision %q, want %q", buildEnvelope.Result.Manifest.PlanRevisionID, active.ID)
	}
	for _, task := range buildEnvelope.Result.Manifest.Tasks {
		if task.ID == doneTaskID || task.Goal == "Build foundation" {
			t.Fatalf("revised build attempted to re-execute completed task: %+v", task)
		}
	}
	harness.assertSourceUnchanged(t)
}

// TestCLIProviderBackedPlanRevisionJourney closes the acceptance gap left by
// the synthetic revision journey: a research question flows through a real
// Oracle provider process into evidence, real Scout and Route-Setter provider
// processes produce the replacement plan, the revision binds that evidence, and
// the revised phase builds and verifies through real provider workers.
func TestCLIProviderBackedPlanRevisionJourney(t *testing.T) {
	t.Parallel()
	harness := newCLIBlackBox(t)
	logPath := filepath.Join(filepath.Dir(harness.repo), "revision-adapter-invocations.jsonl")
	providerEnv := map[string]string{
		"AETHER_ACTIVE_PLATFORM":     "codex",
		"AETHER_CODEX_PATH":          harness.adapter,
		"AETHER_CODEX_REAL_DISPATCH": "real",
		"AETHER_HIVE_POLICY":         "off",
		"AETHER_TEST_ADAPTER_LOG":    logPath,
		"AETHER_TEST_ADAPTER_MODE":   "success",
		"AETHER_WORKER_PLATFORM":     "codex",
	}

	install := harness.run(t, "install", "--package-dir", harness.sourceRoot, "--home-dir", harness.home, "--skip-build-binary")
	assertBlackBoxSuccess(t, "install", install)
	setup := harness.run(t, "lay-eggs", "--repo-dir", harness.repo, "--home-dir", harness.home)
	assertBlackBoxSuccess(t, "lay-eggs", setup)

	if err := os.WriteFile(filepath.Join(harness.repo, "go.mod"), []byte("module example.com/aetherrevision\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write revision go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(harness.repo, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write revision main.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(harness.repo, "main_test.go"), []byte("package main\n\nimport \"testing\"\n\nfunc TestBaseline(t *testing.T) {}\n"), 0644); err != nil {
		t.Fatalf("write revision main_test.go: %v", err)
	}
	harness.runGit(t, "init", "-q")
	harness.runGit(t, "add", ".")
	harness.runGit(t, "-c", "user.name=Aether Test", "-c", "user.email=aether@example.invalid", "commit", "-qm", "revision baseline")

	goal := "Adapt the remaining implementation after research"
	doneTaskID := "1.1"
	futureTaskID := "2.1"
	state := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 2,
		Plan: colony.Plan{
			EvidencePolicy: colony.PlanEvidenceBoundV1,
			Phases: []colony.Phase{
				{ID: 1, Name: "Completed foundation", Description: "Accepted work", Status: colony.PhaseCompleted, Tasks: []colony.Task{{ID: &doneTaskID, Goal: "Build foundation", Status: colony.TaskCompleted}}},
				{ID: 2, Name: "Invalidated approach", Description: "Research made this obsolete", Status: colony.PhaseReady, Tasks: []colony.Task{{ID: &futureTaskID, Goal: "Use old approach", Status: colony.TaskPending}}},
			},
		},
		Memory: colony.Memory{PhaseLearnings: []colony.PhaseLearning{}, Decisions: []colony.Decision{}, Instincts: []colony.Instinct{}},
		Errors: colony.Errors{Records: []colony.ErrorRecord{}, FlaggedPatterns: []colony.FlaggedPattern{}},
		Events: []string{},
	}
	stateData, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(harness.repo, ".aether", "data", "COLONY_STATE.json"), stateData, 0644); err != nil {
		t.Fatalf("write revision state: %v", err)
	}
	completedBefore, _ := json.Marshal(state.Plan.Phases[0])

	research := harness.runWithEnv(t, providerEnv,
		"oracle", "Is the original dependency assumption still valid?",
		"--depth", "quick", "--confidence-target", "1", "--scope", "repo", "--template", "research-brief", "--max-iterations", "1")
	assertBlackBoxSuccess(t, "oracle", research)
	if _, err := os.Stat(filepath.Join(harness.repo, ".aether", "oracle", "synthesis.md")); err != nil {
		t.Fatalf("oracle provider run did not persist synthesis evidence: %v", err)
	}
	// Planning now consumes a verified territory snapshot. This journey is
	// about revision behavior rather than colonize orchestration, so seed the
	// same immutable input a completed colonize-finalize run would publish.
	writeFreshTerritorySnapshot199(t, harness.repo, time.Now().UTC().Add(-time.Minute))

	revise := harness.runWithEnv(t, providerEnv,
		"plan", "--refresh", "--depth", "fast", "--accept",
		"--revision-type", "research",
		"--revision-reason", "Oracle disproved the original dependency assumption",
		"--revision-evidence", ".aether/oracle/synthesis.md",
	)
	assertBlackBoxSuccess(t, "plan --refresh", revise)
	var planEnvelope struct {
		Result struct {
			DispatchMode string              `json:"dispatch_mode"`
			PlanSource   string              `json:"plan_source"`
			PlanRevision colony.PlanRevision `json:"plan_revision"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(revise.Stdout), &planEnvelope); err != nil {
		t.Fatalf("parse provider-backed plan result: %v\n%s", err, revise.Stdout)
	}
	if planEnvelope.Result.DispatchMode != "real" {
		t.Fatalf("provider-backed revision dispatch mode = %q, want real", planEnvelope.Result.DispatchMode)
	}
	if planEnvelope.Result.PlanSource != "worker-artifact" {
		t.Fatalf("provider-backed revision plan source = %q, want worker-artifact", planEnvelope.Result.PlanSource)
	}

	revised := harness.loadColonyState(t)
	completedAfter, _ := json.Marshal(revised.Plan.Phases[0])
	if string(completedBefore) != string(completedAfter) {
		t.Fatalf("completed phase changed across provider-backed revision\nbefore=%s\nafter=%s", completedBefore, completedAfter)
	}
	active, ok := activePlanRevision(revised.Plan)
	if !ok || active.Number != 2 || active.ReasonType != colony.PlanRevisionResearch {
		t.Fatalf("provider-backed revision history = %+v, active=%+v ok=%v", revised.Plan.Revisions, active, ok)
	}
	if active.PlanningRunID == "" {
		t.Fatalf("provider-backed revision lacks a planning run identity: %+v", active)
	}
	if revised.CurrentPhase != 2 || len(revised.Plan.Phases) != 2 || revised.Plan.Phases[1].Status != colony.PhaseReady {
		t.Fatalf("provider-backed revision did not activate replacement future work: %+v", revised)
	}
	if revised.Plan.Phases[1].Name != "Provider-planned replacement approach" {
		t.Fatalf("replacement phase = %q, want the Route-Setter provider artifact", revised.Plan.Phases[1].Name)
	}

	resume := harness.run(t, "resume")
	if resume.ExitCode != 0 || !strings.Contains(resume.Stdout, active.ID) || !strings.Contains(resume.Stdout, "Oracle disproved") {
		t.Fatalf("new process did not restore provider-backed revision context: exit=%d\nstdout:\n%s\nstderr:\n%s", resume.ExitCode, resume.Stdout, resume.Stderr)
	}

	build := harness.runWithEnv(t, providerEnv, "build", "2", "--light", "--worker-timeout", "30s")
	assertBlackBoxSuccess(t, "build 2", build)
	attempt := harness.loadBuildAttempt(t, 2)
	if attempt.Status != buildAttemptBuilt || attempt.Claims == nil {
		t.Fatalf("revised phase lacks durable provider-backed build evidence: %+v", attempt)
	}

	continued := harness.runWithEnv(t, providerEnv, "continue", "--verification-depth", "light", "--worker-timeout", "30s")
	assertBlackBoxSuccess(t, "continue phase 2", continued)
	report := harness.loadVerificationReport(t, 2)
	if !report.Passed || !report.CriteriaEnforced || !report.CriteriaPassed {
		claimsDebug, _ := os.ReadFile(filepath.Join(harness.repo, ".aether", "data", "last-build-claims.json"))
		logDebug, _ := os.ReadFile(logPath)
		t.Fatalf("revised phase verification did not enforce its criteria: %+v\nclaims: %s\nadapter log:\n%s", report, claimsDebug, logDebug)
	}
	final := harness.loadColonyState(t)
	if final.State != colony.StateCOMPLETED {
		t.Fatalf("revised colony ended in %s, want COMPLETED", final.State)
	}

	// Phase 193 (D-08): see the equivalent comment in
	// TestCLICompiledInstallToSealJourney -- the build side no longer
	// dispatches an implicit watcher, which was the only reliable source of
	// a `"caste":"watcher"` tag in this log format.
	logData, err := os.ReadFile(logPath)
	if err != nil ||
		!bytes.Contains(logData, []byte(`"caste":"oracle"`)) ||
		!bytes.Contains(logData, []byte(`"caste":"scout"`)) ||
		!bytes.Contains(logData, []byte(`"caste":"route_setter"`)) ||
		!bytes.Contains(logData, []byte(`"caste":"builder"`)) {
		t.Fatalf("revision journey did not execute oracle, scout, route-setter, and builder provider processes: err=%v\n%s", err, logData)
	}
	harness.assertSourceUnchanged(t)
}

func TestCLICompiledInstallToSealJourney(t *testing.T) {
	t.Parallel()
	harness := newCLIBlackBox(t)
	providerEnv := map[string]string{
		"AETHER_ACTIVE_PLATFORM":     "codex",
		"AETHER_CODEX_PATH":          harness.adapter,
		"AETHER_CODEX_REAL_DISPATCH": "real",
		"AETHER_HIVE_POLICY":         "off",
		"AETHER_TEST_ADAPTER_MODE":   "success",
		"AETHER_WORKER_PLATFORM":     "codex",
	}

	install := harness.run(t, "install", "--package-dir", harness.sourceRoot, "--home-dir", harness.home, "--skip-build-binary")
	assertBlackBoxSuccess(t, "install", install)
	setup := harness.run(t, "lay-eggs", "--repo-dir", harness.repo, "--home-dir", harness.home)
	assertBlackBoxSuccess(t, "lay-eggs", setup)

	if err := os.WriteFile(filepath.Join(harness.repo, "go.mod"), []byte("module example.com/aetherjourney\n\ngo 1.24\n"), 0644); err != nil {
		t.Fatalf("write journey go.mod: %v", err)
	}
	if err := os.WriteFile(filepath.Join(harness.repo, "main.go"), []byte("package main\n\nfunc main() {}\n"), 0644); err != nil {
		t.Fatalf("write journey main.go: %v", err)
	}
	if err := os.WriteFile(filepath.Join(harness.repo, "main_test.go"), []byte("package main\n\nimport \"testing\"\n\nfunc TestBaseline(t *testing.T) {}\n"), 0644); err != nil {
		t.Fatalf("write journey main_test.go: %v", err)
	}
	harness.runGit(t, "init", "-q")
	harness.runGit(t, "add", ".")
	harness.runGit(t, "-c", "user.name=Aether Test", "-c", "user.email=aether@example.invalid", "commit", "-qm", "journey baseline")

	initResult := harness.run(t, "init", "Build a small, verifiable command-line journey")
	assertBlackBoxSuccess(t, "init", initResult)
	discuss := harness.run(t, "discuss")
	assertBlackBoxSuccess(t, "discuss", discuss)
	questions := blackBoxDiscussionQuestions(t, discuss.Stdout)
	if len(questions) == 0 {
		t.Fatal("discussion did not surface ambiguity before planning")
	}
	for _, question := range questions {
		resolved := harness.run(t, "discuss", "--resolve", question.ID, "--answer", "Keep the first delivery inside the existing Go command surface and prove it with automated tests.")
		assertBlackBoxSuccess(t, "discuss --resolve "+question.ID, resolved)
	}

	logPath := filepath.Join(filepath.Dir(harness.repo), "journey-adapter-invocations.jsonl")
	providerEnv["AETHER_TEST_ADAPTER_LOG"] = logPath
	research := harness.runWithEnv(t, providerEnv,
		"oracle", "Which repository evidence should constrain this acceptance journey?",
		"--depth", "quick", "--confidence-target", "1", "--scope", "repo", "--template", "research-brief", "--max-iterations", "1")
	assertBlackBoxSuccess(t, "oracle", research)
	if _, err := os.Stat(filepath.Join(harness.repo, ".aether", "oracle", "research-plan.md")); err != nil {
		t.Fatalf("oracle did not persist a reusable research plan: %v", err)
	}
	// Keep this acceptance journey focused on plan/build/continue/seal while
	// satisfying planning's verified-territory precondition.
	writeFreshTerritorySnapshot199(t, harness.repo, time.Now().UTC().Add(-time.Minute))

	plan := harness.run(t, "plan", "--synthetic", "--depth", "fast", "--accept")
	assertBlackBoxSuccess(t, "plan", plan)
	state := harness.loadColonyState(t)
	if state.Plan.EvidencePolicy != colony.PlanEvidenceBoundV1 || len(state.Plan.Phases) == 0 {
		t.Fatalf("plan did not persist the bound evidence contract: %+v", state.Plan)
	}

	focus := harness.run(t, "focus", "preserve deterministic acceptance evidence")
	assertBlackBoxSuccess(t, "focus", focus)
	for phaseNumber := 1; phaseNumber <= len(state.Plan.Phases); phaseNumber++ {
		build := harness.runWithEnv(t, providerEnv, "build", fmt.Sprintf("%d", phaseNumber), "--light", "--worker-timeout", "30s")
		assertBlackBoxSuccess(t, fmt.Sprintf("build %d", phaseNumber), build)
		attempt := harness.loadBuildAttempt(t, phaseNumber)
		if attempt.Status != buildAttemptBuilt || attempt.Claims == nil {
			t.Fatalf("phase %d lacks durable provider-backed build evidence: %+v", phaseNumber, attempt)
		}
		if state.Plan.Phases[phaseNumber-1].Mode == colony.PhaseModeDiscovery {
			if !buildAttemptHasWorkerReport(attempt) {
				t.Fatalf("discovery phase %d lacks durable worker-report evidence: %+v", phaseNumber, attempt)
			}
		} else if len(attempt.Claims.ArtifactEvidence) == 0 {
			t.Fatalf("implementation phase %d lacks project artifact evidence: %+v", phaseNumber, attempt)
		}

		continued := harness.runWithEnv(t, providerEnv, "continue", "--verification-depth", "light", "--worker-timeout", "30s")
		assertBlackBoxSuccess(t, fmt.Sprintf("continue phase %d", phaseNumber), continued)
		report := harness.loadVerificationReport(t, phaseNumber)
		if !report.Passed || !report.CriteriaEnforced || !report.CriteriaPassed {
			t.Fatalf("phase %d verification did not enforce its criteria: %+v", phaseNumber, report)
		}
		state = harness.loadColonyState(t)
	}
	if state.State != colony.StateCOMPLETED {
		t.Fatalf("journey ended in %s, want COMPLETED", state.State)
	}

	resumed := harness.run(t, "resume")
	assertBlackBoxSuccess(t, "resume", resumed)
	if !strings.Contains(resumed.Stdout, "Crowned Anthill") && !strings.Contains(resumed.Stdout, "aether seal") {
		t.Fatalf("resume did not recover the completed colony's next action:\n%s", resumed.Stdout)
	}
	// D-04's confirmation gate (198-03): the CLI itself asks before finishing.
	// Answer it the same way an owner would -- run seal once to see it stop,
	// record the exact recorded-answer, then rerun.
	firstSealAttempt := harness.runWithEnv(t, providerEnv, "seal")
	assertBlackBoxSuccess(t, "seal (awaiting confirmation)", firstSealAttempt)
	confirm := harness.run(t, "decision-answer", "--question", sealConfirmationQuestionText(nil), "--answer", "yes", "--source", "seal-confirmation")
	assertBlackBoxSuccess(t, "decision-answer (seal confirmation)", confirm)
	seal := harness.runWithEnv(t, providerEnv, "seal")
	assertBlackBoxSuccess(t, "seal", seal)
	state = harness.loadColonyState(t)
	if state.Milestone != "Crowned Anthill" {
		t.Fatalf("sealed milestone = %q, want Crowned Anthill", state.Milestone)
	}
	if _, err := os.Stat(filepath.Join(harness.repo, ".aether", "CROWNED-ANTHILL.md")); err != nil {
		t.Fatalf("seal summary missing: %v", err)
	}
	// Phase 193 (D-08): the build side no longer dispatches an implicit
	// watcher without an explicit Queen proposal (none was made in this
	// journey), which was the only reliable source of a `"caste":"watcher"`
	// tag in this external adapter's log format -- a continue-dispatched
	// watcher still runs (report.Passed above proves verification
	// happened), it just is not caste-tagged in this log shape, a
	// pre-existing gap outside this plan's scope.
	logData, err := os.ReadFile(logPath)
	if err != nil || !bytes.Contains(logData, []byte(`"caste":"oracle"`)) || !bytes.Contains(logData, []byte(`"caste":"builder"`)) {
		t.Fatalf("journey did not execute research and builder provider processes: err=%v\n%s", err, logData)
	}
	harness.assertSourceUnchanged(t)
}

func TestCLICompiledInstallUpdateMigrationContract(t *testing.T) {
	t.Parallel()
	harness := newCLIBlackBox(t)
	install := harness.run(t, "install", "--package-dir", harness.sourceRoot, "--home-dir", harness.home, "--skip-build-binary")
	assertBlackBoxSuccess(t, "install", install)
	setup := harness.run(t, "lay-eggs", "--repo-dir", harness.repo, "--home-dir", harness.home)
	assertBlackBoxSuccess(t, "lay-eggs", setup)

	queenPath := filepath.Join(harness.repo, ".aether", "QUEEN.md")
	queenContent := "# Project Queen\n\nPreserve this repository-specific decision.\n"
	if err := os.WriteFile(queenPath, []byte(queenContent), 0644); err != nil {
		t.Fatalf("write project Queen: %v", err)
	}
	customSkillPath := filepath.Join(harness.repo, ".aether", "skills", "custom-journey", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(customSkillPath), 0755); err != nil {
		t.Fatalf("create custom skill directory: %v", err)
	}
	customSkill := "---\nname: custom-journey\ndescription: Project-owned acceptance behavior.\n---\n\n# Custom Journey\n"
	if err := os.WriteFile(customSkillPath, []byte(customSkill), 0644); err != nil {
		t.Fatalf("write custom skill: %v", err)
	}

	legacyState := []byte(`{
  "version": "3.0",
  "goal": "Preserve legacy state during update",
  "state": "READY",
  "current_phase": 0,
  "plan": {
    "phases": [{
      "id": 1,
      "name": "Legacy phase",
      "description": "A pre-evidence-contract plan",
      "status": "ready",
      "tasks": [{"id": "1.1", "goal": "Preserve the old task", "status": "pending", "success_criteria": ["The old task remains readable"]}],
      "success_criteria": ["The old phase remains readable"]
    }]
  }
}`)
	statePath := filepath.Join(harness.repo, ".aether", "data", "COLONY_STATE.json")
	if err := os.WriteFile(statePath, legacyState, 0644); err != nil {
		t.Fatalf("write legacy state: %v", err)
	}

	update := harness.run(t, "update", "--force")
	assertBlackBoxSuccess(t, "update --force", update)
	if got, err := os.ReadFile(queenPath); err != nil || string(got) != queenContent {
		t.Fatalf("update changed project Queen: err=%v\n%s", err, got)
	}
	if got, err := os.ReadFile(customSkillPath); err != nil || string(got) != customSkill {
		t.Fatalf("update changed project custom skill: err=%v\n%s", err, got)
	}
	if got, err := os.ReadFile(statePath); err != nil || !bytes.Equal(bytes.TrimSpace(got), bytes.TrimSpace(legacyState)) {
		t.Fatalf("update changed canonical colony state before migration: err=%v\n%s", err, got)
	}

	migrate := harness.run(t, "migrate-state")
	assertBlackBoxSuccess(t, "migrate-state", migrate)
	var migrationEnvelope struct {
		Result struct {
			Migrated        bool   `json:"migrated"`
			EvidencePolicy  string `json:"evidence_policy"`
			BackupPath      string `json:"backup_path"`
			RollbackCommand string `json:"rollback_command"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(migrate.Stdout)), &migrationEnvelope); err != nil {
		t.Fatalf("parse migration output: %v\n%s", err, migrate.Stdout)
	}
	migration := migrationEnvelope.Result
	if !migration.Migrated || migration.EvidencePolicy != string(colony.PlanEvidenceLegacy) || migration.BackupPath == "" || migration.RollbackCommand == "" {
		t.Fatalf("migration did not disclose legacy evidence and rollback: %+v", migration)
	}
	backupData, err := os.ReadFile(filepath.Join(harness.repo, ".aether", "data", filepath.FromSlash(migration.BackupPath)))
	if err != nil || !bytes.Equal(bytes.TrimSpace(backupData), bytes.TrimSpace(legacyState)) {
		t.Fatalf("migration backup is not an exact pre-migration state: err=%v\n%s", err, backupData)
	}
	state := harness.loadColonyState(t)
	if state.Plan.EvidencePolicy != colony.PlanEvidenceLegacy {
		t.Fatalf("migrated plan evidence policy = %q, want %q", state.Plan.EvidencePolicy, colony.PlanEvidenceLegacy)
	}
	rollback := harness.run(t, "migrate-state", "--rollback", migration.BackupPath)
	assertBlackBoxSuccess(t, "migrate-state --rollback", rollback)
	var rollbackEnvelope struct {
		Result struct {
			RolledBack   bool   `json:"rolled_back"`
			SafetyBackup string `json:"safety_backup"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(rollback.Stdout)), &rollbackEnvelope); err != nil {
		t.Fatalf("parse migration rollback output: %v\n%s", err, rollback.Stdout)
	}
	if !rollbackEnvelope.Result.RolledBack || rollbackEnvelope.Result.SafetyBackup == "" {
		t.Fatalf("migration rollback did not preserve a safety backup: %+v", rollbackEnvelope.Result)
	}
	rolledBackState, err := os.ReadFile(statePath)
	if err != nil || !bytes.Equal(rolledBackState, legacyState) {
		t.Fatalf("migration rollback did not restore byte-exact legacy state: err=%v\n%s", err, rolledBackState)
	}

	versions := harness.run(t, "versions")
	assertBlackBoxSuccess(t, "versions", versions)
	var versionsEnvelope struct {
		Result struct {
			Binary string `json:"binary"`
			Hub    string `json:"hub"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(versions.Stdout)), &versionsEnvelope); err != nil {
		t.Fatalf("parse versions output: %v\n%s", err, versions.Stdout)
	}
	if versionsEnvelope.Result.Binary == "" || versionsEnvelope.Result.Binary != versionsEnvelope.Result.Hub {
		t.Fatalf("compiled binary and installed hub versions differ: %+v", versionsEnvelope.Result)
	}
	for _, path := range []string{
		filepath.Join(harness.home, ".claude", "agents", "ant", "aether-builder.md"),
		filepath.Join(harness.home, ".config", "opencode", "agents", "aether-builder.md"),
		filepath.Join(harness.home, ".codex", "agents", "aether-builder.toml"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("installed platform agent missing at %s: %v", path, err)
		}
	}
	harness.assertSourceUnchanged(t)
}

func buildAttemptHasWorkerReport(attempt buildAttemptRecord) bool {
	for _, dispatch := range attempt.Dispatches {
		for _, output := range dispatch.Outputs {
			if strings.Contains(filepath.ToSlash(output), "/worker-reports/") {
				return true
			}
		}
	}
	return false
}

func assertBlackBoxSuccess(t *testing.T, operation string, result cliBlackBoxResult) {
	t.Helper()
	if result.ExitCode != 0 {
		t.Fatalf("%s failed: exit=%d\nstdout:\n%s\nstderr:\n%s", operation, result.ExitCode, result.Stdout, result.Stderr)
	}
}

func blackBoxDiscussionQuestions(t *testing.T, output string) []discussQuestion {
	t.Helper()
	var envelope struct {
		Result struct {
			Questions []discussQuestion `json:"questions"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &envelope); err != nil {
		t.Fatalf("parse discussion output: %v\n%s", err, output)
	}
	return envelope.Result.Questions
}

func writeBlackBoxCompletion(t *testing.T, path string, completion codexExternalBuildCompletion) {
	t.Helper()
	data, err := json.MarshalIndent(map[string]interface{}{"result": completion}, "", "  ")
	if err != nil {
		t.Fatalf("marshal black-box completion: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write black-box completion: %v", err)
	}
}

func jsonResultBool(output, key string, want bool) bool {
	var envelope struct {
		Result map[string]interface{} `json:"result"`
	}
	if json.Unmarshal([]byte(output), &envelope) != nil {
		return false
	}
	got, ok := envelope.Result[key].(bool)
	return ok && got == want
}

func waitForBlackBoxFile(t *testing.T, path string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for black-box file %s", path)
}

func killFixtureAdapterProcesses(t *testing.T, logPath string) {
	t.Helper()
	data, err := os.ReadFile(logPath)
	if err != nil {
		return
	}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		var entry struct {
			PID int `json:"pid"`
		}
		if json.Unmarshal([]byte(line), &entry) != nil || entry.PID <= 0 {
			continue
		}
		if process, findErr := os.FindProcess(entry.PID); findErr == nil {
			_ = process.Kill()
		}
	}
}
