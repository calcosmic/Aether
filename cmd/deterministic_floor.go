package cmd

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// deterministicFloorResult is the outcome of the program's own free checks --
// build, types, lint, tests, current-build claims, and each success
// criterion's evidence -- computed with no reviewer worker dispatched or
// consulted for a decision. It is the single source of truth both continue
// lanes build their verification report from (ruling D11 rule 2: "a phase
// with zero reviewer workers advances when those checks pass and blocks when
// they fail").
type deterministicFloorResult struct {
	Steps          []codexVerificationStep
	Claims         codexClaimVerification
	Criteria       criterionEvidenceEvaluation
	ExecutedChecks int
	ChecksPassed   bool
	BlockingIssues []string
	Warnings       []string
}

// runDeterministicFloor computes the deterministic floor: the four shell
// checks, current-build claims, and criterion evidence. It never dispatches
// anything and never consults a depth flag, a review policy, or a caste
// proposal -- its only reviewer input is the already-resolved watcher value
// it is handed, used solely as one possible source of evidence for a
// criterion whose evidence_requirements bind a "watcher" check (D-06,
// FLOOR-03). A reviewer's own pass/fail decision, if one is dispatched by the
// caller after this call returns, can only ever ADD a block on top of this
// result -- never flip ChecksPassed from false to true
// (TestDeterministicFloorIsTheOnlySourceOfAPass). This is the single body
// both runCodexContinueVerification (cmd/codex_continue.go, the in-process
// lane) and runCodexContinueVerificationSnapshot (cmd/codex_continue_plan.go,
// the wrapper/external lane) call, so lane parity is structural rather than a
// discipline (TestBothContinueLanesApplyTheSameFloor).
func runDeterministicFloor(ctx context.Context, root string, phase colony.Phase, manifest codexContinueManifest, watcher codexWatcherVerification, verificationTimeout time.Duration) deterministicFloorResult {
	if ctx == nil {
		ctx = context.Background()
	}
	verificationTimeout = effectiveContinueVerificationTimeout(verificationTimeout)
	commands := resolveCodexVerificationCommands(root)
	requiredChecks := requiredVerificationChecks(phase)
	steps := []codexVerificationStep{
		runVerificationStep(ctx, root, "build", requiredChecks["build"], commands.Build, verificationTimeout),
		runVerificationStep(ctx, root, "types", requiredChecks["types"], commands.Type, verificationTimeout),
		runVerificationStep(ctx, root, "lint", requiredChecks["lint"], commands.Lint, verificationTimeout),
		runVerificationStep(ctx, root, "tests", requiredChecks["tests"], commands.Test, verificationTimeout),
	}
	steps = applyExpectedTestFailure(steps, phase)
	claims := verifyCodexBuildClaims(root, manifest)

	shellChecksPassed := true
	executedChecks := 0
	for _, step := range steps {
		if step.Skipped {
			continue
		}
		executedChecks++
		if !step.Passed {
			shellChecksPassed = false
		}
	}

	checksPassed := shellChecksPassed
	blockers := []string{}
	warnings := []string{}
	if executedChecks == 0 && !phaseHasBoundArtifactRequirements(phase) {
		warnings = append(warnings, "no tests to run in this project; verification relies on the files a worker changed and evidence for each success criterion — add real build/test commands to CLAUDE.md to enable shell checks")
	}
	if !shellChecksPassed {
		for _, step := range steps {
			if !step.Passed && !step.Skipped {
				if step.ErrorClass == ErrorClassEnvironment && phase.Mode != colony.PhaseModeProduction {
					warnings = append(warnings, fmt.Sprintf("%s environment issue (not blocking for %s phase): %s", step.Name, phase.Mode, step.Summary))
				} else {
					blockers = append(blockers, fmt.Sprintf("%s failed: %s", step.Name, step.Summary))
				}
			}
		}
		// If all failures were environment warnings, allow checks to pass.
		if len(blockers) == 0 && len(warnings) > 0 {
			checksPassed = true
			for _, w := range warnings {
				fmt.Fprintf(os.Stderr, "⚠ %s\n", w)
			}
		}
	}

	criteria := evaluatePhaseCriterionEvidence(root, phase, manifest, steps, claims, watcher)
	if criteria.Enforced && !criteria.Passed {
		checksPassed = false
		blockers = append(blockers, criteria.BlockingIssues...)
	}

	return deterministicFloorResult{
		Steps:          steps,
		Claims:         claims,
		Criteria:       criteria,
		ExecutedChecks: executedChecks,
		ChecksPassed:   checksPassed,
		BlockingIssues: blockers,
		Warnings:       warnings,
	}
}
