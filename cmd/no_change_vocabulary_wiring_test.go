package cmd

import (
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/agent"
	"github.com/calcosmic/Aether/pkg/codex"
)

// The no-change vocabulary was added to the build/finalize path in the
// "honest no-change results" work and left unwired everywhere else that
// judges a worker's status. The effect was that a worker following the
// response contract truthfully -- "the behaviour already existed, here is
// the verification I ran" -- was routed into recovery, blocked phase
// advance, dropped out of the closeout tally, vanished from the status
// dashboard, and was reported at finalization as an unexpected status.
//
// These tests exist because a half-wired vocabulary reads as done at every
// individual call site. They assert the vocabulary as an INVARIANT across
// every decision point, plus a source ratchet that fails when a new
// hand-rolled success list is introduced.

// successJudgement is one place in the runtime that decides whether a worker
// status means the work succeeded.
type successJudgement struct {
	name  string
	isWin func(status string) bool
}

func noChangeSuccessJudgements() []successJudgement {
	return []successJudgement{
		{"isSuccessfulExternalBuildStatus", isSuccessfulExternalBuildStatus},
		{"continue review gate does not block", func(s string) bool { return !continueReviewStatusBlocks(s) }},
		{"ceremonyStepCompleted", ceremonyStepCompleted},
		{"recovery does not redispatch", func(s string) bool { return !externalBuildDispatchNeedsRecovery(s) }},
		{"beats a timeout placeholder", func(s string) bool {
			use, ok := preferCompletedResultOverTimeout("timeout", s)
			return use && ok
		}},
		{"handoff verification reads pass", func(s string) bool { return verificationStatusForWorkerStatus(s) == "pass" }},
		{"renders with the success icon", func(s string) bool { return dispatchStatusIcon(s) == "✓" }},
		{"finalization reports no anomaly", func(s string) bool {
			report := buildExternalBuildResultCollectionReport(1, "phase", nil, nil,
				[]codexBuildDispatch{{Name: "w", Status: s}}, parseManifestGeneratedAt(codexBuildManifest{}), nil)
			return len(report.Issues) == 0
		}},
	}
}

func TestNoChangeSuccessIsHonouredWhereverStatusIsJudged(t *testing.T) {
	for _, judgement := range noChangeSuccessJudgements() {
		t.Run(judgement.name, func(t *testing.T) {
			if !judgement.isWin("completed") {
				t.Fatalf("control failed: %s does not treat completed as a success", judgement.name)
			}
			if judgement.isWin("failed") {
				t.Fatalf("control failed: %s treats failed as a success", judgement.name)
			}
			if !judgement.isWin("completed_no_change") {
				t.Fatalf("%s does not treat completed_no_change as a success -- an honest "+
					"no-change result is a first-class success (ruling D6); use "+
					"isSuccessfulExternalBuildStatus instead of a hand-rolled status list", judgement.name)
			}
		})
	}
}

// recognitionJudgements are places that must recognise a status as a
// FINISHED worker at all. Unlike the success table, "failed" belongs here
// too -- the bug being locked out is a status falling through every branch
// and disappearing, not being scored wrongly.
func noChangeRecognitionJudgements() []successJudgement {
	return []successJudgement{
		{"closeout keeps the worker", func(s string) bool {
			return isVerifiedCloseoutWorkerResult(map[string]interface{}{"name": "w", "status": s})
		}},
		{"spawn tree treats it as finished", func(s string) bool { return agent.IsTerminalSpawnStatus(s) }},
	}
}

func TestFinishedWorkerStatusesAreRecognisedNotDropped(t *testing.T) {
	for _, judgement := range noChangeRecognitionJudgements() {
		t.Run(judgement.name, func(t *testing.T) {
			for _, control := range []string{"completed", "failed"} {
				if !judgement.isWin(control) {
					t.Fatalf("control failed: %s does not recognise %s as finished", judgement.name, control)
				}
			}
			if judgement.isWin("running") {
				t.Fatalf("control failed: %s recognises a live status as finished", judgement.name)
			}
			for _, status := range []string{"completed_no_change", "interrupted"} {
				if !judgement.isWin(status) {
					t.Fatalf("%s does not recognise %s as a finished worker -- it is terminal "+
						"(rulings D6/D7), and a status that matches no branch is silently dropped", judgement.name, status)
				}
			}
		})
	}
}

