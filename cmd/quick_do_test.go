package cmd

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// quick_do_test.go proves release 1.0.85's Part A: /ant-quick does the job
// (CAP-029, the owner's 21 Sep decision) with exactly one builder, uses the
// project's own resolved checks rather than a hardcoded pair, raises exactly
// one tracked issue when those checks fail, and leaves a durable record
// aether history can show. The read-only question mode (--question) is kept
// byte-for-byte.

// quickJobCaptureInvoker is a WorkerInvoker spy that records the exact
// WorkerConfig handed to it (proving what the real dispatch wired, not a
// parallel computation that merely agrees) and returns a configurable
// result.
type quickJobCaptureInvoker struct {
	mu      sync.Mutex
	configs []codex.WorkerConfig
	result  codex.WorkerResult
	err     error
}

func (q *quickJobCaptureInvoker) IsAvailable(ctx context.Context) bool { return true }
func (q *quickJobCaptureInvoker) ValidateAgent(path string) error      { return nil }
func (q *quickJobCaptureInvoker) Invoke(ctx context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	q.mu.Lock()
	q.configs = append(q.configs, config)
	q.mu.Unlock()
	if q.err != nil {
		return codex.WorkerResult{}, q.err
	}
	result := q.result
	result.WorkerName = config.WorkerName
	result.Caste = config.Caste
	result.TaskID = config.TaskID
	if result.Status == "" {
		result.Status = "completed"
	}
	return result, nil
}

func (q *quickJobCaptureInvoker) captured() []codex.WorkerConfig {
	q.mu.Lock()
	defer q.mu.Unlock()
	return append([]codex.WorkerConfig{}, q.configs...)
}

// seedQuickJobRealSteering writes one active REDIRECT pheromone signal and
// one QUEEN.md hub preference through the REAL production commands
// (writePheromoneSignal -- the same function /ant-redirect calls -- and
// preferencesCmd's own RunE), never a hand-typed fixture shape, so this test
// proves the real steering channel and not a parallel one.
func seedQuickJobRealSteering(t *testing.T, sentinel string) {
	t.Helper()
	if _, _, err := writePheromoneSignal("REDIRECT", "never touch "+sentinel+" directly", "high", "test", "", "", 0.9, nil); err != nil {
		t.Fatalf("seed REDIRECT signal via the real writer: %v", err)
	}
	t.Setenv("HOME", t.TempDir())
	if err := preferencesCmd.RunE(preferencesCmd, []string{"prefer short commit messages about " + sentinel}); err != nil {
		t.Fatalf("seed preference via the real preferences command: %v", err)
	}
}

func TestQuickSendsOneBuilderWithMemory(t *testing.T) {
	run := func(t *testing.T, setup func(t *testing.T)) {
		t.Helper()
		saveGlobals(t)
		s, root := newTestStore(t)
		store = s
		chdirForTest190_05(t, root)
		if setup != nil {
			setup(t)
		}

		sentinel := "widget-" + t.Name()
		seedQuickJobRealSteering(t, sentinel)

		spy := &quickJobCaptureInvoker{result: codex.WorkerResult{Summary: "renamed the label", FilesModified: []string{"label.go"}}}
		origInvoker := newQuickWorkerInvoker
		newQuickWorkerInvoker = func() codex.WorkerInvoker { return spy }
		t.Cleanup(func() { newQuickWorkerInvoker = origInvoker })
		origChecks := runQuickDeterministicChecks
		runQuickDeterministicChecks = func(root string, files []string) (string, []string, error) {
			return quickChecksPassed, []string{"go build ./...: passed"}, nil
		}
		t.Cleanup(func() { runQuickDeterministicChecks = origChecks })

		result, err := runQuickJob("rename the label", 2*time.Second)
		if err != nil {
			t.Fatalf("runQuickJob: %v", err)
		}

		configs := spy.captured()
		if len(configs) != 1 {
			t.Fatalf("expected exactly one worker dispatch, got %d", len(configs))
		}
		cfg := configs[0]
		if cfg.AgentName != "aether-builder" || cfg.Caste != "builder" {
			t.Fatalf("expected the builder to be dispatched, got AgentName=%q Caste=%q", cfg.AgentName, cfg.Caste)
		}
		if !strings.Contains(cfg.ContextCapsule, "widget-"+t.Name()) {
			t.Fatalf("expected the QUEEN.md preference sentinel in the context capsule, got:\n%s", cfg.ContextCapsule)
		}
		if !strings.Contains(cfg.ContextCapsule, "never touch "+sentinel+" directly") {
			t.Fatalf("expected the active REDIRECT signal in the context capsule, got:\n%s", cfg.ContextCapsule)
		}
		if strings.Contains(cfg.TaskBrief, "Read-only") {
			t.Fatalf("expected the job brief to carry no read-only constraint, got:\n%s", cfg.TaskBrief)
		}

		verdict, _ := result["work_outcome"].(colony.WorkOutcome)
		if verdict != colony.WorkOutcomeSuccess {
			t.Fatalf("verdict = %q, want %q", verdict, colony.WorkOutcomeSuccess)
		}
	}

	t.Run("no project initialised", func(t *testing.T) {
		run(t, nil)
	})

	t.Run("archived project shell", func(t *testing.T) {
		run(t, func(t *testing.T) {
			downstream, _ := runRealLifecycleToSealForTest(t)
			if _, err := runEntombTransaction(entombTransactionInput{
				Root: downstream, DataRoot: store.BasePath(), Confirmed: true, Now: time.Now().UTC(),
			}); err != nil {
				t.Fatalf("archive the sealed colony via the real entomb transaction: %v", err)
			}
			chdirForTest190_05(t, downstream)
		})
	})
}

