package cmd

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
	"github.com/calcosmic/Aether/pkg/storage"
)

// setupSealTestStore creates a fresh temp store with a minimal colony state
// where all phases are completed, ready for seal.
func setupSealTestStore(t *testing.T) (*storage.Store, string) {
	t.Helper()
	tmpDir := t.TempDir()
	dataDir := filepath.Join(tmpDir, ".aether", "data")
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		t.Fatal(err)
	}

	goal := "Test colony goal"
	state := colony.ColonyState{
		Goal:         &goal,
		CurrentPhase: 1,
		State:        colony.StateREADY,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "Discovery", Status: colony.PhaseCompleted},
			},
		},
		Memory: colony.Memory{
			PhaseLearnings: []colony.PhaseLearning{},
		},
		Events: []string{},
	}

	s, err := createTestStore(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	// Create local QUEEN.md for promotion tests
	queenPath := filepath.Join(tmpDir, ".aether", "QUEEN.md")
	if err := os.WriteFile(queenPath, []byte(queenDefaultContent), 0644); err != nil {
		t.Fatal(err)
	}

	return s, tmpDir
}

// runSealCmd runs the seal command with the given args and returns stdout/stderr output.
func runSealCmd(t *testing.T, s *storage.Store, tmpDir string, args []string) (string, string) {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)

	dataDir := filepath.Join(tmpDir, ".aether", "data")
	t.Setenv("COLONY_DATA_DIR", dataDir)
	t.Setenv("AETHER_ROOT", tmpDir)
	// Seal promotes instincts into the hive at the hub. Belt-and-suspenders
	// with the suite-wide TestMain isolation: whatever the global env is in
	// this test's position in the run order, seal tests NEVER write the
	// developer's real ~/.aether/hive.
	t.Setenv("AETHER_HUB_DIR", filepath.Join(tmpDir, ".hub"))

	store = s
	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	stdout = outBuf
	stderr = errBuf

	// The current confirmation is derived from typed preflight facts. Record
	// the exact answer an owner would submit on a second invocation; the old
	// legacy-state helper could not cover forced-incomplete preflights.
	autoRecordSealPreflightConfirmationForTest(t, s, tmpDir, args)

	allArgs := append([]string{"seal"}, args...)
	rootCmd.SetArgs(allArgs)
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	rootCmd.Execute()

	return outBuf.String(), errBuf.String()
}

func autoRecordSealPreflightConfirmationForTest(t *testing.T, s *storage.Store, root string, args []string) {
	t.Helper()
	force := false
	reason := ""
	for index, arg := range args {
		switch arg {
		case "--force":
			force = true
		case "--reason":
			if index+1 < len(args) {
				reason = args[index+1]
			}
		}
	}
	facts, err := loadLifecycleFacts(root, s, time.Now().UTC())
	if err != nil {
		t.Fatalf("load seal facts: %v", err)
	}
	preflight, err := BuildSealPreflight(facts, SealPreflightRequest{Caller: SealCallerDirectOwner, Force: force, Reason: reason})
	if err != nil {
		return // The command must render this preflight refusal itself.
	}
	source := "seal-confirmation"
	if preflight.Disposition == colony.SealDispositionForcedIncomplete {
		source = "seal-force-confirmation"
	}
	if _, err := recordSealConfirmationAnswer(SealConfirmationCopy(preflight), "yes", source); err != nil {
		t.Fatalf("record seal confirmation: %v", err)
	}
}

// executeSealAtPublicRoot exercises the same Execute boundary used by main.
// Recovery menus render their own error payload, so callers must inspect both
// stderr and the renderedCommandError returned after Cobra completes.
func executeSealAtPublicRoot(t *testing.T, s *storage.Store, tmpDir, mode string) (string, string, error) {
	t.Helper()
	saveGlobals(t)
	resetRootCmd(t)
	t.Setenv("COLONY_DATA_DIR", filepath.Join(tmpDir, ".aether", "data"))
	t.Setenv("AETHER_ROOT", tmpDir)
	t.Setenv("AETHER_HUB_DIR", filepath.Join(tmpDir, ".hub"))
	t.Setenv("AETHER_OUTPUT_MODE", mode)
	store = s
	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	stdout = outBuf
	stderr = errBuf
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	rootCmd.SetArgs([]string{"seal"})
	err := Execute()
	return outBuf.String(), errBuf.String(), err
}

func requireRenderedSealExitOne(t *testing.T, err error) {
	t.Helper()
	var rendered renderedCommandError
	if !errors.As(err, &rendered) || rendered.code != 1 {
		t.Fatalf("seal error = %v, want rendered exit code 1", err)
	}
}

func checkpointCapabilitiesInSealOutput(output string) []string {
	const marker = "--checkpoint-capability '"
	capabilities := []string{}
	for {
		start := strings.Index(output, marker)
		if start < 0 {
			return capabilities
		}
		output = output[start+len(marker):]
		end := strings.IndexByte(output, '\'')
		if end < 0 {
			return capabilities
		}
		capabilities = append(capabilities, output[:end])
		output = output[end+1:]
	}
}

