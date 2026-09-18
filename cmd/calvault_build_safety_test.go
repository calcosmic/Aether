package cmd

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

type calvaultTimeoutInvoker struct {
	mu     sync.Mutex
	called []string
}

func (p *calvaultTimeoutInvoker) IsAvailable(context.Context) bool { return true }
func (p *calvaultTimeoutInvoker) ValidateAgent(string) error       { return nil }
func (p *calvaultTimeoutInvoker) Invoke(ctx context.Context, c codex.WorkerConfig) (codex.WorkerResult, error) {
	p.mu.Lock()
	p.called = append(p.called, c.WorkerName)
	p.mu.Unlock()
	if c.WorkerName == "Prerequisite" {
		if err := os.WriteFile(filepath.Join(c.Root, "uncredited-draft.py"), []byte("draft survives\n"), 0600); err != nil {
			return codex.WorkerResult{}, err
		}
		return codex.WorkerResult{WorkerName: c.WorkerName, Status: "timeout"}, context.DeadlineExceeded
	}
	return (&codex.FakeInvoker{}).Invoke(ctx, c)
}

func TestCalVaultTimedOutGroupedPrerequisiteBlocksDependentButNotIndependent(t *testing.T) {
	data := setupBuildFlowTest(t)
	root := filepath.Dir(filepath.Dir(data))
	if out, err := exec.Command("git", "-C", root, "init").CombinedOutput(); err != nil {
		t.Fatalf("git init: %s %v", out, err)
	}
	ids := []string{"2.1", "2.2", "2.3", "2.4", "2.5"}
	phase := colony.Phase{ID: 2, Name: "CalVault safety"}
	for i := range ids {
		phase.Tasks = append(phase.Tasks, colony.Task{ID: &ids[i], Status: colony.TaskPending, Goal: ids[i]})
	}
	phase.Tasks[1].SemanticID = "prove-manifest-driven-transfer"
	phase.Tasks[2].SemanticID = "apply-simple-writing-and-capture"
	phase.Tasks[3].DependsOn = []string{phase.Tasks[1].SemanticID, phase.Tasks[2].SemanticID}
	dispatches := []codexBuildDispatch{
		{Name: "Prerequisite", Caste: "builder", TaskID: ids[0], CoveredTaskIDs: ids[:3], Wave: 11},
		{Name: "Dependent", Caste: "builder", TaskID: ids[3], Wave: 12},
		{Name: "Independent", Caste: "builder", TaskID: ids[4], Wave: 12},
	}
	tree := agent.NewSpawnTree(store, "spawn-tree.txt")
	for _, d := range dispatches {
		if err := tree.RecordSpawn("Queen", d.Caste, d.Name, d.Task, 2); err != nil {
			t.Fatal(err)
		}
	}
	invoker := &calvaultTimeoutInvoker{}
	results, _, _, err := executeCodexBuildDispatches(context.Background(), root, phase, dispatches, time.Now(), invoker, colony.ModeInRepo, time.Second, 3, false, nil)
	if err == nil {
		t.Fatal("expected failed build")
	}
	if len(results) != 3 {
		t.Fatalf("results: %+v, err=%v", results, err)
	}
	calls := strings.Join(invoker.called, ",")
	if strings.Contains(calls, "Dependent") || !strings.Contains(calls, "Independent") || !strings.Contains(calls, "Prerequisite") {
		t.Fatalf("wrong invocations: %s", calls)
	}
	if results[1].Status != "dependency_blocked" {
		t.Fatalf("dependent result: %+v", results[1])
	}
	entries, parseErr := tree.Parse()
	if parseErr != nil {
		t.Fatal(parseErr)
	}
	foundBlocked := false
	for _, entry := range entries {
		if entry.AgentName == "Dependent" {
			foundBlocked = entry.Status == "blocked" && agent.IsTerminalSpawnStatus(entry.Status) && !agent.IsLiveSpawnStatus(entry.Status)
		}
	}
	if !foundBlocked {
		t.Fatalf("dependent reservation still appears active: %+v", entries)
	}
	started := time.Now().UTC()
	active := colony.ColonyState{State: colony.StateEXECUTING, CurrentPhase: 2, BuildStartedAt: &started, Plan: colony.Plan{Phases: []colony.Phase{{ID: 1, Status: colony.PhaseCompleted}, phase}}}
	active.Plan.Phases[1].Status = colony.PhaseInProgress
	if err := store.SaveJSON("COLONY_STATE.json", active); err != nil {
		t.Fatal(err)
	}
	previous := active
	previous.State = colony.StateREADY
	previous.Plan.Phases = append([]colony.Phase(nil), active.Plan.Phases...)
	previous.Plan.Phases[1].Status = colony.PhaseReady
	rollbackCodexBuildFailure(previous, 2, started, context.DeadlineExceeded)
	if message := stderr.(*bytes.Buffer).String(); !strings.Contains(message, "working-tree edits were not rolled back") || !strings.Contains(message, "uncredited-draft.py") {
		t.Fatalf("rollback omitted surviving draft: %s", message)
	}
	report := retainedBuildDraftReport(root)
	if !strings.Contains(report, "uncredited-draft.py") || !strings.Contains(report, "uncredited drafts") || !strings.Contains(report, "pre-existing") {
		t.Fatalf("misleading draft report: %s", report)
	}
	if body, err := os.ReadFile(filepath.Join(root, "uncredited-draft.py")); err != nil || string(body) != "draft survives\n" {
		t.Fatalf("draft was altered: %q %v", body, err)
	}
	summary, err := readWaveSummary(2)
	if err != nil {
		t.Fatal(err)
	}
	if summary.TotalDispatched != 2 {
		t.Fatalf("blocked job counted as dispatched: %+v", summary)
	}
}

