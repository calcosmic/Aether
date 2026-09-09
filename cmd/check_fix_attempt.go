package cmd

import (
	"context"
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/codex"
	"github.com/calcosmic/Aether/pkg/colony"
)

// checkFailureIndexMaxExcerpts and checkFailureIndexMaxExcerptChars bound how
// much of a failing check's raw output this index ever carries. D-02: a
// failed check reaches the fix builder as a compact index, not the whole
// log (spec section 5.1's large-tool-output-externalisation idea, narrowed
// to exactly what this task needs -- the general facility spec section 5.1
// describes is deliberately NOT built here, see buildCheckFailureIndex's doc
// comment).
const (
	checkFailureIndexMaxExcerpts     = 5
	checkFailureIndexMaxExcerptChars = 200
)

// checkFailureIndex is the compact, bounded summary of one failing
// verification step a fix-attempt builder receives instead of the whole
// command log (D-02). It is deliberately small: the check's name and
// command, whether it timed out, a handful of deduplicated failure
// locations, and which task(s) this phase's own claimed changed files tie
// the failure to.
type checkFailureIndex struct {
	Check             string   `json:"check"`
	Command           string   `json:"command,omitempty"`
	ExitCode          int      `json:"exit_code,omitempty"`
	TimedOut          bool     `json:"timed_out,omitempty"`
	Excerpts          []string `json:"excerpts,omitempty"`
	ImplicatedTaskIDs []string `json:"implicated_task_ids,omitempty"`
	Truncated         bool     `json:"truncated,omitempty"`
}

// buildCheckFailureIndex builds the compact failure index for one failing
// verification step. It never carries the whole command output -- only the
// first few distinct failure lines, capped in count and per-line length,
// with Truncated set whenever anything was dropped.
//
// Deliberate scope boundary: this is NOT a general large-tool-output
// externalisation facility (the wider design in the priority spec's section
// 5.1). It builds exactly the index D-02's single bounded fix attempt needs
// -- a general facility, if one is ever built, is later-stage work outside
// this phase (193-CONTEXT.md, Deferred Ideas).
func buildCheckFailureIndex(step codexVerificationStep, claims codexBuildClaims, phase colony.Phase) checkFailureIndex {
	excerpts, truncated := compactFailureExcerpts(step.Output, step.Summary)
	return checkFailureIndex{
		Check:             step.Name,
		Command:           step.Command,
		ExitCode:          step.ExitCode,
		TimedOut:          step.TimedOut,
		Excerpts:          excerpts,
		ImplicatedTaskIDs: implicatedTaskIDsFromExcerpts(excerpts, claims, phase),
		Truncated:         truncated,
	}
}

// compactFailureExcerpts extracts the first few distinct, non-empty lines
// from a failing step's raw output, deduplicated in first-seen order and
// bounded in both count (checkFailureIndexMaxExcerpts) and per-line length
// (checkFailureIndexMaxExcerptChars). When the output produced nothing
// usable (a blocked step with no command output, for example), it falls
// back to the step's own one-line Summary so the index is never empty for a
// genuine failure. Truncated is set whenever excerpts were dropped by count
// or a line was cut short by length -- never when nothing needed dropping.
func compactFailureExcerpts(output, summary string) ([]string, bool) {
	var distinct []string
	seen := map[string]bool{}
	for _, raw := range strings.Split(output, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || seen[line] {
			continue
		}
		seen[line] = true
		distinct = append(distinct, line)
	}

	truncated := len(distinct) > checkFailureIndexMaxExcerpts
	if truncated {
		distinct = distinct[:checkFailureIndexMaxExcerpts]
	}

	excerpts := make([]string, 0, len(distinct))
	for _, line := range distinct {
		if len(line) > checkFailureIndexMaxExcerptChars {
			line = line[:checkFailureIndexMaxExcerptChars] + "…"
			truncated = true
		}
		excerpts = append(excerpts, line)
	}

	if len(excerpts) == 0 {
		if summary := strings.TrimSpace(summary); summary != "" {
			excerpts = []string{summary}
		}
	}
	return excerpts, truncated
}

