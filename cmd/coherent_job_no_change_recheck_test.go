package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// seedRecheckableProject writes a tiny, self-contained Go project into root so
// that the check a receipt names ("go test ./...") is a real command with a
// real, deterministic result when this program re-runs it there. passes picks
// whether that project's own test succeeds or fails.
//
// The file thing.go is also the file the task in these tests declares, so the
// existing "the files this task names are really present" bound is satisfied
// in every case -- which is the point: these tests must fail or pass purely on
// what the re-run did, never on whether a file was there.
func seedRecheckableProject(t *testing.T, root string, passes bool) {
	t.Helper()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("create project dir: %v", err)
	}
	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatalf("seed %s: %v", name, err)
		}
	}
	write("go.mod", "module example.com/recheck\n\ngo 1.21\n")
	write("thing.go", "package thing\n")
	if passes {
		write("thing_test.go", "package thing\n\nimport \"testing\"\n\nfunc TestThing(t *testing.T) {}\n")
		return
	}
	write("thing_test.go", "package thing\n\nimport \"testing\"\n\nfunc TestThing(t *testing.T) { t.Fatal(\"the thing was not already true\") }\n")
}

func noChangeRecheckPhase() colony.Phase {
	return colony.Phase{ID: 1, Tasks: []colony.Task{receiptTestTask("1.1", "make sure the thing is already true", "thing.go")}}
}

func noChangeRecheckDispatch() codexBuildDispatch {
	return codexBuildDispatch{Name: "Mason-1", Status: "failed", CoveredTaskIDs: []string{"1.1"}}
}

func noChangeReceiptNaming(commands ...string) codex.TaskReceipt {
	return codex.TaskReceipt{
		TaskID:  "1.1",
		Status:  codex.TaskReceiptStatusCompletedNoChange,
		Summary: "the thing was already true; nothing needed changing",
		Handoff: passingHandoff(commands...),
	}
}

// TestNoChangeReceiptCreditedWhenItsNamedCheckPassesOnReRun is one half of the
// owner's WR-15 ruling (deferred-items.md, 2026-08-27): when a worker says "I
// looked and nothing needed changing", the program runs the check that worker
// named and believes the RESULT. Here the result is a genuine pass, so the
// task is credited -- through the shared stage-2 boundary and through the lane
// a real build actually uses.
func TestNoChangeReceiptCreditedWhenItsNamedCheckPassesOnReRun(t *testing.T) {
	phase := noChangeRecheckPhase()
	dispatch := noChangeRecheckDispatch()
	root := t.TempDir()
	seedRecheckableProject(t, root, true)

	receipt := noChangeReceiptNaming("go test ./...")

	admission, violations := admitCoherentJobTaskReceipts(root, phase, dispatch, nil, []codex.TaskReceipt{receipt})
	if len(violations) != 0 {
		t.Fatalf("expected an honest no-change receipt to be admitted, got %v", violationRules(violations))
	}
	_, completed, finalViolations := finalizeCoherentJobTaskReceiptEvidence(root, phase, dispatch, admission)
	if len(finalViolations) != 0 {
		t.Fatalf("expected no refusals when the named check really passes on re-run, got %v", violationRules(finalViolations))
	}
	if len(completed) != 1 || completed[0] != "1.1" {
		t.Fatalf("expected task 1.1 credited after the program re-ran its check and the check passed, got %v", completed)
	}

	// Same receipt through the lane both build paths use.
	dispatchWithReceipt := dispatch
	dispatchWithReceipt.TaskReceipts = []codex.TaskReceipt{receipt}
	resolved := resolveCoherentJobDispatchReceipts(root, phase, []codexBuildDispatch{dispatchWithReceipt})
	if len(resolved[0].CompletedTaskIDs) != 1 || resolved[0].CompletedTaskIDs[0] != "1.1" {
		t.Fatalf("the in-repo lane did not credit a no-change receipt whose check the program re-ran and saw pass, got %v", resolved[0].CompletedTaskIDs)
	}
}

