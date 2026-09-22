//go:build journey

package cmd

// 207-01-PLAN.md Task 1 (UED-08): drives one real lifecycle menu command
// through a real `claude -p` chat, inside a script-built messy practice
// project, and asserts what actually happened from the transcript and
// from disk -- never from the chat's own prose.
//
// This is the thin tracer slice: one lifecycle step ("status"), one trial,
// scope "one-step". Plan 04 expands this same TestJourney function to the
// full fourteen-step, three-trial "whole-chain" run and flips the report's
// scope accordingly -- journeyGateVerdict (cmd/journey.go) does not
// change shape between now and then, only the report it is fed does.
//
// Requires a signed-in `claude` CLI on the machine -- this test FAILS
// (never skips) when go/git/jq/claude is missing, per this repository's
// own Definition of Done: a skipped test reads as a pass.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// journeyRequiredTools names every external binary this journey step
// needs. A missing tool FAILS the test by name.
var journeyRequiredTools = []string{"go", "git", "jq", "claude"}

func TestJourney(t *testing.T) {
	for _, tool := range journeyRequiredTools {
		if _, err := exec.LookPath(tool); err != nil {
			t.Fatalf("required tool missing: %s (%v) -- this is a named failure reason, not a skip; a skipped test reads as a pass, which this repository's own Definition of Done forbids", tool, err)
		}
	}

	repoRoot, err := journeyTrapsRepoRoot()
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}

	t.Run("status", func(t *testing.T) {
		testJourneyStatusStep(t, repoRoot)
	})
}