// failureExcerptPathPattern matches a source-file reference inside a
// failure line -- either a repository-relative path ("cmd/foo.go") or a bare
// filename the way Go's own test runner reports it ("foo_test.go:42:").
var failureExcerptPathPattern = regexp.MustCompile(`[\w./-]+\.(?:go|ts|tsx|js|jsx|py|rs|rb)\b`)

// implicatedTaskIDsFromExcerpts intersects the file references found in a
// failing check's excerpts against each task's reported changed files
// (criterionClaimSets, the same normalization every other claim/criterion
// check already uses), matched either by the full normalized path or by
// filename alone -- Go's own test failure lines report only the bare
// filename, not the repository-relative path. When nothing intersects, it
// returns nil rather than guessing which task caused the failure.
func implicatedTaskIDsFromExcerpts(excerpts []string, claims codexBuildClaims, phase colony.Phase) []string {
	var refs []string
	for _, excerpt := range excerpts {
		refs = append(refs, failureExcerptPathPattern.FindAllString(excerpt, -1)...)
	}
	if len(refs) == 0 {
		return nil
	}

	validTaskIDs := map[string]bool{}
	for idx, task := range phase.Tasks {
		validTaskIDs[buildTaskID(task, idx)] = true
	}

	implicated := map[string]bool{}
	for taskID, claimedPaths := range criterionClaimSets(claims) {
		if taskID == "" {
			continue
		}
		if len(validTaskIDs) > 0 && !validTaskIDs[taskID] {
			continue
		}
		for claimedPath := range claimedPaths {
			base := path.Base(claimedPath)
			for _, ref := range refs {
				if ref == claimedPath || path.Base(ref) == base {
					implicated[taskID] = true
				}
			}
		}
	}
	if len(implicated) == 0 {
		return nil
	}
	ids := make([]string, 0, len(implicated))
	for id := range implicated {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// maxAutomaticCheckFixAttempts is D-02's bound: exactly one automatic
// builder fix attempt per phase per failing check, ever. This is the only
// place that number is spelled out -- planCheckFixAttempt reads it, nothing
// else may reimplement the count.
const maxAutomaticCheckFixAttempts = 1

// planCheckFixAttempt decides whether D-02's single bounded automatic
// builder fix attempt should run, and if so, builds the record describing
// it. It returns false -- no attempt -- when the floor already passed (there
// is nothing to fix), when a reviewer was dispatched (the reviewer already
// owns the diagnosis for this phase, D-02: "never a reviewer to diagnose
// first"), when no shell verification step actually failed (a
// claims/criteria-only failure has no check for a builder to fix here), or
// when a fix attempt for this exact phase and check has already been
// recorded maxAutomaticCheckFixAttempts times (D-02: "never a second
// automatic attempt").
func planCheckFixAttempt(state colony.ColonyState, phase colony.Phase, manifest codexContinueManifest, floor deterministicFloorResult, reviewerDispatched bool) (checkFixAttemptRecord, bool) {
	if floor.ChecksPassed || reviewerDispatched {
		return checkFixAttemptRecord{}, false
	}
	failing := firstFailingVerificationStep(floor.Steps)
	if failing == nil {
		return checkFixAttemptRecord{}, false
	}
	if countCheckFixAttempts(phase.ID, failing.Name) >= maxAutomaticCheckFixAttempts {
		return checkFixAttemptRecord{}, false
	}

	parentAttemptID := ""
	if _, record, ok := loadLatestBuildAttempt(phase.ID); ok {
		parentAttemptID = record.ID
	}
	// WR-01 (193-REVIEW.md): the real manifest for this continue run must be
	// threaded through, not a zero-value codexContinueManifest{} -- a
	// non-default ClaimsPath (the external/wrapper lane's completion
	// packets can set one) would otherwise be silently ignored, and
	// loadRawBuildClaimsForScope would always fall back to the default
	// last-build-claims.json, reading stale or missing claims.
	claims := loadRawBuildClaimsForScope(manifest)
	index := buildCheckFailureIndex(*failing, claims, phase)

	return checkFixAttemptRecord{
		Phase:           phase.ID,
		Check:           failing.Name,
		Reason:          fmt.Sprintf("fixing the failed %s check", failing.Name),
		ParentAttemptID: parentAttemptID,
		FailureIndex:    index,
	}, true
}

// firstFailingVerificationStep returns the first verification step (in the
// fixed build/types/lint/tests order) that actually ran and did not pass --
// skipping steps with no resolved command, which are not eligible for a fix
// attempt (there is no command a builder run could make pass). Returns nil
// when every step that ran, passed.
func firstFailingVerificationStep(steps []codexVerificationStep) *codexVerificationStep {
	for i := range steps {
		if steps[i].Skipped {
			continue
		}
		if !steps[i].Passed {
			return &steps[i]
		}
	}
	return nil
}

// countCheckFixAttempts counts how many recorded build attempts for this
// phase already carry a CheckFix record naming this exact check --
// planCheckFixAttempt's dedup guard against ever sending a second automatic
// attempt for the same phase and check.
func countCheckFixAttempts(phaseNum int, check string) int {
	count := 0
	for _, record := range listBuildAttemptsForPhase(phaseNum) {
		if record.CheckFix != nil && strings.EqualFold(strings.TrimSpace(record.CheckFix.Check), check) {
			count++
		}
	}
	return count
}

// applyAutomaticCheckFixAttempt is the entry point runCodexContinueVerification
// calls after resolving the reviewer decision (D-02/D-03's wiring): plan the
// attempt, and if one is eligible AND a worker provider is actually
// available (the same auto-skip discipline continueWatcherDecision already
// applies for the reviewer path -- there is no point recording an attempt
// that cannot run), dispatch exactly one builder carrying the compact
// failure index, record it as a NEW append-only attempt journal entry, and
// re-run the floor exactly once. Never dispatches a reviewer. Never loops --
// the re-run's result is used as-is, whether it fixed the check or not.
func applyAutomaticCheckFixAttempt(ctx context.Context, root string, state colony.ColonyState, phase colony.Phase, manifest codexContinueManifest, floor deterministicFloorResult, buildWatcher codexWatcherVerification, workerTimeout, verificationTimeout time.Duration, reviewerDispatched bool) (deterministicFloorResult, *checkFixAttemptRecord) {
	record, ok := planCheckFixAttempt(state, phase, manifest, floor, reviewerDispatched)
	if !ok {
		return floor, nil
	}

	invoker := newCodexWorkerInvoker()
	if _, isFake := invoker.(*codex.FakeInvoker); !isFake && !invoker.IsAvailable(ctx) {
		// No worker provider available -- the same auto-skip continue
		// already applies to the reviewer path. Nothing to record: an
		// attempt that never ran is not evidence of anything.
		return floor, nil
	}

	dispatch := plannedCheckFixBuilderDispatch(root, phase, record, invoker, workerTimeout)
	journalDispatch := codexBuildDispatch{
		Stage:  "check-fix",
		Wave:   1,
		Caste:  "builder",
		Name:   dispatch.WorkerName,
		Task:   record.Reason,
		Status: "spawned",
	}
	state, authority, err := resolveCodexBuildPlanAuthority(root, state)
	if err != nil {
		return floor, nil
	}
	startedAt := time.Now().UTC()
	initialRecord := record
	initialRecord.Outcome = "pending"
	request, err := newBuildStartRequest(root, buildStartAutomaticCheckFix, state, authority, phase.ID, record.FailureIndex.ImplicatedTaskIDs, "check-fix-attempt", "check-fix-attempt", startedAt, []codexBuildDispatch{journalDispatch}, buildStartEffects{
		MakeLatest:     true,
		CheckFix:       &initialRecord,
		ReviewerWindow: buildStartReviewerClose,
	})
	if err != nil {
		return floor, nil
	}
	receipt, err := commitBuildStart(root, request, buildStartOptions{})
	if err != nil {
		// D-03 requires attempt and provenance to be durable together. A
		// refused/stale transaction therefore never reaches the worker.
		return floor, nil
	}
	attemptRel := receipt.AttemptPath

	_, _ = dispatchBatchByWaveWithVisuals(ctx, invoker, []codex.WorkerDispatch{dispatch}, colony.ModeInRepo, "Check Fix Attempt", true, nil)

	newFloor := runDeterministicFloor(ctx, root, phase, manifest, buildWatcher, verificationTimeout)
	outcome := "still_failing"
	if newFloor.ChecksPassed {
		outcome = "fixed"
	}
	record.Outcome = outcome
	record.RecordedAt = time.Now().UTC().Format(time.RFC3339)

	journalDispatch.Status = "completed"
	journalDispatch.Summary = fmt.Sprintf("check fix attempt %s", outcome)
	_ = transitionBuildAttempt(attemptRel, buildAttemptTerminal, "check fix attempt: "+outcome, []codexBuildDispatch{journalDispatch}, nil, "check-fix-attempt", nil)
	_ = attachCheckFixAttempt(attemptRel, record)

	return newFloor, &record
}

// plannedCheckFixBuilderDispatch builds the single builder dispatch D-02's
// fix attempt sends, modeled on plannedContinueWatcherDispatch's shape but
// for the "builder" caste and a brief describing the compact failure index
// instead of a verification report.
func plannedCheckFixBuilderDispatch(root string, phase colony.Phase, record checkFixAttemptRecord, invoker codex.WorkerInvoker, workerTimeout time.Duration) codex.WorkerDispatch {
	agentName := codexAgentNameForCaste("builder")
	seed := fmt.Sprintf("phase:%d:check-fix:%s", phase.ID, record.Check)
	return codex.WorkerDispatch{
		ID:             fmt.Sprintf("check-fix-%d-%s", phase.ID, record.Check),
		WorkerName:     deterministicAntName("builder", seed),
		AgentName:      agentName,
		AgentTOMLPath:  dispatchAgentPath(root, invoker, agentName),
		Caste:          "builder",
		TaskID:         fmt.Sprintf("check-fix-%d", phase.ID),
		TaskBrief:      renderCheckFixAttemptBrief(phase, record),
		ContextCapsule: resolveCodexWorkerContext(),
		SkillSection:   resolveSkillSectionForWorkflow("continue", "builder", "Fix a failing verification check"),
		HandoffSection: renderRelatedWorkflowHandoffSection("continue", phase.ID, deterministicAntName("builder", seed)),
		Root:           root,
		Timeout:        effectiveContinueReviewTimeout(workerTimeout),
		Wave:           1,
	}
}

// renderCheckFixAttemptBrief renders the fix builder's task brief from the
// compact failure index -- never the whole command log (D-02).
func renderCheckFixAttemptBrief(phase colony.Phase, record checkFixAttemptRecord) string {
	var b strings.Builder
	b.WriteString("# Fix a Failing Verification Check\n\n")
	fmt.Fprintf(&b, "- Phase: %d — %s\n", phase.ID, phase.Name)
	fmt.Fprintf(&b, "- Failing check: %s\n", record.Check)
	if strings.TrimSpace(record.FailureIndex.Command) != "" {
		fmt.Fprintf(&b, "- Command: %s\n", record.FailureIndex.Command)
	}
	if record.FailureIndex.TimedOut {
		b.WriteString("- This check timed out rather than failing outright.\n")
	}
	b.WriteString("\nThis is a single, bounded, automatic fix attempt (D-02). Nobody else will be asked to diagnose this first, and no second automatic attempt will run if this one does not fix it -- make it count.\n\n")
	if len(record.FailureIndex.ImplicatedTaskIDs) > 0 {
		fmt.Fprintf(&b, "Task(s) whose changed files this failure appears to implicate: %s\n\n", strings.Join(record.FailureIndex.ImplicatedTaskIDs, ", "))
	}
	b.WriteString("Failure excerpts (compact -- not the whole log):\n")
	for _, excerpt := range record.FailureIndex.Excerpts {
		b.WriteString("- ")
		b.WriteString(excerpt)
		b.WriteString("\n")
	}
	if record.FailureIndex.Truncated {
		b.WriteString("\n(Some output was truncated. Re-run the check yourself with the command above for the full picture.)\n")
	}
	return b.String()
}
