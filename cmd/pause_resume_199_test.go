package cmd

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

type pauseResume199Fixture struct {
	root    string
	dataDir string
	now     time.Time
}

func newPauseResume199Fixture(t *testing.T) pauseResume199Fixture {
	t.Helper()
	saveGlobals(t)
	pauseResumeLifecycleFault = nil
	t.Cleanup(func() { pauseResumeLifecycleFault = nil })
	root, dataDir := crashSafetyFixture(t)
	now := time.Date(2026, time.September, 4, 9, 0, 0, 0, time.UTC)
	goal := "Preserve one trustworthy recovery point"
	sessionID := "session-199-13"
	runID := "run-199-13"
	taskDone := "task-done"
	taskActive := "task-active"
	state := colony.ColonyState{
		Version:      "1",
		Goal:         &goal,
		State:        colony.StateREADY,
		CurrentPhase: 1,
		SessionID:    &sessionID,
		RunID:        &runID,
		Plan: colony.Plan{Phases: []colony.Phase{{
			ID:     1,
			Name:   "front door",
			Status: colony.PhaseInProgress,
			Tasks: []colony.Task{
				{ID: &taskDone, Goal: "finished work", Status: colony.TaskCompleted},
				{ID: &taskActive, Goal: "partial work", Status: colony.TaskInProgress},
			},
		}}},
		Memory: colony.Memory{Decisions: []colony.Decision{{
			ID:        "memory-decision-1",
			Claim:     "Resume from the partial task",
			Rationale: "Completed work must not run twice",
			Phase:     1,
		}}},
		GateResults: []colony.GateResultEntry{{
			Name: "tests", Passed: true, Timestamp: now.Add(-time.Minute).Format(time.RFC3339), Detail: "focused checks passed",
		}},
		Worktrees: []colony.WorktreeEntry{{
			ID: "worker-tree-1", Branch: "worktree-agent-1", Path: ".aether/worktrees/worker-tree-1",
			Status: colony.WorktreeInProgress, Phase: 1, Agent: "builder-1",
			CreatedAt: now.Add(-time.Hour).Format(time.RFC3339), UpdatedAt: now.Add(-time.Minute).Format(time.RFC3339),
		}},
	}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save state: %v", err)
	}
	session := colony.SessionFile{
		SessionID: sessionID, StartedAt: now.Add(-time.Hour).Format(time.RFC3339),
		LastCommand: "build", LastCommandAt: now.Add(-time.Minute).Format(time.RFC3339),
		ColonyGoal: goal, CurrentPhase: 1, SuggestedNext: "aether continue",
		Summary: "Builder completed task-done and is partway through task-active.",
	}
	if err := store.SaveJSON("session.json", session); err != nil {
		t.Fatalf("save session: %v", err)
	}
	write199File(t, filepath.Join(root, ".aether", "CONTEXT.md"), "# Context\n\nKeep the partial implementation and its proof.\n")
	write199File(t, filepath.Join(dataDir, "spawn-tree.txt"), strings.Join([]string{
		now.Add(-2 * time.Minute).Format(time.RFC3339), "queen", "builder", "builder-1", "task-active", "1", "working",
	}, "|")+"\n")
	strength := 1.0
	signalContent := json.RawMessage(`{"text":"preserve recovery evidence"}`)
	if err := store.SaveJSON("pheromones.json", colony.PheromoneFile{Signals: []colony.PheromoneSignal{{
		ID: "signal-1", Type: "FOCUS", Priority: "normal", Source: "user",
		CreatedAt: now.Add(-time.Minute).Format(time.RFC3339), Active: true, Strength: &strength, Content: signalContent,
	}}}); err != nil {
		t.Fatalf("save signals: %v", err)
	}
	phase := 1
	if err := store.SaveJSON("pending-decisions.json", colony.FlagsFile{Version: "1", Decisions: []colony.FlagEntry{{
		ID: "decision-1", Type: "decision", Description: "Keep the completed task", Phase: &phase,
		Source: "owner", CreatedAt: now.Add(-time.Minute).Format(time.RFC3339),
	}}}); err != nil {
		t.Fatalf("save decisions: %v", err)
	}
	return pauseResume199Fixture{root: root, dataDir: dataDir, now: now}
}