func TestSealPendingDecisionStorageFailureFailsClosed(t *testing.T) {
	assertLegacyStorageRefused := func(t *testing.T, s *storage.Store, tmpDir string) {
		t.Helper()
		_, errOut, err := executeSealAtPublicRoot(t, s, tmpDir, "json")
		requireRenderedSealExitOne(t, err)
		if !strings.Contains(errOut, "flags.json") {
			t.Fatalf("legacy blocker-storage failure did not identify flags.json:\n%s", errOut)
		}
		var state colony.ColonyState
		if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
			t.Fatalf("load state after refused seal: %v", err)
		}
		if state.State == colony.StateCOMPLETED {
			t.Fatal("seal completed while legacy blocker truth was unavailable")
		}
	}

	t.Run("malformed file overrides valid legacy fallback", func(t *testing.T) {
		s, tmpDir := setupSealTestStore(t)
		if err := s.SaveJSON("flags.json", colony.FlagsFile{Version: "1", Decisions: []colony.FlagEntry{}}); err != nil {
			t.Fatalf("seed legacy flags: %v", err)
		}
		if err := os.WriteFile(filepath.Join(s.BasePath(), pendingDecisionsFile), []byte("{not-json"), 0o644); err != nil {
			t.Fatalf("seed malformed pending decisions: %v", err)
		}

		for _, mode := range []string{"json", "visual"} {
			_, errOut, err := executeSealAtPublicRoot(t, s, tmpDir, mode)
			requireRenderedSealExitOne(t, err)
			if !strings.Contains(errOut, pendingDecisionsFile) || !strings.Contains(strings.ToLower(errOut), "unmarshal") {
				t.Fatalf("%s seal did not identify pending-decision corruption:\n%s", mode, errOut)
			}
		}

		var state colony.ColonyState
		if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
			t.Fatalf("load state after refused seal: %v", err)
		}
		if state.State == colony.StateCOMPLETED {
			t.Fatal("corrupt pending decisions were hidden by flags.json and seal completed")
		}
	})

	t.Run("directory-backed file", func(t *testing.T) {
		s, tmpDir := setupSealTestStore(t)
		if err := os.Mkdir(filepath.Join(s.BasePath(), pendingDecisionsFile), 0o755); err != nil {
			t.Fatalf("seed directory-backed pending decisions: %v", err)
		}
		_, errOut, err := executeSealAtPublicRoot(t, s, tmpDir, "json")
		requireRenderedSealExitOne(t, err)
		if !strings.Contains(errOut, pendingDecisionsFile) {
			t.Fatalf("directory-backed failure did not identify %s: %s", pendingDecisionsFile, errOut)
		}
	})

	t.Run("missing current with malformed legacy file", func(t *testing.T) {
		s, tmpDir := setupSealTestStore(t)
		if err := os.WriteFile(filepath.Join(s.BasePath(), "flags.json"), []byte("{not-json"), 0o644); err != nil {
			t.Fatalf("seed malformed legacy flags: %v", err)
		}
		assertLegacyStorageRefused(t, s, tmpDir)
	})

	t.Run("missing current with directory-backed legacy file", func(t *testing.T) {
		s, tmpDir := setupSealTestStore(t)
		if err := os.Mkdir(filepath.Join(s.BasePath(), "flags.json"), 0o755); err != nil {
			t.Fatalf("seed directory-backed legacy flags: %v", err)
		}
		assertLegacyStorageRefused(t, s, tmpDir)
	})

	t.Run("missing current with symlink-backed legacy file", func(t *testing.T) {
		s, tmpDir := setupSealTestStore(t)
		target := filepath.Join(tmpDir, "outside-legacy-flags.json")
		if err := os.WriteFile(target, []byte(`{"version":"1","decisions":[]}`), 0o644); err != nil {
			t.Fatalf("seed symlink target: %v", err)
		}
		if err := os.Symlink(target, filepath.Join(s.BasePath(), "flags.json")); err != nil {
			t.Fatalf("seed symlink-backed legacy flags: %v", err)
		}
		assertLegacyStorageRefused(t, s, tmpDir)
	})

	t.Run("missing current with unreadable legacy file", func(t *testing.T) {
		s, tmpDir := setupSealTestStore(t)
		legacyPath := filepath.Join(s.BasePath(), "flags.json")
		if err := os.WriteFile(legacyPath, []byte(`{"version":"1","decisions":[]}`), 0o600); err != nil {
			t.Fatalf("seed unreadable legacy flags: %v", err)
		}
		if err := os.Chmod(legacyPath, 0); err != nil {
			t.Fatalf("make legacy flags unreadable: %v", err)
		}
		t.Cleanup(func() { _ = os.Chmod(legacyPath, 0o600) })
		assertLegacyStorageRefused(t, s, tmpDir)
		if err := os.Chmod(legacyPath, 0o600); err != nil {
			t.Fatalf("restore legacy flags permissions: %v", err)
		}
	})

	t.Run("capability binding cannot be persisted", func(t *testing.T) {
		saveGlobals(t)
		s, tmpDir := setupSealTestStore(t)
		store = s
		criterion := codexCriterionVerification{
			TaskID: "1.1", Criterion: "The owner experience feels correct", State: criterionStateNeedsOwnerConfirmation,
		}
		if refs, err := materializeRuntimeVerificationCheckpoints(1, []codexCriterionVerification{criterion}, checkpointTestGeneration(t, "seal-unwritable", "seal-unwritable-evidence")); err != nil || len(refs) != 1 {
			t.Fatalf("seed checkpoint: refs=%#v err=%v", refs, err)
		}
		if err := os.Chmod(s.BasePath(), 0o555); err != nil {
			t.Fatalf("make pending-decision directory unwritable: %v", err)
		}
		t.Cleanup(func() { _ = os.Chmod(s.BasePath(), 0o755) })

		_, errOut, err := executeSealAtPublicRoot(t, s, tmpDir, "json")
		if restoreErr := os.Chmod(s.BasePath(), 0o755); restoreErr != nil {
			t.Fatalf("restore pending-decision directory: %v", restoreErr)
		}
		requireRenderedSealExitOne(t, err)
		if !strings.Contains(errOut, pendingDecisionsFile) || !strings.Contains(strings.ToLower(errOut), "durably updated") {
			t.Fatalf("capability-write failure was not surfaced as owner-work durability:\n%s", errOut)
		}
	})
}

func TestSealPendingDecisionAbsenceAllowsLegacyFallback(t *testing.T) {
	s, _ := setupSealTestStore(t)
	legacy := colony.FlagEntry{ID: "legacy-blocker", Type: "blocker", Description: "legacy blocker", Resolved: false}
	if err := s.SaveJSON("flags.json", colony.FlagsFile{Version: "1", Decisions: []colony.FlagEntry{legacy}}); err != nil {
		t.Fatalf("seed legacy flags: %v", err)
	}
	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load state: %v", err)
	}
	blockers, _ := checkSealBlockers(s, state)
	if len(blockers) != 1 || blockers[0].ID != legacy.ID {
		t.Fatalf("absent pending decisions did not allow legacy fallback: %#v", blockers)
	}
}

func TestSealCheckpointCapabilityDeduplicatesAndStaysTransient(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := setupSealTestStore(t)
	store = s
	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load state: %v", err)
	}
	session := "seal-capability-dedup"
	state.SessionID = &session
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("scope seal fixture: %v", err)
	}
	criterion := codexCriterionVerification{
		TaskID: "1.1", Criterion: "The final owner interaction feels correct", State: criterionStateNeedsOwnerConfirmation,
	}
	generationA := checkpointTestGeneration(t, "seal-generation-a", "seal-generation-evidence-a")
	refs, err := materializeRuntimeVerificationCheckpoints(1, []codexCriterionVerification{criterion}, generationA)
	if err != nil || len(refs) != 1 {
		t.Fatalf("materialize checkpoint: refs=%#v err=%v", refs, err)
	}
	replayed, err := materializeRuntimeVerificationCheckpoints(1, []codexCriterionVerification{criterion}, generationA)
	if err != nil || len(replayed) != 1 || replayed[0].ID != refs[0].ID || len(loadCheckpointDecisions(t)) != 1 {
		t.Fatalf("exact generation replay was not deduplicated: first=%#v replay=%#v err=%v", refs, replayed, err)
	}
	initialCapability := checkpointCapabilityFromReference(t, refs[0])
	report := codexContinueVerificationReport{
		Phase: 1, GeneratedAt: time.Now().UTC().Format(time.RFC3339), Criteria: []codexCriterionVerification{criterion},
	}
	if err := s.SaveJSON(continuePlanArtifactsPath(1, "verification.json"), report); err != nil {
		t.Fatalf("seed legacy verification projection: %v", err)
	}

	decision := loadCheckpointDecisions(t)[0]
	compatibilityID := stableAutopilotCheckpointID(decision.CheckpointCompatibilityKey)
	legacy := ownerConfirmationSealBlockers(state)
	if len(legacy) != 1 || legacy[0].ID != compatibilityID || legacy[0].ID == refs[0].ID {
		t.Fatalf("legacy projection identity = %#v, want compatibility ID %s distinct from durable row %s", legacy, compatibilityID, refs[0].ID)
	}
	blockers, _ := checkSealBlockers(s, state)
	if len(blockers) != 1 || blockers[0].ID != refs[0].ID {
		t.Fatalf("durable and legacy checkpoint projections were not deduplicated by identity: %#v", blockers)
	}
	if got := checkpointCapabilitiesInSealOutput(blockers[0].RecoveryCommand); len(got) != 1 || got[0] == initialCapability {
		t.Fatalf("deduplicated blocker command has no fresh capability: %#v", blockers[0])
	}

	_, jsonErr, jsonExecErr := executeSealAtPublicRoot(t, s, tmpDir, "json")
	requireRenderedSealExitOne(t, jsonExecErr)
	jsonCaps := checkpointCapabilitiesInSealOutput(jsonErr)
	if len(jsonCaps) != 1 {
		t.Fatalf("JSON seal rendered %d capability commands, want exactly one:\n%s", len(jsonCaps), jsonErr)
	}
	if !strings.Contains(jsonErr, refs[0].ID) || !strings.Contains(jsonErr, `"ok":false`) {
		t.Fatalf("JSON refusal omitted checkpoint identity or error envelope:\n%s", jsonErr)
	}

	visualOut, visualErr, visualExecErr := executeSealAtPublicRoot(t, s, tmpDir, "visual")
	requireRenderedSealExitOne(t, visualExecErr)
	if visualOut != "" {
		t.Fatalf("visual seal refusal leaked to stdout:\n%s", visualOut)
	}
	visualCaps := checkpointCapabilitiesInSealOutput(visualErr)
	if len(visualCaps) != 1 {
		t.Fatalf("visual seal rendered %d capability commands, want exactly one:\n%s", len(visualCaps), visualErr)
	}
	if visualCaps[0] == jsonCaps[0] {
		t.Fatal("visual seal reused the JSON invocation's raw capability")
	}

	pendingRaw := pendingDecisionBytes(t)
	eventRaw, err := os.ReadFile(filepath.Join(s.BasePath(), "event-bus.jsonl"))
	if err != nil {
		t.Fatalf("read lifecycle events: %v", err)
	}
	for _, capability := range []string{initialCapability, jsonCaps[0], visualCaps[0]} {
		if bytes.Contains(pendingRaw, []byte(capability)) || bytes.Contains(eventRaw, []byte(capability)) {
			t.Fatalf("raw capability %q reached a durable sink", capability)
		}
		digest := sha256.Sum256([]byte(capability))
		if !bytes.Contains(pendingRaw, []byte(hex.EncodeToString(digest[:]))) {
			t.Fatalf("pending decisions omitted SHA-256 binding for emitted capability %q", capability)
		}
	}
	if bytes.Contains(eventRaw, []byte("--checkpoint-capability")) {
		t.Fatalf("lifecycle telemetry persisted the raw blocker summary:\n%s", eventRaw)
	}
	if !bytes.Contains(eventRaw, []byte(refs[0].ID)) {
		t.Fatalf("lifecycle telemetry omitted safe checkpoint identity %s:\n%s", refs[0].ID, eventRaw)
	}

	generationB := checkpointTestGeneration(t, "seal-generation-b", "seal-generation-evidence-b")
	secondGeneration, err := materializeRuntimeVerificationCheckpoints(1, []codexCriterionVerification{criterion}, generationB)
	if err != nil || len(secondGeneration) != 1 || secondGeneration[0].ID == refs[0].ID {
		t.Fatalf("distinct durable generation was conflated: first=%#v second=%#v err=%v", refs, secondGeneration, err)
	}
	blockers, _ = checkSealBlockers(s, state)
	if len(blockers) != 2 {
		t.Fatalf("legacy suppression hid a durable generation or leaked a legacy blocker: %#v", blockers)
	}
	seen := map[string]bool{}
	for _, blocker := range blockers {
		seen[blocker.ID] = true
		if got := checkpointCapabilitiesInSealOutput(blocker.RecoveryCommand); len(got) != 1 {
			t.Fatalf("durable generation %s did not retain one capability command: %#v", blocker.ID, blocker)
		}
	}
	if !seen[refs[0].ID] || !seen[secondGeneration[0].ID] || seen[compatibilityID] {
		t.Fatalf("seal blockers did not preserve both durable row IDs while suppressing compatibility ID %s: %#v", compatibilityID, blockers)
	}
}

