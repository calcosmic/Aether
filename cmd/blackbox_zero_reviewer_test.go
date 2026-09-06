package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// writeZeroReviewerVerificationCommands writes a "## Verification Commands"
// CLAUDE.md section into the harness repo so a fixture can be given a
// passing command set or a failing one, without modifying
// cmd/blackbox_harness_test.go. An empty command string omits that line
// entirely, leaving the check unresolved.
func (h *cliBlackBox) writeZeroReviewerVerificationCommands(t *testing.T, build, types, lint, tests string) {
	t.Helper()
	var b strings.Builder
	b.WriteString("## Verification Commands\n\n")
	for _, entry := range []struct{ label, command string }{
		{"build", build},
		{"types", types},
		{"lint", lint},
		{"tests", tests},
	} {
		if strings.TrimSpace(entry.command) == "" {
			continue
		}
		fmt.Fprintf(&b, "- %s: %s\n", entry.label, entry.command)
	}
	if err := os.WriteFile(filepath.Join(h.repo, "CLAUDE.md"), []byte(b.String()), 0644); err != nil {
		t.Fatalf("write CLAUDE.md verification commands: %v", err)
	}
}

// prepareZeroReviewerChecksOnlyFixture is a variant of prepareBuildFixture
// (cmd/blackbox_harness_test.go) whose one success criterion is bound to a
// check ("claims") rather than an artifact, so phaseHasBoundArtifactRequirements
// is false and the "no tests to run in this project" warning fires when no
// verification command resolves for the repository -- the exact shape
// TestZeroExecutedChecksStillRunsClaimsAndCriteria needs. Kept in this file
// (not cmd/blackbox_harness_test.go) per the plan's read_first note.
func (h *cliBlackBox) prepareZeroReviewerChecksOnlyFixture(t *testing.T) string {
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

	goal := "Prove the deterministic floor without a bound artifact"
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
				Name:            "Checks-only criterion proof",
				Description:     "Create app.txt through a provider-backed worker process",
				Status:          colony.PhaseReady,
				SuccessCriteria: []string{"app.txt exists with deterministic content"},
				EvidenceRequirements: []colony.CriterionEvidenceRequirement{
					{
						Criterion: "app.txt exists with deterministic content",
						Checks:    []string{"claims"},
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
	logPath := filepath.Join(filepath.Dir(h.repo), "checks-only-adapter-invocations.jsonl")
	if err := os.Remove(logPath); err != nil && !os.IsNotExist(err) {
		t.Fatalf("reset adapter invocation log: %v", err)
	}
	return logPath
}

// TestZeroReviewerPhaseAdvancesWhenFreeChecksPass proves FLOOR-02's advance
// direction end to end through the real compiled binary: a phase with no
// reviewer worker dispatched (--skip-watchers) advances when the project's
// own build/types/lint/tests commands all resolve and pass.
func TestZeroReviewerPhaseAdvancesWhenFreeChecksPass(t *testing.T) {
	t.Parallel()
	harness := newCLIBlackBox(t)
	logPath := harness.prepareBuildFixture(t)
	harness.writeZeroReviewerVerificationCommands(t, "true", "true", "true", "true")
	env := map[string]string{
		"AETHER_ACTIVE_PLATFORM":     "codex",
		"AETHER_CODEX_PATH":          harness.adapter,
		"AETHER_CODEX_REAL_DISPATCH": "real",
		"AETHER_TEST_ADAPTER_MODE":   "success",
		"AETHER_WORKER_PLATFORM":     "codex",
		"AETHER_TEST_ADAPTER_LOG":    logPath,
	}
	build := harness.runWithEnv(t, env, "build", "1", "--light", "--worker-timeout", "1s")
	if build.ExitCode != 0 {
		t.Fatalf("build failed before continue: exit=%d\nstdout:\n%s\nstderr:\n%s", build.ExitCode, build.Stdout, build.Stderr)
	}

	result := harness.runWithEnv(t, env, "continue", "--skip-watchers", "--verification-depth", "light")
	if result.ExitCode != 0 {
		t.Fatalf("zero-reviewer continue did not exit 0: exit=%d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}
	report := harness.loadVerificationReport(t, 1)
	if !report.ChecksPassed || !report.CriteriaPassed || !report.Passed {
		t.Fatalf("zero-reviewer phase with green checks did not pass: %+v", report)
	}
	if !report.Watcher.Present || !strings.EqualFold(strings.TrimSpace(report.Watcher.Status), "skipped") {
		t.Fatalf("expected zero reviewer workers dispatched, got watcher = %+v", report.Watcher)
	}
	for _, name := range []string{"build", "types", "lint", "tests"} {
		found := false
		for _, step := range report.Steps {
			if step.Name != name {
				continue
			}
			found = true
			if step.Skipped || !step.Passed {
				t.Fatalf("step %s did not run green: %+v", name, step)
			}
		}
		if !found {
			t.Fatalf("expected step %s in report, got %+v", name, report.Steps)
		}
	}
	state := harness.loadColonyState(t)
	if len(state.Plan.Phases) != 1 || state.Plan.Phases[0].Status != colony.PhaseCompleted || state.State != colony.StateCOMPLETED {
		t.Fatalf("zero-reviewer green run did not complete the phase: state=%s phases=%+v", state.State, state.Plan.Phases)
	}
	harness.assertSourceUnchanged(t)
}

// TestZeroReviewerPhaseIsBlockedWhenFreeChecksFail proves FLOOR-02's block
// direction: with no reviewer worker dispatched, a failing project command
// (tests) blocks advancement and the failing check is named in the report.
func TestZeroReviewerPhaseIsBlockedWhenFreeChecksFail(t *testing.T) {
	t.Parallel()
	harness := newCLIBlackBox(t)
	logPath := harness.prepareBuildFixture(t)
	harness.writeZeroReviewerVerificationCommands(t, "true", "true", "true", "false")
	env := map[string]string{
		"AETHER_ACTIVE_PLATFORM":     "codex",
		"AETHER_CODEX_PATH":          harness.adapter,
		"AETHER_CODEX_REAL_DISPATCH": "real",
		"AETHER_TEST_ADAPTER_MODE":   "success",
		"AETHER_WORKER_PLATFORM":     "codex",
		"AETHER_TEST_ADAPTER_LOG":    logPath,
	}
	build := harness.runWithEnv(t, env, "build", "1", "--light", "--worker-timeout", "1s")
	if build.ExitCode != 0 {
		t.Fatalf("build failed before continue: exit=%d\nstdout:\n%s\nstderr:\n%s", build.ExitCode, build.Stdout, build.Stderr)
	}

	harness.runWithEnv(t, env, "continue", "--skip-watchers", "--verification-depth", "light")
	report := harness.loadVerificationReport(t, 1)
	if report.ChecksPassed || report.Passed {
		t.Fatalf("zero-reviewer phase with a failing check advanced: %+v", report)
	}
	issues := strings.Join(report.BlockingIssues, "\n")
	if !strings.Contains(issues, "tests failed") {
		t.Fatalf("blocking issues did not name the failing check: %+v", report.BlockingIssues)
	}
	state := harness.loadColonyState(t)
	if len(state.Plan.Phases) != 1 || state.Plan.Phases[0].Status == colony.PhaseCompleted || state.State == colony.StateCOMPLETED {
		t.Fatalf("zero-reviewer failing run advanced the phase: state=%s phases=%+v", state.State, state.Plan.Phases)
	}
	harness.assertSourceUnchanged(t)
}

// TestZeroExecutedChecksStillRunsClaimsAndCriteria proves D-01/FLOOR-01
// empty: a project with no resolvable verification command at all still
// runs claimed-files-exist and each-criterion-has-evidence, advances on
// those alone, and warns in plain English rather than handing verification
// responsibility to a reviewer.
func TestZeroExecutedChecksStillRunsClaimsAndCriteria(t *testing.T) {
	t.Parallel()
	harness := newCLIBlackBox(t)
	logPath := harness.prepareZeroReviewerChecksOnlyFixture(t)
	env := map[string]string{
		"AETHER_ACTIVE_PLATFORM":     "codex",
		"AETHER_CODEX_PATH":          harness.adapter,
		"AETHER_CODEX_REAL_DISPATCH": "real",
		"AETHER_TEST_ADAPTER_MODE":   "success",
		"AETHER_WORKER_PLATFORM":     "codex",
		"AETHER_TEST_ADAPTER_LOG":    logPath,
	}
	build := harness.runWithEnv(t, env, "build", "1", "--light", "--worker-timeout", "1s")
	if build.ExitCode != 0 {
		t.Fatalf("build failed before continue: exit=%d\nstdout:\n%s\nstderr:\n%s", build.ExitCode, build.Stdout, build.Stderr)
	}

	result := harness.runWithEnv(t, env, "continue", "--skip-watchers", "--verification-depth", "light")
	if result.ExitCode != 0 {
		t.Fatalf("continue with nothing mechanical to check did not exit 0: exit=%d\nstdout:\n%s\nstderr:\n%s", result.ExitCode, result.Stdout, result.Stderr)
	}
	report := harness.loadVerificationReport(t, 1)
	if !report.Claims.Present || !report.Claims.Passed {
		t.Fatalf("report lacks a claims result: %+v", report.Claims)
	}
	if !report.CriteriaEnforced || !report.CriteriaPassed || len(report.Criteria) == 0 {
		t.Fatalf("report lacks a criteria result: %+v", report)
	}
	if !report.ChecksPassed || !report.Passed {
		t.Fatalf("run did not advance on claims and criteria alone: %+v", report)
	}
	foundPlainEnglishWarning := false
	for _, warning := range report.Warnings {
		if strings.Contains(warning, "no tests to run in this project") {
			foundPlainEnglishWarning = true
		}
		if strings.Contains(strings.ToLower(warning), "watcher") {
			t.Fatalf("warning hands verification responsibility to a reviewer: %q", warning)
		}
	}
	if !foundPlainEnglishWarning {
		t.Fatalf("expected a plain-English %q warning, got %+v", "no tests to run in this project", report.Warnings)
	}
	state := harness.loadColonyState(t)
	if len(state.Plan.Phases) != 1 || state.Plan.Phases[0].Status != colony.PhaseCompleted || state.State != colony.StateCOMPLETED {
		t.Fatalf("run with nothing mechanical to check did not complete the phase: state=%s phases=%+v", state.State, state.Plan.Phases)
	}
	harness.assertSourceUnchanged(t)
}

// TestDispatchedReviewerThatFailedStillBlocks proves that a reviewer verdict
// can still be the sole source of a block even though every free check is
// green -- the deterministic floor and a dispatched reviewer's failure do
// not merge into a pass (D-06 adjacency).
func TestDispatchedReviewerThatFailedStillBlocks(t *testing.T) {
	t.Parallel()
	harness := newCLIBlackBox(t)
	logPath := harness.prepareBuildFixture(t)
	harness.writeZeroReviewerVerificationCommands(t, "true", "true", "true", "true")
	buildEnv := map[string]string{
		"AETHER_ACTIVE_PLATFORM":     "codex",
		"AETHER_CODEX_PATH":          harness.adapter,
		"AETHER_CODEX_REAL_DISPATCH": "real",
		"AETHER_TEST_ADAPTER_MODE":   "success",
		"AETHER_WORKER_PLATFORM":     "codex",
		"AETHER_TEST_ADAPTER_LOG":    logPath,
	}
	build := harness.runWithEnv(t, buildEnv, "build", "1", "--light", "--worker-timeout", "1s")
	if build.ExitCode != 0 {
		t.Fatalf("build failed before continue: exit=%d\nstdout:\n%s\nstderr:\n%s", build.ExitCode, build.Stdout, build.Stderr)
	}

	continueEnv := make(map[string]string, len(buildEnv))
	for k, v := range buildEnv {
		continueEnv[k] = v
	}
	continueEnv["AETHER_TEST_ADAPTER_MODE"] = "crash"

	harness.runWithEnv(t, continueEnv, "continue", "--verification-depth", "light", "--worker-timeout", "2s")
	report := harness.loadVerificationReport(t, 1)
	for _, step := range report.Steps {
		if step.Skipped || !step.Passed {
			t.Fatalf("free checks were not green ahead of the dispatched-reviewer failure: %+v", report.Steps)
		}
	}
	if !report.Watcher.Present || report.Watcher.Passed || strings.EqualFold(strings.TrimSpace(report.Watcher.Status), "skipped") {
		t.Fatalf("expected a dispatched reviewer that failed, got %+v", report.Watcher)
	}
	if report.ChecksPassed || report.Passed {
		t.Fatalf("a dispatched reviewer that failed did not block: %+v", report)
	}
	state := harness.loadColonyState(t)
	if len(state.Plan.Phases) != 1 || state.Plan.Phases[0].Status == colony.PhaseCompleted || state.State == colony.StateCOMPLETED {
		t.Fatalf("dispatched-reviewer failure advanced the phase: state=%s phases=%+v", state.State, state.Plan.Phases)
	}
	harness.assertSourceUnchanged(t)
}