func write199File(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func read199File(t *testing.T, path string) []byte {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return content
}

func load199Handoff(t *testing.T, dataDir string) colony.PauseHandoff {
	t.Helper()
	var handoff colony.PauseHandoff
	content := read199File(t, filepath.Join(dataDir, pauseHandoffDataPath))
	if err := json.Unmarshal(content, &handoff); err != nil {
		t.Fatalf("parse pause handoff: %v", err)
	}
	if err := handoff.Validate(); err != nil {
		t.Fatalf("validate pause handoff: %v", err)
	}
	return handoff
}

func load199State(t *testing.T) colony.ColonyState {
	t.Helper()
	var state colony.ColonyState
	if err := store.LoadJSON("COLONY_STATE.json", &state); err != nil {
		t.Fatalf("load state: %v", err)
	}
	return state
}

func load199Session(t *testing.T) colony.SessionFile {
	t.Helper()
	var session colony.SessionFile
	if err := store.LoadJSON("session.json", &session); err != nil {
		t.Fatalf("load session: %v", err)
	}
	return session
}

func transactionReceiptCount199(t *testing.T, dataDir string) int {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(dataDir, "transactions", "*", "receipt.json"))
	if err != nil {
		t.Fatalf("glob receipts: %v", err)
	}
	return len(matches)
}

func runnableFingerprint199(t *testing.T, root, dataDir string) string {
	t.Helper()
	paths := []string{
		filepath.Join(dataDir, "COLONY_STATE.json"),
		filepath.Join(dataDir, "session.json"),
		filepath.Join(dataDir, pauseHandoffDataPath),
		filepath.Join(dataDir, "spawn-tree.txt"),
		filepath.Join(root, ".aether", "CONTEXT.md"),
		filepath.Join(root, ".aether", "HANDOFF.md"),
	}
	sort.Strings(paths)
	var joined bytes.Buffer
	for _, path := range paths {
		joined.WriteString(path)
		joined.WriteByte('\n')
		content, err := os.ReadFile(path)
		if os.IsNotExist(err) {
			joined.WriteString("<missing>\n")
			continue
		}
		if err != nil {
			t.Fatalf("fingerprint %s: %v", path, err)
		}
		joined.Write(content)
		joined.WriteByte('\n')
	}
	return lifecycleDigest(joined.Bytes())
}

