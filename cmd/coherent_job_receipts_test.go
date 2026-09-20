package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// receiptTestTask builds a colony.Task fixture with an explicit ID and
// optional file hints, mirroring the shape declaredPathsForTask expects.
func receiptTestTask(id, goal string, hints ...string) colony.Task {
	taskID := id
	return colony.Task{ID: &taskID, Goal: goal, Status: colony.TaskPending, Hints: hints}
}

func passingHandoff(commands ...string) codex.WorkerHandoff {
	return codex.WorkerHandoff{VerificationStatus: "pass", CommandsRun: commands}
}

func violationRules(violations []contractViolation) []string {
	rules := make([]string, 0, len(violations))
	for _, v := range violations {
		rules = append(rules, v.Rule)
	}
	sort.Strings(rules)
	return rules
}

func containsRule(violations []contractViolation, rule string) bool {
	for _, v := range violations {
		if v.Rule == rule {
			return true
		}
	}
	return false
}

// TestCoherentJobReceiptAdmission is the adversarial table for stage 1
// (D-08, D-09). Admission never reads root and never populates
// CompletedTaskIDs -- it only decides which receipts are even trustworthy
// candidates.
func TestCoherentJobReceiptAdmission(t *testing.T) {
	basePhase := colony.Phase{
		ID: 1,
		Tasks: []colony.Task{
			receiptTestTask("1.1", "step one", "a.go"),
			receiptTestTask("1.2", "step two", "b.go"),
			receiptTestTask("1.3", "not covered by this dispatch"),
		},
	}
	dispatch := codexBuildDispatch{
		Name:           "Hammer-1",
		CoveredTaskIDs: []string{"1.1", "1.2"},
		Outputs:        []string{"a.go", "b.go"},
	}
	validReceipt := func(taskID, summary string, files ...string) codex.TaskReceipt {
		return codex.TaskReceipt{
			TaskID:        taskID,
			Status:        codex.TaskReceiptStatusCompleted,
			Summary:       summary,
			FilesModified: files,
			Handoff:       passingHandoff("go test ./..."),
		}
	}

	tests := []struct {
		name           string
		receipts       []codex.TaskReceipt
		wantCandidates []string // task IDs expected to be admitted
		wantRule       string   // at least one violation must carry this rule
	}{
		{
			name:           "well-formed receipt is admitted",
			receipts:       []codex.TaskReceipt{validReceipt("1.1", "did step one", "a.go")},
			wantCandidates: []string{"1.1"},
		},
		{
			name:     "missing task_id is refused by name",
			receipts: []codex.TaskReceipt{{Status: codex.TaskReceiptStatusCompleted, Summary: "x", Handoff: passingHandoff("go test ./...")}},
			wantRule: violationRuleTaskReceiptIDRequired,
		},
		{
			name: "duplicate task_id admits only the first and refuses the rest",
			receipts: []codex.TaskReceipt{
				validReceipt("1.1", "did step one", "a.go"),
				validReceipt("1.1", "did step one again", "a.go"),
			},
			wantCandidates: []string{"1.1"},
			wantRule:       violationRuleTaskReceiptDuplicate,
		},
		{
			name:     "unknown task_id is refused by name",
			receipts: []codex.TaskReceipt{validReceipt("9.9", "did something", "a.go")},
			wantRule: violationRuleTaskReceiptUnknown,
		},
		{
			name:     "a real task not covered by this dispatch is out of scope",
			receipts: []codex.TaskReceipt{validReceipt("1.3", "did the uncovered task", "a.go")},
			wantRule: violationRuleTaskReceiptOutOfScope,
		},
		{
			name: "a failed-status receipt is refused, never admitted as a candidate",
			receipts: []codex.TaskReceipt{{
				TaskID: "1.1", Status: "failed", Summary: "tried and failed",
				Handoff: passingHandoff("go test ./..."),
			}},
			wantRule: violationRuleTaskReceiptStatusInvalid,
		},
		{
			name: "an empty summary is refused",
			receipts: []codex.TaskReceipt{{
				TaskID: "1.1", Status: codex.TaskReceiptStatusCompleted, Summary: "",
				FilesModified: []string{"a.go"}, Handoff: passingHandoff("go test ./..."),
			}},
			wantRule: violationRuleTaskReceiptSummaryRequired,
		},
		{
			name: "an invalid verification_status enum is refused",
			receipts: []codex.TaskReceipt{{
				TaskID: "1.1", Status: codex.TaskReceiptStatusCompleted, Summary: "did it",
				FilesModified: []string{"a.go"},
				Handoff:       codex.WorkerHandoff{VerificationStatus: "bogus", CommandsRun: []string{"go test ./..."}},
			}},
			wantRule: violationRuleTaskReceiptHandoffInvalid,
		},
		{
			name: "a receipt with no passing concrete verification is unevidenced",
			receipts: []codex.TaskReceipt{{
				TaskID: "1.1", Status: codex.TaskReceiptStatusCompleted, Summary: "did it",
				FilesModified: []string{"a.go"},
				Handoff:       codex.WorkerHandoff{VerificationStatus: "pass"},
			}},
			wantRule: violationRuleTaskReceiptUnevidenced,
		},
		{
			name: "an absolute path is path-laundering, not a real claim",
			receipts: []codex.TaskReceipt{{
				TaskID: "1.1", Status: codex.TaskReceiptStatusCompleted, Summary: "did it",
				FilesModified: []string{"/etc/passwd"},
				Handoff:       passingHandoff("go test ./..."),
			}},
			wantRule: violationRuleTaskReceiptPathLaundered,
		},
		{
			name: "a parent-escaping path is path-laundering, not a real claim",
			receipts: []codex.TaskReceipt{{
				TaskID: "1.1", Status: codex.TaskReceiptStatusCompleted, Summary: "did it",
				FilesModified: []string{"../../etc/passwd"},
				Handoff:       passingHandoff("go test ./..."),
			}},
			wantRule: violationRuleTaskReceiptPathLaundered,
		},
		{
			name:     "a path the worker's own result never reported touching is refused",
			receipts: []codex.TaskReceipt{validReceipt("1.1", "did step one", "c.go")},
			wantRule: violationRuleTaskReceiptPathOutOfClaims,
		},
		{
			name:     "a receipt touching none of the task's own declared files is a requirement mismatch",
			receipts: []codex.TaskReceipt{validReceipt("1.1", "did step one", "b.go")},
			wantRule: violationRuleTaskReceiptRequirementMismatch,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			admission, violations := admitCoherentJobTaskReceipts("", basePhase, dispatch, dispatch.Outputs, tc.receipts)
			if len(admission.Candidates) != 0 {
				// codexBuildClaims must never carry CompletedTaskIDs -- admission has no such field to check,
				// but assert the type itself has no completion-shaped output leaking through Candidates.
				for _, c := range admission.Candidates {
					if c.TaskID == "" {
						t.Fatalf("admitted candidate with empty TaskID: %+v", c)
					}
				}
			}
			gotCandidates := make([]string, 0, len(admission.Candidates))
			for _, c := range admission.Candidates {
				gotCandidates = append(gotCandidates, c.TaskID)
			}
			sort.Strings(gotCandidates)
			wantCandidates := append([]string{}, tc.wantCandidates...)
			sort.Strings(wantCandidates)
			if fmt.Sprint(gotCandidates) != fmt.Sprint(wantCandidates) {
				t.Fatalf("candidates = %v, want %v (violations: %v)", gotCandidates, wantCandidates, violationRules(violations))
			}
			if tc.wantRule != "" && !containsRule(violations, tc.wantRule) {
				t.Fatalf("violations %v do not contain expected rule %q", violationRules(violations), tc.wantRule)
			}
		})
	}
}