// TestSealBlockerCheck verifies that seal blocks when blocker flags exist.
func TestSealBlockerCheck(t *testing.T) {
	s, tmpDir := setupSealTestStore(t)

	// Add a blocker flag
	flags := colony.FlagsFile{
		Version: "1",
		Decisions: []colony.FlagEntry{
			{
				ID:          "blk-001",
				Type:        "blocker",
				Description: "Critical issue blocking seal",
				Resolved:    false,
				CreatedAt:   "2026-04-27",
				Source:      "test",
			},
		},
	}
	if err := s.SaveJSON("pending-decisions.json", flags); err != nil {
		t.Fatal(err)
	}

	_, errOut := runSealCmd(t, s, tmpDir, nil)

	// Should output error containing BLOCKED
	if !strings.Contains(errOut, "BLOCKED") {
		t.Errorf("expected error output to contain 'BLOCKED', got: %s", errOut)
	}
	if !strings.Contains(errOut, "blk-001") {
		t.Errorf("expected error output to contain blocker ID 'blk-001', got: %s", errOut)
	}

	// Verify colony state was NOT mutated (still READY, not COMPLETED)
	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatal(err)
	}
	if state.State == colony.StateCOMPLETED {
		t.Error("seal should not have mutated colony state when blockers exist")
	}
}

// TestSealForceBlockers verifies that seal --force proceeds despite blockers.
func TestSealForceBlockers(t *testing.T) {
	s, tmpDir := setupSealTestStore(t)

	// Add a blocker flag
	flags := colony.FlagsFile{
		Version: "1",
		Decisions: []colony.FlagEntry{
			{
				ID:          "blk-002",
				Type:        "blocker",
				Description: "Critical issue",
				Resolved:    false,
				CreatedAt:   "2026-04-27",
				Source:      "test",
			},
		},
	}
	if err := s.SaveJSON("pending-decisions.json", flags); err != nil {
		t.Fatal(err)
	}

	// A force that overrides something now requires a written reason — the
	// override is recorded, never waved through silently.
	out, _ := runSealCmd(t, s, tmpDir, []string{"--force", "--reason", "issue tracked externally; shipping"})

	// Should contain the warning about overriding
	if !strings.Contains(out, "WARNING: Overriding") {
		t.Errorf("expected stdout to contain override warning, got: %s", out)
	}

	// Verify colony state WAS mutated (COMPLETED)
	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatal(err)
	}
	if state.State != colony.StateCOMPLETED {
		t.Errorf("expected state COMPLETED, got: %s", state.State)
	}
}

// TestSealIssueWarning verifies that seal proceeds with a warning when issues exist but no blockers.
func TestSealIssueWarning(t *testing.T) {
	s, tmpDir := setupSealTestStore(t)

	// Add an issue (not blocker) flag
	flags := colony.FlagsFile{
		Version: "1",
		Decisions: []colony.FlagEntry{
			{
				ID:          "issue-001",
				Type:        "issue",
				Description: "Non-critical issue",
				Resolved:    false,
				CreatedAt:   "2026-04-27",
				Source:      "test",
			},
		},
	}
	if err := s.SaveJSON("pending-decisions.json", flags); err != nil {
		t.Fatal(err)
	}

	out, _ := runSealCmd(t, s, tmpDir, nil)

	// Should contain the NOTE about unresolved issues
	if !strings.Contains(out, "NOTE:") {
		t.Errorf("expected stdout to contain NOTE about issues, got: %s", out)
	}

	// Verify colony state WAS mutated (seal proceeded)
	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatal(err)
	}
	if state.State != colony.StateCOMPLETED {
		t.Errorf("expected state COMPLETED, got: %s", state.State)
	}
}

// TestCheckSealBlockers unit tests the checkSealBlockers helper.
func TestCheckSealBlockers(t *testing.T) {
	s, _ := setupSealTestStore(t)

	// No flags file: should return empty
	blockers, issues := checkSealBlockers(s, colony.ColonyState{})
	if len(blockers) != 0 || len(issues) != 0 {
		t.Errorf("expected empty with no flags file, got %d blockers, %d issues", len(blockers), len(issues))
	}

	// Mixed flags
	flags := colony.FlagsFile{
		Version: "1",
		Decisions: []colony.FlagEntry{
			{ID: "b1", Type: "blocker", Resolved: false},
			{ID: "b2", Type: "blocker", Resolved: true},
			{ID: "i1", Type: "issue", Resolved: false},
			{ID: "n1", Type: "note", Resolved: false},
		},
	}
	_ = s.SaveJSON("pending-decisions.json", flags)

	blockers, issues = checkSealBlockers(s, colony.ColonyState{})
	if len(blockers) != 1 || blockers[0].ID != "b1" {
		t.Errorf("expected 1 unresolved blocker 'b1', got %d: %v", len(blockers), blockers)
	}
	if len(issues) != 1 || issues[0].ID != "i1" {
		t.Errorf("expected 1 unresolved issue 'i1', got %d: %v", len(issues), issues)
	}
}

