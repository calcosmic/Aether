package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// middenCategoryCheckFailed is the midden.json category a failing free check
// (build, types, lint, tests, or a criterion-evidence gap) is filed under --
// distinct from middenCategoryWorkerFailed (cmd/memory_feed.go), which is
// reserved for a worker's own reported failure, not the program's own
// deterministic checks.
const middenCategoryCheckFailed = "check_failed"

// middenCategoryQuickFailed is the midden.json category a failed /ant-quick
// dispatch's record is filed under.
const middenCategoryQuickFailed = "quick_failed"

// middenCategorySwarmWorkerFailed is the midden.json category a failed or
// timed-out swarm worker's record is filed under.
const middenCategorySwarmWorkerFailed = "swarm_worker_failed"

// captureContinueMemory is the replacement entry point for both continue
// call sites that previously called captureContinueLearning directly
// (cmd/codex_continue.go's default lane, cmd/codex_continue_finalize.go's
// wrapper lane). It calls captureContinueLearning first, with the identical
// arguments and unchanged behaviour -- so the existing eligibility gate (all
// workers succeeded AND gates passed AND learning enabled) is untouched --
// and then unconditionally calls feedContinueWorkerMemory with the same
// workerFlow, regardless of whether captureContinueLearning itself was
// eligible to run. A check that blocked still records what it learned and
// what broke.
//
// TestOnlyOneContinueMemoryEntryPoint (cmd/memory_feed_continue_test.go)
// asserts captureContinueLearning is called ONLY from inside this function --
// a future caller that bypasses it fails that guard by name.
func captureContinueMemory(phase colony.Phase, workerFlow []codexContinueWorkerFlowStep, gates codexContinueGateReport, runID string, noLearn bool, now time.Time) {
	captureContinueLearning(phase, workerFlow, gates, runID, noLearn, now)
	feedContinueWorkerMemory(phase, workerFlow)
}

// feedContinueWorkerMemory is the check-side half of the memory feed (Task 1
// of 198.1-02, mirroring feedMemoryFromWorkerOutcome's build-side shape,
// cmd/memory_feed.go). For each completed check worker it captures every
// ReusableLessons entry as a "pattern" observation and every WeakSpots entry
// as a "redirect" observation. For a worker whose status is not "completed"
// it writes one failure record whose message leads with that worker's own
// first blocker sentence (falling back to its summary).
//
// The two synthetic ceremony steps continueHousekeepingFlowStep and
// continueLearningFlowStep (cmd/codex_continue.go) are skipped: they are
// bookkeeping records the runtime itself appends to the flow, never a
// worker's own reported lessons or failures, and feeding them here would
// misfile ceremony text as worker-authored content.
func feedContinueWorkerMemory(phase colony.Phase, workerFlow []codexContinueWorkerFlowStep) {
	for _, step := range workerFlow {
		if isSyntheticContinueCeremonyStep(step) {
			continue
		}
		if step.Status == "completed" {
			feedCompletedCheckWorkerLessons(step)
			continue
		}
		feedFailedCheckWorker(phase, step)
	}
}

// isSyntheticContinueCeremonyStep reports whether a flow step is one of the
// two runtime-appended ceremony records (housekeeping, learning) rather than
// an actual dispatched worker. Identified by Stage, which
// continueHousekeepingFlowStep and continueLearningFlowStep each set to a
// value no dispatched worker step ever carries ("housekeeping", "learning").
func isSyntheticContinueCeremonyStep(step codexContinueWorkerFlowStep) bool {
	switch step.Stage {
	case "housekeeping", "learning":
		return true
	default:
		return false
	}
}

func feedCompletedCheckWorkerLessons(step codexContinueWorkerFlowStep) {
	for _, lesson := range step.ReusableLessons {
		if ok, reason := captureWorkerObservation(lesson, "pattern", observationSourceTypeForOutcome(true), "single_phase"); !ok {
			fmt.Fprintf(os.Stderr, "warning: check worker reusable lesson rejected from memory: %s\n", reason)
		}
	}
	for _, weakSpot := range step.WeakSpots {
		if ok, reason := captureWorkerObservation(weakSpot, "redirect", observationSourceTypeForOutcome(false), "single_phase"); !ok {
			fmt.Fprintf(os.Stderr, "warning: check worker weak spot rejected from memory: %s\n", reason)
		}
	}
}

