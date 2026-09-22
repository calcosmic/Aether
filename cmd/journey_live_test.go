//go:build journey

package cmd

// 207-04-PLAN.md (UED-08): grows the tracer's one step (207-01) into the
// whole fourteen-step journey, three trials, failures sorted into flaky
// and real, caps on every call, and a report journeyGateVerdict
// (cmd/journey.go) can honestly judge.
//
// Session chaining: the first step ("start") runs with
// --output-format json and the harness reads .session_id from the result.
// Every later step runs its own `claude -p` invocation carrying
// --resume "<that session id>" plus --output-format stream-json --verbose,
// so each step keeps its own transcript file and its own exit code -- the
// failing step can therefore be named -- while remaining one continuous
// conversation (207-RESEARCH.md Q1).
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
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// journeyRequiredTools names every external binary this journey step
// needs. A missing tool FAILS the test by name.
var journeyRequiredTools = []string{"go", "git", "jq", "claude"}

// journeyFirstGoal is the goal scripts/build-messy-practice-project.sh
// already passes to its own `aether init` while constructing the practice
// project (207-01-SUMMARY.md) -- the "start" step's own on-disk fact check
// reads this back, it never re-derives it.
const journeyFirstGoal = "Practice the daily lifecycle on a messy real project"

// journeySecondGoal is the goal the "start-again" step's own /ant-init
// call carries, distinct from journeyFirstGoal so the fact check after
// start-again can prove a genuinely NEW colony state exists rather than
// the first one lingering.
const journeySecondGoal = "Practice the daily lifecycle a second time after the first was archived"

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
	builderScript := filepath.Join(repoRoot, "scripts", "build-messy-practice-project.sh")

	targetStep := strings.TrimSpace(os.Getenv("AETHER_JOURNEY_STEP"))
	scope := "whole-chain"
	var steps []journeyStep
	if targetStep != "" {
		step := journeyStep(targetStep)
		if !journeyStepDeclared(step) {
			t.Fatalf("AETHER_JOURNEY_STEP names an undeclared step: %q (want one of %v)", targetStep, journeyStepNames())
		}
		scope = "one-step"
		steps = []journeyStep{step}
	} else {
		steps = journeyStepVocabulary
	}

	trialCount := journeyResolveTrialCount(t, scope)

	traps, err := loadJourneyTraps()
	if err != nil {
		t.Fatalf("load journey traps: %v", err)
	}
	// Fail fast, before spending any money on a live trial, if the
	// committed register itself is malformed -- journeyRunExpectedRedCheck
	// re-reads it fresh at status-step time regardless (never a stale
	// in-memory copy), this is purely an early sanity check.
	if _, err := loadJourneyExpectedRed(); err != nil {
		t.Fatalf("load journey expected-red register: %v", err)
	}

	budgetFlagSupported := journeyClaudeSupportsBudgetFlag(t)

	var trials []journeyTrial
	var expectedRed []journeyExpectedRedResult
	for i := 0; i < trialCount; i++ {
		trial, tExpectedRed := journeyRunOneTrial(t, i, trialCount, repoRoot, builderScript, traps, steps, scope == "one-step", budgetFlagSupported)
		trials = append(trials, trial)
		if tExpectedRed != nil {
			expectedRed = tExpectedRed
		}
	}

	executed := 0
	if len(trials) > 0 {
		executed = trials[len(trials)-1].StepsExecuted
	}
	report := journeyReport{
		Scope:         scope,
		Mode:          "live",
		StepsDeclared: len(steps),
		StepsExecuted: executed,
		Trials:        trials,
		ExpectedRed:   expectedRed,
		Verdict:       "pending",
	}

	// Each concurrent test process gets its own report path via its own
	// t.TempDir() -- two journeys running at once can never overwrite each
	// other's results.
	reportPath := filepath.Join(t.TempDir(), journeyReportSchemaVersion, "journey-report.json")
	if err := writeJourneyReport(reportPath, report); err != nil {
		t.Fatalf("write journey report: %v", err)
	}
	readBack, err := readJourneyReport(reportPath)
	if err != nil {
		t.Fatalf("read journey report back: %v", err)
	}

	verdictErr := journeyGateVerdict(readBack)
	t.Logf("%s", journeyReportSummary(readBack))
	if verdictErr != nil {
		t.Logf("journeyGateVerdict: refused -- %v", verdictErr)
	} else {
		t.Logf("journeyGateVerdict: PASS")
	}

	switch {
	case scope == "one-step":
		// This IS the proof that the gate cannot be satisfied by a partial
		// run: a one-step report must be REFUSED, naming the scope.
		if verdictErr == nil {
			t.Fatalf("journeyGateVerdict accepted a one-step report -- it must refuse a partial run")
		}
		if !strings.Contains(verdictErr.Error(), "one-step") {
			t.Fatalf("journeyGateVerdict's refusal does not name the scope: %v", verdictErr)
		}
	case trialCount < journeyMinimumTrials:
		// A reduced-trial-count whole-chain run (this plan's own money-aware
		// verification, per 207-04-PLAN.md's <assumptions>) proves the chain
		// works end to end, but must still be refused by the gate on trial
		// count alone -- Plan 06 is what spends the money on the real three
		// trials.
		if verdictErr == nil {
			t.Fatalf("journeyGateVerdict accepted a report with only %d trial(s) -- it must refuse fewer than %d trials", trialCount, journeyMinimumTrials)
		}
	default:
		// A genuine trialCount >= journeyMinimumTrials whole-chain run: a
		// real product defect discovered by a real trial already surfaced
		// as a t.Run subtest failure above (the correct place for it to
		// fail); this final block does not additionally assert
		// journeyGateVerdict returns nil, since doing so would let a
		// single flaky-classified trial across three PASS this test while
		// still correctly registering as "not yet a proven release" in the
		// verdict itself -- the two questions ("did the harness run
		// correctly" and "does this report satisfy the release gate") are
		// deliberately kept separate.
	}
}