// TestRenderBlockerSummary verifies the blocker summary table output.
func TestRenderBlockerSummary(t *testing.T) {
	blockers := []colony.FlagEntry{
		{ID: "blk-001", Description: "Critical blocker", Type: "blocker", CreatedAt: "2026-04-27"},
	}
	issues := []colony.FlagEntry{
		{ID: "issue-001", Description: "Non-critical", Type: "issue", CreatedAt: "2026-04-27"},
	}

	out := renderBlockerSummary(blockers, issues)

	if !strings.Contains(out, "blk-001") {
		t.Error("summary should contain blocker ID")
	}
	if !strings.Contains(out, "BLOCKED") {
		t.Error("summary should contain BLOCKED message")
	}
	if !strings.Contains(out, "flag-resolve --id") {
		t.Error("summary should contain resolution hint")
	}
	if !strings.Contains(out, "issue-severity") {
		t.Error("summary should mention issue-severity flags")
	}
}

// TestRenderBlockerSummaryUsesOwnBlockerRecoveryCommand proves WR-01
// (193-REVIEW.md): an owner-confirmation blocker is computed live and never
// written to pending-decisions.json, so the generic "aether flag-resolve
// --id <ID>" line would always fail for it. The summary must print that
// blocker's own RecoveryCommand instead, and must not offer the
// unresolvable flag-resolve invocation for it -- while an ordinary
// persisted-flag blocker (no RecoveryCommand set) keeps the existing
// flag-resolve line unchanged.
func TestRenderBlockerSummaryUsesOwnBlockerRecoveryCommand(t *testing.T) {
	ownerBlocker := colony.FlagEntry{
		ID:              "owner-confirm-1-abc123",
		Description:     `Phase 1: "A human judged this looks right" needs your confirmation.`,
		Type:            "blocker",
		Source:          "owner_confirmation",
		RecoveryCommand: `aether decision-answer --question 'no program check or reviewer could verify it' --answer 'confirmed' --phase 1`,
	}
	persistedBlocker := colony.FlagEntry{
		ID:          "blk-002",
		Description: "A persisted blocker flag",
		Type:        "blocker",
	}

	out := renderBlockerSummary([]colony.FlagEntry{ownerBlocker, persistedBlocker}, nil)

	if !strings.Contains(out, "aether decision-answer") {
		t.Fatalf("summary must offer the owner-confirmation blocker's real recovery command, got:\n%s", out)
	}
	if strings.Contains(out, "flag-resolve --id owner-confirm") {
		t.Fatalf("summary must not offer an unresolvable flag-resolve command for an owner-confirmation blocker, got:\n%s", out)
	}
	if !strings.Contains(out, "aether flag-resolve --id blk-002") {
		t.Fatalf("summary must keep the flag-resolve line for an ordinary persisted-flag blocker, got:\n%s", out)
	}
}

// TestCountResolvedFlags unit tests the countResolvedFlags helper.
func TestCountResolvedFlags(t *testing.T) {
	s, _ := setupSealTestStore(t)

	// No flags file
	count := countResolvedFlags(s)
	if count != 0 {
		t.Errorf("expected 0 with no flags file, got %d", count)
	}

	flags := colony.FlagsFile{
		Version: "1",
		Decisions: []colony.FlagEntry{
			{ID: "b1", Resolved: true},
			{ID: "b2", Resolved: true},
			{ID: "i1", Resolved: false},
		},
	}
	_ = s.SaveJSON("pending-decisions.json", flags)

	count = countResolvedFlags(s)
	if count != 2 {
		t.Errorf("expected 2 resolved, got %d", count)
	}
}

// TestSealBlockerSummaryJSON verifies the error output is valid JSON.
func TestSealBlockerSummaryJSON(t *testing.T) {
	s, _ := setupSealTestStore(t)

	flags := colony.FlagsFile{
		Version: "1",
		Decisions: []colony.FlagEntry{
			{ID: "blk-json", Type: "blocker", Description: "JSON test blocker", Resolved: false, CreatedAt: "2026-04-27", Source: "test"},
		},
	}
	_ = s.SaveJSON("pending-decisions.json", flags)

	// Use JSON output mode (no visual rendering)
	saveGlobals(t)
	resetRootCmd(t)
	store = s
	dataDir := s.BasePath()
	t.Setenv("COLONY_DATA_DIR", dataDir)
	outBuf := &bytes.Buffer{}
	errBuf := &bytes.Buffer{}
	stdout = outBuf
	stderr = errBuf
	os.Setenv("AETHER_OUTPUT_MODE", "json")
	t.Cleanup(func() { os.Unsetenv("AETHER_OUTPUT_MODE") })

	rootCmd.SetArgs([]string{"seal"})
	rootCmd.SetOut(outBuf)
	rootCmd.SetErr(errBuf)
	rootCmd.Execute()

	// stderr should be valid JSON envelope
	var envelope map[string]interface{}
	if err := json.Unmarshal([]byte(strings.TrimSpace(errBuf.String())), &envelope); err != nil {
		t.Fatalf("expected valid JSON error, got: %s", errBuf.String())
	}
	if ok, _ := envelope["ok"].(bool); ok {
		t.Error("expected ok:false in error envelope")
	}

	// State should not be mutated
	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatal(err)
	}
	if state.State == colony.StateCOMPLETED {
		t.Error("seal should not have completed when blockers exist")
	}
}

// TestSealPromoteInstincts verifies that seal promotes high-confidence instincts
// to local QUEEN.md only (not global).
func TestSealPromoteInstincts(t *testing.T) {
	s, tmpDir := setupSealTestStore(t)

	// Add instincts with confidence >= 0.8
	instincts := colony.InstinctsFile{
		Version: "1",
		Instincts: []colony.InstinctEntry{
			{
				ID:         "inst-001",
				Trigger:    "test pattern",
				Action:     "Run go vet ./... before commit; it catches shadowed err returns",
				Domain:     "testing",
				Confidence: 0.9,
				Archived:   false,
			},
			{
				ID:         "inst-002",
				Trigger:    "low confidence pattern",
				Action:     "Maybe do something",
				Domain:     "general",
				Confidence: 0.5,
				Archived:   false,
			},
		},
	}
	if err := s.SaveJSON("instincts.json", instincts); err != nil {
		t.Fatal(err)
	}

	out, _ := runSealCmd(t, s, tmpDir, nil)

	// Should NOT contain SUGGESTION (replaced with hive promotion confirmation/warning)
	if strings.Contains(out, "SUGGESTION:") {
		t.Errorf("stdout should NOT contain 'SUGGESTION:' after hive wiring, got: %s", out)
	}

	// Verify local QUEEN.md has the promoted instinct
	queenPath := filepath.Join(tmpDir, ".aether", "QUEEN.md")
	queenData, err := os.ReadFile(queenPath)
	if err != nil {
		t.Fatal(err)
	}
	queenText := string(queenData)
	if !strings.Contains(queenText, "inst-001") {
		t.Error("local QUEEN.md should contain promoted instinct inst-001")
	}
	if !strings.Contains(queenText, "Run go vet ./... before commit; it catches shadowed err returns") {
		t.Error("local QUEEN.md should contain the instinct action text")
	}
	// Low-confidence instinct should NOT be promoted
	if strings.Contains(queenText, "inst-002") {
		t.Error("local QUEEN.md should NOT contain low-confidence instinct inst-002")
	}
}

