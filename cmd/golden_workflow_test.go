package cmd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// goldenTestdataDir returns the absolute path to cmd/testdata/ in the source tree.
// This is needed because some tests change the working directory.
func goldenTestdataDir() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		return "testdata"
	}
	return filepath.Join(filepath.Dir(filename), "testdata")
}

// stripANSI removes ANSI escape sequences from a string.
// Reimplements pkg/codex/platform_dispatch.go:stripANSIEscapeCodes (unexported).
func stripANSI(s string) string {
	var b strings.Builder
	inEscape := false
	for i := 0; i < len(s); i++ {
		ch := s[i]
		if inEscape {
			if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') {
				inEscape = false
			}
			continue
		}
		if ch == 0x1b {
			inEscape = true
			continue
		}
		b.WriteByte(ch)
	}
	return b.String()
}

// workerNameRe matches deterministic worker name patterns like "Hammer-22", "Forge-41".
// The prefix varies based on temp directory paths used as hash seeds, so we replace
// the entire match with a caste-agnostic placeholder for golden file stability.
var workerNameRe = regexp.MustCompile(`\b[A-Z][a-z]+-\d{1,3}\b`)

// stepElapsedRe matches rendered workflow step durations, which vary with host
// load and race instrumentation even when the workflow output is otherwise identical.
var stepElapsedRe = regexp.MustCompile(`(?m)(Step \d+/\d+: [^\n]+) \(\d+s\)$`)

var ceremonyElapsedRe = regexp.MustCompile(`(?m)(Ceremony complete in )\d+s$`)

// liveCheckLineDurationRe matches SHOW-03's live verification finish lines
// ("Build ✓ (0.3s)", "Tests ✗ (0.0s)") -- the measured duration varies with
// host load even when every other byte of the check's outcome is identical.
var liveCheckLineDurationRe = regexp.MustCompile(`(?m)^(\s*)(Build|Types|Lint|Tests) (✓|✗) \(\d+\.\d+s\)`)

// normalizeWorkerNames replaces all worker name patterns (CapitalWord-Number)
// with a fixed placeholder so golden files are stable across test runs.
// Worker names are hash-based on temp directory paths, making them non-deterministic.
func normalizeWorkerNames(s string) string {
	return workerNameRe.ReplaceAllString(s, "Worker-XX")
}

// normalizeForGolden prepares output for golden comparison by:
// 1. Stripping ANSI escape codes
// 2. Normalizing non-deterministic worker names
// 3. Normalizing non-deterministic workflow step durations
// 4. Removing ceremony activity lines (non-deterministic concurrent output)
func normalizeForGolden(s string) string {
	clean := stripANSI(s)
	clean = normalizeWorkerNames(clean)
	clean = stepElapsedRe.ReplaceAllString(clean, "$1 (0s)")
	clean = ceremonyElapsedRe.ReplaceAllString(clean, "${1}0s")
	clean = liveCheckLineDurationRe.ReplaceAllString(clean, "$1$2 $3 (0.0s)")

	var filtered strings.Builder
	for _, line := range strings.Split(clean, "\n") {
		trimmed := strings.TrimSpace(line)

		// Skip ceremony log lines -- these are concurrent and non-deterministic
		if strings.HasPrefix(trimmed, "[CEREMONY]") {
			continue
		}
		// Skip COLONY ACTIVITY blocks and their indented content
		if strings.HasPrefix(trimmed, "COLONY ACTIVITY") {
			continue
		}
		// Skip watch-status wave progress lines (e.g., "Wave 11: 0/1 starting")
		if matched, _ := regexp.MatchString(`^Wave \d+: \d+/\d+ `, trimmed); matched {
			continue
		}
		// Skip activity block section headers
		if trimmed == "Context:" || trimmed == "Completed:" || trimmed == "Active:" ||
			strings.HasPrefix(trimmed, "Workers:") {
			continue
		}
		// Skip indented ceremony context lines (activity block content)
		if strings.HasPrefix(trimmed, "ceremony.") {
			continue
		}
		// Skip indented worker status within activity blocks
		// (e.g., "  🔨 Builder:Worker-XX completed task=g-task-1...")
		if strings.HasPrefix(line, "  ") && (strings.Contains(line, "completed task=") ||
			strings.Contains(line, "starting task=") ||
			strings.Contains(line, "running task=") ||
			strings.Contains(line, "simulated worker heartbeat")) {
			continue
		}
		// Skip indented worker name references in activity blocks
		// (e.g., "  🔨 Builder:Worker-XX starting task=g-task-1 Worker: Worker-XX")
		if strings.HasPrefix(line, "  ") && strings.Contains(line, "Worker: Worker-XX") {
			continue
		}
		// Skip wave progress table lines
		if strings.Contains(trimmed, "WAVE") && strings.Contains(trimmed, "DISPATCHED") {
			continue
		}
		if strings.Contains(trimmed, "Total") && strings.Contains(trimmed, "dispatches") && strings.Contains(trimmed, "succeeded") {
			continue
		}
		// Skip table separator lines (pure +---+---+)
		if matched, _ := regexp.MatchString(`^\+[-+]+\+$`, trimmed); matched {
			continue
		}
		if matched, _ := regexp.MatchString(`^\|.*\|.*\|.*\|`, trimmed); matched {
			continue
		}

		filtered.WriteString(line)
		filtered.WriteString("\n")
	}
	return strings.TrimSpace(filtered.String()) + "\n"
}