// TestNoChangeReceiptRefusedWhenItsNamedCheckFailsOnReRun is the other half,
// and the one the ruling exists for. Everything the worker can write about
// itself is identical to the passing case, and every file the task declares is
// present -- the old bar. The only difference is that the check the worker
// named genuinely fails when the program runs it. No credit.
func TestNoChangeReceiptRefusedWhenItsNamedCheckFailsOnReRun(t *testing.T) {
	phase := noChangeRecheckPhase()
	dispatch := noChangeRecheckDispatch()
	root := t.TempDir()
	seedRecheckableProject(t, root, false)

	// The old bar is satisfied: the file this task declares really is there.
	if _, err := os.Stat(filepath.Join(root, "thing.go")); err != nil {
		t.Fatalf("the file this task declares must exist for this test to prove anything: %v", err)
	}

	receipt := noChangeReceiptNaming("go test ./...")

	admission, violations := admitCoherentJobTaskReceipts(root, phase, dispatch, nil, []codex.TaskReceipt{receipt})
	if len(violations) != 0 {
		t.Fatalf("stage 1 must still admit this receipt -- it is well-formed; got %v", violationRules(violations))
	}
	claims, completed, finalViolations := finalizeCoherentJobTaskReceiptEvidence(root, phase, dispatch, admission)
	if len(completed) != 0 {
		t.Fatalf("a no-change receipt was credited even though the check it named FAILS when the program runs it, got %v", completed)
	}
	if len(claims) != 0 {
		t.Fatalf("expected no task claims, got %d", len(claims))
	}
	if !containsRule(finalViolations, violationRuleTaskReceiptNoChangeCheckFailed) {
		t.Fatalf("expected refusal rule %q, got %v", violationRuleTaskReceiptNoChangeCheckFailed, violationRules(finalViolations))
	}

	dispatchWithReceipt := dispatch
	dispatchWithReceipt.TaskReceipts = []codex.TaskReceipt{receipt}
	resolved := resolveCoherentJobDispatchReceipts(root, phase, []codexBuildDispatch{dispatchWithReceipt})
	if len(resolved[0].CompletedTaskIDs) != 0 {
		t.Fatalf("the in-repo lane credited %v from a no-change receipt whose own check fails", resolved[0].CompletedTaskIDs)
	}
	if _, ok := completedBuildTaskIDs(resolved)["1.1"]; ok {
		t.Fatal("task 1.1 reached the credited set although the check it named fails when actually run")
	}
}

// TestNoChangeReceiptCheckIsNeverHandedToAShell keeps Phase 193's CR-01 ruling
// intact at this new call site: a check named by a worker is untrusted text.
// It is only ever executed when it names a recognised build/test runner and
// carries no shell metacharacters, and it is run as a plain argument list --
// never through a shell. Both refused shapes below would create a sentinel
// file if this program had handed them to a shell; the test fails if either
// sentinel appears.
func TestNoChangeReceiptCheckIsNeverHandedToAShell(t *testing.T) {
	for _, tc := range []struct {
		name    string
		command func(sentinel string) string
	}{
		{
			name:    "not a build or test runner",
			command: func(sentinel string) string { return "touch " + sentinel },
		},
		{
			name:    "chains a second command",
			command: func(sentinel string) string { return "go test ./... && touch " + sentinel },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			phase := noChangeRecheckPhase()
			dispatch := noChangeRecheckDispatch()
			root := t.TempDir()
			// The project here PASSES its own tests, so the only reason this
			// receipt can be refused is the shape of the command it named.
			seedRecheckableProject(t, root, true)
			sentinel := filepath.Join(t.TempDir(), "executed-through-a-shell")

			receipt := noChangeReceiptNaming(tc.command(sentinel))

			admission, _ := admitCoherentJobTaskReceipts(root, phase, dispatch, nil, []codex.TaskReceipt{receipt})
			_, completed, finalViolations := finalizeCoherentJobTaskReceiptEvidence(root, phase, dispatch, admission)

			if _, err := os.Stat(sentinel); err == nil {
				t.Fatal("the worker-named command was actually executed through a shell -- Phase 193's CR-01 ruling (argv only, allowlisted runners only) has been regressed at the receipt boundary")
			}
			if len(completed) != 0 {
				t.Fatalf("a no-change receipt was credited from a command this program refuses to run, got %v", completed)
			}
			if !containsRule(finalViolations, violationRuleTaskReceiptNoChangeCommandRefused) {
				t.Fatalf("expected refusal rule %q, got %v", violationRuleTaskReceiptNoChangeCommandRefused, violationRules(finalViolations))
			}
		})
	}
}

// TestNoChangeRefusalsAreOwnerVisible proves the refusal travels the same
// owner-facing path every other receipt refusal uses, in plain words, rather
// than being dropped silently.
func TestNoChangeRefusalsAreOwnerVisible(t *testing.T) {
	phase := noChangeRecheckPhase()
	dispatch := noChangeRecheckDispatch()
	root := t.TempDir()
	seedRecheckableProject(t, root, false)
	dispatch.TaskReceipts = []codex.TaskReceipt{noChangeReceiptNaming("go test ./...")}

	restore := stderr
	t.Cleanup(func() { stderr = restore })
	stderr = &bytes.Buffer{}
	resolveCoherentJobDispatchReceipts(root, phase, []codexBuildDispatch{dispatch})

	output := stderr.(*bytes.Buffer).String()
	if strings.TrimSpace(output) == "" {
		t.Fatal("a refused no-change receipt produced no owner-facing message at all")
	}
	for _, want := range []string{"Mason-1", "1.1", "go test ./..."} {
		if !strings.Contains(output, want) {
			t.Fatalf("owner-facing refusal does not mention %q; got:\n%s", want, output)
		}
	}
}