func TestInterruptedIsTerminalButNeverCountedAsSuccess(t *testing.T) {
	if !isTerminalExternalBuildStatus("interrupted") {
		t.Fatal("interrupted must be terminal: the worker stopped and the record is final (ruling D7)")
	}
	if !agent.IsTerminalSpawnStatus("interrupted") {
		t.Fatal("the spawn tree must treat interrupted as finished, or the worker shows as running forever")
	}
	if isSuccessfulExternalBuildStatus("interrupted") {
		t.Fatal("interrupted must not count as success: the work is unfinished, and completing its tasks would credit work nobody did")
	}
	if !continueReviewStatusBlocks("interrupted") {
		t.Fatal("an interrupted review must still block advancement -- it never finished")
	}
	if got := verificationStatusForWorkerStatus("interrupted"); got != "not_run" {
		t.Fatalf("verification status for interrupted = %q, want not_run: the worker stopped before verifying, which is not a failure", got)
	}
	report := buildExternalBuildResultCollectionReport(1, "phase", nil, nil,
		[]codexBuildDispatch{{Name: "w", Status: "interrupted"}}, parseManifestGeneratedAt(codexBuildManifest{}), nil)
	if len(report.Issues) != 1 || report.Issues[0].Kind != "worker_interrupted" {
		t.Fatalf("finalization must name an interruption as such, got %+v", report.Issues)
	}
}

func TestContinueWatcherPassesOnAnHonestNoChange(t *testing.T) {
	manifest := codexContinueManifest{
		Present: true,
		Data: codexBuildManifest{
			Dispatches: []codexBuildDispatch{{
				Name:    "Sentry-01",
				Stage:   "verification",
				Caste:   "watcher",
				Status:  "completed_no_change",
				Summary: "verified the behaviour already holds; ran the suite",
			}},
		},
	}
	watcher := evaluateContinueWatcherVerification(manifest)
	if !watcher.Present {
		t.Fatal("watcher verification was not found in the manifest")
	}
	if !watcher.Passed {
		t.Fatalf("a watcher that honestly reported completed_no_change must pass verification, got status %q summary %q", watcher.Status, watcher.Summary)
	}
}

// TestNoHandRolledWorkerSuccessLists is the ratchet. Every previous site that
// judged worker success by naming statuses inline drifted the moment the
// vocabulary grew. A new one must not appear.
func TestNoHandRolledWorkerSuccessLists(t *testing.T) {
	// A success list is hand-rolled when it names both "completed" and
	// "manually-reconciled" (the pair that always travelled together) without
	// the no-change status.
	inline := regexp.MustCompile(`"completed"\s*(,|\|\|[^\n]*==)\s*"manually-reconciled"|"manually-reconciled"\s*(,|\|\|[^\n]*==)\s*"completed"`)

	var offenders []string
	roots := []string{".", filepath.Join("..", "pkg")}
	for _, root := range roots {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
			if err != nil || info.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return nil
			}
			for i, line := range strings.Split(string(data), "\n") {
				if !inline.MatchString(line) {
					continue
				}
				if strings.Contains(line, "completed_no_change") {
					continue
				}
				offenders = append(offenders, filepath.ToSlash(path)+":"+strconv.Itoa(i+1)+": "+strings.TrimSpace(line))
			}
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
	if len(offenders) > 0 {
		t.Fatalf("hand-rolled worker success list(s) found -- call isSuccessfulExternalBuildStatus so the "+
			"vocabulary stays in one place:\n  %s", strings.Join(offenders, "\n  "))
	}
}

// The evidence rule is what keeps completed_no_change from becoming a free
// pass. It was enforced only on the external/wrapper lane; the in-process
// runtime lane granted the same exemption from file evidence with nothing
// checked at all.
func TestRuntimeLaneDemandsNoChangeEvidence(t *testing.T) {
	evidenced := codex.WorkerResult{
		WorkerName: "Mason-01",
		Status:     "completed_no_change",
		Summary:    "the retry already backs off; verified rather than changed",
		Handoff: codex.WorkerHandoff{
			VerificationStatus: "pass",
			CommandsRun:        []string{"go test ./pkg/retry"},
		},
	}
	unevidenced := codex.WorkerResult{
		WorkerName: "Mason-02",
		Status:     "completed_no_change",
	}

	if err := validateRuntimeNoChangeEvidence([]codex.DispatchResult{
		{WorkerName: "Mason-01", Status: "completed_no_change", WorkerResult: &evidenced},
	}); err != nil {
		t.Fatalf("an evidenced no-change must be accepted on the runtime lane: %v", err)
	}

	err := validateRuntimeNoChangeEvidence([]codex.DispatchResult{
		{WorkerName: "Mason-02", Status: "completed_no_change", WorkerResult: &unevidenced},
	})
	if err == nil {
		t.Fatal("an evidence-free completed_no_change must be rejected on the runtime lane -- " +
			"it buys an exemption from the file-changes requirement, which is the phantom-build loophole")
	}
	for _, want := range []string{"summary", "verification_status", "commands_run"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("rejection must name the missing evidence %q, got: %v", want, err)
		}
	}

	if err := validateRuntimeNoChangeEvidence([]codex.DispatchResult{
		{WorkerName: "Mason-03", Status: "completed_no_change", WorkerResult: nil},
	}); err == nil {
		t.Fatal("a no-change claim with no result payload must be rejected")
	}
}

