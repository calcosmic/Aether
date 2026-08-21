package cmd

import (
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/codex"
)

// externalResultsForManifest builds a well-formed completion packet body: one
// terminal, handoff-carrying result per dispatch in the manifest.
func externalResultsForManifest(manifest codexBuildManifest) []codexExternalBuildWorkerResult {
	results := make([]codexExternalBuildWorkerResult, 0, len(manifest.Dispatches))
	for _, dispatch := range manifest.Dispatches {
		worker := codexExternalBuildWorkerResult{
			Stage:         dispatch.Stage,
			Wave:          dispatch.Wave,
			ExecutionWave: normalizedDispatchWave(dispatch),
			Caste:         dispatch.Caste,
			Name:          dispatch.Name,
			TaskID:        dispatch.TaskID,
			Status:        "completed",
			Summary:       dispatch.Name + " completed externally",
			Handoff: codex.WorkerHandoff{
				CommandsRun:            []string{"go test ./..."},
				VerificationStatus:     "pass",
				NextWorkerInstructions: []string{dispatch.Name + " done"},
			},
		}
		if dispatch.Caste == "builder" {
			worker.FilesModified = []string{"external-evidence.txt"}
		}
		results = append(results, worker)
	}
	return results
}

// TestForcedRedispatchAfterBuiltIsNotADeadlock is the regression lock for the
// stage/finalize deadlock reported from a live colony on 2026-08-21.
//
// Reproduction, which is an ordinary supported sequence: build a phase, let it
// finalize, then redispatch it with --force (a reviewer found something; the
// user re-ran the build). The colony still reads BUILT from the first attempt
// while the second attempt has committed nothing — its completion digest is
// empty, its claims are nil, and its dispatches are still `planned`.
//
// Finalize used to route that second attempt into committed-attempt
// reconciliation purely because the colony sat at BUILT, and refuse:
//
//	"attempt X is already committed, so this different completion packet
//	 cannot replace it ... run `aether continue`"
//
// while continue refused in the other direction:
//
//	"no completed worker dispatches found -- build did not produce
//	 verifiable results"
//
// Neither exit worked and each named the other. Escape required `aether
// recover`, out of band.
//
// The fix is to stop misclassifying the attempt: reconciliation now requires
// the attempt's OWN terminal evidence, so a fresh attempt finalizes normally
// and the stale BUILT is replaced, which is what --force is for.
func TestForcedRedispatchAfterBuiltIsNotADeadlock(t *testing.T) {
	root := setupExternalBuildAttemptTest(t)

	_, completion := prepareExternalBuildCompletion(t, root)
	if _, state, _, _, err := runCodexBuildFinalize(root, 1, completion, false); err != nil {
		t.Fatalf("first finalize: %v", err)
	} else if state.State != "BUILT" {
		t.Fatalf("fixture did not reach BUILT, so the deadlock's precondition is absent: state=%s", state.State)
	}

	result, _, _, _, err := runCodexBuildPlanOnlyWithOptions(root, 1, nil, codexBuildOptions{Force: true})
	if err != nil {
		t.Fatalf("forced redispatch of an already-built phase: %v", err)
	}
	manifest := result["dispatch_manifest"].(codexBuildManifest)

	// Assert the precondition rather than assume it: this test is only
	// meaningful while the second attempt genuinely has committed nothing.
	_, record, ok := loadLatestBuildAttempt(1)
	if !ok {
		t.Fatal("no attempt recorded for the forced redispatch")
	}
	if record.CompletionSHA256 != "" || record.Claims != nil {
		t.Fatalf("fixture broken: the redispatched attempt already carries terminal evidence (digest=%q claims=%v), so it is a partial commit, not the fresh attempt this test is about", record.CompletionSHA256, record.Claims != nil)
	}
	planned := 0
	for _, dispatch := range record.Dispatches {
		if strings.TrimSpace(dispatch.Status) == "planned" {
			planned++
		}
	}
	if planned != len(record.Dispatches) || planned == 0 {
		t.Fatalf("fixture broken: %d of %d dispatches are `planned`; the reported state had all of them planned", planned, len(record.Dispatches))
	}

	if _, _, _, _, err := runCodexBuildFinalize(root, 1, codexExternalBuildCompletion{
		DispatchManifest: &manifest,
		Dispatches:       externalResultsForManifest(manifest),
	}, false); err != nil {
		t.Fatalf("deadlock regressed: a forced redispatch of an already-built phase cannot be finalized (%v).\n\nThe colony reads BUILT from the PREVIOUS attempt; this attempt has committed nothing. If finalize also refuses here, `aether continue` refuses too — its dispatches are still `planned` — and the only way out is `aether recover`.", err)
	}
}