// compareGolden strips ANSI from got, normalizes non-deterministic content, and
// (if -update-golden) or reads and compares against the existing golden file.
// Uses the shared updateGolden flag from audit_catalog_test.go.
func compareGolden(t *testing.T, goldenPath, got string) {
	t.Helper()
	clean := normalizeForGolden(got)

	if *updateGolden {
		if err := os.WriteFile(goldenPath, []byte(clean), 0644); err != nil {
			t.Fatalf("write golden file %s: %v", goldenPath, err)
		}
		t.Logf("golden file updated: %s", goldenPath)
		return
	}

	data, err := os.ReadFile(goldenPath)
	if err != nil {
		t.Fatalf("read golden file %s: %v (run with -update-golden to create)", goldenPath, err)
	}
	want := normalizeForGolden(string(data))

	if clean != want {
		t.Errorf("golden mismatch for %s; run with -update-golden to refresh", goldenPath)
		// Show first difference for debugging
		gotLines := strings.Split(clean, "\n")
		wantLines := strings.Split(want, "\n")
		maxLen := len(gotLines)
		if len(wantLines) > maxLen {
			maxLen = len(wantLines)
		}
		for i := 0; i < maxLen; i++ {
			var g, w string
			if i < len(gotLines) {
				g = gotLines[i]
			}
			if i < len(wantLines) {
				w = wantLines[i]
			}
			if g != w {
				t.Logf("  first diff at line %d:\n    got:  %q\n    want: %q", i+1, g, w)
				return
			}
		}
	}
}

func TestGoldenPlanVisualOutput(t *testing.T) {
	goldenPath := filepath.Join(goldenTestdataDir(), "golden_plan.txt")
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	// Pin the platform: command naming in visual output is platform-specific,
	// so an unpinned golden records whatever host the suite happened to run on.
	// Claude Code is the primary platform, so the golden locks its naming.
	// Codex and OpenCode naming is covered by TestVisualOutputNeverLeaksRawWrapperCommands.
	t.Setenv("AETHER_PLATFORM", "claude")

	goal := "Golden workflow test colony"
	selection, err := resolvePlanningPreset(codexPlanOptions{})
	if err != nil {
		t.Fatal(err)
	}
	output := renderPlanVisual(planningPresetRequiredResult(colony.ColonyState{Goal: &goal}, selection))
	compareGolden(t, goldenPath, output)

	// Verify golden content expectations (only when not updating)
	if !*updateGolden {
		clean := normalizeForGolden(output)
		for _, want := range []string{"P L A N", "Choose Planning Preset", "Fast", "Balanced", "Deep", "Exhaustive", "Planning did not start. State: unchanged."} {
			if !strings.Contains(clean, want) {
				t.Errorf("plan golden output missing %q", want)
			}
		}
		for _, forbidden := range []string{"P L A N   D I S P A T C H", "Planning Wave", "Choice: Run `/ant-build 1`"} {
			if strings.Contains(clean, forbidden) {
				t.Errorf("unselected preset golden crossed the planning boundary via %q", forbidden)
			}
		}
	}
}