func TestPauseResume199SafeBoundary(t *testing.T) {
	faults := []string{
		"after_validation",
		"after_stage:target-0001",
		"after_intent",
		"after_target_commit:target-0001",
		"after_global_verification",
	}
	for _, command := range []string{"pause", "resume"} {
		for _, point := range faults {
			t.Run(command+"/"+strings.ReplaceAll(point, ":", "_"), func(t *testing.T) {
				fixture := newPauseResume199Fixture(t)
				if command == "resume" {
					if _, err := pauseColonyAt(fixture.now); err != nil {
						t.Fatalf("prepare pause: %v", err)
					}
				}
				before := runnableFingerprint199(t, fixture.root, fixture.dataDir)
				pauseResumeLifecycleFault = func(seen string) error {
					if seen == point {
						return errors.New("injected crash")
					}
					return nil
				}
				var err error
				if command == "pause" {
					_, err = pauseColonyAt(fixture.now)
				} else {
					_, err = resumeColonyAt(fixture.now.Add(time.Minute))
				}
				if err == nil {
					t.Fatalf("expected injected failure at %s", point)
				}
				interrupted := runnableFingerprint199(t, fixture.root, fixture.dataDir)
				if point == "after_validation" || strings.HasPrefix(point, "after_stage:") || point == "after_intent" {
					if interrupted != before {
						t.Fatalf("%s changed runnable bytes before any target commit at %s: before=%s after=%s", command, point, before, interrupted)
					}
				}
				pauseResumeLifecycleFault = nil
				var outcome pauseResumeLifecycleOutcome
				if command == "pause" {
					outcome, err = pauseColonyAt(fixture.now.Add(2 * time.Minute))
				} else {
					outcome, err = resumeColonyAt(fixture.now.Add(2 * time.Minute))
				}
				if err != nil {
					t.Fatalf("recover %s transaction after %s: %v", command, point, err)
				}
				if outcome.Receipt.ReceiptID == "" {
					t.Fatal("recovered transaction did not return its durable receipt")
				}
				after := runnableFingerprint199(t, fixture.root, fixture.dataDir)
				if after == before {
					t.Fatalf("%s recovery did not reach the committed safe point", command)
				}
				if transactionReceiptCount199(t, fixture.dataDir) != map[string]int{"pause": 1, "resume": 2}[command] {
					t.Fatalf("unexpected receipt count after recovered %s", command)
				}
				if command == "pause" {
					_, err = pauseColonyAt(fixture.now.Add(3 * time.Minute))
				} else {
					_, err = resumeColonyAt(fixture.now.Add(3 * time.Minute))
				}
				if err != nil {
					t.Fatalf("replay recovered %s: %v", command, err)
				}
				if replayed := runnableFingerprint199(t, fixture.root, fixture.dataDir); replayed != after {
					t.Fatalf("%s replay changed restored bytes: restored=%s replay=%s", command, after, replayed)
				}
			})
		}
	}

	t.Run("active_work_waits", func(t *testing.T) {
		fixture := newPauseResume199Fixture(t)
		state := load199State(t)
		state.State = colony.StateEXECUTING
		startedAt := fixture.now.Add(-time.Minute)
		state.BuildStartedAt = &startedAt
		if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
			t.Fatalf("save executing state: %v", err)
		}
		attemptID := "attempt-live-199"
		attemptRel := filepath.ToSlash(filepath.Join("build", "phase-1", "attempts", attemptID+".json"))
		if err := store.SaveJSON(attemptRel, buildAttemptRecord{
			SchemaVersion: buildAttemptSchemaVersion, ID: attemptID, Phase: 1,
			Status: buildAttemptDispatching, ProcessID: os.Getpid(), UpdatedAt: fixture.now.Format(time.RFC3339),
		}); err != nil {
			t.Fatalf("save live attempt: %v", err)
		}
		if err := store.SaveJSON(latestBuildAttemptPointerPath(1), latestBuildAttemptPointer{
			SchemaVersion: buildAttemptSchemaVersion, AttemptID: attemptID, Path: attemptRel, UpdatedAt: fixture.now.Format(time.RFC3339),
		}); err != nil {
			t.Fatalf("save latest attempt: %v", err)
		}
		before := runnableFingerprint199(t, fixture.root, fixture.dataDir)
		_, err := pauseColonyAt(fixture.now)
		var pending pauseBoundaryPendingError
		if !errors.As(err, &pending) || pending.Boundary != "worker_completion_boundary" || pending.Attempt != attemptID {
			t.Fatalf("active pause error = %#v, want named worker boundary for %s", err, attemptID)
		}
		if after := runnableFingerprint199(t, fixture.root, fixture.dataDir); after != before {
			t.Fatalf("active pause mutated before safe boundary: before=%s after=%s", before, after)
		}
		if transactionReceiptCount199(t, fixture.dataDir) != 0 {
			t.Fatal("active pause wrote a receipt before reaching its safe boundary")
		}
	})
}

