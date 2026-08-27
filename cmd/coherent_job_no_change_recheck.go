package cmd

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// WR-15 (owner decision, 2026-08-27; see
// .planning/phases/195-coherent-jobs/deferred-items.md).
//
// A "nothing needed changing" receipt is the one shape of finished work that
// leaves no file behind, so there is nothing it wrote that the program can go
// and look at. Until this ruling it was credited when every file its TASK
// declares was present in the checkout -- but for any task that edits existing
// code those files already existed before the build started, so the bar was
// effectively "the file it names is there".
//
// The owner's answer: the worker names the check it ran, and the program runs
// that check again and believes the RESULT, not the claim.
//
// This deliberately reuses Phase 193's already-security-reviewed machinery for
// re-running worker-reported commands (commandSafeToReRun and
// reRunOneBuilderCommand in cmd/criterion_evidence.go) rather than growing a
// second, divergent way to execute worker-authored text. Its rules carry over
// unchanged: the command must name a runner from a fixed allowlist, it must
// contain no shell metacharacters, and it is executed as a plain argument list
// -- never through a shell (Phase 193 code-review finding CR-01).
const (
	// noChangeRecheckCommandTimeout bounds ONE re-run. A check that has not
	// answered by then has produced no result, and a report nothing confirms
	// is not credited.
	noChangeRecheckCommandTimeout = 2 * time.Minute
	// noChangeRecheckTotalBudget bounds ALL re-runs in a single finalize pass,
	// however many receipts say nothing needed changing. Identical commands are
	// run once and their result reused, so the ceiling is per build, not per
	// receipt. A build with no such receipts spends nothing: the runner is only
	// created when the first one is reached.
	noChangeRecheckTotalBudget = 5 * time.Minute
	// noChangeRecheckMaxCommands caps how many of a single receipt's named
	// checks are re-run. A receipt listing a dozen commands does not get to
	// spend the whole budget on its own.
	noChangeRecheckMaxCommands = 3
)

// noChangeRecheckOutcome is what actually happened when the program ran one
// worker-named check itself.
type noChangeRecheckOutcome struct {
	// Passed: the command ran and exited successfully.
	Passed bool
	// Unresolvable: no result was obtained -- the tool is not installed here,
	// or the re-check budget ran out. Never treated as a pass.
	Unresolvable bool
	Summary      string
}

// noChangeRecheckVerdict is the re-check's answer for one receipt: either the
// report was confirmed, or it was not and here is the named rule and the
// plain-English reason to hand back to the owner.
type noChangeRecheckVerdict struct {
	Confirmed bool
	Rule      string
	Message   string
}

// noChangeRecheckRunner re-runs worker-named checks for one finalize pass,
// under one shared wall-clock budget, remembering each command's result so the
// same check is never run twice in a single pass.
type noChangeRecheckRunner struct {
	root   string
	ctx    context.Context
	cancel context.CancelFunc
	seen   map[string]noChangeRecheckOutcome
}

func newNoChangeRecheckRunner(root string) *noChangeRecheckRunner {
	ctx, cancel := context.WithTimeout(context.Background(), noChangeRecheckTotalBudget)
	return &noChangeRecheckRunner{
		root:   root,
		ctx:    ctx,
		cancel: cancel,
		seen:   map[string]noChangeRecheckOutcome{},
	}
}

// close releases the shared budget. Safe on a nil runner, which is the normal
// case for a build where no receipt reported "nothing needed changing".
func (r *noChangeRecheckRunner) close() {
	if r == nil || r.cancel == nil {
		return
	}
	r.cancel()
}

// run executes exactly one already-vetted command and records what happened.
func (r *noChangeRecheckRunner) run(command string) noChangeRecheckOutcome {
	if cached, ok := r.seen[command]; ok {
		return cached
	}
	passed, unresolvable, summary := reRunOneBuilderCommand(r.ctx, r.root, command, noChangeRecheckCommandTimeout)
	if !passed && !unresolvable && r.ctx.Err() != nil {
		// The shared budget expired mid-run, so the command was killed rather
		// than genuinely failing. A killed check has no result; it is reported
		// as unresolvable, which withholds credit without accusing the worker
		// of a failing check.
		unresolvable = true
		summary = fmt.Sprintf("%s did not finish within the time this program allows for re-checking", command)
	}
	outcome := noChangeRecheckOutcome{Passed: passed, Unresolvable: unresolvable, Summary: summary}
	r.seen[command] = outcome
	return outcome
}

// verdict decides whether a "nothing needed changing" report for taskID is
// confirmed by re-running the checks that report named.
//
// Confirmed requires all three: every named check is a shape this program is
// willing to run at all, at least one of them actually ran and passed, and
// none of them ran and failed. Anything else withholds credit -- refusing is
// always the safe direction, because the thing being decided is whether to
// hand out completion credit for work nobody can see.
//
// The safety pass over every command happens BEFORE any of them is executed,
// so a receipt that pairs a legitimate check with an illegitimate one never
// gets the legitimate one run either.
func (r *noChangeRecheckRunner) verdict(taskID string, commands []string) noChangeRecheckVerdict {
	named := make([]string, 0, len(commands))
	seen := make(map[string]struct{}, len(commands))
	for _, command := range commands {
		command = strings.TrimSpace(command)
		if command == "" {
			continue
		}
		if _, dup := seen[command]; dup {
			continue
		}
		seen[command] = struct{}{}
		named = append(named, command)
		if len(named) == noChangeRecheckMaxCommands {
			break
		}
	}

	// Defence in depth. Stage 1 already refuses any receipt whose handoff
	// carries no commands_run entry at all (violationRuleTaskReceiptUnevidenced),
	// so this is only reachable if every entry it carried was blank.
	if len(named) == 0 {
		return noChangeRecheckVerdict{
			Rule:    violationRuleTaskReceiptUnevidenced,
			Message: fmt.Sprintf("task %s reports that nothing needed changing, but names no check the program could run to confirm it; not credited", taskID),
		}
	}

	for _, command := range named {
		if commandSafeToReRun(command) {
			continue
		}
		return noChangeRecheckVerdict{
			Rule: violationRuleTaskReceiptNoChangeCommandRefused,
			Message: fmt.Sprintf(
				"task %s reports that nothing needed changing, but the check it says it ran (%s) is not something this program will run for itself -- it has to be a plain build or test command with no shell tricks in it -- so the report could not be confirmed; not credited",
				taskID, command),
		}
	}

	confirmed := make([]string, 0, len(named))
	unavailable := make([]string, 0, len(named))
	for _, command := range named {
		outcome := r.run(command)
		switch {
		case outcome.Passed:
			confirmed = append(confirmed, command)
		case outcome.Unresolvable:
			unavailable = append(unavailable, command)
		default:
			return noChangeRecheckVerdict{
				Rule: violationRuleTaskReceiptNoChangeCheckFailed,
				Message: fmt.Sprintf(
					"task %s reports that nothing needed changing, but the program ran that check itself (%s) and it did not pass, so the report is not true right now; not credited",
					taskID, command),
			}
		}
	}
	if len(confirmed) == 0 {
		return noChangeRecheckVerdict{
			Rule: violationRuleTaskReceiptNoChangeCheckUnavailable,
			Message: fmt.Sprintf(
				"task %s reports that nothing needed changing, but the program could not get a result from the check it named (%s) here, so nothing confirms the report; not credited",
				taskID, strings.Join(unavailable, ", ")),
		}
	}
	return noChangeRecheckVerdict{Confirmed: true}
}