func TestGoldenBuildVisualOutput(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	goal := "Golden workflow test colony"
	taskOneID := "1.1"
	taskTwoID := "1.2"
	accepted := createApprovedAcceptedBuildTestColony(t, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:     1,
			Name:   "Golden phase",
			Status: colony.PhaseReady,
			Tasks: []colony.Task{
				{ID: &taskOneID, Goal: "First golden task", Status: colony.TaskPending},
				{ID: &taskTwoID, Goal: "Second golden task", Status: colony.TaskPending, DependsOn: []string{taskOneID}},
			},
		}}},
	})
	root := accepted.Root

	goldenPath := filepath.Join(goldenTestdataDir(), "golden_build.txt")

	withTestWorkspace(t, root)
	withWorkingDir(t, root)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	// Pin the platform: command naming in visual output is platform-specific,
	// so an unpinned golden records whatever host the suite happened to run on.
	// Claude Code is the primary platform, so the golden locks its naming.
	// Codex and OpenCode naming is covered by TestVisualOutputNeverLeaksRawWrapperCommands.
	t.Setenv("AETHER_PLATFORM", "claude")

	assertGoldenAcceptedPlanAuthority(t, root)

	stdout = &bytes.Buffer{}
	rootCmd.SetArgs([]string{"build", "1"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("build returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()
	assertGoldenBuiltThroughCanonicalAttempt(t, root, 1)

	// Verify semantics before the textual snapshot may be refreshed.
	//
	// Plan 194-02 (D-07): "Watcher" is no longer among these -- the build
	// floor shrank to the builder alone, and this fixture's wording does not
	// score watcher above the relevance threshold either.
	clean := normalizeForGolden(output)
	for _, want := range []string{
		"B U I L D   D I S P A T C H   1", "S P A W N   P L A N",
		"Builder",
		"── Context ──", "── Tasks ──", "── Dispatch ──",
		"── Verification", "── Housekeeping ──",
		"── Colony Complete ──",
		"it is safe to close this chat",
	} {
		if !strings.Contains(clean, want) {
			t.Errorf("build golden output missing %q", want)
		}
	}
	compareGolden(t, goldenPath, output)
}

func TestGoldenContinueVisualOutput(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	goal := "Golden workflow test colony"
	taskID := "1.1"
	nextTaskID := "2.1"
	acceptedState := colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "Golden phase",
					Status: colony.PhaseReady,
					Tasks:  []colony.Task{{ID: &taskID, Goal: "Golden builder task", Status: colony.TaskPending}},
				},
				{
					ID:     2,
					Name:   "Next golden phase",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &nextTaskID, Goal: "Next golden task", Status: colony.TaskPending}},
				},
			},
		},
	}
	accepted := createApprovedAcceptedBuildTestColony(t, acceptedState)
	root := accepted.Root
	now := accepted.Candidate.CreatedAt.Add(2 * time.Minute).UTC()

	goldenPath := filepath.Join(goldenTestdataDir(), "golden_continue.txt")

	withTestWorkspace(t, root)
	withWorkingDir(t, root)
	t.Setenv("AETHER_OUTPUT_MODE", "visual")
	// Pin the platform: command naming in visual output is platform-specific,
	// so an unpinned golden records whatever host the suite happened to run on.
	// Claude Code is the primary platform, so the golden locks its naming.
	// Codex and OpenCode naming is covered by TestVisualOutputNeverLeaksRawWrapperCommands.
	t.Setenv("AETHER_PLATFORM", "claude")
	assertGoldenAcceptedPlanAuthority(t, root)

	dispatches := []codexBuildDispatch{
		{Stage: "wave", Wave: 1, Caste: "builder", Name: "Forge-41", Task: "Golden builder task", Status: "completed", TaskID: taskID, Outputs: []string{"main.go"}},
		{Stage: "verification", Caste: "watcher", Name: "Keen-42", Task: "Independent verification", Status: "completed", Outputs: []string{"main.go"}},
	}
	acceptedPhase := accepted.State.Plan.Phases[0]
	manifest := codexBuildManifest{
		Phase: 1, PhaseName: "Golden phase", Goal: goal, Root: root,
		ColonyDepth: "standard", DispatchMode: "direct", ExecutionOwner: "runtime-worker-dispatch",
		GeneratedAt: now.Format(time.RFC3339), State: string(colony.StateBUILT),
		ClaimsPath: displayDataPath("last-build-claims.json"), SelectedTasks: []string{taskID},
		Tasks:                   []codexBuildTaskPlan{{ID: taskID, Goal: "Golden builder task", Status: colony.TaskCompleted}},
		SuccessCriteria:         append([]string(nil), acceptedPhase.SuccessCriteria...),
		CriterionEvidencePolicy: phaseCriterionEvidencePolicy(acceptedPhase),
		EvidenceRequirements:    flattenPhaseCriterionEvidenceRequirements(acceptedPhase),
		Dispatches:              dispatches,
	}
	fixture := commitTestBuildStartAt(t, root, 1, now, testBuildStartOptions{
		Variant: buildStartDirect, Phase: 1, GeneratedAt: now, ProcessState: testBuildProcessDead,
		SelectedTasks: []string{taskID}, Dispatches: dispatches,
		ExecutionOwner: "runtime-worker-dispatch", DispatchMode: "direct", Manifest: &manifest,
	})
	builtState := acceptedState
	builtState.State = colony.StateBUILT
	builtState.BuildStartedAt = &now
	builtState.Plan.Phases[0].Status = colony.PhaseInProgress
	builtState.Plan.Phases[0].Tasks[0].Status = colony.TaskInProgress
	completeCanonicalContinueAttempt200(t, fixture, builtState, dispatches)
	assertGoldenBuiltThroughCanonicalAttempt(t, root, 1)

	// Seed empty-but-VALID instincts/observations files so phase-end
	// consolidation (D-04) runs cleanly to a deterministic zero-state beat
	// instead of a load failure whose error text embeds this test's
	// t.TempDir() path -- which would make the checked-in golden fixture
	// mismatch on every run since that path is different each time.
	if err := store.SaveJSON("instincts.json", colony.InstinctsFile{Instincts: []colony.InstinctEntry{}}); err != nil {
		t.Fatalf("seed instincts.json: %v", err)
	}
	if err := store.SaveJSON("learning-observations.json", colony.LearningFile{Observations: []colony.Observation{}}); err != nil {
		t.Fatalf("seed learning-observations.json: %v", err)
	}

	stdout = &bytes.Buffer{}
	rootCmd.SetArgs([]string{"continue"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}

	output := stdout.(*bytes.Buffer).String()

	// Verify semantics before the textual snapshot may be refreshed.
	clean := normalizeForGolden(output)
	for _, want := range []string{
		"Verification",
		"Next phase ready: 2",
		"Choice: Run `/ant-build 2`",
		"Choice: Run `/ant-run`",
	} {
		if !strings.Contains(clean, want) {
			t.Errorf("continue golden output missing %q", want)
		}
	}
	state := loadTestColonyState(t)
	if state.State != colony.StateREADY || state.CurrentPhase != 2 || state.Plan.Phases[0].Status != colony.PhaseCompleted {
		t.Fatalf("continue did not advance canonical state: state=%s current_phase=%d phase_1=%s", state.State, state.CurrentPhase, state.Plan.Phases[0].Status)
	}
	assertGoldenAcceptedPlanAuthority(t, root)
	compareGolden(t, goldenPath, output)
}