func TestPauseResume199HandoffContents(t *testing.T) {
	fixture := newPauseResume199Fixture(t)
	outcome, err := pauseColonyAt(fixture.now)
	if err != nil {
		t.Fatalf("pause: %v", err)
	}
	handoff := load199Handoff(t, fixture.dataDir)
	if handoff.HandoffID != outcome.Handoff.HandoffID || handoff.Transaction.ID != outcome.Receipt.Transaction.ID {
		t.Fatalf("handoff/receipt transaction cross-reference mismatch: %#v %#v", handoff.Transaction, outcome.Receipt.Transaction)
	}
	state, session := load199State(t), load199Session(t)
	if state.PauseHandoff == nil || session.PauseHandoff == nil || !reflect.DeepEqual(*state.PauseHandoff, *session.PauseHandoff) {
		t.Fatalf("state and session do not share one handoff reference: %#v %#v", state.PauseHandoff, session.PauseHandoff)
	}
	if state.PauseHandoff.ID != handoff.HandoffID || state.PauseHandoff.TransactionID != handoff.Transaction.ID {
		t.Fatalf("state reference does not identify handoff transaction: %#v", state.PauseHandoff)
	}
	checks := map[string]bool{
		"safe boundary": handoff.SafeBoundary != "",
		"restart point": handoff.RestartPoint != "",
		"partial task":  len(handoff.TaskIDs) > 0,
		"attempt/run":   handoff.AttemptID != "" || handoff.RunID != "",
		"decisions":     len(handoff.Decisions) > 0,
		"context":       handoff.ContextDigest != "",
		"repository":    handoff.Repository.Head != "" && handoff.Repository.DirtyDigest != "",
		"worktrees":     len(handoff.Worktrees) > 0,
		"lineage":       len(handoff.Lineage) > 0,
		"blockers":      len(handoff.Blockers) > 0,
		"signals":       len(handoff.Signals) > 0,
		"evidence":      len(handoff.Evidence) > 0,
		"verification":  len(handoff.Verification) > 0,
		"receipt":       handoff.Receipt != nil && handoff.Receipt.ID == outcome.Receipt.ReceiptID,
	}
	for name, ok := range checks {
		if !ok {
			t.Errorf("handoff is missing %s evidence: %#v", name, handoff)
		}
	}
}

func TestPauseResume199ReplayExactlyOnce(t *testing.T) {
	fixture := newPauseResume199Fixture(t)
	firstPause, err := pauseColonyAt(fixture.now)
	if err != nil {
		t.Fatalf("pause: %v", err)
	}
	pausedFingerprint := runnableFingerprint199(t, fixture.root, fixture.dataDir)
	secondPause, err := pauseColonyAt(fixture.now.Add(time.Minute))
	if err != nil {
		t.Fatalf("replay pause: %v", err)
	}
	if !secondPause.Replay || secondPause.Message != pauseReplayMessage {
		t.Fatalf("pause replay = %#v, want exact retained message", secondPause)
	}
	if secondPause.Handoff.HandoffID != firstPause.Handoff.HandoffID || secondPause.Receipt.ReceiptID != firstPause.Receipt.ReceiptID {
		t.Fatal("pause replay minted a second handoff or receipt")
	}
	if got := runnableFingerprint199(t, fixture.root, fixture.dataDir); got != pausedFingerprint {
		t.Fatal("pause replay changed runnable state")
	}
	if transactionReceiptCount199(t, fixture.dataDir) != 1 {
		t.Fatal("pause replay created duplicate transaction evidence")
	}

	firstResume, err := resumeColonyAt(fixture.now.Add(2 * time.Minute))
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	resumedFingerprint := runnableFingerprint199(t, fixture.root, fixture.dataDir)
	secondResume, err := resumeColonyAt(fixture.now.Add(3 * time.Minute))
	if err != nil {
		t.Fatalf("replay resume: %v", err)
	}
	if !secondResume.Replay || secondResume.Receipt.ReceiptID != firstResume.Receipt.ReceiptID {
		t.Fatal("resume replay did not return the original receipt")
	}
	if got := runnableFingerprint199(t, fixture.root, fixture.dataDir); got != resumedFingerprint {
		t.Fatal("resume replay duplicated a lifecycle effect")
	}
	if transactionReceiptCount199(t, fixture.dataDir) != 2 {
		t.Fatal("pause/resume replay created duplicate receipts")
	}
}

func TestPauseResume199Confirmed(t *testing.T) {
	fixture := newPauseResume199Fixture(t)
	if _, err := pauseColonyAt(fixture.now); err != nil {
		t.Fatalf("pause: %v", err)
	}
	outcome, err := resumeColonyAt(fixture.now.Add(time.Minute))
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if outcome.Provenance != colony.RecoveryProvenanceConfirmed || outcome.StateEffect != colony.LifecycleStateEffectCommitted {
		t.Fatalf("confirmed resume outcome = %#v", outcome)
	}
	state, session := load199State(t), load199Session(t)
	if state.Paused || state.State != colony.StateREADY || session.ContextCleared || session.ResumedAt == nil {
		t.Fatalf("resume did not restore a runnable point: state=%#v session=%#v", state, session)
	}
	if _, err := os.Stat(filepath.Join(fixture.root, ".aether", "HANDOFF.md")); !os.IsNotExist(err) {
		t.Fatalf("resume did not close the human handoff: %v", err)
	}
}