// journeyResolveTrialCount returns how many trials this run should attempt:
// always 1 for single-step debugging (running one step three times debugs
// nothing extra and only spends money), otherwise AETHER_JOURNEY_TRIALS
// when set (development-only reduction, explicitly instructed to be
// possible -- 207-04-PLAN.md Task 2), else journeyMinimumTrials.
func journeyResolveTrialCount(t *testing.T, scope string) int {
	t.Helper()
	if scope == "one-step" {
		return 1
	}
	raw := strings.TrimSpace(os.Getenv("AETHER_JOURNEY_TRIALS"))
	if raw == "" {
		return journeyMinimumTrials
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 {
		t.Fatalf("AETHER_JOURNEY_TRIALS must be a positive integer, got %q", raw)
	}
	return n
}

// journeyRunOneTrial builds a fresh practice project (every trial gets its
// own, per 207-04-PLAN.md Task 2's own instruction), establishes a fresh
// session, and drives steps through it in order, stopping the trial the
// moment a step fails and recording every later step "not-reached". trialCount
// controls subtest naming: a single-trial run (the norm for this plan's own
// money-aware verification and for AETHER_JOURNEY_STEP debugging) names
// subtests by step alone; a multi-trial run (Plan 06's real three-trial
// gate run) qualifies each subtest with its trial index so three trials'
// worth of "build", say, are never confused with one another in -v output.
func journeyRunOneTrial(t *testing.T, index, trialCount int, repoRoot, builderScript string, traps journeyTrapFile, steps []journeyStep, singleStep bool, budgetFlagSupported bool) (journeyTrial, []journeyExpectedRedResult) {
	t.Helper()
	startedAt := time.Now().UTC()

	dest := t.TempDir()
	buildCtx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	buildCmd := exec.CommandContext(buildCtx, builderScript, dest)
	buildOut, err := buildCmd.CombinedOutput()
	if err != nil {
		t.Fatalf("trial %d: scripts/build-messy-practice-project.sh failed: %v\n%s", index, err, buildOut)
	}

	repo := filepath.Join(dest, "repo")
	settingsPath := filepath.Join(repo, ".claude", "settings.json")
	if _, err := os.Stat(settingsPath); err != nil {
		t.Fatalf("trial %d: practice project has no .claude/settings.json (aether update --force did not register hooks/commands): %v", index, err)
	}
	binDir := filepath.Join(dest, ".journey-bin")
	if _, err := os.Stat(filepath.Join(binDir, "aether")); err != nil {
		t.Fatalf("trial %d: built aether binary not found at %s: %v", index, filepath.Join(binDir, "aether"), err)
	}

	if singleStep {
		journeyFastForwardToStep(t, repo, binDir, steps[0])
	}

	firstCaps := journeyCapsForStep(steps[0], budgetFlagSupported)
	sessionID := journeyCaptureSessionID(t, repo, binDir, settingsPath, budgetFlagSupported)

	var stepResults []journeyStepResult
	var expectedRed []journeyExpectedRedResult
	stopped := false
	for _, step := range steps {
		if stopped {
			stepResults = append(stepResults, journeyStepResult{Name: string(step), Status: "not-reached"})
			continue
		}

		menuCommand, ok := journeyStepMenuCommand(step)
		if !ok {
			t.Fatalf("step %q has no declared menu command", step)
		}
		result := journeyStepResult{Name: string(step), MenuCommand: menuCommand, TrapIDs: journeyTrapIDsForStep(traps, step)}
		caps := journeyCapsForStep(step, budgetFlagSupported)
		prompt := journeyStepPrompt(step)

		subtestName := string(step)
		if trialCount > 1 {
			subtestName = fmt.Sprintf("trial-%d/%s", index, step)
		}
		t.Run(subtestName, func(t *testing.T) {
			journeyDriveStep(t, &result, repo, repoRoot, binDir, settingsPath, sessionID, prompt, menuCommand, caps, budgetFlagSupported, dest, index)
		})

		if step == journeyStepStatus && result.Status == "pass" {
			expectedRed = journeyRunExpectedRedCheck(t, repo, repoRoot)
		}

		stepResults = append(stepResults, result)
		if result.Status != "pass" {
			stopped = true
		}
	}

	executed := 0
	for _, s := range stepResults {
		if s.Status != "not-reached" {
			executed++
		}
	}
	outcome, incompleteAt := journeyDeriveTrialOutcome(stepResults, len(steps), executed)

	trial := journeyTrial{
		Index:            index,
		SessionID:        sessionID,
		StartedAt:        startedAt.Format(time.RFC3339),
		EndedAt:          time.Now().UTC().Format(time.RFC3339),
		Caps:             firstCaps,
		Steps:            stepResults,
		StepsDeclared:    len(steps),
		StepsExecuted:    executed,
		IncompleteAtStep: incompleteAt,
		Outcome:          string(outcome),
	}
	return trial, expectedRed
}

// journeyDriveStep runs one lifecycle step's prompt through a real chat,
// resuming sessionID, then asserts (a) the transcript's <command-name> tag
// names menuCommand, (b) at least one Bash tool call happened, and (c) the
// step's own on-disk fact -- never the chat's own prose. result's fields are
// set before any t.Fatalf call so the caller always has a usable,
// classified result even when the subtest itself is marked failed (Fatalf
// halts only this goroutine via runtime.Goexit, after which the already-set
// fields on result remain visible to the caller).
func journeyDriveStep(t *testing.T, result *journeyStepResult, repo, repoRoot, binDir, settingsPath, sessionID, prompt, menuCommand string, caps journeyCaps, budgetFlagSupported bool, dest string, trialIndex int) {
	t.Helper()

	streamOutPath := filepath.Join(dest, fmt.Sprintf("trial-%d-%s-stream.jsonl", trialIndex, result.Name))
	status, retried := journeyRunStep(t, repo, binDir, settingsPath, prompt, sessionID, budgetFlagSupported, caps, streamOutPath)
	result.Retried = retried
	if status != 0 {
		streamed, _ := os.ReadFile(streamOutPath)
		result.Status = "fail"
		result.FailureKind = string(classifyJourneyFailure(string(streamed)))
		result.Detail = fmt.Sprintf("claude -p invocation failed (exit %d, retried=%v)", status, retried)
		t.Fatalf("step %q: %s", result.Name, result.Detail)
	}

	transcriptPath, err := journeyFindSessionTranscript(sessionID)
	if err != nil {
		result.Status = "fail"
		result.FailureKind = string(journeyFailureReal)
		result.Detail = fmt.Sprintf("locate the real on-disk session transcript: %v", err)
		t.Fatalf("step %q: %s", result.Name, result.Detail)
	}

	names, err := journeyMenuCommandNames(transcriptPath)
	if err != nil {
		result.Status = "fail"
		result.FailureKind = string(journeyFailureReal)
		result.Detail = fmt.Sprintf("parse menu command names from transcript: %v", err)
		t.Fatalf("step %q: %s", result.Name, result.Detail)
	}
	if !journeyContainsString(names, menuCommand) {
		result.Status = "fail"
		result.FailureKind = string(journeyFailureReal)
		result.Detail = fmt.Sprintf("transcript never carries a <command-name>%s</command-name> tag -- the menu command did not drive this step (found: %v)", menuCommand, names)
		t.Fatalf("step %q: %s", result.Name, result.Detail)
	}

	bashCalls, err := journeyBashToolCallCount(transcriptPath)
	if err != nil {
		result.Status = "fail"
		result.FailureKind = string(journeyFailureReal)
		result.Detail = fmt.Sprintf("count Bash tool calls in transcript: %v", err)
		t.Fatalf("step %q: %s", result.Name, result.Detail)
	}
	result.BashToolCalls = bashCalls
	// The "start" step is the one legitimate exception to "at least one Bash
	// tool call": scripts/build-messy-practice-project.sh's own `aether
	// init` already established this practice project (so every trap has a
	// .aether/ to seed into) before the chat ever runs -- verified live
	// (2026-09-22): the real /ant-init wrapper, seeing an already-active
	// colony in its own injected context, correctly explains why it is
	// declining and names the real next step (/ant-plan) WITHOUT running
	// any Bash command at all, avoiding a wasted, guaranteed-to-fail
	// invocation. That is the wrapper behaving correctly, not a defect --
	// requiring a Bash call here would fail a step that did exactly the
	// right thing. Every other step drives a freshly-reached lifecycle
	// state where a real Bash call is the only way the step's own on-disk
	// fact could ever become true, so the requirement stays load-bearing
	// everywhere else.
	if bashCalls < 1 && journeyStep(result.Name) != journeyStepStart {
		result.Status = "fail"
		result.FailureKind = string(journeyFailureReal)
		result.Detail = "the chat never ran a Bash tool call -- it never ran the underlying aether command at all"
		t.Fatalf("step %q: %s", result.Name, result.Detail)
	}

	journeyAssertStepFact(t, result, repo, repoRoot, journeyStep(result.Name))
	if result.Status == "fail" {
		// journeyAssertStepFact already called t.Fatalf when it set this --
		// unreachable in practice, kept only so a future edit that forgets
		// to Fatalf still leaves result correctly classified.
		return
	}
	result.Status = "pass"
}

// journeyAssertStepFact checks the one on-disk fact 207-04-PLAN.md names
// for step, derived from reading the real runtime code that writes it
// (never a plausible-looking guess). Sets result's failure fields and calls
// t.Fatalf before returning on failure; leaves result untouched (the caller
// sets Status="pass") on success.
func journeyAssertStepFact(t *testing.T, result *journeyStepResult, repo, repoRoot string, step journeyStep) {
	t.Helper()
	fail := func(detail string) {
		result.Status = "fail"
		result.FailureKind = string(journeyFailureReal)
		result.Detail = detail
		t.Fatalf("step %q: %s", step, detail)
	}

	switch step {
	case journeyStepStart:
		state := journeyReadColonyState(t, repo)
		if state.Goal == nil || strings.TrimSpace(*state.Goal) == "" {
			fail("COLONY_STATE.json parses but carries no goal")
			return
		}
		if strings.TrimSpace(*state.Goal) != journeyFirstGoal {
			fail(fmt.Sprintf("COLONY_STATE.json goal is %q after start, want %q -- state does not match what the practice project was actually built with", *state.Goal, journeyFirstGoal))
		}

	case journeyStepSurvey:
		snapPath := filepath.Join(repo, filepath.FromSlash(territorySnapshotRelativePath))
		data, err := os.ReadFile(snapPath)
		if err != nil {
			fail(fmt.Sprintf("read %s: %v", snapPath, err))
			return
		}
		var snap territorySnapshotMetadata
		if err := json.Unmarshal(data, &snap); err != nil {
			fail(fmt.Sprintf("%s does not parse: %v", snapPath, err))
			return
		}
		head, err := journeyGitRevParseHead(repo)
		if err != nil {
			fail(fmt.Sprintf("git rev-parse HEAD: %v", err))
			return
		}
		if snap.SourceRevision != head {
			fail(fmt.Sprintf("territory snapshot source_revision is %s, want current HEAD %s -- the out-of-date-code-map trap's snapshot should have been refreshed by the survey step", snap.SourceRevision, head))
		}

	case journeyStepDiscuss:
		// The shared decision store (CLAUDE.md's own name for it) is the
		// one place a real /ant-discuss run would leave a clarification --
		// if this messy project genuinely produced nothing to clarify, the
		// file may legitimately not exist; a real product defect here would
		// instead surface as the transcript/Bash-call checks above already
		// failing, or as a malformed file below.
		path := filepath.Join(repo, ".aether", "data", "pending-decisions.json")
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				result.Detail = "no pending-decisions.json after discuss (nothing needed clarifying in this run)"
				return
			}
			fail(fmt.Sprintf("read %s: %v", path, err))
			return
		}
		var probe interface{}
		if err := json.Unmarshal(data, &probe); err != nil {
			fail(fmt.Sprintf("%s does not parse as JSON: %v", path, err))
		}

	case journeyStepSpecification:
		state := journeyReadColonyState(t, repo)
		if state.Specification == nil || strings.TrimSpace(state.Specification.CurrentRevisionID) == "" {
			fail("COLONY_STATE.json carries no approved specification after the specification step")
		}

	case journeyStepPlanFirst:
		path := filepath.Join(repo, ".aether", "data", "planning", "phase-plan.json")
		if _, err := os.Stat(path); err != nil {
			fail(fmt.Sprintf("no plan artifact at %s: %v", path, err))
		}

	case journeyStepPlanSecond:
		journeyAssertPlanSecondFact(t, result, repo, fail)

	case journeyStepBuild:
		path := filepath.Join(repo, ".aether", "data", "handoffs", "worker-handoffs.json")
		data, err := os.ReadFile(path)
		if err != nil {
			fail(fmt.Sprintf("read %s: %v", path, err))
			return
		}
		var handoffs []interface{}
		if err := json.Unmarshal(data, &handoffs); err != nil {
			fail(fmt.Sprintf("%s does not parse as a JSON array: %v", path, err))
			return
		}
		if len(handoffs) == 0 {
			fail(fmt.Sprintf("%s parses but carries no handoffs after the build step", path))
		}

	case journeyStepCheck:
		matches, err := filepath.Glob(filepath.Join(repo, ".aether", "data", "gate-results-*.json"))
		if err != nil {
			fail(fmt.Sprintf("glob gate-results-*.json: %v", err))
			return
		}
		if len(matches) == 0 {
			fail("no .aether/data/gate-results-*.json after the check step")
		}

	case journeyStepStatus:
		// The expected-red register's own genuine result (journeyRunExpectedRedCheck,
		// called by the caller once this subtest passes) IS this step's
		// on-disk fact -- recording it in the report, per 207-04-PLAN.md,
		// not a pass/fail condition of the subtest itself.

	case journeyStepPause:
		state := journeyReadColonyState(t, repo)
		if state.PauseHandoff == nil {
			fail("COLONY_STATE.json carries no pause_handoff after the pause step")
			return
		}
		handoffPath := filepath.Join(repo, ".aether", "HANDOFF.md")
		if _, err := os.Stat(handoffPath); err != nil {
			fail(fmt.Sprintf("no %s after the pause step: %v", handoffPath, err))
		}

	case journeyStepResume:
		state := journeyReadColonyState(t, repo)
		if state.RecoveryProvenance == nil {
			fail("COLONY_STATE.json carries no recovery_provenance after the resume step")
		}

	case journeyStepFinish:
		state := journeyReadColonyState(t, repo)
		if state.SealOutcome == nil {
			fail("COLONY_STATE.json carries no seal_outcome after the finish step")
		}

	case journeyStepArchive:
		chambers, err := filepath.Glob(filepath.Join(repo, ".aether", "chambers", "*"))
		if err != nil {
			fail(fmt.Sprintf("glob .aether/chambers/*: %v", err))
			return
		}
		if len(chambers) == 0 {
			fail("no chamber directory under .aether/chambers after the archive step")
			return
		}
		// The live flags file holds only the carried subset (issue/note,
		// never a blocker) once entomb has run -- best-effort, since the
		// file may legitimately not exist if nothing survived the carry
		// filter (CLAUDE.md's "Flags" section: filterCarriedFlags).
		flagsPath := filepath.Join(repo, ".aether", "data", "pending-decisions.json")
		if data, err := os.ReadFile(flagsPath); err == nil {
			var flags []colony.FlagEntry
			if err := json.Unmarshal(data, &flags); err == nil {
				for _, f := range flags {
					if f.Type == "blocker" {
						fail(fmt.Sprintf("%s still carries a blocker flag after archive -- only issue/note may survive the carry-forward filter", flagsPath))
						return
					}
				}
			}
		}

	case journeyStepStartAgain:
		state := journeyReadColonyState(t, repo)
		if state.Goal == nil || strings.TrimSpace(*state.Goal) != journeySecondGoal {
			got := "<nil>"
			if state.Goal != nil {
				got = *state.Goal
			}
			fail(fmt.Sprintf("COLONY_STATE.json goal is %q after start-again, want %q", got, journeySecondGoal))
		}
	}
}

