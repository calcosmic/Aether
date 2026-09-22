package cmd

import (
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// twoTaskVerdictFixture builds the two-task phase, manifest, and options
// shared by every test in this file: two "completed", trusted, real
// dispatches -- so the only thing that can make a task unverified is the
// per-task criterion logic under test, not a missing dispatch or an
// untrusted manifest.
func twoTaskVerdictFixture(t *testing.T) (colony.Phase, string, string, codexContinueManifest) {
	t.Helper()
	phase := colony.Phase{
		ID:     1,
		Name:   "Two task phase",
		Status: colony.PhaseInProgress,
		Tasks: []colony.Task{
			{Goal: "Task one"},
			{Goal: "Task two"},
		},
	}
	taskID1 := buildTaskID(phase.Tasks[0], 0)
	taskID2 := buildTaskID(phase.Tasks[1], 1)

	manifest := codexContinueManifest{
		Present: true,
		Path:    "build/phase-1/manifest.json",
		Data: codexBuildManifest{
			Phase:        phase.ID,
			DispatchMode: "real",
			Dispatches: []codexBuildDispatch{
				{Stage: "wave", Caste: "builder", Name: "Forge-1", Task: "Task one", Status: "completed", TaskID: taskID1},
				{Stage: "wave", Caste: "builder", Name: "Forge-2", Task: "Task two", Status: "completed", TaskID: taskID2},
			},
		},
	}
	return phase, taskID1, taskID2, manifest
}

func TestOneFailingCriterionMarksOnlyItsOwnTask(t *testing.T) {
	phase, taskID1, taskID2, manifest := twoTaskVerdictFixture(t)

	criterionText := "the export button downloads a real file"
	verification := codexContinueVerificationReport{
		ChecksPassed: true,
		Passed:       true,
		Claims:       codexClaimVerification{Skipped: true},
		Criteria: []codexCriterionVerification{
			{
				Criterion:      criterionText,
				TaskID:         taskID1,
				Policy:         criterionEvidencePolicyBoundV1,
				Enforced:       true,
				Passed:         false,
				BlockingIssues: []string{"artifact export.csv was not claimed by the current build"},
			},
		},
	}

	assessment := assessCodexContinue(phase, manifest, verification, codexContinueOptions{}, time.Now().UTC())

	if len(assessment.Tasks) != 2 {
		t.Fatalf("expected 2 tasks in assessment, got %d: %+v", len(assessment.Tasks), assessment.Tasks)
	}
	task1 := taskAssessmentByID(t, assessment, taskID1)
	task2 := taskAssessmentByID(t, assessment, taskID2)

	if task1.Verified {
		t.Fatalf("expected task 1 (whose own criterion failed) to be Verified=false, got true: %+v", task1)
	}
	if !task2.Verified {
		t.Fatalf("expected task 2 (no criterion bound to it) to remain Verified=true, got false: %+v", task2)
	}
}

func TestPhaseLevelCriterionDoesNotUnverifyEveryTask(t *testing.T) {
	phase, taskID1, taskID2, manifest := twoTaskVerdictFixture(t)

	verification := codexContinueVerificationReport{
		ChecksPassed: true,
		Passed:       true,
		Claims:       codexClaimVerification{Skipped: true},
		Criteria: []codexCriterionVerification{
			{
				Criterion:      "the whole phase ships without a security regression",
				TaskID:         "",
				Policy:         criterionEvidencePolicyBoundV1,
				Enforced:       true,
				Passed:         false,
				BlockingIssues: []string{"security scan step failed"},
			},
		},
	}

	assessment := assessCodexContinue(phase, manifest, verification, codexContinueOptions{}, time.Now().UTC())

	task1 := taskAssessmentByID(t, assessment, taskID1)
	task2 := taskAssessmentByID(t, assessment, taskID2)

	if !task1.Verified {
		t.Fatalf("expected task 1 to stay Verified=true when only a phase-level criterion failed, got false: %+v", task1)
	}
	if !task2.Verified {
		t.Fatalf("expected task 2 to stay Verified=true when only a phase-level criterion failed, got false: %+v", task2)
	}
	if task1.Outcome == "implemented_unverified" || task2.Outcome == "implemented_unverified" {
		t.Fatalf("expected neither task to be implemented_unverified on account of a phase-level criterion, got task1=%q task2=%q", task1.Outcome, task2.Outcome)
	}
}

func TestFailingCriterionProducesOneNamedFailureNotTwo(t *testing.T) {
	phase, taskID1, _, manifest := twoTaskVerdictFixture(t)

	criterionText := "the export button downloads a real file"
	verification := codexContinueVerificationReport{
		ChecksPassed: true,
		Passed:       true,
		Claims:       codexClaimVerification{Skipped: true},
		Criteria: []codexCriterionVerification{
			{
				Criterion:      criterionText,
				TaskID:         taskID1,
				Policy:         criterionEvidencePolicyBoundV1,
				Enforced:       true,
				Passed:         false,
				BlockingIssues: []string{"artifact export.csv was not claimed by the current build"},
			},
		},
	}

	assessment := assessCodexContinue(phase, manifest, verification, codexContinueOptions{}, time.Now().UTC())

	matches := 0
	for _, issue := range assessment.BlockingIssues {
		if strings.Contains(issue, criterionText) {
			matches++
		}
	}
	if matches != 1 {
		t.Fatalf("expected exactly one blocking issue naming the failing criterion, got %d: %+v", matches, assessment.BlockingIssues)
	}
	for _, issue := range assessment.BlockingIssues {
		if strings.Contains(issue, "verification passed but no implementation evidence was recorded") {
			t.Fatalf("expected the generic no-implementation-evidence line to be absent when a criterion names the cause, got %+v", assessment.BlockingIssues)
		}
	}
}

func taskAssessmentByID(t *testing.T, assessment codexContinueAssessment, taskID string) codexContinueTaskAssessment {
	t.Helper()
	for _, task := range assessment.Tasks {
		if task.TaskID == taskID {
			return task
		}
	}
	t.Fatalf("no task assessment found for task id %q in %+v", taskID, assessment.Tasks)
	return codexContinueTaskAssessment{}
}