// TestSealHiveEligibleLog verifies that seal attempts hive promotion for eligible instincts.
func TestSealHiveEligibleLog(t *testing.T) {
	s, tmpDir := setupSealTestStore(t)

	// Set up temp hive directory so promotion succeeds
	hubDir := filepath.Join(tmpDir, ".hub") // runSealCmd pins AETHER_HUB_DIR here
	hiveDir := filepath.Join(hubDir, "hive")
	if err := os.MkdirAll(hiveDir, 0755); err != nil {
		t.Fatal(err)
	}
	wisdomPath := filepath.Join(hiveDir, "wisdom.json")
	if err := os.WriteFile(wisdomPath, []byte(`{"entries":[]}
`), 0644); err != nil {
		t.Fatal(err)
	}

	// Add instincts with confidence >= 0.8
	instincts := colony.InstinctsFile{
		Version: "1",
		Instincts: []colony.InstinctEntry{
			{ID: "hive-1", Trigger: "t1", Action: "Run go test ./cmd before merging colony state changes", Domain: "d1", Confidence: 0.85, Archived: false},
			{ID: "hive-2", Trigger: "t2", Action: "Guard nil pointers in cmd/hive.go before dereference", Domain: "d2", Confidence: 0.95, Archived: false},
		},
	}
	_ = s.SaveJSON("instincts.json", instincts)

	out, _ := runSealCmd(t, s, tmpDir, nil)

	// Should NOT contain SUGGESTION (replaced with actual promotion)
	if strings.Contains(out, "SUGGESTION:") {
		t.Error("stdout should NOT contain 'SUGGESTION:' after hive wiring")
	}
	if !strings.Contains(out, "Promoted") || !strings.Contains(out, "Hive Brain") {
		t.Errorf("expected confirmation about Hive Brain promotion, got: %s", out)
	}
}

// TestSealExpireFocus verifies that seal expires all FOCUS pheromones
// while preserving REDIRECT pheromones.
func TestSealExpireFocus(t *testing.T) {
	s, tmpDir := setupSealTestStore(t)

	now := time.Now().UTC().Format(time.RFC3339)
	pheromones := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "focus-1", Type: "FOCUS", Content: json.RawMessage(`"pay attention here"`), Active: true, CreatedAt: now},
			{ID: "focus-2", Type: "FOCUS", Content: json.RawMessage(`"another focus"`), Active: true, CreatedAt: now},
			{ID: "redirect-1", Type: "REDIRECT", Content: json.RawMessage(`"never do this"`), Active: true, CreatedAt: now},
			{ID: "feedback-1", Type: "FEEDBACK", Content: json.RawMessage(`"adjust this"`), Active: true, CreatedAt: now},
			{ID: "focus-3", Type: "FOCUS", Content: json.RawMessage(`"expired focus"`), Active: false, CreatedAt: now, ExpiresAt: &now},
		},
	}
	if err := s.SaveJSON("pheromones.json", pheromones); err != nil {
		t.Fatal(err)
	}

	runSealCmd(t, s, tmpDir, nil)

	// Verify FOCUS signals are expired
	var pf colony.PheromoneFile
	if err := s.LoadJSON("pheromones.json", &pf); err != nil {
		t.Fatal(err)
	}
	for _, sig := range pf.Signals {
		switch sig.ID {
		case "focus-1", "focus-2":
			if sig.Active {
				t.Errorf("FOCUS signal %s should be expired after seal", sig.ID)
			}
		case "focus-3":
			if sig.Active {
				t.Error("already-expired FOCUS signal should remain expired")
			}
		case "redirect-1":
			if !sig.Active {
				t.Error("REDIRECT signal should be preserved after seal")
			}
		case "feedback-1":
			if !sig.Active {
				t.Error("FEEDBACK signal should be preserved after seal")
			}
		}
	}
}

// TestCrownedAnthillEnrichment verifies that CROWNED-ANTHILL.md contains
// the Colony Statistics table with all 5 metrics.
func TestCrownedAnthillEnrichment(t *testing.T) {
	s, tmpDir := setupSealTestStore(t)

	// Add some learnings
	state := colony.ColonyState{}
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatal(err)
	}
	state.Memory.PhaseLearnings = []colony.PhaseLearning{
		{Phase: 1, Learnings: []colony.Learning{{Claim: "Learned something useful"}}},
		{Phase: 1, Learnings: []colony.Learning{{Claim: "Another learning"}}},
	}
	if err := s.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}

	// Add resolved flags
	flags := colony.FlagsFile{
		Version: "1",
		Decisions: []colony.FlagEntry{
			{ID: "r1", Resolved: true},
			{ID: "r2", Resolved: true},
		},
	}
	_ = s.SaveJSON("pending-decisions.json", flags)

	runSealCmd(t, s, tmpDir, nil)

	// Read CROWNED-ANTHILL.md
	anthillPath := filepath.Join(tmpDir, ".aether", "CROWNED-ANTHILL.md")
	data, err := os.ReadFile(anthillPath)
	if err != nil {
		t.Fatalf("CROWNED-ANTHILL.md not found: %v", err)
	}
	content := string(data)

	// Verify Colony Statistics table
	if !strings.Contains(content, "## Colony Statistics") {
		t.Error("CROWNED-ANTHILL.md should contain '## Colony Statistics' section")
	}
	if !strings.Contains(content, "| Learnings captured | 2 |") {
		t.Error("CROWNED-ANTHILL.md should show 2 learnings captured")
	}
	// 3, not 2: the seal confirmation gate's own recorded "yes" answer
	// (198-03, recorded by autoRecordSealConfirmationForTest the same way
	// an owner's real answer would be) is itself a resolved decision in
	// the same pending-decisions.json store, and countResolvedFlags counts
	// any resolved decision there regardless of type -- an accurate count
	// of what a real seal run now leaves resolved, not a test artifact.
	if !strings.Contains(content, "| Flags resolved | 3 |") {
		t.Error("CROWNED-ANTHILL.md should show 3 flags resolved (2 fixture flags + the recorded seal confirmation)")
	}
	if !strings.Contains(content, "| FOCUS signals expired | 0 |") {
		t.Error("CROWNED-ANTHILL.md should show FOCUS signals expired metric")
	}
	if !strings.Contains(content, "| Hive-eligible instincts | 0 |") {
		t.Error("CROWNED-ANTHILL.md should show hive-eligible instincts metric")
	}
	if !strings.Contains(content, "| Instincts promoted | 0 |") {
		t.Error("CROWNED-ANTHILL.md should show instincts promoted metric")
	}

	// Verify Signal Cleanup section
	if !strings.Contains(content, "### Signal Cleanup") {
		t.Error("CROWNED-ANTHILL.md should contain '### Signal Cleanup' section")
	}
	if !strings.Contains(content, "REDIRECT signals preserved") {
		t.Error("CROWNED-ANTHILL.md should mention REDIRECT signals preserved")
	}
}

// TestExpireSignalsByType unit tests the expireSignalsByType helper.
func TestExpireSignalsByType(t *testing.T) {
	s, _ := setupSealTestStore(t)

	now := time.Now().UTC().Format(time.RFC3339)
	pf := colony.PheromoneFile{
		Signals: []colony.PheromoneSignal{
			{ID: "f1", Type: "FOCUS", Active: true, CreatedAt: now},
			{ID: "f2", Type: "FOCUS", Active: true, CreatedAt: now},
			{ID: "r1", Type: "REDIRECT", Active: true, CreatedAt: now},
			{ID: "f3", Type: "FOCUS", Active: false, CreatedAt: now, ExpiresAt: &now},
		},
	}
	_ = s.SaveJSON("pheromones.json", pf)

	// Expire FOCUS signals
	count, _ := expireSignalsByType(s, "FOCUS")
	if count != 2 {
		t.Errorf("expected 2 FOCUS signals expired, got %d", count)
	}

	// Verify REDIRECT still active
	var loaded colony.PheromoneFile
	_ = s.LoadJSON("pheromones.json", &loaded)
	for _, sig := range loaded.Signals {
		if sig.ID == "r1" && !sig.Active {
			t.Error("REDIRECT signal should still be active")
		}
	}

	// Expiring again should return 0
	count2, _ := expireSignalsByType(s, "FOCUS")
	if count2 != 0 {
		t.Errorf("expected 0 on second expire, got %d", count2)
	}
}