// TestFinalizeNeverSendsUserToACommandThatRefuses is the ratchet over the
// attempt-journal / dispatch-status store pair, stated as the property the
// deadlock violated rather than as the routing rule that violated it:
//
//	finalize may only tell someone to run `aether continue` in a state where
//	`aether continue` actually runs.
//
// It sweeps every state this pair can be left in that ends with finalize
// refusing, and for each one where the refusal names a recovery command, runs
// that command's own gate. A refusal that names an exit which itself refuses
// is a deadlock however it was reached, so this catches the next instance
// without needing to predict which of the two commands will be at fault.
//
// Stated as a routing assertion instead, this test would have passed
// throughout the bug it exists for.
func TestFinalizeNeverSendsUserToACommandThatRefuses(t *testing.T) {
	cases := []struct {
		name  string
		setup func(t *testing.T, root string) codexExternalBuildCompletion
	}{
		{
			// The case the refusal was written for: a genuinely committed
			// attempt handed a different packet, e.g. files amended after
			// sign-off. Here `aether continue` is the right advice.
			name: "different packet for a committed attempt",
			setup: func(t *testing.T, root string) codexExternalBuildCompletion {
				t.Helper()
				_, completion := prepareExternalBuildCompletion(t, root)
				if _, _, _, _, err := runCodexBuildFinalize(root, 1, completion, false); err != nil {
					t.Fatalf("first finalize: %v", err)
				}
				changed := completion
				workers := append([]codexExternalBuildWorkerResult(nil), completion.Dispatches...)
				workers[0].Summary = workers[0].Summary + " (amended after review)"
				changed.Dispatches = workers
				return changed
			},
		},
		{
			// The case that produced the deadlock: a fresh attempt on a phase
			// that was already built. It must not refuse at all — but if some
			// future change makes it refuse again, the refusal must not name
			// a command that cannot run.
			name: "forced redispatch of an already-built phase",
			setup: func(t *testing.T, root string) codexExternalBuildCompletion {
				t.Helper()
				_, completion := prepareExternalBuildCompletion(t, root)
				if _, _, _, _, err := runCodexBuildFinalize(root, 1, completion, false); err != nil {
					t.Fatalf("first finalize: %v", err)
				}
				result, _, _, _, err := runCodexBuildPlanOnlyWithOptions(root, 1, nil, codexBuildOptions{Force: true})
				if err != nil {
					t.Fatalf("forced redispatch: %v", err)
				}
				manifest := result["dispatch_manifest"].(codexBuildManifest)
				return codexExternalBuildCompletion{
					DispatchManifest: &manifest,
					Dispatches:       externalResultsForManifest(manifest),
				}
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			root := setupExternalBuildAttemptTest(t)
			completion := tc.setup(t, root)

			_, _, _, _, err := runCodexBuildFinalize(root, 1, completion, false)
			if err == nil {
				return // Finalize succeeded; there is no advice to honour.
			}
			if !strings.Contains(err.Error(), "aether continue") {
				return // Refused without naming continue; not this invariant's business.
			}

			_, record, ok := loadLatestBuildAttempt(1)
			if !ok {
				t.Fatalf("finalize refused with %v but no attempt is recorded to inspect", err)
			}
			if provErr := traceContinueProvenance(record.Dispatches); provErr != nil {
				t.Fatalf("deadlock: finalize refused and told the user to run `aether continue`, which refuses on this same attempt.\n\nfinalize: %v\ncontinue:  %v\n\nNeither exit works and each names the other; the only way out is `aether recover`, which is out of band. Either this state must not reach that refusal, or that refusal must not name continue.", err, provErr)
			}
		})
	}
}