// loadTestColonyState reads COLONY_STATE.json from the store for golden state tests.
func loadTestColonyState(t *testing.T) colony.ColonyState {
	t.Helper()
	state, err := loadColonyState()
	if err != nil {
		t.Fatalf("failed to load colony state: %v", err)
	}
	if state == nil {
		t.Fatal("colony state is nil after load")
	}
	return *state
}

func assertGoldenAcceptedPlanAuthority(t *testing.T, root string) {
	t.Helper()
	facts, err := loadLifecycleFacts(root, store, time.Now().UTC())
	if err != nil {
		t.Fatalf("load golden lifecycle authority facts: %v", err)
	}
	decision, err := preflightCodexBuildPlanAuthority(facts, loadPlanAuthorityVerifiedBindings(root, facts))
	if err != nil {
		t.Fatalf("resolve golden accepted-plan authority: %v", err)
	}
	if !decision.Eligible || decision.Classification != planAuthorityCurrentAccepted {
		t.Fatalf("golden workflow lacks current accepted-plan authority: %+v", decision)
	}
}

func assertGoldenBuiltThroughCanonicalAttempt(t *testing.T, root string, phase int) {
	t.Helper()
	state := loadTestColonyState(t)
	if state.State != colony.StateBUILT || state.CurrentPhase != phase || state.BuildStartedAt == nil {
		t.Fatalf("golden build state did not cross canonical start: state=%s current_phase=%d build_started_at=%v", state.State, state.CurrentPhase, state.BuildStartedAt)
	}
	attemptPath, attempt, ok := loadLatestBuildAttempt(phase)
	if !ok {
		t.Fatalf("golden build phase %d has no canonical latest attempt", phase)
	}
	if attempt.Status != buildAttemptBuilt || attempt.PlanManifest == nil || attempt.Claims == nil {
		t.Fatalf("golden build attempt is not terminal and evidence-backed: %+v", attempt)
	}
	var receipt buildStartReceipt
	if err := store.LoadJSON(buildStartReceiptPath(phase, attempt.ID), &receipt); err != nil {
		t.Fatalf("load golden build-start receipt: %v", err)
	}
	if receipt.AttemptID != attempt.ID || receipt.AttemptPath != attemptPath ||
		!receipt.PlanAuthority.Eligible || receipt.PlanAuthority.Classification != planAuthorityCurrentAccepted {
		t.Fatalf("golden build receipt lost attempt or accepted-plan binding: receipt=%+v attempt_path=%q", receipt, attemptPath)
	}
	assertGoldenAcceptedPlanAuthority(t, root)
}