func TestPauseResume199Reconstructed(t *testing.T) {
	fixture := newPauseResume199Fixture(t)
	state := load199State(t)
	pausedAt := fixture.now.Format(time.RFC3339)
	state.Paused = true
	state.PausedAt = &pausedAt
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatalf("save legacy paused state: %v", err)
	}
	write199File(t, filepath.Join(fixture.root, ".aether", "HANDOFF.md"), buildHandoffDocument(fixture.now, state, load199Session(t), "aether resume"))
	outcome, err := resumeColonyAt(fixture.now.Add(time.Minute))
	if err != nil {
		t.Fatalf("reconstruct resume: %v", err)
	}
	if outcome.Provenance != colony.RecoveryProvenanceReconstructed {
		t.Fatalf("provenance = %q, want reconstructed", outcome.Provenance)
	}
	if load199State(t).Paused {
		t.Fatal("reconstructed recovery did not restore a runnable state")
	}
	if outcome.Handoff.HandoffID == "" || outcome.Receipt.ReceiptID == "" {
		t.Fatal("reconstructed recovery was not bound to durable handoff/receipt evidence")
	}
	legacySessionPath := filepath.Join(fixture.dataDir, "colonies", "legacy", "session.json")
	write199File(t, legacySessionPath, string(read199File(t, filepath.Join(fixture.dataDir, "session.json"))))
	if err := os.Remove(filepath.Join(fixture.dataDir, "session.json")); err != nil {
		t.Fatalf("remove top-level session: %v", err)
	}
	restored, err := ensureLegacySessionMirror(store)
	if err != nil || restored {
		t.Fatalf("read-only legacy session loader = restored:%t err:%v", restored, err)
	}
	if _, err := os.Stat(filepath.Join(fixture.dataDir, "session.json")); !os.IsNotExist(err) {
		t.Fatalf("orientation recreated a top-level session outside the lifecycle transaction: %v", err)
	}
}

func TestPauseResume199ConflictZeroWrite(t *testing.T) {
	fixture := newPauseResume199Fixture(t)
	if _, err := pauseColonyAt(fixture.now); err != nil {
		t.Fatalf("pause: %v", err)
	}
	session := load199Session(t)
	session.PauseHandoff.Digest = "sha256:" + strings.Repeat("0", 64)
	if err := store.SaveJSON("session.json", session); err != nil {
		t.Fatalf("inject conflicting provenance: %v", err)
	}
	before := runnableFingerprint199(t, fixture.root, fixture.dataDir)
	outcome, err := resumeColonyAt(fixture.now.Add(time.Minute))
	if err != nil {
		t.Fatalf("conflict should be a rendered stop, not an execution error: %v", err)
	}
	if outcome.Provenance != colony.RecoveryProvenanceConflicting || outcome.StateEffect != colony.LifecycleStateEffectNone {
		t.Fatalf("conflict outcome = %#v", outcome)
	}
	if !strings.Contains(outcome.Message, "state and session") || !strings.Contains(outcome.Message, "aether status") {
		t.Fatalf("conflict lacks exact evidence and decision guidance: %q", outcome.Message)
	}
	after := runnableFingerprint199(t, fixture.root, fixture.dataDir)
	if after != before {
		t.Fatalf("conflicting resume mutated runnable state: before=%s after=%s", before, after)
	}
}