func testJourneyStatusStep(t *testing.T, repoRoot string) {
	dest := t.TempDir()
	builderScript := filepath.Join(repoRoot, "scripts", "build-messy-practice-project.sh")

	buildCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	buildCmd := exec.CommandContext(buildCtx, builderScript, dest)
	buildOut, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("scripts/build-messy-practice-project.sh failed: %v\n%s", err, buildOut)
	}
	t.Logf("practice project build output:\n%s", buildOut)

	repo := filepath.Join(dest, "repo")
	settingsPath := filepath.Join(repo, ".claude", "settings.json")
	if _, err := os.Stat(settingsPath); err != nil {
		t.Fatalf("practice project has no .claude/settings.json at %s (aether update --force did not register hooks/commands): %v", settingsPath, err)
	}

	// The binary the builder script built is what the chat's own Bash tool
	// calls must resolve to -- put its directory first on PATH for every
	// claude invocation, exactly as scripts/proof-screens-reach-the-owner.sh
	// already does, so a stray aether already on the developer's PATH can
	// never be what the journey actually exercises.
	binDir := filepath.Join(dest, ".journey-bin")
	if _, err := os.Stat(filepath.Join(binDir, "aether")); err != nil {
		t.Fatalf("built aether binary not found at %s: %v", filepath.Join(binDir, "aether"), err)
	}

	budgetFlagSupported := journeyClaudeSupportsBudgetFlag(t)

	caps := journeyCaps{
		MaxTurns:      6,
		WallClockSecs: 300,
	}
	if budgetFlagSupported {
		caps.MaxBudgetUSD = 1.00
		caps.BudgetCapped = true
	}
	t.Logf("caps in force for the chat calls: max-turns=%d wall-clock=%ds max-budget-usd=%v (capped=%v)",
		caps.MaxTurns, caps.WallClockSecs, caps.MaxBudgetUSD, caps.BudgetCapped)

	sessionID := journeyCaptureSessionID(t, repo, binDir, settingsPath, budgetFlagSupported)

	menuCommand, ok := journeyStepMenuCommand(journeyStepStatus)
	if !ok {
		t.Fatalf("status step has no declared menu command")
	}

	streamOutPath := filepath.Join(t.TempDir(), "status-stream-stdout.jsonl")
	status, retried := journeyRunStep(t, repo, binDir, settingsPath, menuCommand, sessionID, budgetFlagSupported, caps, streamOutPath)
	if status != 0 {
		t.Fatalf("the status step's claude -p invocation failed (exit %d, retried=%v) -- see logged output above for the named reason", status, retried)
	}

	// The streamed stdout (streamOutPath, written by journeyRunStep) never
	// echoes the outgoing user turn -- only Claude Code's own persisted,
	// on-disk session transcript does, in the exact shape
	// cmd/testdata/stop-hook/menu-command-transcript.jsonl models (verified
	// empirically this session: reading a real captured run confirmed the
	// <command-message>/<command-name> tags exist ONLY in that on-disk
	// file, never in the stream-json stdout). Assert against THAT file --
	// still never against the chat's own prose.
	transcriptPath, err := journeyFindSessionTranscript(sessionID)
	if err != nil {
		t.Fatalf("locate the real on-disk session transcript: %v", err)
	}
	t.Logf("asserting against the on-disk session transcript at %s", transcriptPath)

	names, err := journeyMenuCommandNames(transcriptPath)
	if err != nil {
		t.Fatalf("parse menu command names from transcript: %v", err)
	}
	if !journeyContainsString(names, menuCommand) {
		t.Fatalf("transcript never carries a <command-name>%s</command-name> tag -- the menu command did not drive this step (found: %v)", menuCommand, names)
	}

	bashCalls, err := journeyBashToolCallCount(transcriptPath)
	if err != nil {
		t.Fatalf("count Bash tool calls in transcript: %v", err)
	}
	if bashCalls < 1 {
		t.Fatalf("the chat never ran a Bash tool call -- it never ran the underlying aether command at all, which is a different failure from a wording mismatch")
	}
	t.Logf("transcript carries %d Bash tool call(s) and command names %v", bashCalls, names)

	colonyStatePath := filepath.Join(repo, ".aether", "data", "COLONY_STATE.json")
	stateData, err := os.ReadFile(colonyStatePath)
	if err != nil {
		t.Fatalf("read %s: %v", colonyStatePath, err)
	}
	var stateJSON map[string]interface{}
	if err := json.Unmarshal(stateData, &stateJSON); err != nil {
		t.Fatalf("%s does not parse as JSON: %v", colonyStatePath, err)
	}

	report := journeyReport{
		Scope:         "one-step",
		Mode:          "live",
		StepsDeclared: len(journeyStepVocabulary),
		StepsExecuted: 1,
		Trials: []journeyTrial{
			{
				SessionID: sessionID,
				Caps:      caps,
				Steps: []journeyStepResult{
					{
						Name:          string(journeyStepStatus),
						MenuCommand:   menuCommand,
						Status:        "pass",
						BashToolCalls: bashCalls,
						Retried:       retried,
					},
				},
				Outcome: "pass",
			},
		},
		Verdict: "pending",
	}

	reportPath := filepath.Join(repo, journeyDefaultReportRelativePath)
	if err := writeJourneyReport(reportPath, report); err != nil {
		t.Fatalf("write journey report: %v", err)
	}
	readBack, err := readJourneyReport(reportPath)
	if err != nil {
		t.Fatalf("read journey report back: %v", err)
	}

	// This IS the tracer's proof that the gate cannot be satisfied by a
	// partial run: a one-step report must be REFUSED, naming the scope.
	// Plan 04 flips this run's scope to "whole-chain" and expects the
	// verdict to pass instead.
	verdictErr := journeyGateVerdict(readBack)
	if verdictErr == nil {
		t.Fatalf("journeyGateVerdict accepted a one-step report -- it must refuse a partial run")
	}
	if !strings.Contains(verdictErr.Error(), "one-step") {
		t.Fatalf("journeyGateVerdict's refusal does not name the scope: %v", verdictErr)
	}
	t.Logf("journeyGateVerdict correctly refused the one-step report: %v", verdictErr)
}

// journeyClaudeSupportsBudgetFlag probes `claude --help` for
// --max-budget-usd, exactly as scripts/proof-screens-reach-the-owner.sh
// already does, rather than assuming the flag exists.
func journeyClaudeSupportsBudgetFlag(t *testing.T) bool {
	t.Helper()
	out, err := exec.Command("claude", "--help").CombinedOutput()
	if err != nil {
		t.Fatalf("claude --help failed: %v\n%s", err, out)
	}
	return strings.Contains(string(out), "--max-budget-usd")
}