func feedFailedCheckWorker(phase colony.Phase, step codexContinueWorkerFlowStep) {
	facts := workerOutcomeFacts{
		Workflow:   "check",
		PhaseID:    phase.ID,
		WorkerName: step.Name,
		Caste:      step.Caste,
		Status:     step.Status,
		Summary:    step.Summary,
		Blockers:   append([]string{}, step.Blockers...),
	}
	message := middenMessageForFailedWorker(facts)
	if err := recordWorkerFailureToMidden(middenCategoryWorkerFailed, "aether continue", message, []string{"check", facts.Caste}); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not record check worker failure to memory: %v\n", err)
	}
}

// recordFailedChecksToMidden writes one failure record per genuine blocking
// issue the deterministic floor (cmd/deterministic_floor.go) found -- a
// failing build/type/lint/test step, or a criterion-evidence gap. Called
// unconditionally from the end of runDeterministicFloor -- the single shared
// body both continue lanes (and check_fix_attempt.go's automatic retry, and
// build-finalize's own verify pass) call -- so lane parity is structural
// rather than a discipline.
//
// Entries already downgraded to warnings (an environment issue on a
// non-production phase, see runDeterministicFloor) never reach
// result.BlockingIssues, so no extra filter is needed here: every entry this
// function sees is already a genuine blocker.
func recordFailedChecksToMidden(phase colony.Phase, result deterministicFloorResult) {
	for _, issue := range result.BlockingIssues {
		trimmed := strings.TrimSpace(issue)
		if trimmed == "" {
			continue
		}
		message := fmt.Sprintf("%s — check on phase %d", trimmed, phase.ID)
		if err := recordWorkerFailureToMidden(middenCategoryCheckFailed, "aether continue", message, []string{"check"}); err != nil {
			fmt.Fprintf(os.Stderr, "warning: could not record failed check to memory: %v\n", err)
		}
	}
}

// recordQuickFailureToMidden records a /ant-quick dispatch failure. Only the
// invoker error path (runQuickScout's final return) is recorded: the two
// pre-flight availability returns (missing dispatcher, unavailable scout
// agent) are environment conditions, not colony failures, and are
// deliberately not recorded here.
func recordQuickFailureToMidden(question string, err error) {
	if err == nil {
		return
	}
	errText := strings.TrimSpace(err.Error())
	message := fmt.Sprintf("%s — quick query %q failed", errText, strings.TrimSpace(question))
	if middenErr := recordWorkerFailureToMidden(middenCategoryQuickFailed, "aether quick", message, []string{"quick"}); middenErr != nil {
		fmt.Fprintf(os.Stderr, "warning: could not record quick failure to memory: %v\n", middenErr)
	}
}

// recordSwarmWorkerFailureToMidden records a failed or timed-out swarm
// worker. The message leads with the worker's own summary, then its blocker
// (error) text, before the attribution -- so the worker's own words survive
// the colony-prime capsule's 160-character truncation.
func recordSwarmWorkerFailureToMidden(swarmID, target string, execution swarmWorkerExecution) {
	sentence := firstNonEmptySwarmSentence(execution)
	sanitized, err := colony.SanitizeSignalContent(sentence)
	if err != nil {
		sanitized = "the swarm worker's reported reason could not be safely recorded"
	}
	attribution := fmt.Sprintf("swarm %s, worker %s (%s), target %q, status %s", swarmID, execution.Name, execution.Caste, target, execution.Status)
	message := sanitized + " — " + attribution
	if middenErr := recordWorkerFailureToMidden(middenCategorySwarmWorkerFailed, "aether swarm", message, []string{"swarm", execution.Caste}); middenErr != nil {
		fmt.Fprintf(os.Stderr, "warning: could not record swarm worker failure to memory: %v\n", middenErr)
	}
}

// firstNonEmptySwarmSentence picks a failed swarm worker's own explanation:
// its summary first, else its first blocker (which already carries the
// invoker's error text when the worker never ran), else a literal fallback
// so the message is never empty.
func firstNonEmptySwarmSentence(execution swarmWorkerExecution) string {
	if trimmed := strings.TrimSpace(execution.Summary); trimmed != "" {
		return trimmed
	}
	for _, b := range execution.Blockers {
		if trimmed := strings.TrimSpace(b); trimmed != "" {
			return trimmed
		}
	}
	return "the swarm worker reported no reason"
}