// TestPromoteInstinctLocal unit tests the promoteInstinctLocal helper.
func TestPromoteInstinctLocal(t *testing.T) {
	s, tmpDir := setupSealTestStore(t)
	saveGlobals(t)
	store = s
	t.Setenv("COLONY_DATA_DIR", s.BasePath())

	err := promoteInstinctLocal(s, "test-inst-1", "Write tests before code")
	if err != nil {
		t.Fatalf("promoteInstinctLocal failed: %v", err)
	}

	queenPath := filepath.Join(tmpDir, ".aether", "QUEEN.md")
	data, err := os.ReadFile(queenPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if !strings.Contains(content, "test-inst-1") {
		t.Error("QUEEN.md should contain instinct ID")
	}
	if !strings.Contains(content, "Write tests before code") {
		t.Error("QUEEN.md should contain instinct action")
	}
	if !strings.Contains(content, "## Wisdom") {
		t.Error("Entry should be in Wisdom section")
	}
}

// TestBuildSealSummaryEnrichment unit tests the enriched buildSealSummary.
func TestBuildSealSummaryEnrichment(t *testing.T) {
	goal := "Test goal"
	state := colony.ColonyState{
		Goal:         &goal,
		CurrentPhase: 3,
		Plan: colony.Plan{
			Phases: []colony.Phase{
				{ID: 1, Name: "P1", Status: colony.PhaseCompleted},
				{ID: 2, Name: "P2", Status: colony.PhaseCompleted},
				{ID: 3, Name: "P3", Status: colony.PhaseCompleted},
			},
		},
	}

	enrichment := sealEnrichment{
		LearningsCount:    5,
		InstinctsPromoted: []string{"inst-1", "inst-2"},
		HiveEligible:      3,
		SignalsExpired:    4,
		FlagsResolved:     2,
	}

	summary := buildSealSummary(state, "2026-04-27T12:00:00Z", nil, enrichment)

	if !strings.Contains(summary, "## Colony Statistics") {
		t.Error("summary should contain Colony Statistics section")
	}
	if !strings.Contains(summary, "| Learnings captured | 5 |") {
		t.Error("summary should show 5 learnings")
	}
	if !strings.Contains(summary, "| Instincts promoted | 2 |") {
		t.Error("summary should show 2 promoted instincts")
	}
	if !strings.Contains(summary, "### Promoted Instincts") {
		t.Error("summary should contain Promoted Instincts section")
	}
	if !strings.Contains(summary, "- inst-1") {
		t.Error("summary should list inst-1 in promoted instincts")
	}
	if !strings.Contains(summary, "### Signal Cleanup") {
		t.Error("summary should contain Signal Cleanup section")
	}
	if !strings.Contains(summary, "FOCUS signals expired: 4") {
		t.Error("summary should show 4 expired FOCUS signals")
	}
}

// TestSealHivePromote verifies that seal promotes high-confidence instincts to Hive Brain
// and does NOT promote low-confidence instincts. Verifies the SUGGESTION message is replaced
// with a confirmation message.
func TestSealHivePromote(t *testing.T) {
	s, tmpDir := setupSealTestStore(t)

	// Set up temp hive directory
	hubDir := filepath.Join(tmpDir, ".hub") // runSealCmd pins AETHER_HUB_DIR here
	hiveDir := filepath.Join(hubDir, "hive")
	if err := os.MkdirAll(hiveDir, 0755); err != nil {
		t.Fatal(err)
	}
	wisdomPath := filepath.Join(hiveDir, "wisdom.json")
	if err := os.WriteFile(wisdomPath, []byte(`{"entries":[]}
`), 0644); err != nil {
		t.Fatal(err)
	}

	// Add instincts with mixed confidence
	instincts := colony.InstinctsFile{
		Version: "1",
		Instincts: []colony.InstinctEntry{
			{
				ID:         "hive-high",
				Trigger:    "test pattern",
				Action:     "Run go vet ./... before commit; it catches shadowed err returns",
				Domain:     "testing",
				Confidence: 0.9,
				Archived:   false,
			},
			{
				ID:         "hive-low",
				Trigger:    "low pattern",
				Action:     "Maybe do something",
				Domain:     "general",
				Confidence: 0.5,
				Archived:   false,
			},
		},
	}
	if err := s.SaveJSON("instincts.json", instincts); err != nil {
		t.Fatal(err)
	}

	out, _ := runSealCmd(t, s, tmpDir, nil)

	// Should contain confirmation (not SUGGESTION)
	if strings.Contains(out, "SUGGESTION:") {
		t.Errorf("stdout should NOT contain 'SUGGESTION:' after hive wiring, got: %s", out)
	}
	if !strings.Contains(out, "Promoted") || !strings.Contains(out, "to Hive Brain") {
		t.Errorf("expected confirmation about Hive Brain promotion, got: %s", out)
	}

	// Verify hive wisdom file contains the high-confidence instinct
	wisdomData, err := os.ReadFile(wisdomPath)
	if err != nil {
		t.Fatalf("failed to read wisdom.json: %v", err)
	}
	var wf hiveWisdomData
	if err := json.Unmarshal(wisdomData, &wf); err != nil {
		t.Fatalf("failed to parse wisdom.json: %v", err)
	}

	found := false
	for _, e := range wf.Entries {
		if strings.Contains(e.Text, "Run go vet ./... before commit; it catches shadowed err returns") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("hive wisdom should contain the high-confidence instinct, entries: %v", wf.Entries)
	}

	// Verify low-confidence instinct was NOT promoted to hive
	for _, e := range wf.Entries {
		if strings.Contains(e.Text, "Maybe do something") {
			t.Error("low-confidence instinct should NOT be promoted to hive")
		}
	}
}

// TestSealHivePromoteNonBlocking verifies that seal completes successfully
// even when hive promotion fails (e.g., read-only directory).
func TestSealHivePromoteNonBlocking(t *testing.T) {
	s, tmpDir := setupSealTestStore(t)

	// Create a read-only parent directory to force hive write failure
	roDir := filepath.Join(t.TempDir(), "readonly")
	if err := os.MkdirAll(roDir, 0755); err != nil {
		t.Fatal(err)
	}
	// Make the directory read-only (no write permission)
	if err := os.Chmod(roDir, 0555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(roDir, 0755) })

	// Add a high-confidence instinct
	instincts := colony.InstinctsFile{
		Version: "1",
		Instincts: []colony.InstinctEntry{
			{ID: "hive-fail", Trigger: "t", Action: "This should fail to promote", Domain: "testing", Confidence: 0.9, Archived: false},
		},
	}
	if err := s.SaveJSON("instincts.json", instincts); err != nil {
		t.Fatal(err)
	}

	origHub := os.Getenv("AETHER_HUB_DIR")
	os.Setenv("AETHER_HUB_DIR", roDir)
	defer os.Setenv("AETHER_HUB_DIR", origHub)

	out, _ := runSealCmd(t, s, tmpDir, nil)

	// Seal should still complete
	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatal(err)
	}
	if state.State != colony.StateCOMPLETED {
		t.Errorf("expected state COMPLETED despite hive failure, got: %s", state.State)
	}

	// Should contain warning about hive failure (not BLOCKED)
	if strings.Contains(out, "BLOCKED") {
		t.Errorf("seal should not be BLOCKED by hive failure, got: %s", out)
	}
	if !strings.Contains(out, "WARNING:") || !strings.Contains(out, "hive") {
		t.Errorf("expected warning about hive promotion failure, got: %s", out)
	}
}