func finalizeGoldenBuildFromRuntimePlan(t *testing.T, root string, phase int) {
	t.Helper()
	result, _, _, _, err := runCodexBuildPlanOnly(root, phase, nil)
	if err != nil {
		t.Fatalf("prepare golden runtime build plan: %v", err)
	}
	manifest, ok := result["dispatch_manifest"].(codexBuildManifest)
	if !ok {
		t.Fatalf("golden runtime build omitted typed dispatch manifest: %#v", result["dispatch_manifest"])
	}

	evidencePath := "golden-runtime-evidence.txt"
	if err := os.WriteFile(filepath.Join(root, evidencePath), []byte("canonical golden build completion\n"), 0o644); err != nil {
		t.Fatalf("write golden runtime evidence: %v", err)
	}
	workers := make([]codexExternalBuildWorkerResult, 0, len(manifest.Dispatches))
	for _, dispatch := range manifest.Dispatches {
		worker := codexExternalBuildWorkerResult{
			Stage:         dispatch.Stage,
			Wave:          dispatch.Wave,
			ExecutionWave: normalizedDispatchWave(dispatch),
			Caste:         dispatch.Caste,
			Name:          dispatch.Name,
			TaskID:        dispatch.TaskID,
			Status:        "completed",
			Summary:       dispatch.Name + " completed the golden runtime task",
			Handoff: codex.WorkerHandoff{
				CommandsRun:            []string{"go test ./..."},
				VerificationStatus:     "pass",
				NextWorkerInstructions: []string{"golden runtime work complete"},
			},
		}
		if dispatch.Caste == "builder" {
			worker.Outputs = []string{evidencePath}
			worker.FilesCreated = []string{evidencePath}
		}
		workers = append(workers, worker)
	}
	completion := codexExternalBuildCompletion{DispatchManifest: &manifest, Dispatches: workers}
	completionData, err := json.MarshalIndent(completion, "", "  ")
	if err != nil {
		t.Fatalf("marshal golden runtime completion: %v", err)
	}
	completionPath := filepath.Join(root, "golden-build-completion.json")
	if err := os.WriteFile(completionPath, completionData, 0o644); err != nil {
		t.Fatalf("write golden runtime completion: %v", err)
	}

	stdout = &bytes.Buffer{}
	rootCmd.SetArgs([]string{"build-finalize", fmt.Sprint(phase), "--completion-file", completionPath})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("finalize golden runtime build: %v", err)
	}
}

