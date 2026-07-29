package cmd

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/calcosmic/Aether/pkg/colony"
)

// assembledContextTaskShareFloorPercent is a judgement call, not a
// measurement: the minimum share of the TOTAL assembled worker context
// (manifest capsule + brief + skill section) that must be task-relevant
// content (Assignment, Phase Objective, Dependencies, Task/Constraints,
// Hints, Task/Phase Success Criteria). It protects against the failure
// TestBuildWorkerBriefIsMostlyTask was written for -- scaffolding crowding
// out the assignment -- but scaled down from that test's 40% floor because
// this denominator additionally includes the capsule and skill section,
// which are legitimate grounding, not scaffolding, and can reasonably
// dominate the total. If real usage shows this threshold is wrong in either
// direction, that is a finding to raise, not a number to silently retune to
// whatever the fixture currently produces.
const assembledContextTaskShareFloorPercent = 5.0

// buildBudgetTestFixtureRoot creates a colony fixture with a real charter,
// a codebase-graph.json whose targets match the task's hints, and real phase
// research content -- but deliberately no survey artifacts. Survey content
// is emitted under an "### " (not "## ") heading (see resolveSurveySection),
// so splitBriefSections cannot give it its own top-level section boundary;
// it would otherwise bleed into whatever "## " section precedes it,
// inflating that section's measured size past what the section actually
// contains. Omitting survey artifacts keeps the per-section measurements in
// this test honest.
func buildBudgetTestFixtureRoot(t *testing.T) (root string, phase colony.Phase, state colony.ColonyState) {
	t.Helper()
	state = basePrintBriefState()
	root, _ = printBriefFixture(t, state)
	writeCodegraphFixture(t, root)
	writePhaseResearchFixture(t, root)
	return root, state.Plan.Phases[0], state
}

// assembleBudgetTestContext reproduces exactly what printWorkerBriefs
// computes for a single dispatch: the manifest-level capsule (read once,
// display-only), the composed brief, and the per-dispatch skill section --
// the same three pieces "total assembled context" means throughout this
// plan.
func assembleBudgetTestContext(t *testing.T, phase colony.Phase, state colony.ColonyState) (dispatch codexBuildDispatch, brief, capsule string) {
	t.Helper()

	policy := recommendQueenExecutionPolicy(state, phase, len(state.Plan.Phases), codexQueenExecutionPolicyInput{})
	reviewDepth := colony.NormalizeVerificationDepth(policy.VerificationDepth)
	dispatches := plannedBuildDispatchesForSelectionWithState(phase, state, nil, reviewDepth)
	if len(dispatches) == 0 {
		t.Fatal("fixture produced no dispatches")
	}

	// Resolve root via skillWorkspaceRoot(), the same call the CLI path makes
	// -- not the raw t.TempDir() value the fixture setup returned. On macOS
	// t.TempDir() returns an unresolved /var/... path while os.Getwd() after
	// chdir resolves the /private/var symlink, and that resolution
	// difference alone would show up as a several-char drift in the
	// rendered "- Workspace: %s" line, unrelated to anything this test is
	// meant to measure.
	root := skillWorkspaceRoot()
	single := []codexBuildDispatch{dispatches[0]}
	attachBuildDispatchContext(root, phase, single, time.Now())

	capsule = resolveCodexWorkerContext()
	return single[0], single[0].Brief, capsule
}