// journeyClaudeConfigDir resolves the directory Claude Code stores its own
// session transcripts under: CLAUDE_CONFIG_DIR when set (mirrors this
// repository's own multi-platform config-dir convention), otherwise the
// real $HOME/.claude. HOME is never overridden by this journey -- see the
// ISOLATION RULE in scripts/build-messy-practice-project.sh -- because
// Claude Code's own session history, like its sign-in, lives there.
func journeyClaudeConfigDir() (string, error) {
	if dir := os.Getenv("CLAUDE_CONFIG_DIR"); dir != "" {
		return dir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".claude"), nil
}

// journeySessionTranscriptPollInterval and journeySessionTranscriptPollTimeout
// bound how long journeyFindSessionTranscript waits for the on-disk
// transcript file to appear after the driving claude invocation exits --
// writing it is not guaranteed to be synchronous with process exit.
var (
	journeySessionTranscriptPollInterval = 250 * time.Millisecond
	journeySessionTranscriptPollTimeout  = 10 * time.Second
)

// journeyFindSessionTranscript locates the real, on-disk Claude Code
// session transcript for sessionID: <session-id>.jsonl under
// ~/.claude/projects/<encoded-cwd>/. Session ids are UUIDs, so matching by
// filename alone is unambiguous -- this deliberately avoids reimplementing
// Claude Code's own undocumented cwd-to-directory-name encoding scheme.
func journeyFindSessionTranscript(sessionID string) (string, error) {
	configDir, err := journeyClaudeConfigDir()
	if err != nil {
		return "", err
	}
	projectsDir := filepath.Join(configDir, "projects")
	target := sessionID + ".jsonl"

	deadline := time.Now().Add(journeySessionTranscriptPollTimeout)
	var lastErr error
	for {
		var found string
		walkErr := filepath.WalkDir(projectsDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil // tolerate a transient stat error on one entry, keep walking
			}
			if d.IsDir() {
				return nil
			}
			if d.Name() == target {
				found = path
				return fs.SkipAll
			}
			return nil
		})
		if walkErr != nil {
			lastErr = fmt.Errorf("search %s for session transcript %s: %w", projectsDir, target, walkErr)
		} else if found != "" {
			return found, nil
		} else {
			lastErr = fmt.Errorf("no on-disk session transcript found for session %s under %s", sessionID, projectsDir)
		}
		if time.Now().After(deadline) {
			return "", lastErr
		}
		time.Sleep(journeySessionTranscriptPollInterval)
	}
}

// journeyCaptureSessionID makes the first, session-establishing claude -p
// call -- 207-RESEARCH.md Q1's documented pattern:
//
//	session_id=$(claude -p "Start a review" --output-format json | jq -r '.session_id')
//	claude -p "Continue that review" --resume "$session_id"
//
// so later steps (Plan 04) can chain with --resume. Asserts only the exit
// status and the session_id field -- never the chat's own prose.
func journeyCaptureSessionID(t *testing.T, repo, binDir, settingsPath string, budgetFlagSupported bool) string {
	t.Helper()

	args := []string{
		"-p", "Acknowledge you are ready to work in this practice project. Reply with exactly one word: ready. Take no action.",
		"--output-format", "json",
		"--settings", settingsPath,
		"--permission-mode", "acceptEdits",
		"--allowedTools", "Bash",
		"--max-turns", "3",
	}
	if budgetFlagSupported {
		// $2.00, deliberately more generous than the driven step's own cap
		// below: a session's FIRST call pays the one-time cost of building
		// this repository's own (large) CLAUDE.md/hooks/skills prompt cache
		// from cold -- observed empirically at ~$0.49 for a single-word
		// reply on this machine -- while every later --resume'd call reads
		// from that cache instead of rebuilding it. Provisional, per
		// 207-RESEARCH.md Assumption A2; not a measurement, an instrumented
		// starting point.
		args = append(args, "--max-budget-usd", "2.00")
	}

	out, status, retried := journeyRunClaudeWithRetry(t, repo, binDir, args, 120*time.Second)
	if status != 0 {
		t.Fatalf("session-establishing claude -p call failed (exit %d, retried=%v)", status, retried)
	}

	var envelope struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(out, &envelope); err != nil {
		t.Fatalf("session-establishing call's --output-format json output does not parse: %v\n%s", err, out)
	}
	if envelope.SessionID == "" {
		t.Fatalf("session-establishing call returned an empty session_id")
	}
	return envelope.SessionID
}