// TestCoherentJobReceiptFinalization proves stage 2 only credits candidates
// whose claimed files are actually present in the root checkout right now,
// and that CompletedTaskIDs is populated exclusively here (D-08, D-09).
func TestCoherentJobReceiptFinalization(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.go"), []byte("package a\n"), 0644); err != nil {
		t.Fatalf("write fixture file: %v", err)
	}
	phase := colony.Phase{ID: 1, Tasks: []colony.Task{
		receiptTestTask("1.1", "step one", "a.go"),
		receiptTestTask("1.2", "step two", "b.go"),
	}}
	dispatch := codexBuildDispatch{Name: "Hammer-1", CoveredTaskIDs: []string{"1.1", "1.2"}}

	t.Run("root-backed candidate is credited with artifact evidence", func(t *testing.T) {
		admission := coherentJobReceiptAdmission{Candidates: []coherentJobReceiptCandidate{
			{TaskID: "1.1", Claim: codexBuildTaskClaim{TaskID: "1.1", FilesModified: []string{"a.go"}}},
		}}
		claims, completed, violations := finalizeCoherentJobTaskReceiptEvidence(root, phase, dispatch, admission)
		if len(violations) != 0 {
			t.Fatalf("unexpected violations for a root-backed candidate: %+v", violations)
		}
		if fmt.Sprint(completed) != fmt.Sprint([]string{"1.1"}) {
			t.Fatalf("CompletedTaskIDs = %v, want [1.1]", completed)
		}
		if len(claims) != 1 || claims[0].TaskID != "1.1" || len(claims[0].ArtifactEvidence) != 1 {
			t.Fatalf("claims = %+v, want one root-evidenced claim for 1.1", claims)
		}
		if claims[0].ArtifactEvidence[0].SHA256 == "" {
			t.Fatalf("artifact evidence has no computed hash: %+v", claims[0].ArtifactEvidence[0])
		}
	})

	t.Run("a candidate whose file is missing from root is not credited", func(t *testing.T) {
		admission := coherentJobReceiptAdmission{Candidates: []coherentJobReceiptCandidate{
			{TaskID: "1.2", Claim: codexBuildTaskClaim{TaskID: "1.2", FilesModified: []string{"b.go"}}},
		}}
		claims, completed, violations := finalizeCoherentJobTaskReceiptEvidence(root, phase, dispatch, admission)
		if len(completed) != 0 || len(claims) != 0 {
			t.Fatalf("a missing-artifact candidate must not be credited: completed=%v claims=%+v", completed, claims)
		}
		if !containsRule(violations, violationRuleTaskReceiptRootEvidenceMissing) {
			t.Fatalf("violations %v do not name the missing root evidence", violationRules(violations))
		}
	})

	t.Run("a mix of one root-backed and one missing candidate credits only the root-backed one", func(t *testing.T) {
		admission := coherentJobReceiptAdmission{Candidates: []coherentJobReceiptCandidate{
			{TaskID: "1.1", Claim: codexBuildTaskClaim{TaskID: "1.1", FilesModified: []string{"a.go"}}},
			{TaskID: "1.2", Claim: codexBuildTaskClaim{TaskID: "1.2", FilesModified: []string{"b.go"}}},
		}}
		claims, completed, _ := finalizeCoherentJobTaskReceiptEvidence(root, phase, dispatch, admission)
		if fmt.Sprint(completed) != fmt.Sprint([]string{"1.1"}) {
			t.Fatalf("CompletedTaskIDs = %v, want exactly [1.1]", completed)
		}
		if len(claims) != 1 || claims[0].TaskID != "1.1" {
			t.Fatalf("claims = %+v, want exactly one claim for 1.1", claims)
		}
	})
}