// journeyAssertPlanSecondFact proves the second /ant-plan pass did not get
// stuck ON the specification-corrected-mid-planning trap's parked run
// (journeyTrapSupersededPlanRunID): it finds the most recently modified
// OTHER planning run directory and, when that run's own stage-state.json
// still exists, checks it is bound to the specification's CURRENT approved
// revision, never the superseded one. A run that finished cleanly and had
// its own staging directory cleaned up is treated as informational, not a
// hard failure -- the genuinely provable fact here is "did not get stuck",
// and the mere existence of any OTHER, later-modified run directory (or a
// clean finish with nothing left to inspect) already demonstrates that.
func journeyAssertPlanSecondFact(t *testing.T, result *journeyStepResult, repo string, fail func(string)) {
	t.Helper()
	planningDir := filepath.Join(repo, ".aether", "data", "planning")
	entries, err := os.ReadDir(planningDir)
	if err != nil {
		fail(fmt.Sprintf("read %s: %v", planningDir, err))
		return
	}

	var latestRunID string
	var latestModTime time.Time
	for _, e := range entries {
		if !e.IsDir() || e.Name() == journeyTrapSupersededPlanRunID {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if latestRunID == "" || info.ModTime().After(latestModTime) {
			latestRunID = e.Name()
			latestModTime = info.ModTime()
		}
	}
	if latestRunID == "" {
		fail("no planning run other than the pre-seeded superseded-plan trap's own run exists after the second /ant-plan pass")
		return
	}

	stagePath := filepath.Join(repo, filepath.FromSlash(planningStageStateRepositoryPath(latestRunID)))
	stateData, err := os.ReadFile(stagePath)
	if err != nil {
		result.Detail = fmt.Sprintf("no lingering stage-state.json for run %s (planning may have finished cleanly and cleaned up its own staging directory): %v", latestRunID, err)
		return
	}
	var runState planningStageState
	if err := json.Unmarshal(stateData, &runState); err != nil {
		fail(fmt.Sprintf("parse %s: %v", stagePath, err))
		return
	}

	state := journeyReadColonyState(t, repo)
	currentRevision := ""
	if state.Specification != nil {
		currentRevision = state.Specification.CurrentRevisionID
	}
	if runState.Specification.RevisionID != currentRevision {
		fail(fmt.Sprintf("second-pass planning run %s is bound to specification revision %s, but the current approved revision is %s -- it must not remain pinned to the superseded revision", latestRunID, runState.Specification.RevisionID, currentRevision))
	}
}

// journeyReadColonyState reads and parses the practice project's own
// COLONY_STATE.json -- the authoritative on-disk record most per-step
// facts in this file check.
func journeyReadColonyState(t *testing.T, repo string) colony.ColonyState {
	t.Helper()
	path := filepath.Join(repo, ".aether", "data", "COLONY_STATE.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	var state colony.ColonyState
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("%s does not parse as JSON: %v", path, err)
	}
	return state
}

// journeyRunExpectedRedCheck runs the sixth-blocker check (207-03-PLAN.md)
// for real against the practice project the status step just drove,
// deriving what the status card actually advises from the real guidance
// code, and folds the result into the report's expected_red section via
// journeyEvaluateExpectedRed (cmd/journey.go).
func journeyRunExpectedRedCheck(t *testing.T, repo, repoRoot string) []journeyExpectedRedResult {
	t.Helper()
	s, err := storage.NewStore(filepath.Join(repo, ".aether", "data"))
	if err != nil {
		t.Fatalf("open practice project store for the expected-red check: %v", err)
	}
	missing, err := statusGuidanceCommandsWithoutMenuWrapper(s, repo, repoRoot)
	if err != nil {
		t.Fatalf("statusGuidanceCommandsWithoutMenuWrapper: %v", err)
	}
	// The register is re-read fresh here (cheap) rather than reusing a copy
	// captured once at the top of TestJourney, so a stale in-memory copy can
	// never silently drift from what is actually committed on disk.
	registered, err := loadJourneyExpectedRed()
	if err != nil {
		t.Fatalf("load journey expected-red register: %v", err)
	}
	return journeyEvaluateExpectedRed(registered.Cases, missing)
}

// journeyGitRevParseHead returns the practice project's current HEAD
// revision, the same fact classifySurveyFreshness itself compares a
// territory snapshot's source_revision against.
func journeyGitRevParseHead(repo string) (string, error) {
	out, err := exec.Command("git", "-C", repo, "rev-parse", "HEAD").Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// journeyTrapIDsForStep returns the declared trap ids (in traps' own file
// order) whose journey_steps names step -- the report entry names which
// traps this step exercised, taken from loadJourneyTraps(), never a
// re-typed list.
func journeyTrapIDsForStep(traps journeyTrapFile, step journeyStep) []string {
	var ids []string
	for _, tr := range traps.Traps {
		for _, s := range tr.JourneySteps {
			if s == string(step) {
				ids = append(ids, tr.ID)
				break
			}
		}
	}
	return ids
}

// journeyStepMaxTurns, journeyStepWallClockSecs and journeyStepMaxBudgetUSD
// size each step's own caps to what that step's wrapper actually needs
// (207-04-PLAN.md Task 1): the simple relay steps (status, pause, resume,
// start, start-again) need few turns and a short wall clock; survey,
// discuss, specification, plan-first, plan-second, build and check --
// every step whose own wrapper dispatches real subagent workers -- need
// substantially more of both. These are a documented, instrumented number,
// never a fixed, unquestioned one: the first-draft caps for survey
// (max-budget-usd 1.00) were measured live (2026-09-22) against this exact
// repository's own /ant-colonize wrapper and found genuinely too tight --
// it dispatches four real surveyor subagents (surveyor-provisions,
// surveyor-nest, surveyor-disciplines, surveyor-pathogens) and reached
// $2.09 actual cost before the too-tight cap cut it off mid-run
// (terminal_reason: budget_exhausted). The figures below are that
// measurement times a safety margin, not a guess; a later pass should keep
// tightening or widening them from further real measurements, never
// silently.
var journeyStepMaxTurns = map[journeyStep]int{
	journeyStepStart:         4,
	journeyStepSurvey:        20,
	journeyStepDiscuss:       12,
	journeyStepSpecification: 15,
	journeyStepPlanFirst:     25,
	journeyStepPlanSecond:    25,
	journeyStepBuild:         40,
	journeyStepCheck:         40,
	journeyStepStatus:        4,
	journeyStepPause:         4,
	journeyStepResume:        4,
	journeyStepFinish:        12,
	journeyStepArchive:       12,
	journeyStepStartAgain:    4,
}

var journeyStepWallClockSecs = map[journeyStep]int{
	journeyStepStart:         120,
	journeyStepSurvey:        600,
	journeyStepDiscuss:       400,
	journeyStepSpecification: 400,
	journeyStepPlanFirst:     700,
	journeyStepPlanSecond:    700,
	journeyStepBuild:         1200,
	journeyStepCheck:         1200,
	journeyStepStatus:        120,
	journeyStepPause:         120,
	journeyStepResume:        120,
	journeyStepFinish:        400,
	journeyStepArchive:       400,
	journeyStepStartAgain:    120,
}

var journeyStepMaxBudgetUSD = map[journeyStep]float64{
	journeyStepStart:         0.50,
	journeyStepSurvey:        4.00, // measured live at $2.09 (four real surveyor subagents); this is that measurement with a safety margin, not a guess
	journeyStepDiscuss:       2.00,
	journeyStepSpecification: 2.00,
	journeyStepPlanFirst:     4.00,
	journeyStepPlanSecond:    4.00,
	journeyStepBuild:         6.00,
	journeyStepCheck:         6.00,
	journeyStepStatus:        0.50,
	journeyStepPause:         0.50,
	journeyStepResume:        0.50,
	journeyStepFinish:        2.00,
	journeyStepArchive:       2.00,
	journeyStepStartAgain:    0.50,
}

// journeyCapsForStep returns the caps in force for step's own claude -p
// invocation.
func journeyCapsForStep(step journeyStep, budgetFlagSupported bool) journeyCaps {
	caps := journeyCaps{
		MaxTurns:      journeyStepMaxTurns[step],
		WallClockSecs: journeyStepWallClockSecs[step],
	}
	if caps.MaxTurns == 0 {
		caps.MaxTurns = 6
	}
	if caps.WallClockSecs == 0 {
		caps.WallClockSecs = 300
	}
	if budgetFlagSupported {
		caps.MaxBudgetUSD = journeyStepMaxBudgetUSD[step]
		if caps.MaxBudgetUSD == 0 {
			caps.MaxBudgetUSD = 1.00
		}
		caps.BudgetCapped = true
	}
	return caps
}

// journeyStepPrompt returns the literal prompt driving step: the menu
// command alone, adding nothing else (207-04-PLAN.md Task 1), except for
// start-again, which structurally requires a goal argument distinct from
// the first init -- without one, /ant-init has nothing to start a second
// colony with. This is the one place a step's prompt carries anything
// beyond the bare command name, and it is a required CLI argument, not an
// added instruction about what to assert.
func journeyStepPrompt(step journeyStep) string {
	cmd, _ := journeyStepMenuCommand(step)
	if step == journeyStepStartAgain {
		return fmt.Sprintf("%s %q", cmd, journeySecondGoal)
	}
	return cmd
}

// journeyFastForwardToStep prepares repo for driving target alone, by
// running direct, non-chat `aether` commands for the steps declared before
// target -- never by replaying earlier steps through the chat, which is
// exactly the cost this debugging mode exists to avoid (207-04-PLAN.md
// Task 1: "Money is the reason this mode exists").
//
// Scoped conservatively: only a step whose own command is safe to run
// without genuinely new generative content is actually invoked here
// (survey). A step whose own command requires real planning/build/check
// content (discuss, specification, plan, build, check) is NOT synthesized
// here -- producing that content any other way than the real runtime
// writers would be exactly the false-certificate fixture CLAUDE.md's
// Definition of Done forbids, and scripts/build-messy-practice-project.sh
// already ran a real `aether init` while constructing the practice project
// (207-01-SUMMARY.md), so "start" needs no fast-forward at all.
//
// This omission is verified safe for the one target this plan's own <verify>
// block actually exercises (AETHER_JOURNEY_STEP=pause): pauseSafeBoundary
// (cmd/session_flow_cmds.go) only blocks pause on a
// worker_completion_boundary (an in-flight build), which a freshly built,
// never-built practice project can never be in. A future target step that
// genuinely needs one of the skipped steps to have completed will fail
// loudly, at that step's own subtest, rather than silently -- which is the
// correct failure mode for a debugging aid, not a defect to paper over.
func journeyFastForwardToStep(t *testing.T, repo, binDir string, target journeyStep) {
	t.Helper()
	aether := filepath.Join(binDir, "aether")
	for _, step := range journeyStepVocabulary {
		if step == target {
			return
		}
		switch step {
		case journeyStepStart:
			// Already done by scripts/build-messy-practice-project.sh's own
			// `aether init` call -- nothing further to fast-forward.
		case journeyStepSurvey:
			cmd := exec.Command(aether, "colonize")
			cmd.Dir = repo
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Logf("fast-forward: %q did not complete cleanly -- proceeding anyway (target step %q does not depend on it): %v\n%s", "aether colonize", target, err, out)
			}
		default:
			t.Logf("fast-forward: step %q needs real generative content and is not synthesized here -- if %q genuinely requires it to have run first, that will fail loudly at %q's own subtest rather than being papered over", step, target, target)
		}
	}
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
// so later steps can chain with --resume. Asserts only the exit status and
// the session_id field -- never the chat's own prose.
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
		// $2.00, deliberately more generous than any single driven step's
		// own cap: a session's FIRST call pays the one-time cost of
		// building this repository's own (large) CLAUDE.md/hooks/skills
		// prompt cache from cold -- observed empirically at ~$0.49 for a
		// single-word reply -- while every later --resume'd call reads
		// from that cache instead of rebuilding it (207-01-SUMMARY.md).
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

// journeyRunStep drives one lifecycle step's prompt through a real chat,
// resuming sessionID, writing the raw --output-format stream-json stdout to
// streamOutPath for debugging -- NOT the artifact the caller asserts
// against (see journeyFindSessionTranscript's comment: the <command-name>
// tag only exists in Claude Code's own on-disk session transcript, never in
// this streamed stdout). Never passes the flag that skips hooks, custom
// commands, subagents and CLAUDE.md: hooks and menu commands must genuinely
// fire, and the caller's <command-name> / Bash tool-call assertions against
// the on-disk transcript are what actually prove that -- the flag's mere
// absence is not itself proof (207-RESEARCH.md Pitfall 1).
func journeyRunStep(t *testing.T, repo, binDir, settingsPath, prompt, sessionID string, budgetFlagSupported bool, caps journeyCaps, streamOutPath string) (int, bool) {
	t.Helper()

	args := []string{
		"-p", prompt,
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
// generalized per 207-RESEARCH.md Q6. Returns stdout only (the transcript /
// --output-format json payload), never merged with stderr, so a transient
// failure's stderr noise can never corrupt a JSON parse.
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
