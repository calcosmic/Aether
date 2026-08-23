package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/calcosmic/Aether/pkg/colony"
)

// Phase 175 (Orchestration Visibility): the Queen's team rationale is composed
// on every build, carried in the dispatch contract, and was never rendered to
// a human — "why didn't it use the security one?" required opening a JSON
// manifest. These tests pin the render to the manifest data so it cannot
// drift into hardcoded prose, and pin the brevity guard so 27 castes never
// become a 27-row absentee table.

func teamChoicePolicy() codexQueenExecutionPolicy {
	pruned := 3
	return codexQueenExecutionPolicy{
		SpawnBudget: &codexQueenSpawnBudgetContract{
			SelectedReasons: map[string]string{
				"builder": "Score 12 >= threshold 6 for build flow",
				"watcher": "watcher is always required for build flow",
			},
			PrunedReasons: map[string]string{
				"gatekeeper": "no security surface named in this phase",
				"probe":      "documentation-only phase produces no testable code",
				"measurer":   "no performance work in this phase",
				"includer":   "no interface work in this phase",
			},
			PreservedCastes: []string{"auditor"},
			PrunedCastes:    &pruned,
		},
	}
}

func teamChoiceDispatches() []codexBuildDispatch {
	return []codexBuildDispatch{
		{Caste: "builder", Name: "Mason-1", Task: "implement"},
		{Caste: "watcher", Name: "Sentry-2", Task: "verify"},
	}
}

// TestQueenTeamChoiceRendersManifestRationaleVerbatim is the anti-hardcode
// fence: every clause must come from the contract, so removing the rationale
// from the manifest removes it from the render.
func TestQueenTeamChoiceRendersManifestRationaleVerbatim(t *testing.T) {
	policy := teamChoicePolicy()
	rendered := renderQueenTeamChoice(policy, teamChoiceDispatches())

	for _, want := range []string{
		"Score 12 >= threshold 6 for build flow",
		"watcher is always required for build flow",
		"no security surface named in this phase",
	} {
		if !strings.Contains(rendered, want) {
			t.Errorf("team choice missing manifest rationale %q:\n%s", want, rendered)
		}
	}

	// Blank the manifest rationale: the clauses must vanish with it. If this
	// half fails, the renderer has drifted into composing its own prose.
	policy.SpawnBudget.SelectedReasons = nil
	policy.SpawnBudget.PrunedReasons = nil
	policy.SpawnBudget.PreservedCastes = nil
	blanked := renderQueenTeamChoice(policy, teamChoiceDispatches())
	if strings.Contains(blanked, "threshold") || strings.Contains(blanked, "security surface") {
		t.Errorf("rationale text survived removal from the manifest — the render is hardcoded:\n%s", blanked)
	}
}

// TestNotCalledCastesRenderAsOneShortClause: the brevity guard. Castes a
// reader might expect are named (at most three); the rest are a count.
func TestNotCalledCastesRenderAsOneShortClause(t *testing.T) {
	rendered := renderQueenTeamChoice(teamChoicePolicy(), teamChoiceDispatches())

	if !strings.Contains(rendered, "Not called (4):") {
		t.Errorf("not-called castes missing their count clause:\n%s", rendered)
	}
	if !strings.Contains(rendered, "and 1 more") {
		t.Errorf("overflow beyond three named castes must collapse to a count:\n%s", rendered)
	}
	notCalledLines := 0
	for _, line := range strings.Split(rendered, "\n") {
		if strings.Contains(line, "Not called") {
			notCalledLines++
		}
	}
	if notCalledLines != 1 {
		t.Errorf("not-called castes must be ONE clause, got %d lines:\n%s", notCalledLines, rendered)
	}
}