// A validator nothing calls is the failure mode this repo keeps
// rediscovering, so the evidence gate is asserted at its call site too.
func TestRuntimeNoChangeEvidenceGateIsWiredIntoDispatch(t *testing.T) {
	source, err := os.ReadFile("codex_build.go")
	if err != nil {
		t.Fatalf("read codex_build.go: %v", err)
	}
	const marker = "func executeCodexBuildDispatches("
	start := strings.Index(string(source), marker)
	if start < 0 {
		t.Fatalf("executeCodexBuildDispatches not found -- if it was renamed, retarget this wiring assertion")
	}
	body := string(source)[start:]
	if next := strings.Index(body[len(marker):], "\nfunc "); next >= 0 {
		body = body[:len(marker)+next]
	}
	if !strings.Contains(body, "validateRuntimeNoChangeEvidence(") {
		t.Fatal("executeCodexBuildDispatches does not call validateRuntimeNoChangeEvidence -- " +
			"without it the in-process lane grants the no-change exemption from file evidence with nothing checked")
	}
	if !strings.Contains(body, "validateRuntimeBuildDispatchResults(") {
		t.Fatal("executeCodexBuildDispatches no longer validates its dispatch results")
	}
}

// A no-change worker may cover other dispatches' tasks via covered_task_ids.
// The covered dispatch used to be stamped a plain "completed" with no
// outputs, and continue provenance then halted it as a suspected phantom
// build -- the honest result punished one hop downstream.
func TestCoveredDispatchInheritsTheClaimantsOutcome(t *testing.T) {
	manifest := codexBuildManifest{
		PlanOnly: true,
		Dispatches: []codexBuildDispatch{
			{Name: "Mason-67", Caste: "builder", Stage: "wave", TaskID: "1.1"},
			{Name: "Mason-68", Caste: "builder", Stage: "wave", TaskID: "1.2"},
		},
	}
	claimant := noChangeResultFixture()
	claimant.CoveredTaskIDs = []string{"1.1", "1.2"}

	dispatches, violations, err := mergeExternalBuildResults(manifest, []codexExternalBuildWorkerResult{claimant})
	if err != nil {
		t.Fatalf("mergeExternalBuildResults: %v", err)
	}
	if len(violations) != 0 {
		t.Fatalf("an evidenced no-change claimant covering another task must merge cleanly, got %+v", violations)
	}
	covered := dispatches[1]
	if covered.Status != "completed_no_change" {
		t.Fatalf("covered dispatch status = %q, want completed_no_change inherited from its claimant", covered.Status)
	}
	if covered.Disposition != "verified_existing" {
		t.Fatalf("covered dispatch disposition = %q, want verified_existing", covered.Disposition)
	}
	if err := traceContinueProvenance(dispatches); err != nil {
		t.Fatalf("a covered no-change dispatch must survive continue provenance: %v", err)
	}
}

// A manually reconciled dispatch has no worker outputs by definition.
// Counting it as a success while still demanding outputs halted the manual
// recovery path with a phantom-build accusation.
func TestManuallyReconciledDispatchDoesNotReadAsAPhantomBuild(t *testing.T) {
	if err := traceContinueProvenance([]codexBuildDispatch{
		{Name: "Mason-67", Status: "completed", Outputs: []string{"cmd/x.go"}},
		{Name: "Mason-68", Status: "manually-reconciled"},
	}); err != nil {
		t.Fatalf("a manually reconciled dispatch must not be accused of a phantom build: %v", err)
	}
	if err := traceContinueProvenance([]codexBuildDispatch{
		{Name: "Mason-68", Status: "manually-reconciled"},
	}); err != nil {
		t.Fatalf("a build reconciled entirely by hand must still trace: %v", err)
	}
	// The guard keeps its teeth for an ordinary completion.
	if err := traceContinueProvenance([]codexBuildDispatch{
		{Name: "Mason-67", Status: "completed"},
	}); err == nil {
		t.Fatal("a plain completed dispatch with no outputs must still be rejected")
	}
}