func TestPauseResume199HiddenRedirect(t *testing.T) {
	if pauseColonyCmd.Use != "pause" || resumeColonyCmd.Use != "resume" {
		t.Fatalf("canonical uses are pause/resume: %q %q", pauseColonyCmd.Use, resumeColonyCmd.Use)
	}
	if len(pauseColonyCmd.Aliases) != 0 || len(resumeColonyCmd.Aliases) != 0 {
		t.Fatalf("legacy names leaked into Cobra metadata: %v %v", pauseColonyCmd.Aliases, resumeColonyCmd.Aliases)
	}
	cases := []struct {
		version string
		args    []string
		want    []string
		from    string
	}{
		{"1.28.9", []string{"aether", "pause-colony", "--json"}, []string{"aether", "pause", "--json"}, "pause-colony"},
		{"1.28.9", []string{"aether", "resume-colony"}, []string{"aether", "resume"}, "resume-colony"},
		{"1.28.9", []string{"aether", "x", "pause-colony"}, []string{"aether", "x", "pause-colony"}, ""},
		{"1.29.0", []string{"aether", "pause-colony"}, []string{"aether", "pause-colony"}, ""},
	}
	for _, tc := range cases {
		got, redirect := normalizeLegacySessionInvocation(tc.args, tc.version)
		if !reflect.DeepEqual(got, tc.want) {
			t.Errorf("normalizeLegacySessionInvocation(%v, %q) = %v, want %v", tc.args, tc.version, got, tc.want)
		}
		if tc.from == "" && redirect != nil {
			t.Errorf("unexpected redirect: %#v", redirect)
		}
		if tc.from != "" && (redirect == nil || redirect.From != tc.from || redirect.ExpiresAt != legacySessionRedirectExpiry) {
			t.Errorf("redirect = %#v, want from %q expiring at %s", redirect, tc.from, legacySessionRedirectExpiry)
		}
	}
	if legacySessionRedirectExpiry != "1.29.0" {
		t.Fatalf("machine expiry marker = %q, want 1.29.0", legacySessionRedirectExpiry)
	}
	if got := legacySessionRedirectNotice(&legacySessionRedirect{From: "pause-colony", To: "pause", ExpiresAt: legacySessionRedirectExpiry}); got != "notice: pause-colony is deprecated and will be removed in 1.29; use aether pause." {
		t.Fatalf("deprecation notice = %q", got)
	}
}

func TestPauseWrapperContract199(t *testing.T) {
	repoRoot := filepath.Clean("..")
	canonicalPath := filepath.Join(repoRoot, ".aether", "commands", "pause.yaml")
	canonical := string(read199File(t, canonicalPath))
	const exactDescription = "Stop at a safe boundary and save one resumable handoff."
	for _, required := range []string{exactDescription, "aether pause", "safe boundary", "receipt", "/ant-resume"} {
		if !strings.Contains(canonical, required) {
			t.Errorf("canonical pause source is missing %q", required)
		}
	}
	for _, forbidden := range []string{"pause-colony", "COLONY_STATE.json", "session.json", "HANDOFF.md"} {
		if strings.Contains(canonical, forbidden) {
			t.Errorf("canonical pause source contains forbidden host-owned mutation/alias %q", forbidden)
		}
	}
	generated := []string{
		filepath.Join(repoRoot, ".claude", "commands", "ant-pause.md"),
		filepath.Join(repoRoot, ".claude", "commands", "ant", "pause.md"),
		filepath.Join(repoRoot, ".opencode", "commands", "ant", "pause.md"),
	}
	for _, path := range generated {
		content := string(read199File(t, path))
		for _, required := range []string{exactDescription, "aether pause", "safe boundary", "receipt", "/ant-resume"} {
			if !strings.Contains(content, required) {
				t.Errorf("%s is missing %q", path, required)
			}
		}
		for _, forbidden := range []string{"pause-colony", "COLONY_STATE.json", "session.json", "HANDOFF.md"} {
			if strings.Contains(content, forbidden) {
				t.Errorf("%s contains forbidden host-owned mutation/alias %q", path, forbidden)
			}
		}
		wantHeader := "<!-- Aether-managed: runtime spec at .aether/commands/pause.yaml. Synced by aether update. -->"
		if !strings.HasPrefix(content, wantHeader+"\n") {
			t.Errorf("%s does not declare canonical source linkage", path)
		}
	}
	for _, legacyPath := range []string{
		filepath.Join(repoRoot, ".claude", "commands", "ant-pause-colony.md"),
		filepath.Join(repoRoot, ".claude", "commands", "ant", "pause-colony.md"),
		filepath.Join(repoRoot, ".opencode", "commands", "ant", "pause-colony.md"),
	} {
		if _, err := os.Stat(legacyPath); !os.IsNotExist(err) {
			t.Errorf("legacy public pause wrapper still exists at %s: %v", legacyPath, err)
		}
	}
}

func Example_pauseResume199Contract() {
	fmt.Println(pauseReplayMessage)
	// Output: Already paused; the existing validated handoff was retained.
}