// TestNativeWorkerTransportKeepsTaskReceiptsOnFailure locks D-08/D-09's
// prerequisite: a failed/interrupted native worker's task-specific receipts
// must survive the internal-worker-adapter transport and the durable
// build-attempt worker-run conversion into an external-shaped completion
// packet, not be silently dropped because the worker's OWN status was not a
// success.
func TestNativeWorkerTransportKeepsTaskReceiptsOnFailure(t *testing.T) {
	receipt := codex.TaskReceipt{
		TaskID: "1.1", Status: codex.TaskReceiptStatusCompleted, Summary: "finished step one before crashing",
		FilesModified: []string{"a.go"},
		Handoff:       passingHandoff("go test ./..."),
	}
	result := codex.WorkerResult{
		WorkerName: "Hammer-1", Caste: "builder", TaskID: "1.1",
		Status: "failed", Summary: "crashed after step one",
		TaskReceipts: []codex.TaskReceipt{receipt},
	}

	mapped := mapInternalWorkerResult(result, fmt.Errorf("boom"))
	if len(mapped.TaskReceipts) != 1 || mapped.TaskReceipts[0].TaskID != "1.1" {
		t.Fatalf("mapInternalWorkerResult dropped task receipts on a failed result: %+v", mapped.TaskReceipts)
	}

	record := buildAttemptRecord{
		PlanManifest: &codexBuildManifest{Dispatches: []codexBuildDispatch{
			{Name: "Hammer-1", Caste: "builder", TaskID: "1.1", Status: "planned"},
		}},
		WorkerRuns: []buildAttemptWorkerRun{{
			WorkerName: "Hammer-1", TaskID: "1.1", Status: buildWorkerFailed,
			Result: mapped, ResultSHA256: "test-fixture-nonempty-digest",
		}},
	}
	completion, ok := buildCompletionFromWorkerRuns(record)
	if !ok {
		t.Fatalf("buildCompletionFromWorkerRuns did not produce a completion for one terminal worker run")
	}
	if len(completion.Dispatches) != 1 || len(completion.Dispatches[0].TaskReceipts) != 1 {
		t.Fatalf("buildCompletionFromWorkerRuns dropped task receipts converting a failed native result into an external-shaped packet: %+v", completion.Dispatches)
	}
	if completion.Dispatches[0].TaskReceipts[0].TaskID != "1.1" {
		t.Fatalf("buildCompletionFromWorkerRuns carried the wrong receipt through: %+v", completion.Dispatches[0].TaskReceipts)
	}
}