// TestQuickQuestionModeStaysReadOnlyScout proves --question keeps today's
// scout, brief and capsule byte-for-byte: runQuickScout is untouched by
// Part A's job mode.
func TestQuickQuestionModeStaysReadOnlyScout(t *testing.T) {
	saveGlobals(t)
	s, root := newTestStore(t)
	store = s
	chdirForTest190_05(t, root)

	spy := &quickJobCaptureInvoker{result: codex.WorkerResult{Summary: "the answer"}}
	origInvoker := newQuickWorkerInvoker
	newQuickWorkerInvoker = func() codex.WorkerInvoker { return spy }
	t.Cleanup(func() { newQuickWorkerInvoker = origInvoker })

	result, err := runQuickScout("where is X", 2*time.Second)
	if err != nil {
		t.Fatalf("runQuickScout: %v", err)
	}
	configs := spy.captured()
	if len(configs) != 1 {
		t.Fatalf("expected exactly one worker dispatch, got %d", len(configs))
	}
	cfg := configs[0]
	if cfg.AgentName != "aether-scout" || cfg.Caste != "scout" {
		t.Fatalf("expected the scout to be dispatched, got AgentName=%q Caste=%q", cfg.AgentName, cfg.Caste)
	}
	if !strings.Contains(cfg.TaskBrief, "Read-only") {
		t.Fatalf("expected the question brief to keep its read-only constraint, got:\n%s", cfg.TaskBrief)
	}
	if cfg.ContextCapsule != renderQuickContextCapsule("where is X") {
		t.Fatalf("expected the question capsule to stay byte-for-byte, got:\n%s", cfg.ContextCapsule)
	}
	if result["mode"] != "quick" {
		t.Fatalf("expected mode=quick, got %v", result["mode"])
	}
}

