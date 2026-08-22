package cmd

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// TestTimedOutCheckIsAFailedCheck proves a verification step that hits its
// timeout is reported as failed, not skipped, contributes a blocking issue
// naming the check and the timeout duration, and is eligible for the fix
// path -- while a step whose command simply never resolved stays skipped and
// is NOT eligible. The two must be distinguishable.
func TestTimedOutCheckIsAFailedCheck(t *testing.T) {
	t.Run("a step that times out is failed, not skipped", func(t *testing.T) {
		step := runVerificationStep(context.Background(), ".", "tests", false, "sleep 2", 50*time.Millisecond)
		if !step.TimedOut {
			t.Fatalf("expected TimedOut=true, got %+v", step)
		}
		if step.Passed {
			t.Fatalf("expected Passed=false for a timed-out step, got %+v", step)
		}
		if step.Skipped {
			t.Fatalf("expected Skipped=false for a timed-out step, got %+v", step)
		}
		if step.ErrorClass != ErrorClassTimeout {
			t.Fatalf("expected ErrorClass=%q, got %q", ErrorClassTimeout, step.ErrorClass)
		}
		if !strings.Contains(step.Summary, "timed out") {
			t.Fatalf("expected Summary to name the timeout, got %q", step.Summary)
		}
	})

	t.Run("a timed-out step contributes a blocking issue naming the check and duration", func(t *testing.T) {
		saveGlobals(t)
		s, root := newTestStore(t)
		store = s
		writeAgentsVerificationCommands(t, root, "- build: true", "- types: true", "- lint: true", "- tests: sh -c 'sleep 2'")
		phase := colony.Phase{ID: 1, Name: "Timeout fixture"}
		floor := runDeterministicFloor(context.Background(), root, phase, codexContinueManifest{}, codexWatcherVerification{}, 50*time.Millisecond)
		if floor.ChecksPassed {
			t.Fatalf("expected a timed-out check to fail the floor, got %+v", floor)
		}
		found := false
		for _, issue := range floor.BlockingIssues {
			if strings.Contains(issue, "tests") && strings.Contains(issue, "timed out") {
				found = true
			}
		}
		if !found {
			t.Fatalf("expected a blocking issue naming the tests check and the timeout, got %v", floor.BlockingIssues)
		}
	})

	t.Run("a step with no resolved command stays skipped and is not eligible", func(t *testing.T) {
		step := runVerificationStep(context.Background(), ".", "lint", false, "", 5*time.Second)
		if !step.Skipped {
			t.Fatalf("expected Skipped=true when no command resolved, got %+v", step)
		}
		if step.TimedOut {
			t.Fatalf("expected TimedOut=false when no command resolved, got %+v", step)
		}
		if !step.Passed {
			t.Fatalf("expected an optional check with no command to Pass (D-01 enrichment), got %+v", step)
		}
	})
}

// TestFailureIndexIsCompactAndNamesTheImplicatedTasks proves the index for a
// failing check with a long output is bounded in size (a small proportion of
// the raw output, not a fixed byte count pinned to one fixture), contains
// the failing check's name and the first distinct failure locations, and
// names the task whose reported changed files are implicated.
func TestFailureIndexIsCompactAndNamesTheImplicatedTasks(t *testing.T) {
	var raw strings.Builder
	raw.WriteString("cmd/widget.go:42: assertion failed: expected true got false\n")
	for i := 0; i < 500; i++ {
		fmt.Fprintf(&raw, "line %d: unrelated failure detail padding padding padding padding\n", i)
	}
	output := raw.String()

	claims := codexBuildClaims{
		TaskClaims: []codexBuildTaskClaim{
			{TaskID: "1.1", FilesModified: []string{"cmd/widget.go"}},
		},
	}
	step := codexVerificationStep{Name: "tests", Command: "go test ./...", ExitCode: 1, Output: output, Summary: "tests failed (exit 1)"}

	index := buildCheckFailureIndex(step, claims, colony.Phase{})

	if index.Check != "tests" {
		t.Fatalf("Check = %q, want %q", index.Check, "tests")
	}
	if len(index.Excerpts) == 0 {
		t.Fatalf("expected at least one excerpt")
	}
	joined := strings.Join(index.Excerpts, "\n")
	// The bound is asserted as a proportion of the raw output, not a fixed
	// byte count -- this must hold regardless of how large a real failure
	// log happens to be.
	if len(joined) >= len(output)/10 {
		t.Fatalf("index is not a small proportion of the raw output: index=%d bytes, output=%d bytes", len(joined), len(output))
	}
	if !index.Truncated {
		t.Fatalf("expected Truncated=true for output far exceeding the excerpt bound")
	}
	if len(index.ImplicatedTaskIDs) != 1 || index.ImplicatedTaskIDs[0] != "1.1" {
		t.Fatalf("ImplicatedTaskIDs = %v, want [1.1]", index.ImplicatedTaskIDs)
	}
}

// TestFailureIndexNeverCarriesTheWholeLog proves the index for a large
// output is strictly smaller than that output and does not contain its
// tail.
func TestFailureIndexNeverCarriesTheWholeLog(t *testing.T) {
	var raw strings.Builder
	for i := 0; i < 1000; i++ {
		fmt.Fprintf(&raw, "unique failure line %d with some padding text here\n", i)
	}
	output := raw.String()
	step := codexVerificationStep{Name: "tests", Command: "go test ./...", ExitCode: 1, Output: output, Summary: "tests failed"}

	index := buildCheckFailureIndex(step, codexBuildClaims{}, colony.Phase{})

	joined := strings.Join(index.Excerpts, "\n")
	if len(joined) >= len(output) {
		t.Fatalf("index (%d bytes) is not strictly smaller than the raw output (%d bytes)", len(joined), len(output))
	}
	tail := "unique failure line 999"
	if strings.Contains(joined, tail) {
		t.Fatalf("index carries the tail of the raw output: %v", index.Excerpts)
	}
	if !index.Truncated {
		t.Fatalf("expected Truncated=true for an output this much larger than the excerpt bound")
	}
}
