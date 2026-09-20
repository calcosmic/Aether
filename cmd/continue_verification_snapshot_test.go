package cmd

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

type verificationSnapshotInvoker struct {
	continueWatcherTestInvoker
	snapshot codexContinueVerificationReport
	readErr  error
	brief    string
}

func (i *verificationSnapshotInvoker) Invoke(ctx context.Context, config codex.WorkerConfig) (codex.WorkerResult, error) {
	i.readErr = store.LoadJSON("build/phase-1/verification.json", &i.snapshot)
	i.brief = config.TaskBrief
	return i.continueWatcherTestInvoker.Invoke(ctx, config)
}

func TestContinueVerificationSnapshotPrecedesWatcher(t *testing.T) {
	for _, stale := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing", true: "stale"}[stale], func(t *testing.T) {
			saveGlobals(t)
			s, root := newTestStore(t)
			store = s
			if err := os.WriteFile(filepath.Join(root, "AGENTS.md"), []byte("## Verification Commands\n\nBuild: \x60echo current-build-proof\x60\nTests: \x60echo current-test-proof\x60\n"), 0644); err != nil {
				t.Fatal(err)
			}
			if stale {
				if err := store.SaveJSON("build/phase-1/verification.json", codexContinueVerificationReport{Phase: 1, GeneratedAt: "2000-01-01T00:00:00Z", Passed: true, Watcher: codexWatcherVerification{Present: true, Passed: true, Status: "completed"}}); err != nil {
					t.Fatal(err)
				}
			}
			invoker := &verificationSnapshotInvoker{}
			original := newCodexWorkerInvoker
			newCodexWorkerInvoker = func() codex.WorkerInvoker { return invoker }
			t.Cleanup(func() { newCodexWorkerInvoker = original })
			phase := colony.Phase{ID: 1, Name: "Verification snapshot"}
			state := colony.ColonyState{Plan: colony.Plan{Phases: []colony.Phase{phase}}}
			report, _ := runCodexContinueVerification(context.Background(), root, state, phase, codexContinueManifest{}, time.Second, time.Second, false)
			if invoker.watcherCalls != 1 {
				t.Fatalf("watcher calls=%d, want one", invoker.watcherCalls)
			}
			if invoker.readErr != nil {
				t.Fatalf("watcher could not read current verification: %v", invoker.readErr)
			}
			got := invoker.snapshot
			if got.GeneratedAt == "2000-01-01T00:00:00Z" || got.GeneratedAt == "" || got.Phase != 1 {
				t.Fatalf("snapshot is stale or unbound: %+v", got)
			}
			if got.Passed || got.Watcher.Passed || got.Watcher.Present || got.Watcher.Status != "pending" {
				t.Fatalf("snapshot falsely completes review: %+v", got)
			}
			if !got.ChecksPassed {
				t.Fatalf("current deterministic floor absent: %+v", got)
			}
			found := false
			for _, step := range got.Steps {
				if strings.Contains(step.Output, "current-test-proof") {
					found = true
				}
			}
			if !found {
				t.Fatalf("snapshot omits current command output: %+v", got.Steps)
			}
			if !strings.Contains(invoker.brief, "historical") || !strings.Contains(invoker.brief, "pending") || !strings.Contains(invoker.brief, "result-collection.json") {
				t.Fatalf("watcher brief does not distinguish current pending work from history: %s", invoker.brief)
			}
			if !report.Passed || !report.Watcher.Passed {
				t.Fatalf("successful watcher did not complete report: %+v", report)
			}
		})
	}
}

func TestContinueVerificationSnapshotWriteFailurePreventsWatcher(t *testing.T) {
	saveGlobals(t)
	s, root := newTestStore(t)
	store = s
	if err := os.MkdirAll(filepath.Join(root, ".aether/data/build/phase-1/verification.json"), 0755); err != nil {
		t.Fatal(err)
	}
	invoker := &verificationSnapshotInvoker{}
	original := newCodexWorkerInvoker
	newCodexWorkerInvoker = func() codex.WorkerInvoker { return invoker }
	t.Cleanup(func() { newCodexWorkerInvoker = original })
	phase := colony.Phase{ID: 1, Name: "Cannot persist verification"}
	state := colony.ColonyState{Plan: colony.Plan{Phases: []colony.Phase{phase}}}
	report, _ := runCodexContinueVerification(context.Background(), root, state, phase, codexContinueManifest{}, time.Second, time.Second, false)
	if invoker.watcherCalls != 0 {
		t.Fatalf("launched %d watcher(s) without persisted evidence", invoker.watcherCalls)
	}
	if report.Passed || report.ChecksPassed || !strings.Contains(strings.Join(report.BlockingIssues, " "), "verification snapshot") {
		t.Fatalf("write failure did not block: %+v", report)
	}
}

func TestContinueReviewBriefTreatsPriorOutcomeAsHistorical(t *testing.T) {
	brief := renderCodexContinueReviewBrief("fixture", colony.Phase{ID: 1}, codexContinueManifest{}, codexContinueVerificationReport{}, codexContinueAssessment{}, codexContinueReviewSpec{Caste: "auditor"})
	if !strings.Contains(brief, "historical") || !strings.Contains(brief, "continue.json") || !strings.Contains(brief, "review.json") {
		t.Fatalf("review brief lacks prior-outcome boundary: %s", brief)
	}
}