// TestLightBuildOfProductionPhaseShowsSafetyRestorationLine: criterion 2. When
// the runtime keeps a safety caste the depth flag would have dropped, the
// output names the caste and the action instead of silently correcting.
// TestLightContinueOfSecurityPhaseShowsSafetyRestorationLine used to run
// this at "build" with a production/security phase, relying on the old
// build-side floor (watcher+probe+auditor+gatekeeper all required) to
// generate genuine budget pressure at light depth. Plan 194-02 (D-05, D-07)
// removed that floor: build now requires only the builder, which is never at
// risk of being pruned, so a build-flow fixture can no longer demonstrate a
// real "kept by safety policy" restoration. The forced-reviewer floor this
// line exists to announce now lives at the continue step, so the fixture
// moved there -- a security-signal phase with enough OTHER relevance-scored
// candidates to genuinely exceed the light continue budget, so gatekeeper
// (forced) is provably preserved rather than trivially present.
func TestLightContinueOfSecurityPhaseShowsSafetyRestorationLine(t *testing.T) {
	goal := "ship the payment flow"
	state := colony.ColonyState{
		Goal:              &goal,
		VerificationDepth: string(colony.VerificationDepthLight),
	}
	taskID := "t1"
	phase := colony.Phase{
		ID:   1,
		Name: "Password reset by email",
		// D-06 removed auditor's mode==production relevance boost (it was
		// the same implicit "production ⇒ auditor" floor restated as a
		// score), so this fixture's budget pressure can no longer come from
		// that free 4th candidate. "crash" + "resilience" pulls in Chaos on
		// its own keyword merit instead, restoring genuine pressure against
		// light continue's 3-worker cap (watcher, gatekeeper and measurer
		// already fill it) so a real prune -- not just Chaos's absence --
		// is what this test now measures.
		Description: "Let a user reset their password with an emailed token, and store the credential hash. Harden auth token handling, refactor the legacy parser, optimize latency, and add crash resilience handling.",
		Mode:        colony.PhaseModeProduction,
		Tasks:       []colony.Task{{ID: &taskID, Goal: "harden auth token handling, refactor legacy code, benchmark performance"}},
	}

	// The real contract builder, not a fixture: a light continue of a
	// security-signal phase must preserve its forced reviewer.
	policy := enrichQueenExecutionPolicyWithSpawnBudget(codexQueenExecutionPolicy{}, state, phase, "continue", colony.VerificationDepthLight, nil)
	if policy.SpawnBudget == nil || len(policy.SpawnBudget.PreservedCastes) == 0 {
		t.Fatalf("expected a light continue of a security-signal phase to preserve safety castes, got %+v", policy.SpawnBudget)
	}

	rendered := renderQueenTeamChoice(policy, nil)
	if !strings.Contains(rendered, "Kept by safety policy:") {
		t.Fatalf("safety restoration happened but the output does not say so:\n%s", rendered)
	}
	named := false
	for _, caste := range policy.SpawnBudget.PreservedCastes {
		if strings.Contains(rendered, casteLabel(caste)) {
			named = true
		}
	}
	if !named {
		t.Errorf("the restoration line does not name the preserved caste:\n%s", rendered)
	}
}

// TestBuildVisualCarriesQueenTeamChoice: the clause must reach the output the
// operator actually reads, not merely exist as a helper.
func TestBuildVisualCarriesQueenTeamChoice(t *testing.T) {
	goal := "ship it"
	state := colony.ColonyState{
		Goal: &goal,
		Plan: colony.Plan{Phases: []colony.Phase{{ID: 1, Name: "Build", Tasks: []colony.Task{{Goal: "implement"}}}}},
	}
	rendered := renderBuildVisualWithDispatches(state, state.Plan.Phases[0], teamChoiceDispatches(), colony.VerificationDepthStandard, teamChoicePolicy())
	if !strings.Contains(rendered, "Queen's Team") || !strings.Contains(rendered, "Score 12 >= threshold 6") {
		t.Errorf("build output does not carry the Queen's team choice:\n%s", truncateString(rendered, 800))
	}

	planOnly := renderBuildPlanOnlyVisual(state, state.Plan.Phases[0], teamChoiceDispatches(), colony.VerificationDepthStandard, teamChoicePolicy())
	if !strings.Contains(planOnly, "Queen's Team") {
		t.Errorf("plan-only build output does not carry the Queen's team choice:\n%s", truncateString(planOnly, 800))
	}
}