// TestAssembledContextStaysUnderBudgetCeiling is D-03's guard: it bounds how
// large the total assembled worker context (manifest capsule + brief + skill
// section) can grow relative to the sum of every named budget constant this
// codebase declares, and it bounds each individually budgeted section
// against its own constant. It is a guard against future growth, not a gate
// on today's numbers -- see the SUMMARY for the measured totals this
// fixture produces and the CONTEXT-09/D-08 evidence statement.
func TestAssembledContextStaysUnderBudgetCeiling(t *testing.T) {
	saveGlobals(t)

	_, phase, state := buildBudgetTestFixtureRoot(t)
	dispatch, brief, capsule := assembleBudgetTestContext(t, phase, state)

	skillChars := len(dispatch.SkillSection)
	total := len(capsule) + len(brief) + skillChars
	ceiling := assembledContextBudgetCeilingChars()

	t.Logf("measured: capsule=%d brief=%d skill=%d total=%d ceiling=%d",
		len(capsule), len(brief), skillChars, total, ceiling)

	t.Run("total assembled context stays at or under the derived ceiling", func(t *testing.T) {
		if total > ceiling {
			t.Errorf("assembled context (%d chars) exceeds the derived budget ceiling (%d chars = colonyPrimeBudgetChars(%d) + skillInjectNormalBudgetChars(%d) + phaseResearchBriefBudgetChars(%d) + codegraphWorkerContextBudgetChars(%d) + briefTaskContentAllowanceChars(%d)) -- this is D-03's growth guard tripping, not a number to silently retune; report the measured totals and raise it with the developer",
				total, ceiling, colonyPrimeBudgetChars, skillInjectNormalBudgetChars, phaseResearchBriefBudgetChars, codegraphWorkerContextBudgetChars, briefTaskContentAllowanceChars)
		}
	})

	t.Run("no individually budgeted section exceeds its own declared constant", func(t *testing.T) {
		if len(capsule) > colonyPrimeBudgetChars {
			t.Errorf("capsule is %d chars, exceeds colonyPrimeBudgetChars (%d)", len(capsule), colonyPrimeBudgetChars)
		}
		if skillChars > skillInjectNormalBudgetChars {
			t.Errorf("skill section is %d chars, exceeds skillInjectNormalBudgetChars (%d)", skillChars, skillInjectNormalBudgetChars)
		}

		for _, section := range splitBriefSections(brief) {
			switch section.Name {
			case "Phase Research":
				if section.Chars > phaseResearchBriefBudgetChars {
					t.Errorf("Phase Research section is %d chars, exceeds phaseResearchBriefBudgetChars (%d)", section.Chars, phaseResearchBriefBudgetChars)
				}
			case "Codebase Graph Context":
				if section.Chars > codegraphWorkerContextBudgetChars {
					t.Errorf("Codebase Graph Context section is %d chars, exceeds codegraphWorkerContextBudgetChars (%d)", section.Chars, codegraphWorkerContextBudgetChars)
				}
			}
		}
	})

	t.Run("task-relevant content is at least the named minimum share of the total", func(t *testing.T) {
		taskChars := 0
		for _, section := range splitBriefSections(brief) {
			switch section.Name {
			case "Assignment", "Phase Objective", "Phase Success Criteria",
				"Task Success Criteria", "Dependencies", "Task Constraints",
				"Constraints", "Hints":
				taskChars += section.Chars
			}
		}
		if total == 0 {
			t.Fatal("total assembled context is empty")
		}
		share := float64(taskChars) / float64(total) * 100
		if share < assembledContextTaskShareFloorPercent {
			t.Errorf("task-relevant content is %.1f%% of the total assembled context (%d of %d chars), below the %.1f%% floor",
				share, taskChars, total, assembledContextTaskShareFloorPercent)
		}
	})

	t.Run("the checklist reports the same total and ceiling this test computes", func(t *testing.T) {
		out, errOut := runPrintBriefCmd(t, "--worker", dispatch.Name)
		if errOut != "" {
			t.Fatalf("unexpected stderr: %s", errOut)
		}

		totalLine := lineContaining(out, "TOTAL")
		if totalLine == "" {
			t.Fatalf("checklist output missing a TOTAL line:\n%s", out)
		}
		if !strings.Contains(totalLine, fmt.Sprintf("%d", total)) {
			t.Errorf("checklist TOTAL line %q does not contain the test-computed total %d", totalLine, total)
		}
		if !strings.Contains(totalLine, fmt.Sprintf("%d", ceiling)) {
			t.Errorf("checklist TOTAL line %q does not contain the test-computed ceiling %d", totalLine, ceiling)
		}
	})
}