func TestGoldenStateMutations(t *testing.T) {
	saveGlobals(t)
	resetRootCmd(t)

	goal := "Golden state mutation test"
	taskOneID := "1.1"
	taskTwoID := "2.1"

	// Reach READY through the real specification approval and plan acceptance
	// path. The public build/continue commands own every later lifecycle write.
	accepted := createApprovedAcceptedBuildTestColony(t, colony.ColonyState{
		Version:      "3.0",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{
					ID:     1,
					Name:   "State phase",
					Status: colony.PhaseReady,
					Tasks:  []colony.Task{{ID: &taskOneID, Goal: "State task one", Status: colony.TaskPending}},
				},
				{
					ID:     2,
					Name:   "Final phase",
					Status: colony.PhasePending,
					Tasks:  []colony.Task{{ID: &taskTwoID, Goal: "State task two", Status: colony.TaskPending}},
				},
			},
		},
	})
	root := accepted.Root
	withTestWorkspace(t, root)
	withWorkingDir(t, root)
	assertGoldenAcceptedPlanAuthority(t, root)

	state := loadTestColonyState(t)
	if state.State != colony.StateREADY {
		t.Errorf("before build: expected state READY, got %q", state.State)
	}
	if state.CurrentPhase != 1 || len(state.Plan.Phases) != 2 {
		t.Fatalf("before build: expected accepted phase 1 of 2, got current=%d phases=%d", state.CurrentPhase, len(state.Plan.Phases))
	}

	// The production plan-only + finalize boundary creates the receipt,
	// attempt, runtime-issued manifest, real claims, and BUILT state. The test
	// supplies only the external worker result that this boundary requires.
	finalizeGoldenBuildFromRuntimePlan(t, root, 1)
	assertGoldenBuiltThroughCanonicalAttempt(t, root, 1)

	// Continue consumes the exact artifacts emitted by build; no test-only
	// state rewrite or hand-built manifest bridges the commands.
	stdout = &bytes.Buffer{}
	rootCmd.SetArgs([]string{"continue", "--skip-watchers", "--light"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("continue returned error: %v", err)
	}
	continueOutput := stdout.(*bytes.Buffer).String()

	state = loadTestColonyState(t)
	if state.Plan.Phases[0].Status != colony.PhaseCompleted {
		t.Errorf("after continue: expected phase 1 status PhaseCompleted, got %q\n%s", state.Plan.Phases[0].Status, continueOutput)
	}
	if state.CurrentPhase != 2 {
		t.Errorf("after continue: expected CurrentPhase advanced to 2, got %d", state.CurrentPhase)
	}
	// Multi-phase colony should return to READY (not COMPLETED)
	if state.State != colony.StateREADY {
		t.Errorf("after continue: expected state READY (multi-phase), got %q", state.State)
	}
	assertGoldenAcceptedPlanAuthority(t, root)
}