// TestQuickVerdictFollowsTheProjectsOwnChecks is the rigorous table proving
// CAP-029's verdict rule: files changed and checks pass is success; files
// changed and checks fail is blocker, with the files kept on disk, exactly
// one unresolved issue flag raised (source quick), and one failure-log
// entry; files changed and nothing resolvable to check is reported as
// "changed, not checked" -- never success.
func TestQuickVerdictFollowsTheProjectsOwnChecks(t *testing.T) {
	for _, test := range []struct {
		name          string
		checksOutcome string
		wantVerdict   colony.WorkOutcome
		wantFlag      bool
	}{
		{"checks pass", quickChecksPassed, colony.WorkOutcomeSuccess, false},
		{"checks fail", quickChecksFailed, colony.WorkOutcomeBlocker, true},
		{"nothing resolvable", quickChecksNotResolved, colony.WorkOutcomePartial, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			saveGlobals(t)
			s, root := newTestStore(t)
			store = s
			chdirForTest190_05(t, root)

			origInvoker := newQuickWorkerInvoker
			newQuickWorkerInvoker = func() codex.WorkerInvoker { return &fileChangingWorkerInvoker{} }
			t.Cleanup(func() { newQuickWorkerInvoker = origInvoker })

			origChecks := runQuickDeterministicChecks
			runQuickDeterministicChecks = func(root string, files []string) (string, []string, error) {
				return test.checksOutcome, []string{"evidence line"}, nil
			}
			t.Cleanup(func() { runQuickDeterministicChecks = origChecks })

			result, err := runQuickJob("fix the typo in the readme", 2*time.Second)
			if err != nil {
				t.Fatalf("runQuickJob: %v", err)
			}
			verdict, _ := result["work_outcome"].(colony.WorkOutcome)
			if verdict != test.wantVerdict {
				t.Fatalf("verdict = %q, want %q", verdict, test.wantVerdict)
			}
			if verdict == colony.WorkOutcomeSuccess || verdict == colony.WorkOutcomePartial {
				statusWords := stringValue(result["checks_status"])
				if statusWords == "" {
					t.Fatalf("expected a checks_status in the result, got %v", result)
				}
			}
			if test.checksOutcome == quickChecksNotResolved {
				words := quickChecksStatusWords(quickChecksNotResolved)
				if strings.Contains(words, "success") {
					t.Fatalf("expected the not-checked wording to never say success, got %q", words)
				}
			}

			var ff colony.FlagsFile
			_ = store.LoadJSON("pending-decisions.json", &ff)
			openQuickIssues := 0
			for _, entry := range ff.Decisions {
				if entry.Resolved {
					continue
				}
				if strings.EqualFold(strings.TrimSpace(entry.Source), "quick") && entry.Type == "issue" {
					openQuickIssues++
				}
			}
			wantIssues := 0
			if test.wantFlag {
				wantIssues = 1
			}
			if openQuickIssues != wantIssues {
				t.Fatalf("open quick issue flags = %d, want %d (decisions=%+v)", openQuickIssues, wantIssues, ff.Decisions)
			}

			if test.wantFlag {
				mf, err := loadMiddenFile(store)
				if err != nil {
					t.Fatalf("load midden: %v", err)
				}
				if len(mf.Entries) == 0 {
					t.Fatalf("expected at least one failure-log entry for a failed quick job")
				}

				// Re-running the same failing job must not add a second
				// identical flag.
				if _, err := runQuickJob("fix the typo in the readme", 2*time.Second); err != nil {
					t.Fatalf("runQuickJob (second run): %v", err)
				}
				var ff2 colony.FlagsFile
				_ = store.LoadJSON("pending-decisions.json", &ff2)
				openQuickIssues2 := 0
				for _, entry := range ff2.Decisions {
					if !entry.Resolved && strings.EqualFold(strings.TrimSpace(entry.Source), "quick") && entry.Type == "issue" {
						openQuickIssues2++
					}
				}
				if openQuickIssues2 != 1 {
					t.Fatalf("expected exactly one deduplicated open quick issue flag after re-running, got %d", openQuickIssues2)
				}
			}
		})
	}
}

// TestQuickUsesResolvedChecksNotGoOnly proves the checks seam reads the
// project's own resolved verification commands (resolveCodexVerificationCommands)
// rather than a hardcoded `go build`/`go vet` pair.
func TestQuickUsesResolvedChecksNotGoOnly(t *testing.T) {
	saveGlobals(t)
	_, root := newTestStore(t)
	chdirForTest190_05(t, root)

	claudeMD := "## Verification Commands\n\n- Build: echo build-marker-190-85\n- Test: echo test-marker-190-85\n"
	writeFile(t, root, "CLAUDE.md", []byte(claudeMD))

	status, evidence, err := runQuickDeterministicChecks(root, []string{"main.go"})
	if err != nil {
		t.Fatalf("runQuickDeterministicChecks: %v", err)
	}
	if status != quickChecksPassed {
		t.Fatalf("status = %q, want %q, evidence=%v", status, quickChecksPassed, evidence)
	}
	joined := strings.Join(evidence, "\n")
	if !strings.Contains(joined, "echo build-marker-190-85") {
		t.Fatalf("expected the CLAUDE.md-declared build command to run, got evidence:\n%s", joined)
	}
}