// journeyRunStep drives one lifecycle step's menu command through a real
// chat, resuming sessionID, writing the raw --output-format stream-json
// stdout to streamOutPath for debugging -- NOT the artifact the caller
// asserts against (see journeyFindSessionTranscript's comment: the
// <command-name> tag only exists in Claude Code's own on-disk session
// transcript, never in this streamed stdout). Never passes the flag that
// skips hooks, custom commands, subagents and CLAUDE.md: hooks and menu
// commands must genuinely fire, and the caller's <command-name> / Bash
// tool-call assertions against the on-disk transcript are what actually
// prove that -- the flag's mere absence is not itself proof
// (207-RESEARCH.md Pitfall 1).
func journeyRunStep(t *testing.T, repo, binDir, settingsPath, menuCommand, sessionID string, budgetFlagSupported bool, caps journeyCaps, streamOutPath string) (int, bool) {
	t.Helper()

	args := []string{
		"-p", menuCommand,
		"--resume", sessionID,
		"--output-format", "stream-json",
		"--verbose",
		"--settings", settingsPath,
		"--permission-mode", "acceptEdits",
		"--allowedTools", "Bash",
		"--max-turns", fmt.Sprintf("%d", caps.MaxTurns),
	}
	if budgetFlagSupported {
		args = append(args, "--max-budget-usd", fmt.Sprintf("%.2f", caps.MaxBudgetUSD))
	}

	out, status, retried := journeyRunClaudeWithRetry(t, repo, binDir, args, time.Duration(caps.WallClockSecs)*time.Second)
	if err := os.WriteFile(streamOutPath, out, 0o644); err != nil {
		t.Fatalf("write captured stream stdout to %s: %v", streamOutPath, err)
	}
	return status, retried
}

// journeyRunClaudeWithRetry runs `claude` with args from dir, with binDir
// first on PATH, retrying exactly once if the combined stdout+stderr
// classifies as a transient failure -- the same one-retry-on-transient
// policy scripts/proof-screens-reach-the-owner.sh already implements,
// generalized per 207-RESEARCH.md Q6. Returns stdout only (the transcript
// / --output-format json payload), never merged with stderr, so a
// transient failure's stderr noise can never corrupt a JSON parse.
func journeyRunClaudeWithRetry(t *testing.T, dir, binDir string, args []string, timeout time.Duration) ([]byte, int, bool) {
	t.Helper()

	run := func() ([]byte, []byte, int) {
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		cmd := exec.CommandContext(ctx, "claude", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(), "PATH="+binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
		var stdoutBuf, stderrBuf bytes.Buffer
		cmd.Stdout = &stdoutBuf
		cmd.Stderr = &stderrBuf
		err := cmd.Run()
		status := 0
		if err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) {
				status = exitErr.ExitCode()
			} else {
				status = -1
			}
		}
		return stdoutBuf.Bytes(), stderrBuf.Bytes(), status
	}

	out, errOut, status := run()
	retried := false
	if status != 0 {
		if classifyJourneyFailure(string(out)+"\n"+string(errOut)) == journeyFailureTransient {
			t.Logf("transient failure (exit %d) -- retrying once, as instructed; not counted as a real failure unless it recurs", status)
			retried = true
			out, errOut, status = run()
		}
	}
	if status != 0 {
		t.Logf("claude invocation failed (exit %d): stdout=%s stderr=%s", status, string(out), string(errOut))
	}
	return out, status, retried
}