// TestRunSummaryDistinguishesFindingFromCleanWorkers: criterion 3. One worker
// flagged something, one came back clean — two different lines, never two
// identical "completed" rows.
func TestRunSummaryDistinguishesFindingFromCleanWorkers(t *testing.T) {
	dispatches := []codexBuildDispatch{
		{Caste: "watcher", Name: "Sentry-1", Task: "verify", Status: "completed", Blockers: []string{"auth token logged in plaintext"}},
		{Caste: "builder", Name: "Mason-2", Task: "implement", Status: "completed"},
	}
	rendered := renderSpawnPlanForDispatches(dispatches, colony.ModeInRepo)

	if !strings.Contains(rendered, "flagged 1 issue(s)") {
		t.Errorf("a worker that surfaced a finding renders without saying so:\n%s", rendered)
	}
	if !strings.Contains(rendered, "nothing to flag") {
		t.Errorf("a clean worker renders without saying it found nothing:\n%s", rendered)
	}

	var findingLine, cleanLine string
	for _, line := range strings.Split(rendered, "\n") {
		if strings.Contains(line, "Sentry-1") {
			findingLine = line
		}
		if strings.Contains(line, "Mason-2") {
			cleanLine = line
		}
	}
	if findingLine == "" || cleanLine == "" {
		t.Fatalf("could not locate both worker lines:\n%s", rendered)
	}
	if strings.TrimSpace(strings.ReplaceAll(findingLine, "Sentry-1", "")) == strings.TrimSpace(strings.ReplaceAll(cleanLine, "Mason-2", "")) {
		t.Error("the finding worker and the clean worker render as identical lines")
	}
}

// TestStatusHealthLineReflectsComputedVitals: status shows the same numbers
// colony-vital-signs computes — surfaced, not reimplemented.
func TestStatusHealthLineReflectsComputedVitals(t *testing.T) {
	saveGlobals(t)
	s, tmpDir := newTestStore(t)
	t.Cleanup(func() { os.RemoveAll(tmpDir) })
	store = s

	goal := "ship it"
	state := colony.ColonyState{Goal: &goal, Plan: colony.Plan{Phases: []colony.Phase{{ID: 1, Status: "completed"}}}}
	vitals := computeColonyVitalSigns(s, state)

	line := renderColonyHealthLine(vitals)
	if line == "" {
		t.Fatal("status renders no colony health line from computed vitals")
	}
	if !strings.Contains(line, "Colony health:") {
		t.Errorf("health line missing its label: %q", line)
	}
	wantLabel := stringValue(vitals["health_label"])
	if wantLabel == "" || !strings.Contains(line, wantLabel) {
		t.Errorf("health line %q does not carry the computed label %q — the render must surface the computation, not invent one", line, wantLabel)
	}
}

// TestQuietBuildDoesNotAnnounceSafetyInterventions: PreservedCastes is the
// intersection of selected and required, which on an ordinary build always
// contains the Watcher. Rendering "Kept by safety policy" unconditionally
// would announce an intervention on every build — noise, and untrue: when
// nothing was pruned, nothing was protected from anything.
func TestQuietBuildDoesNotAnnounceSafetyInterventions(t *testing.T) {
	policy := codexQueenExecutionPolicy{
		SpawnBudget: &codexQueenSpawnBudgetContract{
			SelectedReasons: map[string]string{
				"builder": "Score 12 >= threshold 6 for build flow",
				"watcher": "watcher is always required for build flow",
			},
			PreservedCastes: []string{"watcher"},
			// No pruning of any kind: nothing was cut, so nothing was kept
			// against a cut.
		},
	}
	rendered := renderQueenTeamChoice(policy, teamChoiceDispatches())
	if strings.Contains(rendered, "Kept by safety policy") {
		t.Errorf("a build where nothing was pruned announces a safety intervention:\n%s", rendered)
	}
	// The per-caste selection clauses still render — quiet about
	// interventions, not silent about the team.
	if !strings.Contains(rendered, "watcher is always required for build flow") {
		t.Errorf("quieting the safety line must not remove the selection clauses:\n%s", rendered)
	}
}