// TestSealHivePromotedCount verifies that CROWNED-ANTHILL.md contains
// the hive-promoted instincts count in the Colony Statistics table.
func TestSealHivePromotedCount(t *testing.T) {
	s, tmpDir := setupSealTestStore(t)

	// Set up temp hive directory
	hubDir := filepath.Join(tmpDir, ".hub") // runSealCmd pins AETHER_HUB_DIR here
	hiveDir := filepath.Join(hubDir, "hive")
	if err := os.MkdirAll(hiveDir, 0755); err != nil {
		t.Fatal(err)
	}
	wisdomPath := filepath.Join(hiveDir, "wisdom.json")
	if err := os.WriteFile(wisdomPath, []byte(`{"entries":[]}
`), 0644); err != nil {
		t.Fatal(err)
	}

	// Add 2 high-confidence instincts
	instincts := colony.InstinctsFile{
		Version: "1",
		Instincts: []colony.InstinctEntry{
			{ID: "hp1", Trigger: "t1", Action: "Check cmd/seal.go archive paths before renaming chambers", Domain: "d1", Confidence: 0.85, Archived: false},
			{ID: "hp2", Trigger: "t2", Action: "Run go build ./cmd before tagging a release", Domain: "d2", Confidence: 0.95, Archived: false},
		},
	}
	if err := s.SaveJSON("instincts.json", instincts); err != nil {
		t.Fatal(err)
	}

	origHub := os.Getenv("AETHER_HUB_DIR")
	os.Setenv("AETHER_HUB_DIR", hubDir)
	defer os.Setenv("AETHER_HUB_DIR", origHub)

	runSealCmd(t, s, tmpDir, nil)

	// Read CROWNED-ANTHILL.md
	anthillPath := filepath.Join(tmpDir, ".aether", "CROWNED-ANTHILL.md")
	data, err := os.ReadFile(anthillPath)
	if err != nil {
		t.Fatalf("CROWNED-ANTHILL.md not found: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "| Hive-promoted instincts | 2 |") {
		t.Errorf("CROWNED-ANTHILL.md should show 2 hive-promoted instincts, got: %s", content)
	}
}

// TestSealDoesNotDoublePromoteInstincts pins D-09: an instinct that is BOTH
// pkg/memory-QueenEligible (confidence >= 0.75 AND >= 3 recorded
// applications) AND above the seal ceremony's own local-promotion bar
// (confidence >= 0.8) must reach .aether/QUEEN.md exactly once, written by
// the authoritative pkg/memory pipeline under "## Instincts" -- never a
// second time under "## Wisdom" by the seal's subordinate promoteInstinctLocal
// loop. Removing Task 2's ID skip-set check must make this test fail.
func TestSealDoesNotDoublePromoteInstincts(t *testing.T) {
	s, tmpDir := setupSealTestStore(t)

	// A real, valid (empty) observations file so runSealConsolidation's
	// pipeline.RunConsolidation actually runs to completion instead of
	// failing on a missing file and leaving QueenEligible empty.
	if err := s.SaveJSON("learning-observations.json", colony.LearningFile{Observations: []colony.Observation{}}); err != nil {
		t.Fatal(err)
	}

	const actionText = "Run go build ./... before every seal to catch broken compilation"
	instincts := colony.InstinctsFile{
		Version: "1",
		Instincts: []colony.InstinctEntry{
			{
				ID:         "inst-both-001",
				Trigger:    "dual eligibility pattern",
				Action:     actionText,
				Domain:     "testing",
				TrustScore: 0.9,
				TrustTier:  "trusted",
				Confidence: 0.9,
				Provenance: colony.InstinctProvenance{ApplicationCount: 3},
				Archived:   false,
			},
		},
	}
	if err := s.SaveJSON("instincts.json", instincts); err != nil {
		t.Fatal(err)
	}

	runSealCmd(t, s, tmpDir, nil)

	queenPath := filepath.Join(tmpDir, ".aether", "QUEEN.md")
	data, err := os.ReadFile(queenPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)

	count := strings.Count(text, actionText)
	if count != 1 {
		t.Fatalf("expected the instinct's action text to appear exactly once in QUEEN.md, got %d occurrences:\n%s", count, text)
	}

	instinctsSectionHasIt := false
	wisdomSectionHasIt := false
	for _, section := range strings.Split(text, "\n## ") {
		if strings.HasPrefix(section, "Instincts") && strings.Contains(section, actionText) {
			instinctsSectionHasIt = true
		}
		if strings.HasPrefix(section, "Wisdom") && strings.Contains(section, actionText) {
			wisdomSectionHasIt = true
		}
	}
	if !instinctsSectionHasIt {
		t.Errorf("expected the action text under '## Instincts' (pkg/memory's authoritative write), got:\n%s", text)
	}
	if wisdomSectionHasIt {
		t.Errorf("action text should NOT also appear under '## Wisdom' (the subordinate write must have been skipped), got:\n%s", text)
	}
}

// TestSealDoesNotDoublePromoteOnCurationFailure pins CR-01's D-09 half on
// the FAILURE path: a corrupt pheromones.json triggers a curation sentinel
// abort while instincts.json is valid -- the asymmetric case where, before
// the short-circuit fix, the mutating pipeline still ran, promoted the
// dual-eligible instinct into "## Instincts", and then the seal's subordinate
// loop (handed an empty skip-set from the Ran:false summary) wrote the SAME
// instinct into "## Wisdom". The action text must appear exactly once in
// QUEEN.md, and the seal must still complete (learning is never a gate).
func TestSealDoesNotDoublePromoteOnCurationFailure(t *testing.T) {
	s, tmpDir := setupSealTestStore(t)

	// Valid observations so the consolidation pipeline itself would run
	// cleanly if (wrongly) invoked despite the sentinel abort.
	if err := s.SaveJSON("learning-observations.json", colony.LearningFile{Observations: []colony.Observation{}}); err != nil {
		t.Fatal(err)
	}

	const actionText = "Run the race detector before sealing colonies with concurrent workers"
	instincts := colony.InstinctsFile{
		Version: "1",
		Instincts: []colony.InstinctEntry{
			{
				ID:         "inst-curfail-001",
				Trigger:    "dual eligibility pattern behind a curation failure",
				Action:     actionText,
				Domain:     "testing",
				TrustScore: 0.9,
				TrustTier:  "trusted",
				Confidence: 0.9,
				Provenance: colony.InstinctProvenance{ApplicationCount: 3},
				Archived:   false,
			},
		},
	}
	if err := s.SaveJSON("instincts.json", instincts); err != nil {
		t.Fatal(err)
	}

	// Corrupt a sentinel-checked store the pipeline does not read.
	dataDir := filepath.Join(tmpDir, ".aether", "data")
	if err := os.WriteFile(filepath.Join(dataDir, "pheromones.json"), []byte("{not valid json"), 0o644); err != nil {
		t.Fatalf("seed invalid pheromones.json: %v", err)
	}

	out, _ := runSealCmd(t, s, tmpDir, nil)

	if !strings.Contains(out, "colony sealed WITHOUT consolidation —") {
		t.Errorf("expected the D-05 loud warning on the curation failure path, got:\n%s", out)
	}

	queenPath := filepath.Join(tmpDir, ".aether", "QUEEN.md")
	data, err := os.ReadFile(queenPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if count := strings.Count(text, actionText); count != 1 {
		t.Fatalf("expected the instinct's action text to appear exactly once in QUEEN.md on the curation-failure path (CR-01/D-09), got %d occurrences:\n%s", count, text)
	}

	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatal(err)
	}
	if state.State != colony.StateCOMPLETED {
		t.Errorf("expected state COMPLETED despite the curation failure, got: %s", state.State)
	}
}

// TestSealStillPromotesInstinctsWithoutApplicationHistory pins D-09's other
// half: an instinct with confidence >= 0.8 but zero recorded applications is
// NOT pkg/memory-QueenEligible (which requires >= 3 applications), so it
// must still reach QUEEN.md via the seal's subordinate promoteInstinctLocal
// loop -- proving Task 2 did not delete that loop, only made it conditional.
// Deleting the subordinate promoteInstinctLocal call must make this test fail.
func TestSealStillPromotesInstinctsWithoutApplicationHistory(t *testing.T) {
	s, tmpDir := setupSealTestStore(t)

	if err := s.SaveJSON("learning-observations.json", colony.LearningFile{Observations: []colony.Observation{}}); err != nil {
		t.Fatal(err)
	}

	const actionText = "Prefer table-driven tests in new Go test files"
	instincts := colony.InstinctsFile{
		Version: "1",
		Instincts: []colony.InstinctEntry{
			{
				ID:         "inst-young-001",
				Trigger:    "fresh pattern with no application history",
				Action:     actionText,
				Domain:     "testing",
				TrustScore: 0.9,
				TrustTier:  "trusted",
				Confidence: 0.9,
				Archived:   false,
			},
		},
	}
	if err := s.SaveJSON("instincts.json", instincts); err != nil {
		t.Fatal(err)
	}

	runSealCmd(t, s, tmpDir, nil)

	queenPath := filepath.Join(tmpDir, ".aether", "QUEEN.md")
	data, err := os.ReadFile(queenPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, actionText) {
		t.Fatalf("expected the young instinct (no application history) to still reach QUEEN.md via the subordinate seal-side promotion loop, got:\n%s", text)
	}
}

// TestSealReportsPipelinePromotedInstinctsBelowLocalBar pins WR-03: an
// instinct that pkg/memory's pipeline promotes into QUEEN.md (QueenEligible:
// post-decay confidence >= 0.75 with >= 3 applications) but whose snapshot
// confidence sits BELOW the seal loop's own 0.8 bar must still appear in
// CROWNED-ANTHILL.md's promoted set. Before the reconciliation, the seal
// report claimed to carry "the full promoted set" while silently omitting
// every pipeline promotion in the [0.75, 0.8) band.
func TestSealReportsPipelinePromotedInstinctsBelowLocalBar(t *testing.T) {
	s, tmpDir := setupSealTestStore(t)

	if err := s.SaveJSON("learning-observations.json", colony.LearningFile{Observations: []colony.Observation{}}); err != nil {
		t.Fatal(err)
	}

	// Snapshot confidence 0.78: below the seal loop's 0.8 bar, but with 3
	// recorded applications the post-decay confidence stays >= 0.75, so the
	// pipeline promotes it into QUEEN.md's "## Instincts" section.
	instincts := colony.InstinctsFile{
		Version: "1",
		Instincts: []colony.InstinctEntry{
			{
				ID:         "inst-mid-001",
				Trigger:    "pattern promoted by the pipeline below the seal bar",
				Action:     "Report every QUEEN.md promotion in CROWNED-ANTHILL.md",
				Domain:     "testing",
				TrustScore: 0.9,
				TrustTier:  "trusted",
				Confidence: 0.78,
				Provenance: colony.InstinctProvenance{ApplicationCount: 3},
				Archived:   false,
			},
		},
	}
	if err := s.SaveJSON("instincts.json", instincts); err != nil {
		t.Fatal(err)
	}

	runSealCmd(t, s, tmpDir, nil)

	// Precondition: the pipeline really did promote it into QUEEN.md.
	queenData, err := os.ReadFile(filepath.Join(tmpDir, ".aether", "QUEEN.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(queenData), "Report every QUEEN.md promotion in CROWNED-ANTHILL.md") {
		t.Fatalf("precondition: expected the pipeline to promote the instinct into QUEEN.md, got:\n%s", string(queenData))
	}

	anthillPath := filepath.Join(tmpDir, ".aether", "CROWNED-ANTHILL.md")
	data, err := os.ReadFile(anthillPath)
	if err != nil {
		t.Fatalf("CROWNED-ANTHILL.md not found: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "| Instincts promoted | 1 |") {
		t.Errorf("CROWNED-ANTHILL.md should count the pipeline-promoted instinct (WR-03), got:\n%s", content)
	}
	if !strings.Contains(content, "- inst-mid-001") {
		t.Errorf("CROWNED-ANTHILL.md's Promoted Instincts section should list inst-mid-001 (WR-03), got:\n%s", content)
	}
}

// sealNamedAntLabels are the eight curation ant labels (D-06) that must
// appear verbatim in seal stdout once Task 3's rendering is wired in.
var sealNamedAntLabels = []string{"Sentinel", "Nurse", "Critic", "Herald", "Janitor", "Archivist", "Librarian", "Scribe"}

// TestSealRendersEightNamedAnts asserts seal stdout contains all eight
// distinct curation ant labels and that no line falls back to the generic
// "🐜 Ant" identity casteLabel/casteEmoji return for an unmapped caste.
func TestSealRendersEightNamedAnts(t *testing.T) {
	s, tmpDir := setupSealTestStore(t)

	if err := s.SaveJSON("learning-observations.json", colony.LearningFile{Observations: []colony.Observation{}}); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveJSON("instincts.json", colony.InstinctsFile{Version: "1", Instincts: []colony.InstinctEntry{}}); err != nil {
		t.Fatal(err)
	}

	out, _ := runSealCmd(t, s, tmpDir, nil)

	for _, label := range sealNamedAntLabels {
		if !strings.Contains(out, label) {
			t.Errorf("expected seal stdout to contain ant label %q, got:\n%s", label, out)
		}
	}
	if strings.Contains(out, "🐜 Ant") {
		t.Errorf("seal stdout contains the generic fallback '🐜 Ant' identity; a curation ant caste is unmapped:\n%s", out)
	}
}

// TestSealRendersReportPath asserts seal stdout and CROWNED-ANTHILL.md both
// name CURATION-REPORT.md, and that the file exists at that path after seal.
func TestSealRendersReportPath(t *testing.T) {
	s, tmpDir := setupSealTestStore(t)

	if err := s.SaveJSON("learning-observations.json", colony.LearningFile{Observations: []colony.Observation{}}); err != nil {
		t.Fatal(err)
	}
	if err := s.SaveJSON("instincts.json", colony.InstinctsFile{Version: "1", Instincts: []colony.InstinctEntry{}}); err != nil {
		t.Fatal(err)
	}

	out, _ := runSealCmd(t, s, tmpDir, nil)

	if !strings.Contains(out, "CURATION-REPORT.md") {
		t.Errorf("expected seal stdout to name CURATION-REPORT.md, got:\n%s", out)
	}

	anthillPath := filepath.Join(tmpDir, ".aether", "CROWNED-ANTHILL.md")
	data, err := os.ReadFile(anthillPath)
	if err != nil {
		t.Fatalf("CROWNED-ANTHILL.md not found: %v", err)
	}
	if !strings.Contains(string(data), "CURATION-REPORT.md") {
		t.Errorf("expected CROWNED-ANTHILL.md to name CURATION-REPORT.md, got:\n%s", string(data))
	}

	reportPath := filepath.Join(tmpDir, ".aether", "CURATION-REPORT.md")
	if _, err := os.Stat(reportPath); err != nil {
		t.Fatalf("CURATION-REPORT.md does not exist at %s: %v", reportPath, err)
	}
}

// TestSealRendersLoudFailure asserts that when consolidation fails, stdout
// contains the D-05 loud warning and the seal still reaches
// colony.StateCOMPLETED -- a learning failure never blocks a seal.
func TestSealRendersLoudFailure(t *testing.T) {
	s, tmpDir := setupSealTestStore(t)

	dataDir := filepath.Join(tmpDir, ".aether", "data")
	if err := os.WriteFile(filepath.Join(dataDir, "instincts.json"), []byte("{not valid json"), 0o644); err != nil {
		t.Fatalf("seed invalid instincts.json: %v", err)
	}

	out, _ := runSealCmd(t, s, tmpDir, nil)

	if !strings.Contains(out, "colony sealed WITHOUT consolidation —") {
		t.Errorf("expected seal stdout to contain the D-05 loud warning, got:\n%s", out)
	}

	var state colony.ColonyState
	if err := s.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatal(err)
	}
	if state.State != colony.StateCOMPLETED {
		t.Errorf("expected state COMPLETED despite consolidation failure, got: %s", state.State)
	}
}