// TestQuickLeavesARecordHistoryCanShow proves a quick job's attempt is
// persisted durably (persistQuickAttempt) and reaches the shared episode
// lineage aether history reads (loadColonyEpisodeIndex).
func TestQuickLeavesARecordHistoryCanShow(t *testing.T) {
	saveGlobals(t)
	s, root := newTestStore(t)
	store = s
	chdirForTest190_05(t, root)

	origInvoker := newQuickWorkerInvoker
	newQuickWorkerInvoker = func() codex.WorkerInvoker { return &fileChangingWorkerInvoker{} }
	t.Cleanup(func() { newQuickWorkerInvoker = origInvoker })

	origChecks := runQuickDeterministicChecks
	runQuickDeterministicChecks = func(root string, files []string) (string, []string, error) {
		return quickChecksPassed, []string{"passed"}, nil
	}
	t.Cleanup(func() { runQuickDeterministicChecks = origChecks })

	if _, err := runQuickJob("fix the typo in the readme", 2*time.Second); err != nil {
		t.Fatalf("runQuickJob: %v", err)
	}

	idx, err := loadColonyEpisodeIndex(root, s)
	if err != nil {
		t.Fatalf("loadColonyEpisodeIndex: %v", err)
	}
	found := false
	for _, entry := range idx.Entries {
		if entry.Kind == colonyEpisodeKindQuick && strings.Contains(entry.Subject, "fix the typo") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected one quick-job entry in the episode index, got %+v", idx.Entries)
	}
}

// silentFileWritingWorkerInvoker writes a real file into the repo but
// reports zero files -- the untrusted-evidence shape TestQuickTrustsDiskOverWorkerReport
// proves the runtime never takes on faith.
type silentFileWritingWorkerInvoker struct {
	root string
	rel  string
}

func (i *silentFileWritingWorkerInvoker) IsAvailable(ctx context.Context) bool { return true }
func (i *silentFileWritingWorkerInvoker) ValidateAgent(path string) error      { return nil }
func (i *silentFileWritingWorkerInvoker) Invoke(ctx context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	if err := os.WriteFile(filepath.Join(i.root, i.rel), []byte("package fixture\n"), 0644); err != nil {
		return codex.WorkerResult{}, err
	}
	return codex.WorkerResult{
		WorkerName: config.WorkerName,
		Caste:      config.Caste,
		TaskID:     config.TaskID,
		Status:     "completed",
		Summary:    "did the job",
		// Deliberately empty -- the worker reports NO files, even though it
		// really wrote one. Worker-reported evidence is untrusted.
	}, nil
}

// TestQuickTrustsDiskOverWorkerReport is release 1.0.85's TRUST fix: a
// helper that changes a file but reports none must still have that file
// checked and must never verdict no_change. The runtime's own trust
// boundary (CLAUDE.md: worker-reported evidence is untrusted) applies to
// quick jobs exactly as it does to build claims.
func TestQuickTrustsDiskOverWorkerReport(t *testing.T) {
	saveGlobals(t)
	s, root := newTestStore(t)
	store = s
	chdirForTest190_05(t, root)
	gitInitForTest(t, root)

	origInvoker := newQuickWorkerInvoker
	newQuickWorkerInvoker = func() codex.WorkerInvoker {
		return &silentFileWritingWorkerInvoker{root: root, rel: "sneaky.go"}
	}
	t.Cleanup(func() { newQuickWorkerInvoker = origInvoker })

	origChecks := runQuickDeterministicChecks
	checkCalls := 0
	var checkedFiles []string
	runQuickDeterministicChecks = func(root string, files []string) (string, []string, error) {
		checkCalls++
		checkedFiles = append([]string(nil), files...)
		return quickChecksPassed, []string{"passed"}, nil
	}
	t.Cleanup(func() { runQuickDeterministicChecks = origChecks })

	result, err := runQuickJob("silently create a file", 2*time.Second)
	if err != nil {
		t.Fatalf("runQuickJob: %v", err)
	}
	if checkCalls != 1 {
		t.Fatalf("expected the checks seam to be called exactly once for the real disk change the worker never reported, got %d calls", checkCalls)
	}
	found := false
	for _, f := range checkedFiles {
		if strings.Contains(f, "sneaky.go") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected sneaky.go (written to disk but never reported) to reach the checks seam, got %v", checkedFiles)
	}
	verdict, _ := result["work_outcome"].(colony.WorkOutcome)
	if verdict == colony.WorkOutcomeNoChange {
		t.Fatalf("expected a real disk change to never verdict no_change just because the worker did not report it")
	}
}