func TestCalVaultAcceptedTaskCreditControlsGroupedDependencies(t *testing.T) {
	ids := []string{"2.2", "2.3", "2.4"}
	phase := colony.Phase{Tasks: []colony.Task{
		{ID: &ids[0], SemanticID: "proof"}, {ID: &ids[1], SemanticID: "apply"},
		{ID: &ids[2], DependsOn: []string{"proof", "apply"}},
	}}
	dispatches := []codex.WorkerDispatch{{WorkerName: "Dependent", TaskID: ids[2]}}
	for _, tc := range []struct {
		name   string
		credit map[string]bool
		want   int
	}{
		{"no receipts", map[string]bool{}, 0},
		{"only one prerequisite", map[string]bool{"proof": true}, 0},
		{"both accepted", map[string]bool{"proof": true, "apply": true}, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ready, _ := partitionReadyBuildDispatches(phase, dispatches, tc.credit)
			if len(ready) != tc.want {
				t.Fatalf("ready=%v", ready)
			}
		})
	}
}

func TestCalVaultPartialReceiptsUnlockOnlyProvenPrerequisites(t *testing.T) {
	root := t.TempDir()
	phase := colony.Phase{ID: 1, Tasks: []colony.Task{receiptTestTask("1.1", "first", "a.go"), receiptTestTask("1.2", "second", "b.go")}}
	phase.Tasks[0].SemanticID = "first"
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package example\n"), 0600); err != nil {
		t.Fatal(err)
	}
	original := codexBuildDispatch{Name: "Grouped", CoveredTaskIDs: []string{"1.1", "1.2"}}
	result := codex.DispatchResult{WorkerName: "Grouped", Status: "timeout", WorkerResult: &codex.WorkerResult{
		Status: "timeout", FilesModified: []string{"a.go"}, TaskReceipts: []codex.TaskReceipt{{TaskID: "1.1", Status: codex.TaskReceiptStatusCompleted, Summary: "first finished", FilesModified: []string{"a.go"}, Handoff: passingHandoff("go test ./...")}},
	}}
	credit := map[string]bool{}
	acceptBuildWaveCredit(root, phase, []codexBuildDispatch{original}, []codex.DispatchResult{result}, nil, credit, false)
	if !credit["1.1"] || !credit["first"] || credit["1.2"] {
		t.Fatalf("wrong partial credit: %v", credit)
	}
}

func TestCalVaultDependencyReferencesNormalizeWhitespace(t *testing.T) {
	setupBuildFlowTest(t)
	ids := []string{"1.1", "1.2"}
	phase := colony.Phase{Tasks: []colony.Task{
		{ID: &ids[0], SemanticID: " prove  transfer ", Status: colony.TaskCompleted},
		{ID: &ids[1], DependsOn: []string{"  prove \ttransfer  "}},
	}}
	// Seed through the same path used for already-credited prerequisites.
	credit := buildDependencyCredit(phase)
	ready, blocked := partitionReadyBuildDispatches(phase, []codex.WorkerDispatch{{TaskID: ids[1]}}, credit)
	if len(ready) != 1 || len(blocked) != 0 {
		t.Fatalf("accepted whitespace alias blocked: ready=%v blocked=%v", ready, blocked)
	}
	// Grouped internal references obey the same normalization.
	ready, blocked = partitionReadyBuildDispatches(phase, []codex.WorkerDispatch{{TaskID: ids[0], CoveredTaskIDs: ids}}, nil)
	if len(ready) != 1 || len(blocked) != 0 {
		t.Fatalf("internal whitespace alias blocked: ready=%v blocked=%v", ready, blocked)
	}
}

func TestCalVaultPriorUnnamedTaskCannotCreditCurrentSemanticAlias(t *testing.T) {
	setupBuildFlowTest(t)
	ids := []string{"2.1", "2.2"}
	current := colony.Phase{ID: 2, Tasks: []colony.Task{
		{ID: &ids[0], SemanticID: "task-1", Status: colony.TaskPending},
		{ID: &ids[1], DependsOn: []string{"task-1"}, Status: colony.TaskPending},
	}}
	previous := colony.Phase{ID: 1, Tasks: []colony.Task{{Status: colony.TaskCompleted}}}
	state := colony.ColonyState{Plan: colony.Plan{Phases: []colony.Phase{previous, current}}}
	if _, err := colony.NewTaskReferenceIndex(state.Plan.Phases); err != nil {
		t.Fatalf("fixture must be canonically unambiguous: %v", err)
	}
	if err := store.SaveJSON("COLONY_STATE.json", state); err != nil {
		t.Fatal(err)
	}
	credit := buildDependencyCredit(current)
	// The current prerequisite timed out without an accepted task receipt.
	acceptBuildWaveCredit(t.TempDir(), current, []codexBuildDispatch{{Name: "Predecessor", TaskID: ids[0]}}, []codex.DispatchResult{{WorkerName: "Predecessor", Status: "timeout"}}, nil, credit, false)
	ready, blocked := partitionReadyBuildDispatches(current, []codex.WorkerDispatch{{WorkerName: "Dependent", TaskID: ids[1]}}, credit)
	if len(ready) != 0 || len(blocked) != 1 || credit["task-1"] {
		t.Fatalf("foreign synthetic task unlocked dependency: credit=%v ready=%v blocked=%v", credit, ready, blocked)
	}
}