// blockerReportingWorkerInvoker completes successfully but reports a
// blocker -- the shape TestQuickBlockerNeverStopsTheProject proves never
// escalates into a project-stopping blocker flag for a quick job.
type blockerReportingWorkerInvoker struct{}

func (i *blockerReportingWorkerInvoker) IsAvailable(ctx context.Context) bool { return true }
func (i *blockerReportingWorkerInvoker) ValidateAgent(path string) error      { return nil }
func (i *blockerReportingWorkerInvoker) Invoke(ctx context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	return codex.WorkerResult{
		WorkerName: config.WorkerName,
		Caste:      config.Caste,
		TaskID:     config.TaskID,
		Status:     "completed",
		Summary:    "could not finish safely",
		Blockers:   []string{"needs a decision about which config file to use"},
	}, nil
}

// TestQuickBlockerNeverStopsTheProject is release 1.0.85's second review
// fix: a helper-reported blocker during a quick job must never raise a
// project-stopping `blocker` flag (which halts an active project's next
// check and flips the what-next advice to "resume"). A quick job raises
// exactly one tracked `issue` instead, carrying the helper's own sentence.
func TestQuickBlockerNeverStopsTheProject(t *testing.T) {
	saveGlobals(t)
	s, root := newTestStore(t)
	store = s
	chdirForTest190_05(t, root)

	origInvoker := newQuickWorkerInvoker
	newQuickWorkerInvoker = func() codex.WorkerInvoker { return &blockerReportingWorkerInvoker{} }
	t.Cleanup(func() { newQuickWorkerInvoker = origInvoker })

	if _, err := runQuickJob("do something risky", 2*time.Second); err != nil {
		t.Fatalf("runQuickJob: %v", err)
	}

	var ff colony.FlagsFile
	_ = store.LoadJSON("pending-decisions.json", &ff)
	unresolvedBlockers := 0
	unresolvedQuickIssues := 0
	for _, entry := range ff.Decisions {
		if entry.Resolved {
			continue
		}
		if entry.Type == "blocker" {
			unresolvedBlockers++
		}
		if entry.Type == "issue" && strings.EqualFold(strings.TrimSpace(entry.Source), "quick") {
			unresolvedQuickIssues++
		}
	}
	if unresolvedBlockers != 0 {
		t.Fatalf("expected zero unresolved blocker rows from a quick job's reported blocker, got %d (decisions=%+v)", unresolvedBlockers, ff.Decisions)
	}
	if unresolvedQuickIssues != 1 {
		t.Fatalf("expected exactly one unresolved issue row with source quick, got %d (decisions=%+v)", unresolvedQuickIssues, ff.Decisions)
	}
}

// TestQuickInvokeErrorLogsOnce is release 1.0.85's third review fix: an
// invoke error was logged twice -- once by recordDispatchWorkerOutcome
// (category worker_failed, falsely attributed to "aether build") and once
// by recordQuickFailureToMidden (category quick_failed, the correct
// quick-specific record). Exactly one entry, with the correct category,
// should reach the failure log.
func TestQuickInvokeErrorLogsOnce(t *testing.T) {
	saveGlobals(t)
	s, root := newTestStore(t)
	store = s
	chdirForTest190_05(t, root)

	wantErr := errors.New("builder invocation failed: connection refused")
	origInvoker := newQuickWorkerInvoker
	newQuickWorkerInvoker = func() codex.WorkerInvoker { return &failingWorkerInvoker{err: wantErr} }
	t.Cleanup(func() { newQuickWorkerInvoker = origInvoker })

	if _, err := runQuickJob("do something", 2*time.Second); err == nil {
		t.Fatal("expected runQuickJob to return the invoker's error")
	}

	mf, err := loadMiddenFile(store)
	if err != nil {
		t.Fatalf("load midden: %v", err)
	}
	if len(mf.Entries) != 1 {
		t.Fatalf("expected exactly one failure-log entry, got %d: %+v", len(mf.Entries), mf.Entries)
	}
	if mf.Entries[0].Category != middenCategoryQuickFailed {
		t.Fatalf("category = %q, want %q", mf.Entries[0].Category, middenCategoryQuickFailed)
	}
}
